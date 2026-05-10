package transport

import "fmt"

// APIError represents a non-2xx response.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Body       []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("api error: %s %s (%d)", e.Method, e.URL, e.StatusCode)
}
