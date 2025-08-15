import asyncio
import sys
from enum import Enum
from typing import List, Optional
from dataclasses import dataclass
from loguru import logger
from playwright.async_api import (
    async_playwright,
    Page,
    Frame,
    TimeoutError as PlaywrightTimeoutError,
)

# Configure logger for console output
logger.remove()
logger.add(
    sys.stderr,
    level="INFO",
    format="<green>{time:HH:mm:ss}</green> | <level>{level: <8}</level> | <cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - <level>{message}</level>",
)


class ParserType(Enum):
    """Enumeration of available parser types."""

    DEFAULT = "default"
    GREENHOUSE = "greenhouse"
    ANGULAR = "angular"


@dataclass
class ElementResult:
    """Data class to hold element extraction results."""

    selector: str
    found: bool
    text_content: Optional[str] = None
    html_content: Optional[str] = None
    error_message: Optional[str] = None
    context: str = "main_page"


@dataclass
class ParseContext:
    """Context information for parsing operations."""

    page: Page
    frame: Optional[Frame] = None
    parser_type: ParserType = ParserType.DEFAULT

    @property
    def target(self) -> Page | Frame:
        """Returns the appropriate target for selector queries."""
        return self.frame if self.frame else self.page


class SelectorParser:
    """Base class for different parser implementations."""

    def __init__(self, page: Page, selectors: List[str]):
        self.page = page
        self.selectors = selectors
        self.results: List[ElementResult] = []

    async def setup(self) -> ParseContext:
        """Setup parsing context. Override in subclasses for specific setup."""
        return ParseContext(page=self.page)

    async def wait_for_content(self, context: ParseContext) -> None:
        """Wait for content to load. Override in subclasses for specific wait logic."""
        pass

    async def extract_element(
        self, context: ParseContext, selector: str, timeout: int = 5000
    ) -> ElementResult:
        """Extract content from a single element."""
        try:
            element = await context.target.wait_for_selector(selector, timeout=timeout)

            if element:
                text_content = await element.inner_text()
                html_content = await element.inner_html()

                return ElementResult(
                    selector=selector,
                    found=True,
                    text_content=text_content,
                    html_content=html_content,
                    context=self._get_context_name(context),
                )
            else:
                return ElementResult(
                    selector=selector,
                    found=False,
                    error_message="Element not found",
                    context=self._get_context_name(context),
                )

        except Exception as e:
            return ElementResult(
                selector=selector,
                found=False,
                error_message=str(e),
                context=self._get_context_name(context),
            )

    def _get_context_name(self, context: ParseContext) -> str:
        """Get human-readable context name."""
        if context.frame:
            return f"{context.parser_type.value}_frame"
        return context.parser_type.value

    async def parse(self) -> List[ElementResult]:
        """Main parsing method."""
        try:
            context = await self.setup()
            await self.wait_for_content(context)

            for selector in self.selectors:
                result = await self.extract_element(context, selector)
                self.results.append(result)
                self._log_result(result)

        except Exception as e:
            logger.error(f"Error during parsing: {e}")
            # Add error result for remaining selectors
            for selector in self.selectors:
                if not any(r.selector == selector for r in self.results):
                    self.results.append(
                        ElementResult(
                            selector=selector,
                            found=False,
                            error_message=f"Parser error: {str(e)}",
                            context=self.parser_type.value,
                        )
                    )

        return self.results

    def _log_result(self, result: ElementResult) -> None:
        """Log the result of element extraction."""
        if result.found:
            logger.success(f"Found element with selector: {result.selector}")
        else:
            logger.warning(
                f"Failed to find element: {result.selector} - {result.error_message}"
            )


class DefaultParser(SelectorParser):
    """Parser for standard HTML pages."""

    async def setup(self) -> ParseContext:
        """Setup for default parsing - no special handling needed."""
        logger.info("Using default parser for standard HTML")
        return ParseContext(page=self.page, parser_type=ParserType.DEFAULT)

    async def wait_for_content(self, context: ParseContext) -> None:
        """Wait for standard page load."""
        try:
            await context.page.wait_for_load_state("domcontentloaded", timeout=30000)
            logger.debug("Page reached network idle state")
        except PlaywrightTimeoutError:
            logger.warning("Network idle timeout - proceeding with available content")
            # Continue anyway - content might still be available


