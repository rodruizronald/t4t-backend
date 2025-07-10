package httpservice

import (
	"fmt"
	"strings"
)

// RequestParseError represents an error that occurred while parsing HTTP request parameters.
// This typically happens when query parameters cannot be bound to the request struct,
// indicating malformed or invalid client input.
// Results in HTTP 400 Bad Request.
type RequestParseError struct {
	Err error
}

func (e *RequestParseError) Error() string {
	return fmt.Sprintf("request parse error: %v", e.Err)
}

// ValidationError represents validation failures for search request parameters.
// This occurs when the request is well-formed but contains invalid values
// (e.g., invalid enum values, out-of-range numbers, malformed dates).
// Results in HTTP 400 Bad Request.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation errors: %s", strings.Join(e.Errors, ", "))
}

// SearchError represents an error that occurred during search execution.
// This typically indicates infrastructure issues like database connectivity problems,
// query execution failures, or other server-side issues.
// Results in HTTP 500 Internal Server Error.
type SearchError struct {
	Operation string
	Err       error
}

func (e *SearchError) Error() string {
	return fmt.Sprintf("search error during %s: %v", e.Operation, e.Err)
}

// ConversionError represents an error that occurred while converting request data
// to search parameters. This happens when the request contains data that cannot
// be properly converted to the expected types (e.g., invalid date formats).
// Results in HTTP 400 Bad Request.
type ConversionError struct {
	Field string
	Value string
	Err   error
}

func (e *ConversionError) Error() string {
	return fmt.Sprintf("conversion error for field %s with value %s: %v", e.Field, e.Value, e.Err)
}
