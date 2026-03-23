package api

import "fmt"

// ListResponse is the generic response for API7 EE list endpoints.
// API7 EE returns paginated results with a "list" array and "total" count.
type ListResponse[T any] struct {
	Total int `json:"total"`
	List  []T `json:"list"`
}

// SingleResponse is a generic wrapper for single-resource responses.
// API7 EE typically returns the resource object directly, but some
// endpoints wrap it. This provides flexibility.
type SingleResponse[T any] struct {
	Value T `json:"value"`
}

// APIError represents an error response from the API7 EE Admin API.
type APIError struct {
	StatusCode int    `json:"-"`
	ErrorMsg   string `json:"error_msg"`
}

func (e *APIError) Error() string {
	if e.ErrorMsg != "" {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.ErrorMsg)
	}
	return fmt.Sprintf("API error: status %d", e.StatusCode)
}

// DeleteResponse is the response for delete endpoints.
type DeleteResponse struct {
	Message string `json:"message"`
}
