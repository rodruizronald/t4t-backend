import asyncio
import sys
from enum import Enum
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from loguru import logger
from playwright.async_api import async_playwright, Page, Frame, Browser

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
        context = await self.setup()
        await self.wait_for_content(context)

        for selector in self.selectors:
            result = await self.extract_element(context, selector)
            self.results.append(result)
            self._log_result(result)

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
        await context.page.wait_for_load_state("networkidle")
        logger.debug("Page reached network idle state")


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
        if context.frame:
            await context.frame.wait_for_load_state("networkidle")
            logger.debug("Iframe content loaded")
        else:
            await context.page.wait_for_load_state("networkidle")

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
        # Initial network idle
        await context.page.wait_for_load_state("networkidle")

        try:
            # Wait for Angular-specific indicators
            await context.page.wait_for_function(
                """
               () => {
                   // Check for Angular rendered content
                   const angularElements = document.querySelectorAll('.ng-star-inserted');
                   const appComponents = document.querySelectorAll('[_ngcontent-ng-c]');
                   return angularElements.length > 0 || appComponents.length > 0;
               }
               """,
                timeout=10000,
            )
            logger.success("Angular dynamic content detected")

            # Additional wait for lazy-loaded components
            await context.page.wait_for_timeout(2000)

        except Exception as e:
            logger.warning(f"Angular content indicators not found: {e}")
            # Continue anyway - the page might still work

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

    try:
        async with async_playwright() as p:
            browser = await p.chromium.launch(headless=headless)
            page = await browser.new_page()

            # Navigate to URL
            logger.info(f"Navigating to {url}")
            await page.goto(url, wait_until="networkidle", timeout=60000)

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

            await browser.close()

    except Exception as e:
        logger.error(f"Error testing selectors on {url}: {str(e)}")
        raise

    return results


async def main():
    """Example usage demonstrating different parser types."""

    url = "https://www.golaunchpad.io/company/careers"
    selectors = [
        "body > app-root > div > main > mat-sidenav-container > mat-sidenav-content > app-career > app-career-container > div"
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
