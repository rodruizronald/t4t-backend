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
    INPUT_DIR / "prompts/job_description.md"
)  # File containing the prompt template

# Define global output directory path
OUTPUT_DIR = Path("data/output")
timestamp = datetime.now().strftime("%Y%m%d")
PIPELINE_INPUT_DIR = OUTPUT_DIR / timestamp / "pipeline_stage_2"
PIPELINE_OUTPUT_DIR = OUTPUT_DIR / timestamp / "pipeline_stage_3"
PIPELINE_OUTPUT_DIR.mkdir(exist_ok=True, parents=True)

INPUT_FILE = "jobs_stage_2.json"  # JSON file with job eligibility data
OUTPUT_FILE = "jobs_stage_3.json"  # JSON file with job descriptions

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
    """Process a single job URL to extract job description."""
    logger.info(f"Processing job at {job_url} for {company_name}...")

    # Extract HTML content
    html_content = await extract_html_content(job_url, selectors)
    if not html_content:
        logger.warning(f"Could not fetch content for job at {job_url}")
        return {
            "description": None,
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
                    "content": "You extract job descriptions from HTML content.",
                },
                {"role": "user", "content": filled_prompt},
            ],
            response_format={"type": "json_object"},
        )

        # Parse response
        response_text = response.choices[0].message.content
        description_data = json.loads(response_text)

        # Add metadata
        result = {
            "description": description_data.get("description"),
            "timestamp": datetime.now().isoformat(),
        }

        logger.success(f"Successfully processed job at {job_url}")
        return result

    except Exception as e:
        logger.error(f"Error processing job at {job_url} with OpenAI: {str(e)}")
        return {
            "description": None,
            "error": str(e),
        }


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
    jobs_with_descriptions = 0

    for job in data.get("jobs", []):
        job_url = job.get("application_url", "")
        job_title = job.get("title", "")
        company_name = job.get("company", "")
        job_description_selector = job.get("job_description_selector", [])

        # Only process eligible jobs
        if not job.get("eligible", False):
            logger.debug(f"Skipping ineligible job: {job_title}")
            processed_jobs.append(job)
            continue

        if not job_url:
            logger.warning(f"Job missing URL, skipping: {job_title}")
            processed_jobs.append(job)
            continue

        logger.info(f"Processing new eligible job: {job_title} at {job_url}")
        result = await process_job(job_url, job_description_selector, company_name)

        total_jobs_processed += 1

        # Check if there was an error
        if "error" in result:
            logger.error(f"Error processing job {job_title}: {result['error']}")
            # Add job without description
            processed_jobs.append(job)
            continue

        if result and result["description"]:
            jobs_with_descriptions += 1
            # Add description to job data
            job["description"] = result["description"]
            logger.info(f"Added description to job: {job_title}")
        else:
            logger.warning(f"Failed to extract description for job: {job_title}")

        # Add job to processed jobs list (whether description was extracted or not)
        processed_jobs.append(job)

        # Delay to avoid rate limiting
        await asyncio.sleep(1)

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
    logger.info(f"Jobs with descriptions: {jobs_with_descriptions}")


if __name__ == "__main__":
    # Check for API key
    if not OPENAI_API_KEY:
        logger.error("OPENAI_API_KEY environment variable is not set")
        exit(1)

    logger.info("Starting job description extraction process")
    # Run the async main function
    asyncio.run(main())
    logger.info("Process completed")
