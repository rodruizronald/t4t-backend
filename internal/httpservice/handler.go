package httpservice

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SearchHandler - GENERIC HANDLER that consumers can use
type SearchHandler[TRequest SearchRequest, TParams SearchParams, TResult SearchResult] struct {
	parser          RequestParser[TRequest]
	service         SearchService[TParams, TResult]
	responseBuilder ResponseBuilder[TResult, TParams]
}

// NewSearchHandler creates a new search handler using the provided parser, service, and response builder
func NewSearchHandler[TRequest SearchRequest, TParams SearchParams, TResult SearchResult](
	parser RequestParser[TRequest],
	service SearchService[TParams, TResult],
	responseBuilder ResponseBuilder[TResult, TParams],
) *SearchHandler[TRequest, TParams, TResult] {
	return &SearchHandler[TRequest, TParams, TResult]{
		parser:          parser,
		service:         service,
		responseBuilder: responseBuilder,
	}
}

// NewSearchHandlerWithDefaults - CONVENIENCE CONSTRUCTOR with default implementations
func NewSearchHandlerWithDefaults[TRequest SearchRequest, TParams SearchParams, TResult SearchResult](
	createRequest func() TRequest,
	service SearchService[TParams, TResult],
) *SearchHandler[TRequest, TParams, TResult] {
	return NewSearchHandler(
		NewDefaultRequestParser(createRequest),
		service,
		NewDefaultResponseBuilder[TResult, TParams](),
	)
}

// HandleSearch handles HTTP requests for job search operations
func (h *SearchHandler[TRequest, TParams, TResult]) HandleSearch(c *gin.Context) {
	// Parse request using generic parser
	req, err := h.parser.ParseSearchRequest(c)
	if err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Validate request
	if err = req.Validate(); err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Convert to search params
	searchParams, err := req.ToSearchParams()
	if err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Execute search using consumer's business logic
	results, total, err := h.service.ExecuteSearch(c.Request.Context(), searchParams.(TParams))
	if err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Build and send response using generic builder
	response := h.responseBuilder.BuildSearchResponse(results, total, searchParams.(TParams))
	c.JSON(http.StatusOK, response)
}

// HandleSearchCount handles HTTP requests for count-only operations
func (h *SearchHandler[TRequest, TParams, TResult]) HandleSearchCount(c *gin.Context) {
	// Parse request using generic parser (same as HandleSearch)
	req, err := h.parser.ParseSearchRequest(c)
	if err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Validate request (same as HandleSearch)
	if err = req.Validate(); err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Convert to search params (same as HandleSearch)
	searchParams, err := req.ToSearchParams()
	if err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Execute count using consumer's business logic
	count, err := h.service.ExecuteSearchCount(c.Request.Context(), searchParams.(TParams))
	if err != nil {
		statusCode, errorResp := h.responseBuilder.BuildErrorResponse(err)
		c.JSON(statusCode, errorResp)
		return
	}

	// Return simple count response
	c.JSON(http.StatusOK, map[string]int{"count": count})
}
