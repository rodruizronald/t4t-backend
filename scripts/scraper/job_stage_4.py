import asyncio
import json
import os
import sys
from datetime import datetime
from pathlib import Path

from dotenv import load_dotenv
import openai
from loguru import logger
from playwright.async_api import async_playwright

# Get the root directory
root_dir = Path(__file__).parent.parent.parent

# Load environment variables from .env file
load_dotenv(root_dir / ".env")

# Configuration
OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY")
MODEL = "o4-mini"  # OpenAI model to use

# Define input directory path and input file name
INPUT_DIR = Path("data/input")
PROMPT_FILE = (
    INPUT_DIR / "prompts/job_technologies.md"
)  # File containing the prompt template

# Define global output directory path
OUTPUT_DIR = Path("data/output")
timestamp = datetime.now().strftime("%Y%m%d")
PIPELINE_INPUT_DIR = OUTPUT_DIR / timestamp / "pipeline_stage_3"
PIPELINE_OUTPUT_DIR = OUTPUT_DIR / timestamp / "pipeline_stage_4"
PIPELINE_OUTPUT_DIR.mkdir(exist_ok=True, parents=True)

INPUT_FILE = "jobs_stage_3.json"  # JSON file with job descriptions
OUTPUT_FILE = "jobs_stage_4.json"  # JSON file with job technologies

# Configure logger
LOG_LEVEL = "DEBUG"  # Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
logger.remove()  # Remove default handler
logger.add(sys.stderr, level=LOG_LEVEL)  # Add stderr handler with desired log level
logger.add(
    f"{PIPELINE_OUTPUT_DIR}/logs.log",
    rotation="10 MB",
    level=LOG_LEVEL,
)  # Add file handler

# Initialize OpenAI client
client = openai.OpenAI(api_key=OPENAI_API_KEY)


async def extract_html_content(url: str, selectors: list[str] = None) -> str:
    """
    Use Playwright to fetch HTML content from multiple selectors on a webpage.

    Args:
        url: The URL to fetch content from
        selectors: List of CSS selectors to extract content from (optional)

    Returns:
        String containing concatenated HTML content from all selectors,
        or full page HTML if no selectors provided
    """
    try:
        async with async_playwright() as p:
            browser = await p.chromium.launch()
            page = await browser.new_page()

            # Navigate to the URL
            logger.info(f"Navigating to {url}")
            await page.goto(url, wait_until="networkidle", timeout=60000)

            # Wait a bit for any dynamic content to load
            await page.wait_for_timeout(3000)

            if selectors:
                contents = []
                for selector in selectors:
                    try:
                        # Wait for the specific element to be available
                        element = await page.wait_for_selector(selector, timeout=5000)
                        if element:
                            # Get the HTML content of this element
                            content = await element.inner_html()
                            contents.append(content)
                            logger.info(
                                f"Successfully extracted content from selector: {selector}"
                            )
                        else:
                            logger.warning(f"Selector not found: {selector}")
                    except Exception as e:
                        logger.error(
                            f"Error extracting content from selector {selector}: {str(e)}"
                        )

                # Concatenate all contents with a newline between them
                content = "\n".join(contents) if contents else None
            else:
                # Get the full page content if no selectors specified
                content = await page.content()

            await browser.close()
            return content
    except Exception as e:
        logger.error(f"Error fetching {url}: {str(e)}")
        return None


def read_prompt_template():
    """Read the prompt template from a file."""
    try:
        with open(PROMPT_FILE, "r") as f:
            return f.read()
    except FileNotFoundError:
        logger.error(f"Error: Prompt template file '{PROMPT_FILE}' not found.")
        exit(1)
    except Exception as e:
        logger.error(f"Error reading prompt template: {str(e)}")
        exit(1)


