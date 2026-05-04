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
//
// Any error that occurs during construction (e.g. an invalid base URL) is
// stored inside the Client and returned the first time Do or Scan is called.
// Call client.Error() to inspect it eagerly before making any requests.
type Client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	defaultHeaders map[string]string
	hooks          interceptors
	// raiseOnError causes Do/Scan to return *FetchError for non-2xx responses
	// when true (default). Set to false to handle status codes yourself.
	raiseOnError bool
	// err stores the first error that occurred during client construction.
	// It is returned by every subsequent Do / Scan call.
	err error
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

// New creates a Client bound to baseURL. It never returns an error directly —
// any construction error (e.g. an invalid URL) is stored inside the Client and
// surfaced the first time Do or Scan is called on any request built from it.
//
// Check eagerly with client.Error() if you need to fail fast at startup.
//
//	client := fetch.New("https://api.example.com")
//	if err := client.Error(); err != nil {
//	    log.Fatal(err)
//	}
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		httpClient:     &http.Client{Timeout: defaultTimeout},
		defaultHeaders: make(map[string]string),
		raiseOnError:   true,
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		c.err = fmt.Errorf("fetch: invalid base URL %q: %w", baseURL, err)
		return c
	}
	c.baseURL = u

	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Error returns the first error that occurred during client construction,
// or nil if the client was created successfully.
//
// Use this when you want to validate the client at startup rather than
// waiting for the first request to fail:
//
//	client := fetch.New(os.Getenv("API_URL"))
//	if err := client.Error(); err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) Error() error {
	return c.err
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
