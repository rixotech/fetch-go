package fetch

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client is a reusable HTTP client bound to a base URL — the Go equivalent of
// an axios instance.  Create one via New; share it across goroutines freely.
type Client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	defaultHeaders map[string]string
	hooks          interceptors
	// raiseOnError causes Do/Scan to return *FetchError for non-2xx responses
	// when true (default). Set to false to handle status codes yourself.
	raiseOnError bool
}

// ─── Functional options ───────────────────────────────────────────────────────

// Option is a functional option for Client.
type Option func(*Client)

// WithTimeout sets the total timeout for every request made by this client.
// Default: 30 s.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithHTTPClient replaces the underlying *http.Client entirely.  Useful when
// you need custom TLS config, a proxy, or cookie jars.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithDefaultHeaders merges h into the client's default headers.  Per-request
// headers always take precedence over these.
func WithDefaultHeaders(h map[string]string) Option {
	return func(c *Client) {
		for k, v := range h {
			c.defaultHeaders[k] = v
		}
	}
}

// WithoutErrorOnStatus disables automatic *FetchError wrapping for non-2xx
// responses.  Use when you want to inspect every status code yourself.
func WithoutErrorOnStatus() Option {
	return func(c *Client) { c.raiseOnError = false }
}

// ─── Constructor ──────────────────────────────────────────────────────────────

// New creates a Client bound to baseURL.
//
//	client, err := fetch.New("https://api.example.com")
func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch: invalid base URL %q: %w", baseURL, err)
	}

	c := &Client{
		baseURL:        u,
		httpClient:     &http.Client{Timeout: defaultTimeout},
		defaultHeaders: make(map[string]string),
		raiseOnError:   true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// MustNew is like New but panics on error.  Intended for package-level
// initialization where an invalid URL is a programming mistake.
func MustNew(baseURL string, opts ...Option) *Client {
	c, err := New(baseURL, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// ─── Interceptor registration ─────────────────────────────────────────────────

// UseRequest appends one or more request interceptors.  They are executed in
// registration order before every outgoing request.
func (c *Client) UseRequest(fns ...RequestInterceptorFn) {
	c.hooks.request = append(c.hooks.request, fns...)
}

// UseResponse appends one or more response interceptors.  They are executed in
// registration order after every response is received.
func (c *Client) UseResponse(fns ...ResponseInterceptorFn) {
	c.hooks.response = append(c.hooks.response, fns...)
}
