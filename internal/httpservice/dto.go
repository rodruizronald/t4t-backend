package httpservice

// SearchResponse represents the search response with pagination
type SearchResponse struct {
	Data       []any             `json:"data"`
	Pagination PaginationDetails `json:"pagination"`
}

// PaginationDetails contains pagination metadata
type PaginationDetails struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

// ErrorDetails contains error information
type ErrorDetails struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}
