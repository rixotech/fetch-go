package fetch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rixotech/fetch-go"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

func jsonHandler(code int, v any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
}

func echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type echo struct {
			Method  string      `json:"method"`
			Path    string      `json:"path"`
			Query   string      `json:"query"`
			Headers http.Header `json:"headers"`
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(echo{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Headers: r.Header,
		})
	}
}

func ctx() context.Context { return context.Background() }

// ─── Client construction ──────────────────────────────────────────────────────

func TestNew_InvalidURL(t *testing.T) {
	_, err := fetch.New("://bad-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestMustNew_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid URL")
		}
	}()
	fetch.MustNew("://bad-url")
}

// ─── GET ──────────────────────────────────────────────────────────────────────

func TestGet_200(t *testing.T) {
	want := map[string]string{"hello": "world"}
	srv := httptest.NewServer(jsonHandler(200, want))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)

	var got map[string]string
	if err := client.Get(ctx(), "/").Scan(&got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hello"] != "world" {
		t.Fatalf("got %v", got)
	}
}

func TestGet_WithParams(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.RawQuery
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	_, err := client.Get(ctx(), "/search").
		WithParam("q", "golang").
		WithParam("page", "2").
		Do()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(captured, "q=golang") {
		t.Fatalf("params not set: %q", captured)
	}
}

// ─── POST ─────────────────────────────────────────────────────────────────────

func TestPost_JSONBody(t *testing.T) {
	type payload struct{ Name string }

	var received payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected content-type: %s", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(received)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	var echo payload
	err := client.Post(ctx(), "/", payload{Name: "Alice"}).Scan(&echo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if echo.Name != "Alice" {
		t.Fatalf("body not echoed: %+v", echo)
	}
}

// ─── Error handling ───────────────────────────────────────────────────────────

func TestDo_FetchError_404(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(404, map[string]string{"error": "not found"}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	_, err := client.Get(ctx(), "/missing").Do()
	if err == nil {
		t.Fatal("expected error for 404")
	}

	fe, ok := fetch.AsFetchError(err)
	if !ok {
		t.Fatalf("expected *FetchError, got %T", err)
	}
	if fe.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", fe.StatusCode)
	}
	if !fe.IsNotFound() {
		t.Fatal("IsNotFound() should be true")
	}
}

func TestDo_FetchError_500(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(500, map[string]string{"error": "internal"}))
	defer srv.Close()

	_, err := fetch.MustNew(srv.URL).Get(ctx(), "/").Do()
	fe, _ := fetch.AsFetchError(err)
	if !fe.IsServerError() {
		t.Fatal("IsServerError() should be true")
	}
}

func TestWithoutErrorOnStatus(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(404, map[string]string{"reason": "gone"}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL, fetch.WithoutErrorOnStatus())
	resp, err := client.Get(ctx(), "/").Do()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ─── Headers ──────────────────────────────────────────────────────────────────

func TestHeaders_DefaultAndPerRequest(t *testing.T) {
	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL,
		fetch.WithDefaultHeaders(map[string]string{"X-App": "test-app"}),
	)
	_, err := client.Get(ctx(), "/").
		WithHeader("X-Request-ID", "abc123").
		Do()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Get("X-App") != "test-app" {
		t.Errorf("default header missing: %v", captured)
	}
	if captured.Get("X-Request-ID") != "abc123" {
		t.Errorf("per-request header missing: %v", captured)
	}
}

func TestBearerToken(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	_, err := client.Get(ctx(), "/").WithBearerToken("my-secret-token").Do()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authHeader != "Bearer my-secret-token" {
		t.Fatalf("unexpected auth header: %q", authHeader)
	}
}

// ─── Body variants ────────────────────────────────────────────────────────────

func TestWithFormBody(t *testing.T) {
	var ct, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		body = r.Form.Get("username")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	_, err := client.Post(ctx(), "/login", nil).
		WithFormBody(map[string]string{"username": "alice", "password": "s3cr3t"}).
		Do()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected content-type: %q", ct)
	}
	if body != "alice" {
		t.Fatalf("form field not received: %q", body)
	}
}

// ─── Interceptors ─────────────────────────────────────────────────────────────

func TestRequestInterceptor(t *testing.T) {
	var xCustom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xCustom = r.Header.Get("X-Custom")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	client.UseRequest(func(req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Custom", "injected")
		return req, nil
	})

	if _, err := client.Get(ctx(), "/").Do(); err != nil {
		t.Fatal(err)
	}
	if xCustom != "injected" {
		t.Fatalf("interceptor did not run: %q", xCustom)
	}
}

func TestResponseInterceptor(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(200, map[string]string{"k": "v"}))
	defer srv.Close()

	var interceptedStatus int
	client := fetch.MustNew(srv.URL)
	client.UseResponse(func(resp *http.Response) (*http.Response, error) {
		interceptedStatus = resp.StatusCode
		return resp, nil
	})

	if _, err := client.Get(ctx(), "/").Do(); err != nil {
		t.Fatal(err)
	}
	if interceptedStatus != 200 {
		t.Fatalf("response interceptor did not run, got %d", interceptedStatus)
	}
}

// ─── Timeout ──────────────────────────────────────────────────────────────────

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL, fetch.WithTimeout(50*time.Millisecond))
	_, err := client.Get(ctx(), "/").Do()
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func TestResponse_Text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello fetch"))
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	resp, err := client.Get(ctx(), "/").Do()
	if err != nil {
		t.Fatal(err)
	}
	text, err := resp.Text()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello fetch" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestResponse_Bytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte{0x01, 0x02, 0x03})
	}))
	defer srv.Close()

	client := fetch.MustNew(srv.URL)
	resp, err := client.Get(ctx(), "/").Do()
	if err != nil {
		t.Fatal(err)
	}
	b, err := resp.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 3 || b[0] != 0x01 {
		t.Fatalf("unexpected bytes: %v", b)
	}
}

// ─── Package-level API ────────────────────────────────────────────────────────

func TestPackageLevel_Get(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(200, map[string]string{"ok": "true"}))
	defer srv.Close()

	var got map[string]string
	if err := fetch.Get(ctx(), srv.URL+"/").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != "true" {
		t.Fatalf("unexpected body: %v", got)
	}
}
