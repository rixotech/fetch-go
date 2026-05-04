package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// ─── Body helpers ─────────────────────────────────────────────────────────────

// body carries an encoded request body and its Content-Type.
type body struct {
	reader      io.Reader
	contentType string
}

// ─── Request builder ──────────────────────────────────────────────────────────

// Request is a single pending HTTP request built via the fluent API.
// Call Do or Scan to execute it.
type Request struct {
	client  *Client
	ctx     context.Context
	method  string
	rawPath string
	params  url.Values
	headers map[string]string
	body    *body
	err     error // first error captured during building
}

// newRequest allocates a Request attached to c.
func newRequest(c *Client, ctx context.Context, method, path string) *Request {
	return &Request{
		client:  c,
		ctx:     ctx,
		method:  method,
		rawPath: path,
		params:  make(url.Values),
		headers: make(map[string]string),
	}
}

// ─── HTTP method constructors ─────────────────────────────────────────────────

// Get returns a GET Request for the given path.
func (c *Client) Get(ctx context.Context, path string) *Request {
	return newRequest(c, ctx, http.MethodGet, path)
}

// Post returns a POST Request with a JSON-encoded body.
func (c *Client) Post(ctx context.Context, path string, v any) *Request {
	r := newRequest(c, ctx, http.MethodPost, path)
	return r.jsonBody(v)
}

// Put returns a PUT Request with a JSON-encoded body.
func (c *Client) Put(ctx context.Context, path string, v any) *Request {
	r := newRequest(c, ctx, http.MethodPut, path)
	return r.jsonBody(v)
}

// Patch returns a PATCH Request with a JSON-encoded body.
func (c *Client) Patch(ctx context.Context, path string, v any) *Request {
	r := newRequest(c, ctx, http.MethodPatch, path)
	return r.jsonBody(v)
}

// Delete returns a DELETE Request. An optional body may be passed (nil is fine).
func (c *Client) Delete(ctx context.Context, path string, v any) *Request {
	r := newRequest(c, ctx, http.MethodDelete, path)
	if v != nil {
		return r.jsonBody(v)
	}
	return r
}

// Head returns a HEAD Request.
func (c *Client) Head(ctx context.Context, path string) *Request {
	return newRequest(c, ctx, http.MethodHead, path)
}

// Options returns an OPTIONS Request.
func (c *Client) Options(ctx context.Context, path string) *Request {
	return newRequest(c, ctx, http.MethodOptions, path)
}

// ─── Builder methods ──────────────────────────────────────────────────────────

// WithParams merges query parameters into the request URL.
// May be called multiple times; later values overwrite earlier ones for the
// same key.
//
//	r.WithParams(map[string]string{"page": "2", "limit": "50"})
func (r *Request) WithParams(params map[string]string) *Request {
	if r.err != nil {
		return r
	}
	for k, v := range params {
		r.params.Set(k, v)
	}
	return r
}

// WithParam sets a single query parameter.
func (r *Request) WithParam(key, value string) *Request {
	if r.err != nil {
		return r
	}
	r.params.Set(key, value)
	return r
}

// WithHeaders merges HTTP headers into the request.
// Per-request headers override client-level default headers.
func (r *Request) WithHeaders(headers map[string]string) *Request {
	if r.err != nil {
		return r
	}
	for k, v := range headers {
		r.headers[k] = v
	}
	return r
}

// WithHeader sets a single HTTP header.
func (r *Request) WithHeader(key, value string) *Request {
	if r.err != nil {
		return r
	}
	r.headers[key] = value
	return r
}

// WithBearerToken is a convenience shortcut for Authorization: Bearer <token>.
func (r *Request) WithBearerToken(token string) *Request {
	return r.WithHeader("Authorization", "Bearer "+token)
}

// WithBasicAuth sets the Authorization header using HTTP Basic Auth encoding.
func (r *Request) WithBasicAuth(username, password string) *Request {
	req, _ := http.NewRequest("GET", "/", nil) // cheap way to use stdlib helper
	req.SetBasicAuth(username, password)
	return r.WithHeader("Authorization", req.Header.Get("Authorization"))
}

// WithJSONBody replaces the request body with a JSON-encoded v.
// It also sets Content-Type: application/json.
func (r *Request) WithJSONBody(v any) *Request {
	return r.jsonBody(v)
}

// WithTextBody replaces the request body with a plain-text string.
func (r *Request) WithTextBody(text string) *Request {
	if r.err != nil {
		return r
	}
	r.body = &body{
		reader:      strings.NewReader(text),
		contentType: "text/plain; charset=utf-8",
	}
	return r
}

