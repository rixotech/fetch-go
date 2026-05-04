// Package fetch is an ergonomic HTTP client for Go, inspired by axios.
//
// # Quick start
//
//	// Create a reusable client bound to a base URL.
//	// New never returns an error — any problem is stored and returned by Do/Scan.
//	client := fetch.New("https://api.example.com",
//	    fetch.WithTimeout(10*time.Second),
//	    fetch.WithDefaultHeaders(map[string]string{
//	        "Accept": "application/json",
//	    }),
//	)
//
//	// Optionally check for construction errors up front (e.g. bad env var URL).
//	if err := client.Error(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// GET with query params — decode JSON response.
//	var users []User
//	err := client.Get(ctx, "/users").
//	    WithParam("page", "1").
//	    Scan(&users)
//
//	// POST with JSON body.
//	var created User
//	err = client.Post(ctx, "/users", User{Name: "Alice"}).
//	    WithBearerToken(token).
//	    Scan(&created)
//
//	// Request + response interceptors (e.g. logging, auth injection).
//	client.UseRequest(func(req *http.Request) (*http.Request, error) {
//	    req.Header.Set("X-Request-ID", uuid.New().String())
//	    return req, nil
//	})
//
// # Error handling
//
// By default, any non-2xx response is returned as *FetchError, which carries
// StatusCode, Status, Body, and Header fields.
//
//	resp, err := client.Get(ctx, "/secret").Do()
//	if fe, ok := fetch.AsFetchError(err); ok {
//	    if fe.IsUnauthorized() { ... }
//	}
package fetch

import (
	"context"
	"net/http"
	"time"

	"github.com/rixotech/fetch-go/helpers"
)

// ─── Package-level default client ────────────────────────────────────────────
// These functions mirror the Client methods but operate on a shared default
// client with no base URL — useful for one-off requests or scripts.

var defaultClient = &Client{
	httpClient:     &http.Client{Timeout: defaultTimeout},
	defaultHeaders: make(map[string]string),
	raiseOnError:   true,
}

// SetDefaultTimeout changes the timeout of the package-level default client.
func SetDefaultTimeout(d time.Duration) { defaultClient.httpClient.Timeout = d }

// Get issues a one-off GET request without a base URL.
func Get(rawURL string) *Request {
	ctx := helpers.Ctx()
	return defaultClient.Get(ctx, rawURL)
}

// Post issues a one-off POST request without a base URL.
func Post(rawURL string, v any) *Request {
	ctx := helpers.Ctx()
	return defaultClient.Post(ctx, rawURL, v)
}

// Put issues a one-off PUT request without a base URL.
func Put(rawURL string, v any) *Request {
	ctx := helpers.Ctx()
	return defaultClient.Put(ctx, rawURL, v)
}

// Patch issues a one-off PATCH request without a base URL.
func Patch(rawURL string, v any) *Request {
	ctx := helpers.Ctx()
	return defaultClient.Patch(ctx, rawURL, v)
}

// Delete issues a one-off DELETE request without a base URL.
func Delete(rawURL string, v any) *Request {
	ctx := helpers.Ctx()
	return defaultClient.Delete(ctx, rawURL, v)
}

// Get issues a one-off GET request without a base URL.
func GetWithContext(ctx context.Context, rawURL string) *Request {
	return defaultClient.Get(ctx, rawURL)
}

// Post issues a one-off POST request without a base URL.
func PostWithContext(ctx context.Context, rawURL string, v any) *Request {
	return defaultClient.Post(ctx, rawURL, v)
}

// Put issues a one-off PUT request without a base URL.
func PutWithContext(ctx context.Context, rawURL string, v any) *Request {
	return defaultClient.Put(ctx, rawURL, v)
}

// Patch issues a one-off PATCH request without a base URL.
func PatchWithContext(ctx context.Context, rawURL string, v any) *Request {
	return defaultClient.Patch(ctx, rawURL, v)
}

// Delete issues a one-off DELETE request without a base URL.
func DeleteWithContext(ctx context.Context, rawURL string, v any) *Request {
	return defaultClient.Delete(ctx, rawURL, v)
}
