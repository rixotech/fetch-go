package fetch

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// Response wraps *http.Response and exposes ergonomic decode helpers.
// The underlying body is always closed after any Scan / JSON / Text / Bytes
// call — callers must not read the body a second time.
type Response struct {
	// StatusCode is the HTTP response status code (e.g. 200, 404).
	StatusCode int
	// Status is the raw status string (e.g. "200 OK").
	Status string
	// Header contains the response headers.
	Header http.Header
	// Raw is the underlying *http.Response for callers who need lower-level
	// access. Body has NOT been consumed; call exactly one of the decode
	// helpers, or read Body yourself and close it.
	Raw *http.Response
}

func newResponse(r *http.Response) *Response {
	return &Response{
		StatusCode: r.StatusCode,
		Status:     r.Status,
		Header:     r.Header,
		Raw:        r,
	}
}

// ─── Decode helpers ──────────────────────────────────────────────────────────

// JSON decodes the response body as JSON into v.
func (r *Response) JSON(v any) error {
	defer r.Raw.Body.Close()
	if err := json.NewDecoder(r.Raw.Body).Decode(v); err != nil {
		return fmt.Errorf("fetch: JSON decode: %w", err)
	}
	return nil
}

// XML decodes the response body as XML into v.
func (r *Response) XML(v any) error {
	defer r.Raw.Body.Close()
	if err := xml.NewDecoder(r.Raw.Body).Decode(v); err != nil {
		return fmt.Errorf("fetch: XML decode: %w", err)
	}
	return nil
}

// Text reads and returns the response body as a UTF-8 string.
func (r *Response) Text() (string, error) {
	b, err := r.Bytes()
	return string(b), err
}

// Bytes reads and returns the raw response body bytes.
func (r *Response) Bytes() ([]byte, error) {
	defer r.Raw.Body.Close()
	b, err := io.ReadAll(r.Raw.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch: read body: %w", err)
	}
	return b, nil
}

// ─── Status helpers ───────────────────────────────────────────────────────────

// IsOK returns true for any 2xx status code.
func (r *Response) IsOK() bool { return r.StatusCode >= 200 && r.StatusCode < 300 }

// IsRedirect returns true for any 3xx status code.
func (r *Response) IsRedirect() bool { return r.StatusCode >= 300 && r.StatusCode < 400 }

// IsClientError returns true for any 4xx status code.
func (r *Response) IsClientError() bool { return r.StatusCode >= 400 && r.StatusCode < 500 }

// IsServerError returns true for any 5xx status code.
func (r *Response) IsServerError() bool { return r.StatusCode >= 500 }