class GreenhouseParser(SelectorParser):
    """Parser for Greenhouse iframe-based job boards."""

    async def setup(self) -> ParseContext:
        """Setup Greenhouse iframe context."""
        logger.info("Using Greenhouse parser - looking for iframe")

        try:
            # Look for Greenhouse iframe
            greenhouse_iframe = await self.page.wait_for_selector(
                "#grnhse_iframe", timeout=5000
            )

            if greenhouse_iframe:
                frame = await greenhouse_iframe.content_frame()
                if frame:
                    logger.success("Successfully accessed Greenhouse iframe")
                    return ParseContext(
                        page=self.page, frame=frame, parser_type=ParserType.GREENHOUSE
                    )
                else:
                    logger.warning(
                        "Could not access iframe content, falling back to main page"
                    )
        except Exception as e:
            logger.warning(f"Greenhouse iframe not found: {e}, using main page")

        return ParseContext(page=self.page, parser_type=ParserType.GREENHOUSE)

    async def wait_for_content(self, context: ParseContext) -> None:
        """Wait for iframe content to load."""
        try:
            if context.frame:
                await context.frame.wait_for_load_state(
                    "domcontentloaded", timeout=30000
                )
                logger.debug("Iframe content loaded")
            else:
                await context.page.wait_for_load_state(
                    "domcontentloaded", timeout=30000
                )
        except PlaywrightTimeoutError:
            logger.warning("Load state timeout - proceeding with available content")

    async def extract_element(
        self, context: ParseContext, selector: str, timeout: int = 5000
    ) -> ElementResult:
        """Try iframe first, then fall back to main page if needed."""
        # Try iframe first if available
        if context.frame:
            result = await super().extract_element(context, selector, timeout)
            if result.found:
                return result

            # Fallback to main page
            logger.info(f"Selector not found in iframe, trying main page: {selector}")
            main_context = ParseContext(
                page=context.page, parser_type=context.parser_type
            )
            return await super().extract_element(main_context, selector, timeout=2000)

        # No iframe, use main page
        return await super().extract_element(context, selector, timeout)


class AngularParser(SelectorParser):
    """Parser for Angular applications with dynamic content."""

    async def setup(self) -> ParseContext:
        """Setup for Angular parsing."""
        logger.info("Using Angular parser for dynamic content")
        return ParseContext(page=self.page, parser_type=ParserType.ANGULAR)

    async def wait_for_content(self, context: ParseContext) -> None:
        """Wait for Angular to render dynamic content."""
        try:
            # For Angular, use 'domcontentloaded' or 'commit' instead of 'networkidle'
            # as Angular apps often never reach networkidle state
            await context.page.wait_for_load_state("domcontentloaded", timeout=30000)
            logger.debug("DOM content loaded")

            # Wait a bit for initial Angular bootstrapping
            await context.page.wait_for_timeout(2000)

            # Try to wait for Angular-specific indicators with a shorter timeout
            try:
                await context.page.wait_for_function(
                    """
                    () => {
                        // Check for any Angular indicators
                        const hasAngularElements =
                            document.querySelector('[ng-version]') !== null ||
                            document.querySelector('app-root') !== null ||
                            document.querySelectorAll('[_ngcontent-ng-c]').length > 0 ||
                            document.querySelectorAll('.ng-star-inserted').length > 0;

                        // Also check if there's actual content (not just Angular shell)
                        const hasContent = document.body.innerText.trim().length > 100;

                        return hasAngularElements || hasContent;
                    }
                    """,
                    timeout=10000,
                )
                logger.success("Angular content detected")

            except PlaywrightTimeoutError:
                logger.warning("Angular indicators not found, but proceeding anyway")

            # Give Angular components time to render
            await context.page.wait_for_timeout(3000)

        except PlaywrightTimeoutError as e:
            logger.warning(f"Angular content wait timeout: {e}")
            # Don't re-raise - continue with what we have
        except Exception as e:
            logger.error(f"Unexpected error waiting for Angular content: {e}")
            # Don't re-raise - continue with what we have

    async def extract_element(
        self,
        context: ParseContext,
        selector: str,
        timeout: int = 10000,  # Longer timeout for Angular
    ) -> ElementResult:
        """Extract element with extended timeout for Angular."""
        return await super().extract_element(context, selector, timeout)


class ParserFactory:
    """Factory class to create appropriate parser instances."""

    _parsers = {
        ParserType.DEFAULT: DefaultParser,
        ParserType.GREENHOUSE: GreenhouseParser,
        ParserType.ANGULAR: AngularParser,
    }

    @classmethod
    def create_parser(
        cls, parser_type: ParserType, page: Page, selectors: List[str]
    ) -> SelectorParser:
        """Create a parser instance based on the specified type."""
        parser_class = cls._parsers.get(parser_type, DefaultParser)
        return parser_class(page, selectors)

    @classmethod
    def register_parser(cls, parser_type: ParserType, parser_class: type) -> None:
        """Register a new parser type (for extensibility)."""
        cls._parsers[parser_type] = parser_class


