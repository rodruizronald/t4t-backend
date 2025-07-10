// Package httpservice provides a generic HTTP service framework for implementing search endpoints.
// It includes default implementations for request parsing, response building, and error handling
// that can be used across different domain services in the application.
package httpservice

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Constants for error codes and messages
const (
	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeInvalidRequest  = "INVALID_REQUEST"
	ErrCodeValidationError = "VALIDATION_ERROR"
	ErrCodeSearchError     = "SEARCH_ERROR"
)

// DefaultRequestParser - GENERIC IMPLEMENTATION that consumers can use
type DefaultRequestParser[T SearchRequest] struct {
	createRequest func() T // Factory function to create new request instance
}

// NewDefaultRequestParser - CONVENIENCE CONSTRUCTOR with default implementation
func NewDefaultRequestParser[T SearchRequest](createRequest func() T) RequestParser[T] {
	return &DefaultRequestParser[T]{createRequest: createRequest}
}

// ParseSearchRequest - parses incoming request from Gin context
func (p *DefaultRequestParser[T]) ParseSearchRequest(c *gin.Context) (T, error) {
	req := p.createRequest()
	if err := c.ShouldBindQuery(req); err != nil {
		var zero T
		return zero, &RequestParseError{Err: err}
	}
	return req, nil
}

// DefaultResponseBuilder - GENERIC IMPLEMENTATION that consumers can use
type DefaultResponseBuilder[TResult SearchResult, TParams SearchParams] struct{}

// NewDefaultResponseBuilder - CONVENIENCE CONSTRUCTOR with default implementation
func NewDefaultResponseBuilder[TResult SearchResult, TParams SearchParams]() ResponseBuilder[TResult, TParams] {
	return &DefaultResponseBuilder[TResult, TParams]{}
}

// BuildSearchResponse - GENERIC IMPLEMENTATION that consumers can use
func (b *DefaultResponseBuilder[TResult, TParams]) BuildSearchResponse(results TResult, total int,
	params TParams) SearchResponse {
	hasMore := params.GetOffset()+len(results.GetItems()) < total

	return SearchResponse{
		Data: results.GetItems(),
		Pagination: PaginationDetails{
			Total:   total,
			Limit:   params.GetLimit(),
			Offset:  params.GetOffset(),
			HasMore: hasMore,
		},
	}
}

// BuildErrorResponse - GENERIC IMPLEMENTATION that consumers can use
func (b *DefaultResponseBuilder[TResult, TParams]) BuildErrorResponse(err error) (int, ErrorResponse) {
	var e *RequestParseError
	var e1 *ValidationError
	var e2 *SearchError
	var e3 *ConversionError
	switch {
	case errors.As(err, &e):
		return http.StatusBadRequest,
			ErrorResponse{
				Error: ErrorDetails{
					Code:    ErrCodeInvalidRequest,
					Message: "Invalid request parameters",
					Details: []string{e.Error()},
				},
			}
	case errors.As(err, &e1):
		return http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetails{
				Code:    ErrCodeValidationError,
				Message: "Invalid search parameters",
				Details: e1.Errors,
			},
		}
	case errors.As(err, &e2):
		return http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetails{
				Code:    ErrCodeSearchError,
				Message: fmt.Sprintf("Failed to %s", e2.Operation),
				Details: []string{e2.Error()},
			},
		}
	case errors.As(err, &e3):
		return http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetails{
				Code:    ErrCodeValidationError,
				Message: "Invalid search parameters",
				Details: []string{e3.Error()},
			},
		}
	default:
		return http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetails{
				Code:    ErrCodeInternalError,
				Message: "Internal server error",
				Details: []string{err.Error()},
			},
		}
	}
}
