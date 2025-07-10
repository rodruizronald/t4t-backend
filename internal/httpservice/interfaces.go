package httpservice

import (
	"context"

	"github.com/gin-gonic/gin"
)

// SearchRequest represents the incoming request from the client
type SearchRequest interface {
	Validate() error
	ToSearchParams() (SearchParams, error)
}

// SearchParams represents the parameters for a search operation
type SearchParams interface {
	GetLimit() int
	GetOffset() int
}

// SearchResult represents the result of a search operation
type SearchResult interface {
	GetItems() []any
	GetTotal() int
}

// SearchService handles business logic (domain layer concern) - THIS IS WHAT CONSUMERS MUST IMPLEMENT
type SearchService[TParams SearchParams, TResult SearchResult] interface {
	ExecuteSearch(ctx context.Context, params TParams) (TResult, int, error)
}

// RequestParser handles HTTP request parsing (HTTP layer concern) - WITH DEFAULT IMPLEMENTATION PROVIDED
type RequestParser[T SearchRequest] interface {
	ParseSearchRequest(c *gin.Context) (T, error)
}

// ResponseBuilder handles response formatting (HTTP layer concern) - WITH DEFAULT IMPLEMENTATION PROVIDED
type ResponseBuilder[TResult SearchResult, TParams SearchParams] interface {
	BuildSearchResponse(results TResult, total int, params TParams) SearchResponse
	BuildErrorResponse(err error) (int, ErrorResponse)
}
