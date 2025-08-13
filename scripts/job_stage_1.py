import asyncio
import json
import os
import sys
from datetime import datetime
from pathlib import Path
import hashlib

from dotenv import load_dotenv
import openai
from loguru import logger
from playwright.async_api import async_playwright

# Get the root directory
root_dir = Path(__file__).parent.parent

# Load environment variables from .env file
load_dotenv(root_dir / ".env")

# Configuration
OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY")
MODEL = "gpt-5-mini"  # OpenAI model to use

# Define input directory path and input file name
INPUT_DIR = Path("input")
COMPANIES_FILE = INPUT_DIR / "companies.json"  # JSON file with company career URLs
PROMPT_FILE = (
    INPUT_DIR / "prompts/job_title_url_parser.md"
)  # File containing the prompt template

# Define global output directory path
OUTPUT_DIR = Path("data")
timestamp = datetime.now().strftime("%Y%m%d")
PIPELINE_OUTPUT_DIR = OUTPUT_DIR / timestamp / "pipeline_stage_1"
PIPELINE_OUTPUT_DIR.mkdir(exist_ok=True, parents=True)

JOBS_FILE = "jobs_stage_1.json"  # JSON file with job details (just the filename)

# Configure logger
LOG_LEVEL = "DEBUG"  # Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
logger.remove()  # Remove default handler
logger.add(sys.stderr, level=LOG_LEVEL)  # Add stderr handler with desired log level
logger.add(
    f"{PIPELINE_OUTPUT_DIR}/logs.log", rotation="10 MB", level=LOG_LEVEL
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


async def process_company(
    company_name: str, career_url: str, selectors: list[str] = None
):
    """Process a single company's career page."""
    logger.info(f"Processing {company_name}...")

    # Extract HTML content
    html_content = await extract_html_content(career_url, selectors)
    if not html_content:
        logger.warning(f"Could not fetch content for {company_name}")
        return {
            "jobs": [],
            "error": "Failed to fetch HTML content",
        }

    # Read prompt template and fill it with HTML content
    prompt_template = read_prompt_template()
    # Escape any curly braces in the HTML content
    escaped_html = html_content.replace("{", "{{").replace("}", "}}")
    filled_prompt = prompt_template.replace("{html_content}", escaped_html).replace(
        "{career_url}", career_url
    )

    # Send to OpenAI
    try:
        logger.info(f"Sending content to OpenAI for {company_name}...")
        response = client.chat.completions.create(
            model=MODEL,
            messages=[
                {
                    "role": "system",
                    "content": "You extract job href links from HTML content.",
                },
                {"role": "user", "content": filled_prompt},
            ],
            response_format={"type": "json_object"},
        )

        # Parse response
        response_text = response.choices[0].message.content
        job_data = json.loads(response_text)

        # Add company metadata
        result = {
            "jobs": job_data.get("jobs", []),
            "timestamp": datetime.now().isoformat(),
        }

        logger.success(f"Found {len(result['jobs'])} job links for {company_name}")
        return result

    except Exception as e:
        logger.error(f"Error processing {company_name} with OpenAI: {str(e)}")
        return {
            "jobs": [],
            "error": str(e),
        }


def generate_job_signature(url: str) -> str:
    """
    Generate a unique hash signature for a job URL.

    Args:
        url: The job URL to hash

    Returns:
        A string containing the hexadecimal hash signature
    """
    if not url:
        return ""

    # Create a hash of the URL
    return hashlib.sha256(url.encode()).hexdigest()


def filter_new_jobs(companies_jobs: dict) -> dict:
    """
    Filter out jobs that were processed the previous day based on signatures.

    Args:
        companies_jobs: Dictionary containing companies and their jobs

    Returns:
        Dictionary with filtered companies_jobs containing only new jobs
    """
    from datetime import datetime, timedelta

    # Get previous day timestamp
    previous_date = datetime.now() - timedelta(days=1)
    previous_timestamp = previous_date.strftime("%Y%m%d")

    # Define path to previous day's historical_jobs.json
    previous_day_dir = OUTPUT_DIR / previous_timestamp / "pipeline_stage_4"
    previous_historical_jobs_file = previous_day_dir / "historical_jobs.json"

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
            logger.info("Proceeding without filtering")
            return companies_jobs
    else:
        logger.info(
            f"No previous day's historical_jobs.json found at {previous_historical_jobs_file}"
        )
        logger.info("Proceeding without filtering - all jobs will be processed")
        return companies_jobs

    # If no previous signatures found, return original data
    if not previous_signatures:
        logger.info("No previous signatures to filter against")
        return companies_jobs

    # Filter jobs for each company
    filtered_companies_jobs = {"companies": []}
    total_original_jobs = 0
    total_filtered_jobs = 0
    total_duplicate_jobs = 0

    for company_data in companies_jobs.get("companies", []):
        company_name = company_data.get("company", "")
        job_description_selector = company_data.get("job_description_selector", [])
        original_jobs = company_data.get("jobs", [])

        # Filter out jobs with signatures that exist in previous day's data
        filtered_jobs = []
        duplicate_count = 0

        for job in original_jobs:
            job_signature = job.get("signature", "")
            if job_signature and job_signature in previous_signatures:
                duplicate_count += 1
                logger.debug(
                    f"Filtering out duplicate job: {job.get('title', 'Unknown')} (signature: {job_signature[:8]}...)"
                )
            else:
                filtered_jobs.append(job)

        # Update counters
        original_count = len(original_jobs)
        filtered_count = len(filtered_jobs)

        total_original_jobs += original_count
        total_filtered_jobs += filtered_count
        total_duplicate_jobs += duplicate_count

        # Log filtering results for this company
        if original_count > 0:
            logger.info(
                f"{company_name}: {filtered_count} new jobs, {duplicate_count} duplicate jobs filtered out"
            )

        # Add company data with filtered jobs (even if empty)
        filtered_companies_jobs["companies"].append(
            {
                "company": company_name,
                "job_description_selector": job_description_selector,
                "jobs": filtered_jobs,
            }
        )

    # Log overall filtering results
    logger.info(f"Filtering summary:")
    logger.info(f"  Original jobs: {total_original_jobs}")
    logger.info(f"  New jobs after filtering: {total_filtered_jobs}")
    logger.info(f"  Duplicate jobs filtered out: {total_duplicate_jobs}")

    return filtered_companies_jobs


async def main():
    """Main function to process all companies."""
    # Check if prompt template file exists
    if not Path(PROMPT_FILE).exists():
        logger.error(f"Prompt template file '{PROMPT_FILE}' not found")
        return

    # Read input file with company data
    try:
        with open(COMPANIES_FILE, "r") as f:
            companies = json.load(f)
    except Exception as e:
        logger.error(f"Error reading input file: {str(e)}")
        return

    # Process each company
    companies_jobs = {"companies": []}  # Initialize the structure for all jobs

    # Process each company
    for company in companies:
        company_name = company.get("name")
        career_url = company.get("career_url")
        job_board_selector = company.get("html_selectors", {}).get(
            "job_board_selector", []
        )
        job_eligibility_selector = company.get("html_selectors", {}).get(
            "job_eligibility_selector", []
        )
        job_description_selector = company.get("html_selectors", {}).get(
            "job_description_selector", []
        )

        if not company_name or not career_url:
            logger.warning("Skipping entry with missing name or URL")
            continue

        result = await process_company(company_name, career_url, job_board_selector)

        # Check if there was an error or no jobs found
        if "error" in result:
            logger.error(f"Error processing {company_name}: {result['error']}")
            # Skip this company entirely if there was an error
            continue

        if not result.get("jobs"):
            logger.warning(f"No jobs found for {company_name}")

        # Generate signature for each job and filter out existing ones
        jobs_with_signatures = []
        for job in result.get("jobs", []):
            signature = generate_job_signature(job.get("url", ""))
            job["signature"] = signature
            jobs_with_signatures.append(job)

        # Add to the all_jobs structure with filtered jobs
        companies_jobs["companies"].append(
            {
                "company": company_name,
                "job_eligibility_selector": job_eligibility_selector,
                "job_description_selector": job_description_selector,
                "jobs": jobs_with_signatures,
            }
        )

        # Delay to avoid rate limiting
        await asyncio.sleep(1)

    # Filter out jobs that were processed the previous day
    companies_jobs = filter_new_jobs(companies_jobs)

    output_file = PIPELINE_OUTPUT_DIR / JOBS_FILE
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(companies_jobs, f, indent=2, ensure_ascii=False)

    logger.info(f"Processing complete. Results saved to {output_file}")


if __name__ == "__main__":
    # Check for API key
    if not OPENAI_API_KEY:
        logger.error("OPENAI_API_KEY environment variable is not set")
        exit(1)

    logger.info("Starting job link extraction process")
    # Run the async main function
    asyncio.run(main())
    logger.info("Process completed")
