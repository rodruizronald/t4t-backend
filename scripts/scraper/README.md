# Job Link Extractor

This tool extracts job listing links from company career websites using Playwright for web scraping and OpenAI for intelligent link extraction.

## Requirements

- Python 3.8+
- Playwright
- OpenAI Python SDK

## Installation

1. Clone this repository or download the script files
2. Install the required packages:

```bash
pip install playwright openai
playwright install  # Install browser binaries
```

3. Set up your OpenAI API key as an environment variable:

```bash
# On macOS/Linux
export OPENAI_API_KEY="your-api-key-here"

# On Windows (Command Prompt)
set OPENAI_API_KEY=your-api-key-here

# On Windows (PowerShell)
$env:OPENAI_API_KEY="your-api-key-here"
```

## Usage

1. Create a `companies.json` file with the list of companies and their career URLs:

```json
[
  {
    "name": "Company Name",
    "career_url": "https://company.com/careers"
  },
  ...
]
```

2. Create a `prompt_template.txt` file with your prompt (a template is provided in this repo)

3. Run the script:

```bash
python job_link_extractor.py
```

4. Find the results in the `extracted_jobs` directory:
   - Individual company results: `extracted_jobs/company_name_jobs.json`
   - Combined results: `extracted_jobs/all_jobs_TIMESTAMP.json`

## How It Works

1. The script reads the list of companies from `companies.json`
2. For each company, it uses Playwright to fetch the HTML content of the career page
3. It sends the HTML content to OpenAI's API with a specialized prompt for extracting job links
4. OpenAI returns a JSON response with extracted job links
5. The script saves both individual company results and a combined file with all results

## Configuration

You can modify these variables at the top of the script:

- `INPUT_FILE`: Path to the JSON file with company data (default: `companies.json`)
- `OUTPUT_DIR`: Directory where results will be saved (default: `extracted_jobs`)
- `MODEL`: OpenAI model to use (default: `gpt-4-turbo`)
- `PROMPT_FILE`: Path to the file containing the prompt template (default: `prompt_template.txt`)

## Limitations

- The script includes a 3-second delay after page load to allow dynamic content to render, but some websites may need more time
- Very large HTML content might exceed OpenAI's token limits
- The quality of extracted links depends on OpenAI's ability to identify job links based on the prompt

## Troubleshooting

If you encounter issues:

1. **Timeout errors**: Increase the timeout value in the `page.goto()` method
2. **Missing links**: Some websites might load job listings dynamically with JavaScript - adjust the delay
3. **Rate limiting**: If processing many companies, increase the delay between API calls