async def process_job(job_url: str, selectors: list[str], company_name: str):
    """Process a single job URL to extract technologies."""
    logger.info(f"Processing job at {job_url} for {company_name}...")

    # Extract HTML content
    html_content = await extract_html_content(job_url, selectors)
    if not html_content:
        logger.warning(f"Could not fetch content for job at {job_url}")
        return {
            "technologies": [],
            "error": "Failed to fetch HTML content",
        }

    # Read prompt template and fill it with HTML content
    prompt_template = read_prompt_template()
    # Escape any curly braces in the HTML content
    escaped_html = html_content.replace("{", "{{").replace("}", "}}")
    filled_prompt = prompt_template.replace("{html_content}", escaped_html)

    # Send to OpenAI
    try:
        logger.info(f"Sending content to OpenAI for job at {job_url}...")
        response = client.chat.completions.create(
            model=MODEL,
            messages=[
                {
                    "role": "system",
                    "content": "You extract technologies from job postings.",
                },
                {"role": "user", "content": filled_prompt},
            ],
            response_format={"type": "json_object"},
        )

        # Parse response
        response_text = response.choices[0].message.content
        tech_data = json.loads(response_text)

        # Add metadata
        result = {
            "technologies": tech_data.get("technologies", []),
            "timestamp": datetime.now().isoformat(),
        }

        logger.success(f"Successfully processed job at {job_url}")
        return result

    except Exception as e:
        logger.error(f"Error processing job at {job_url} with OpenAI: {str(e)}")
        return {
            "technologies": [],
            "error": str(e),
        }


def manage_past_jobs_signatures(combined_signatures: set) -> None:
    """
    Manage historical jobs signatures by combining with previous day's data and detecting duplicates.

    Args:
        combined_signatures: Set of current signatures to process
    """
    from datetime import datetime, timedelta

    # Get current and previous day timestamps
    current_date = datetime.now()
    previous_date = current_date - timedelta(days=1)

    current_timestamp = current_date.strftime("%Y%m%d")
    previous_timestamp = previous_date.strftime("%Y%m%d")

    # Define paths
    previous_day_dir = OUTPUT_DIR / previous_timestamp / "pipeline_stage_4"
    previous_historical_jobs_file = previous_day_dir / "historical_jobs.json"

    current_day_dir = OUTPUT_DIR / current_timestamp / "pipeline_stage_4"
    current_historical_jobs_file = current_day_dir / "historical_jobs.json"
    duplicates_file = current_day_dir / "duplicated_signatures.json"

    # Load previous day's signatures if they exist
    previous_signatures = set()
    if previous_historical_jobs_file.exists():
        try:
            with open(previous_historical_jobs_file, "r") as f:
                previous_data = json.load(f)
                previous_signatures = set(previous_data.get("signatures", []))
            logger.info(
                f"Loaded {len(previous_signatures)} signatures from previous day: {previous_historical_jobs_file}"
            )
        except Exception as e:
            logger.error(f"Error loading previous day's signatures: {str(e)}")
    else:
        logger.info(
            f"No previous day's historical_jobs.json found at {previous_historical_jobs_file}"
        )

    # Detect duplicates between current and previous signatures
    duplicated_signatures = combined_signatures.intersection(previous_signatures)
    unique_current_signatures = combined_signatures - duplicated_signatures

    # Combine all unique signatures (previous + unique current)
    all_unique_signatures = previous_signatures.union(unique_current_signatures)

    # Log duplicate detection results
    if duplicated_signatures:
        logger.warning(f"Found {len(duplicated_signatures)} duplicate signatures")

        # Save duplicated signatures to file
        try:
            with open(duplicates_file, "w") as f:
                json.dump(
                    {
                        "duplicated_signatures": list(duplicated_signatures),
                        "count": len(duplicated_signatures),
                        "timestamp": current_date.isoformat(),
                    },
                    f,
                    indent=2,
                )
            logger.info(f"Duplicated signatures saved to {duplicates_file}")
        except Exception as e:
            logger.error(f"Error saving duplicated signatures: {str(e)}")
    else:
        logger.info("No duplicate signatures found")

    # Save all unique signatures to current day's historical_jobs.json
    try:
        with open(current_historical_jobs_file, "w") as f:
            json.dump(
                {
                    "signatures": list(all_unique_signatures),
                    "count": len(all_unique_signatures),
                    "previous_day_count": len(previous_signatures),
                    "new_unique_count": len(unique_current_signatures),
                    "duplicates_count": len(duplicated_signatures),
                    "timestamp": current_date.isoformat(),
                },
                f,
                indent=2,
            )

        logger.info(f"Historical jobs signatures saved to {current_historical_jobs_file}")
        logger.info(f"Total unique signatures: {len(all_unique_signatures)}")
        logger.info(f"Previous day signatures: {len(previous_signatures)}")
        logger.info(f"New unique signatures: {len(unique_current_signatures)}")
        logger.info(f"Duplicate signatures: {len(duplicated_signatures)}")

    except Exception as e:
        logger.error(f"Error saving historical jobs signatures: {str(e)}")


