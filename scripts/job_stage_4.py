import asyncio
import json
import os
import sys
from datetime import datetime
from pathlib import Path

from dotenv import load_dotenv
import openai
from loguru import logger

# Get the root directory
root_dir = Path(__file__).parent.parent

# Load environment variables from .env file
load_dotenv(root_dir / ".env")

# Configuration
OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY")
MODEL = "gpt-5-mini"  # OpenAI model to use

# Define input directory path and input file name
INPUT_DIR = Path("input")
PROMPT_FILE = (
    INPUT_DIR / "prompts/job_technologies.md"
)  # File containing the prompt template

# Define global output directory path
OUTPUT_DIR = Path("data")
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


async def process_job(job_requirements: dict, job_title: str, company_name: str):
    """Process a single job's requirements to extract technologies.

    Args:
        job_requirements: Dictionary containing 'must_have' and 'nice_to_have' arrays
        job_title: Title of the job for logging purposes
        company_name: Company name for context

    Returns:
        Dictionary with extracted technologies or error information
    """
    logger.info(f"Processing job requirements for {job_title} at {company_name}...")

    # Validate requirements structure
    if not job_requirements:
        logger.warning(f"No requirements found for job: {job_title}")
        return {
            "technologies": [],
            "error": "No requirements provided",
        }

    # Convert requirements to JSON string for the prompt
    requirements_json = json.dumps({"requirements": job_requirements}, indent=2)

    # Read prompt template and fill it with requirements JSON
    prompt_template = read_prompt_template()
    filled_prompt = prompt_template.replace("{requirements_json}", requirements_json)

    # Send to OpenAI
    try:
        logger.info(f"Sending requirements to OpenAI for job: {job_title}...")
        response = client.chat.completions.create(
            model=MODEL,
            messages=[
                {
                    "role": "system",
                    "content": "You extract and categorize technologies from structured job requirements.",
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
            "main_technologies": tech_data.get("main_technologies", []),
            "timestamp": datetime.now().isoformat(),
        }

        logger.success(f"Successfully processed job: {job_title}")
        return result

    except Exception as e:
        logger.error(f"Error processing job {job_title} with OpenAI: {str(e)}")
        return {
            "technologies": [],
            "main_technologies": [],
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

        logger.info(
            f"Historical jobs signatures saved to {current_historical_jobs_file}"
        )
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
        job_title = job.get("title", "")
        company_name = job.get("company", "")
        job_signature = job.get("signature", "")
        job_requirements = job.get("requirements", {})

        if not job_requirements:
            logger.warning(f"Job missing requirements, skipping: {job_title}")
            # Create a clean job object without the excluded fields
            clean_job = {
                k: v
                for k, v in job.items()
                if k not in ["job_description_selector", "eligible"]
            }
            clean_job["technologies"] = []
            clean_job["main_technologies"] = []
            processed_jobs.append(clean_job)
            continue

        logger.info(f"Processing job: {job_title} at {company_name}")
        result = await process_job(job_requirements, job_title, company_name)

        total_jobs_processed += 1

        # Check if there was an error
        if "error" in result:
            logger.error(f"Error processing job {job_title}: {result['error']}")
            # Add empty technologies array if extraction failed
            job["technologies"] = []
            job["main_technologies"] = []
        else:
            if result and result["technologies"]:
                jobs_with_technologies += 1
                # Add technologies to job data
                job["technologies"] = result["technologies"]
                job["main_technologies"] = result.get("main_technologies", [])
                logger.info(
                    f"Added {len(result['technologies'])} technologies to job: {job_title}"
                )
            else:
                # Add empty technologies array if extraction failed
                job["technologies"] = []
                job["main_technologies"] = []
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