def format_result_output(result: ElementResult) -> str:
    """Format a single result for console output."""
    output = []
    output.append(f"SELECTOR: {result.selector}")
    output.append("-" * 60)

    if result.found:
        output.append(f"✅ FOUND ELEMENT in {result.context}")

        if result.text_content:
            text_preview = result.text_content[:500]
            if len(result.text_content) > 500:
                text_preview += "..."
            output.append(f"📝 TEXT CONTENT ({len(result.text_content)} chars):")
            output.append(text_preview)

        if result.html_content:
            html_preview = result.html_content[:300]
            if len(result.html_content) > 300:
                html_preview += "..."
            output.append(f"\n🏷️  HTML CONTENT ({len(result.html_content)} chars):")
            output.append(html_preview)
    else:
        output.append(f"❌ ELEMENT NOT FOUND")
        output.append(f"Error: {result.error_message}")

    return "\n".join(output)


async def test_html_selectors(
    url: str,
    selectors: List[str],
    parser: ParserType = ParserType.DEFAULT,
    headless: bool = True,
) -> List[ElementResult]:
    """
    Test HTML selectors on a webpage using the specified parser.

    Args:
        url: The URL to test selectors on
        selectors: List of CSS selectors to test
        parser: Parser type to use (DEFAULT, GREENHOUSE, or ANGULAR)
        headless: Whether to run browser in headless mode

    Returns:
        List of ElementResult objects containing extraction results
    """
    results = []
    browser = None

    try:
        async with async_playwright() as p:
            browser = await p.chromium.launch(headless=headless)
            page = await browser.new_page()

            # Navigate to URL with different strategies based on parser type
            logger.info(f"Navigating to {url}")

            try:
                await page.goto(url, wait_until="domcontentloaded", timeout=30000)
                logger.info("Initial page load complete (domcontentloaded)")

            except PlaywrightTimeoutError:
                logger.warning(
                    f"Page load timeout for {url} - proceeding with partial content"
                )
                # Don't re-raise - the page might still be usable

            except Exception as e:
                logger.error(f"Failed to navigate to {url}: {e}")
                # Return empty results with error messages
                for selector in selectors:
                    results.append(
                        ElementResult(
                            selector=selector,
                            found=False,
                            error_message=f"Navigation failed: {str(e)}",
                            context="error",
                        )
                    )
                return results

            # Create appropriate parser
            parser_instance = ParserFactory.create_parser(parser, page, selectors)

            # Print header
            print(f"\n{'='*80}")
            print(f"TESTING SELECTORS ON: {url}")
            print(f"📋 PARSER: {parser.value}")
            print(f"📊 SELECTORS TO TEST: {len(selectors)}")
            print(f"{'='*80}\n")

            # Parse and collect results
            results = await parser_instance.parse()

            # Print results
            for i, result in enumerate(results, 1):
                print(f"\n[{i}/{len(results)}] {format_result_output(result)}")
                print(f"\n{'='*80}")

    except Exception as e:
        logger.error(f"Unexpected error testing selectors: {str(e)}")
        # Ensure we return results even on error
        if not results:
            for selector in selectors:
                results.append(
                    ElementResult(
                        selector=selector,
                        found=False,
                        error_message=f"Unexpected error: {str(e)}",
                        context="error",
                    )
                )
    finally:
        if browser:
            await browser.close()

    return results


async def main():
    """Example usage demonstrating different parser types."""

    url = "https://www.linkedin.com/jobs/search/?currentJobId=4276872405&f_C=293703&geoId=92000000&origin=COMPANY_PAGE_JOBS_CLUSTER_EXPANSION&originToLandingJobPostings=4276872405%2C4281876153%2C4281806618%2C4283549850%2C4274331743"
    selectors = [
        "#main > div > div.scaffold-layout__list-detail-inner.scaffold-layout__list-detail-inner--grow > div.scaffold-layout__list > div > ul"
    ]
    results = await test_html_selectors(
        url=url,
        selectors=selectors,
        parser=ParserType.ANGULAR,
    )

    # Print summary
    successful = sum(1 for r in results if r.found)
    print(f"\n📊 SUMMARY: {successful}/{len(results)} selectors found successfully")


if __name__ == "__main__":
    asyncio.run(main())