// WithFormBody replaces the request body with URL-encoded form data and sets
// Content-Type: application/x-www-form-urlencoded.
func (r *Request) WithFormBody(fields map[string]string) *Request {
	if r.err != nil {
		return r
	}
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	r.body = &body{
		reader:      strings.NewReader(form.Encode()),
		contentType: "application/x-www-form-urlencoded",
	}
	return r
}

// WithMultipartBody replaces the request body with a multipart/form-data
// payload.  Pass file content as []byte values; all other values are treated
// as text fields.
func (r *Request) WithMultipartBody(fields map[string]any) *Request {
	if r.err != nil {
		return r
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		switch val := v.(type) {
		case []byte:
			fw, err := w.CreateFormFile(k, k)
			if err != nil {
				r.err = fmt.Errorf("fetch: multipart create file field %q: %w", k, err)
				return r
			}
			if _, err = fw.Write(val); err != nil {
				r.err = fmt.Errorf("fetch: multipart write field %q: %w", k, err)
				return r
			}
		default:
			if err := w.WriteField(k, fmt.Sprintf("%v", val)); err != nil {
				r.err = fmt.Errorf("fetch: multipart write field %q: %w", k, err)
				return r
			}
		}
	}
	if err := w.Close(); err != nil {
		r.err = fmt.Errorf("fetch: multipart close: %w", err)
		return r
	}
	r.body = &body{reader: &buf, contentType: w.FormDataContentType()}
	return r
}

// WithRawBody sets an arbitrary body reader and Content-Type.
func (r *Request) WithRawBody(reader io.Reader, contentType string) *Request {
	if r.err != nil {
		return r
	}
	r.body = &body{reader: reader, contentType: contentType}
	return r
}

// ─── Internal body helper ─────────────────────────────────────────────────────

func (r *Request) jsonBody(v any) *Request {
	if r.err != nil {
		return r
	}
	b, err := json.Marshal(v)
	if err != nil {
		r.err = fmt.Errorf("fetch: JSON encode body: %w", err)
		return r
	}
	r.body = &body{
		reader:      bytes.NewReader(b),
		contentType: "application/json",
	}
	return r
}

// ─── Execution ────────────────────────────────────────────────────────────────

// Do executes the request and returns the raw *Response.
//
// Unless WithoutErrorOnStatus was set on the client, a *FetchError is returned
// for any non-2xx status code so the caller does not need to check
// resp.IsOK() manually.
func (r *Request) Do() (*Response, error) {
	if r.err != nil {
		return nil, r.err
	}

	// ── Resolve URL ────────────────────────────────────────────────────────
	// When the client has no base URL (package-level default client), treat
	// rawPath as a complete absolute URL.
	var u *url.URL
	if r.client.baseURL == nil {
		var err error
		u, err = url.Parse(r.rawPath)
		if err != nil {
			return nil, fmt.Errorf("fetch: parse URL %q: %w", r.rawPath, err)
		}
	} else {
		u = r.client.baseURL.ResolveReference(&url.URL{Path: r.rawPath})
	}
	if len(r.params) > 0 {
		q := u.Query()
		for k, vs := range r.params {
			for _, v := range vs {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}

	// ── Body ───────────────────────────────────────────────────────────────
	var bodyReader io.Reader
	if r.body != nil {
		bodyReader = r.body.reader
	}

	// ── Build *http.Request ────────────────────────────────────────────────
	req, err := http.NewRequestWithContext(r.ctx, r.method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("fetch: build request: %w", err)
	}

	// ── Merge headers: defaults → request-level ────────────────────────────
	for k, v := range r.client.defaultHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	if r.body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", r.body.contentType)
	}

	// ── Request interceptors ───────────────────────────────────────────────
	req, err = r.client.hooks.applyRequest(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: request interceptor: %w", err)
	}

	// ── Execute ────────────────────────────────────────────────────────────
	raw, err := r.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: http: %w", err)
	}

	// ── Response interceptors ──────────────────────────────────────────────
	raw, err = r.client.hooks.applyResponse(raw)
	if err != nil {
		raw.Body.Close()
		return nil, fmt.Errorf("fetch: response interceptor: %w", err)
	}

	// ── Non-2xx error wrapping ─────────────────────────────────────────────
	resp := newResponse(raw)
	if r.client.raiseOnError && !resp.IsOK() {
		bodyBytes, _ := io.ReadAll(raw.Body)
		raw.Body.Close()
		return nil, &FetchError{
			StatusCode: raw.StatusCode,
			Status:     raw.Status,
			Body:       bodyBytes,
			Header:     raw.Header,
		}
	}

	return resp, nil
}

// Scan executes the request and JSON-decodes a successful response body into v.
// It is shorthand for:
//
//	resp, err := r.Do()
//	if err != nil { return err }
//	return resp.JSON(v)
func (r *Request) Scan(v any) error {
	resp, err := r.Do()
	if err != nil {
		return err
	}
	return resp.JSON(v)
}