async def main():
    """Main function to process all jobs."""
    # Check if prompt template file exists
    if not Path(PROMPT_FILE).exists():
        logger.error(f"Prompt template file '{PROMPT_FILE}' not found")
        return

    # Check if input directory exists
    input_file_path = PIPELINE_INPUT_DIR / INPUT_FILE
    if not input_file_path.exists():
        logger.error(f"Input file {input_file_path} does not exist")
        return

    # Read input file with jobs data
    try:
        with open(input_file_path, "r") as f:
            data = json.load(f)
    except Exception as e:
        logger.error(f"Error reading input file: {str(e)}")
        return

    # Process each job
    processed_jobs = []
    total_jobs_processed = 0
    jobs_with_technologies = 0
    processed_signatures = set()

    for job in data.get("jobs", []):
        job_url = job.get("application_url", "")
        job_title = job.get("title", "")
        company_name = job.get("company", "")
        job_signature = job.get("signature", "")
        job_description_selector = job.get("job_description_selector", [])

        if not job_url:
            logger.warning(f"Job missing URL, skipping: {job_title}")
            # Create a clean job object without the excluded fields
            clean_job = {
                k: v
                for k, v in job.items()
                if k not in ["job_description_selector", "eligible"]
            }
            processed_jobs.append(clean_job)
            continue

        logger.info(f"Processing new job: {job_title} at {job_url}")
        result = await process_job(job_url, job_description_selector, company_name)

        total_jobs_processed += 1

        # Check if there was an error
        if "error" in result:
            logger.error(f"Error processing job {job_title}: {result['error']}")
            # Add empty technologies array if extraction failed
            job["technologies"] = []
        else:
            if result and result["technologies"]:
                jobs_with_technologies += 1
                # Add technologies to job data
                job["technologies"] = result["technologies"]
                logger.info(
                    f"Added {len(result['technologies'])} technologies to job: {job_title}"
                )
            else:
                # Add empty technologies array if extraction failed
                job["technologies"] = []
                logger.warning(f"Failed to extract technologies for job: {job_title}")

        # Add signature to processed signatures
        if job_signature:
            processed_signatures.add(job_signature)

        # Create a clean job object without the excluded fields
        clean_job = {
            k: v
            for k, v in job.items()
            if k not in ["job_description_selector", "eligible"]
        }

        # Add job to final jobs list
        processed_jobs.append(clean_job)

        # Delay to avoid rate limiting
        await asyncio.sleep(1)

    # Manage past jobs signatures with duplicate detection
    manage_past_jobs_signatures(processed_signatures)

    # Save results
    output_file = PIPELINE_OUTPUT_DIR / OUTPUT_FILE
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(
            {"jobs": processed_jobs},
            f,
            indent=2,
            ensure_ascii=False,  # This will prevent Unicode escaping
        )

    logger.info(f"Processing complete. Results saved to {output_file}")
    logger.info(f"Processed {total_jobs_processed} jobs")
    logger.info(f"Jobs with technologies: {jobs_with_technologies}")


if __name__ == "__main__":
    # Check for API key
    if not OPENAI_API_KEY:
        logger.error("OPENAI_API_KEY environment variable is not set")
        exit(1)

    logger.info("Starting job technologies extraction process")
    # Run the async main function
    asyncio.run(main())
    logger.info("Process completed")
