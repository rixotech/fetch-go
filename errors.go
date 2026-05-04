package fetch

import (
	"fmt"
	"net/http"
)

// FetchError represents an HTTP-level error (non-2xx response).
// It is distinct from a network/transport error so callers can branch
// on response status without string-matching.
type FetchError struct {
	StatusCode int
	Status     string
	Body       []byte
	Header     http.Header
}

func (e *FetchError) Error() string {
	if len(e.Body) > 0 {
		return fmt.Sprintf("fetch: request failed [%s]: %s", e.Status, truncate(e.Body, 256))
	}
	return fmt.Sprintf("fetch: request failed [%s]", e.Status)
}

// IsNotFound returns true when the server returned 404.
func (e *FetchError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

// IsUnauthorized returns true when the server returned 401.
func (e *FetchError) IsUnauthorized() bool { return e.StatusCode == http.StatusUnauthorized }

// IsForbidden returns true when the server returned 403.
func (e *FetchError) IsForbidden() bool { return e.StatusCode == http.StatusForbidden }

// IsServerError returns true for any 5xx status.
func (e *FetchError) IsServerError() bool {
	return e.StatusCode >= http.StatusInternalServerError
}

// AsFetchError unwraps err into *FetchError. Returns (nil, false) when err is
// not (or does not wrap) a *FetchError.
func AsFetchError(err error) (*FetchError, bool) {
	if err == nil {
		return nil, false
	}
	fe, ok := err.(*FetchError)
	return fe, ok
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "…"
}
