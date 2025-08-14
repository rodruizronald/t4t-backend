import asyncio
import sys
from loguru import logger
from playwright.async_api import async_playwright

# Configure logger for console output
logger.remove()
logger.add(
    sys.stderr,
    level="INFO",
    format="<green>{time:HH:mm:ss}</green> | <level>{level: <8}</level> | <cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - <level>{message}</level>",
)


async def test_html_selectors(url: str, selectors: list[str]):
    """
    Test HTML selectors on a webpage and print the extracted content.
    Automatically detects and handles Greenhouse iframes if present.

    Args:
        url: The URL to test selectors on
        selectors: List of CSS selectors to test
    """
    try:
        async with async_playwright() as p:
            browser = await p.chromium.launch()
            page = await browser.new_page()

            # Navigate to the URL
            logger.info(f"Navigating to {url}")
            await page.goto(url, wait_until="networkidle", timeout=60000)

            # Wait for dynamic content to load
            await page.wait_for_timeout(3000)

            # Check for Greenhouse iframe
            greenhouse_iframe = None
            frame_context = page  # Default to main page

            try:
                # Look for Greenhouse iframe (common ID)
                greenhouse_iframe = await page.query_selector("#grnhse_iframe")
                if greenhouse_iframe:
                    logger.info("Detected Greenhouse iframe - switching context")
                    frame = await greenhouse_iframe.content_frame()
                    if frame:
                        # Wait for iframe content to load
                        await frame.wait_for_load_state("networkidle")
                        frame_context = frame  # Switch context to iframe
                        logger.success(
                            "Successfully switched to Greenhouse iframe context"
                        )
                    else:
                        logger.warning(
                            "Could not access Greenhouse iframe content, using main page"
                        )
            except Exception as e:
                logger.debug(f"No Greenhouse iframe found or accessible: {e}")

            print(f"\n{'='*80}")
            print(f"TESTING SELECTORS ON: {url}")
            if greenhouse_iframe and frame_context != page:
                print(f"📋 CONTEXT: Greenhouse iframe")
            else:
                print(f"📋 CONTEXT: Main page")
            print(f"{'='*80}\n")

            for i, selector in enumerate(selectors, 1):
                print(f"SELECTOR {i}: {selector}")
                print("-" * 60)

                try:
                    # Use the appropriate context (main page or iframe)
                    element = await frame_context.wait_for_selector(
                        selector, timeout=5000
                    )
                    if element:
                        # Get the text content (clean text without HTML tags)
                        text_content = await element.inner_text()
                        # Get the HTML content (with HTML tags)
                        html_content = await element.inner_html()

                        print(f"✅ FOUND ELEMENT")
                        print(f"📝 TEXT CONTENT ({len(text_content)} chars):")
                        print(
                            f"{text_content[:500]}{'...' if len(text_content) > 500 else ''}"
                        )
                        print(f"\n🏷️  HTML CONTENT ({len(html_content)} chars):")
                        print(
                            f"{html_content[:300]}{'...' if len(html_content) > 300 else ''}"
                        )

                        logger.success(
                            f"Successfully extracted content from selector: {selector}"
                        )
                    else:
                        print("❌ ELEMENT NOT FOUND")
                        logger.warning(f"Selector not found: {selector}")

                except Exception as e:
                    # If selector fails in iframe, optionally try main page as fallback
                    if greenhouse_iframe and frame_context != page:
                        try:
                            logger.info(
                                f"Selector failed in iframe, trying main page..."
                            )
                            element = await page.wait_for_selector(
                                selector, timeout=2000
                            )
                            if element:
                                text_content = await element.inner_text()
                                html_content = await element.inner_html()

                                print(f"✅ FOUND ELEMENT (in main page as fallback)")
                                print(f"📝 TEXT CONTENT ({len(text_content)} chars):")
                                print(
                                    f"{text_content[:500]}{'...' if len(text_content) > 500 else ''}"
                                )
                                print(f"\n🏷️  HTML CONTENT ({len(html_content)} chars):")
                                print(
                                    f"{html_content[:300]}{'...' if len(html_content) > 300 else ''}"
                                )
                                logger.success(
                                    f"Found selector in main page as fallback"
                                )
                        except:
                            print(f"❌ ERROR: {str(e)}")
                            logger.error(f"Error with selector {selector}: {str(e)}")
                    else:
                        print(f"❌ ERROR: {str(e)}")
                        logger.error(f"Error with selector {selector}: {str(e)}")

                print(f"\n{'='*80}\n")

            await browser.close()

    except Exception as e:
        logger.error(f"Error testing selectors on {url}: {str(e)}")


async def main():
    """Main function to test selectors."""
    url = "https://www.luflox.com/career"
    selectors = ["main div.career-container"]

    logger.info(f"Testing {len(selectors)} selector(s) on {url}")
    await test_html_selectors(url, selectors)
    logger.info("Testing completed")


if __name__ == "__main__":
    asyncio.run(main())
