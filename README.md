<div align="center">

# 🌐 fetch

**An ergonomic, axios-inspired HTTP client for Go**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Go Reference](https://pkg.go.dev/badge/github.com/rixotech/fetch-go.svg)](https://pkg.go.dev/github.com/rixotech/fetch-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/rixotech/fetch-go)](https://goreportcard.com/report/github.com/rixotech/fetch-go)
[![Tests](https://github.com/rixotech/fetch-go/actions/workflows/test.yml/badge.svg)](https://github.com/rixotech/fetch-go/actions)

</div>

---

`fetch` is a lightweight, zero-dependency HTTP client for Go that brings the
ergonomics of JavaScript's [axios](https://axios-http.com) to idiomatic Go.
It wraps `net/http` with a fluent builder API, typed errors, interceptors, and
first-class JSON support — without hiding the standard library from you.

```go
resp, err := fetch.Get(context.Background(), "https://jsonplaceholder.typicode.com/posts").Do()
	if err != nil {
		log.Fatal("Fetch error:", err)
	}

	posts := []struct {
		UserID int    `json:"userId"`
		ID     int    `json:"id"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}{}

	if err := resp.JSON(&posts); err != nil {
		log.Fatal("JSON parse error:", err)
	}
	log.Printf("Fetched %d posts", len(posts))
	log.Printf("First post: %+v", posts)


// -------------> With Client <------------- //

client := fetch.New("https://api.example.com")

var users []User
err := client.Get(ctx, "/users").
    WithParam("page", "1").
    WithBearerToken(token).
    Scan(&users)

// POST with JSON body
resp, err := client.Post(ctx, "/users", User{Name: "Alice"}).Do()

// PUT
resp, err := client.Put(ctx, "/users/1", User{Name: "Alice Updated"}).Do()

// PATCH
resp, err := client.Patch(ctx, "/users/1", map[string]string{"name": "Alice"}).Do()

// DELETE (body is optional — pass nil)
resp, err := client.Delete(ctx, "/users/1", nil).Do()
```

---

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Guide](#usage-guide)
  - [Creating a Client](#creating-a-client)
  - [Making Requests](#making-requests)
  - [Query Parameters](#query-parameters)
  - [Headers & Auth](#headers--auth)
  - [Request Bodies](#request-bodies)
  - [Reading Responses](#reading-responses)
  - [Error Handling](#error-handling)
  - [Interceptors](#interceptors)
  - [Package-level API](#package-level-api)
- [API Reference](#api-reference)
- [Contributing](#contributing)
- [Changelog](#changelog)
- [License](#license)

---

## Features

- **Fluent builder API** — chain methods naturally, no boilerplate
- **All HTTP verbs** — `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`
- **Body formats** — JSON, form-encoded, multipart, plain text, raw `io.Reader`
- **Typed errors** — non-2xx responses become `*FetchError` with status helpers
- **Interceptors** — hook into every request and response (logging, auth, retries)
- **First-class JSON** — `Scan(&v)` encodes the body and decodes the response in one call
- **Auth helpers** — `WithBearerToken`, `WithBasicAuth`
- **Zero external dependencies** — only the Go standard library
- **Context-aware** — every request respects `context.Context` for cancellation and deadlines
- **Safe for concurrent use** — share one `Client` across all goroutines

---

## Requirements

| Requirement | Version |
|---|---|
| Go | 1.22 or higher |

No external dependencies. `go.sum` will be empty.

---

## Installation

```bash
go get github.com/rixotech/fetch-go@latest
```

Then import it in your code:

```go
import "github.com/rixotech/fetch-go"
```

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/rixotech/fetch-go"
)

type Post struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
    Body  string `json:"body"`
}

func main() {
    // Create a reusable client
    client, err := fetch.New("https://jsonplaceholder.typicode.com",
        fetch.WithTimeout(10*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // GET — decode response into a struct
    var post Post
    if err := client.Get(ctx, "/posts/1").Scan(&post); err != nil {
        log.Fatal(err)
    }
    fmt.Println(post.Title)

    // POST — send JSON body, decode created resource
    var created Post
    err = client.Post(ctx, "/posts", Post{Title: "Hello", Body: "World"}).
        Scan(&created)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(created.ID)
}
```

---

## Usage Guide

### Creating a Client

A `Client` is bound to a base URL and reused across requests. Create it once and share it freely — it is safe for concurrent use.

```go
client, err := fetch.New("https://api.example.com")
```

**With options:**

```go
client, err := fetch.New("https://api.example.com",
    // Total timeout per request (default: 30s)
    fetch.WithTimeout(15*time.Second),

    // Headers sent with every request
    fetch.WithDefaultHeaders(map[string]string{
        "Accept":     "application/json",
        "User-Agent": "my-app/1.0",
    }),

    // Supply your own *http.Client (custom TLS, proxy, cookie jar, etc.)
    fetch.WithHTTPClient(myHTTPClient),

    // Disable automatic *FetchError for non-2xx responses
    fetch.WithoutErrorOnStatus(),
)
```

**`MustNew` — for package-level initialization:**

```go
// Panics on invalid URL; safe to call at package init time.
var apiClient = fetch.MustNew("https://api.example.com")
```

---

### Making Requests

Every method returns a `*Request` builder. Nothing is sent until you call `Do()` or `Scan()`.

```go
ctx := context.Background()

// GET
resp, err := client.Get(ctx, "/users").Do()

// POST with JSON body
resp, err := client.Post(ctx, "/users", User{Name: "Alice"}).Do()

// PUT
resp, err := client.Put(ctx, "/users/1", User{Name: "Alice Updated"}).Do()

// PATCH
resp, err := client.Patch(ctx, "/users/1", map[string]string{"name": "Alice"}).Do()

// DELETE (body is optional — pass nil)
resp, err := client.Delete(ctx, "/users/1", nil).Do()

// HEAD
resp, err := client.Head(ctx, "/users").Do()

// OPTIONS
resp, err := client.Options(ctx, "/users").Do()
```

---

### Query Parameters

```go
// Set multiple at once
resp, err := client.Get(ctx, "/search").
    WithParams(map[string]string{
        "q":     "golang",
        "page":  "1",
        "limit": "20",
    }).
    Do()

// Or set individually
resp, err := client.Get(ctx, "/search").
    WithParam("q", "golang").
    WithParam("page", "1").
    Do()

// Resulting URL: https://api.example.com/search?limit=20&page=1&q=golang
```

---

### Headers & Auth

```go
// Set multiple headers
resp, err := client.Get(ctx, "/data").
    WithHeaders(map[string]string{
        "X-Request-ID": "abc-123",
        "X-Tenant-ID":  "acme",
    }).
    Do()

// Set a single header
resp, err := client.Get(ctx, "/data").
    WithHeader("X-Request-ID", "abc-123").
    Do()

// Bearer token shorthand
resp, err := client.Get(ctx, "/profile").
    WithBearerToken("your-jwt-token").
    Do()

// HTTP Basic Auth
resp, err := client.Get(ctx, "/admin").
    WithBasicAuth("username", "password").
    Do()
```

> Per-request headers **always override** client-level default headers for the same key.

---

### Request Bodies

**JSON** (default for `Post`, `Put`, `Patch`):

```go
type CreateUserReq struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

resp, err := client.Post(ctx, "/users", CreateUserReq{
    Name:  "Alice",
    Email: "alice@example.com",
}).Do()
```

**Override / replace the body at any point in the chain:**

```go
// JSON
resp, err := client.Post(ctx, "/data", nil).
    WithJSONBody(map[string]any{"key": "value"}).
    Do()

// URL-encoded form
resp, err := client.Post(ctx, "/login", nil).
    WithFormBody(map[string]string{
        "username": "alice",
        "password": "s3cr3t",
    }).
    Do()

// Multipart form (file upload)
fileBytes, _ := os.ReadFile("avatar.png")
resp, err := client.Post(ctx, "/upload", nil).
    WithMultipartBody(map[string]any{
        "avatar": fileBytes,  // []byte → file field
        "caption": "My photo", // string → text field
    }).
    Do()

// Plain text
resp, err := client.Post(ctx, "/logs", nil).
    WithTextBody("something happened at 12:00").
    Do()

// Raw reader (e.g. a file, a buffer)
f, _ := os.Open("data.bin")
defer f.Close()
resp, err := client.Post(ctx, "/upload", nil).
    WithRawBody(f, "application/octet-stream").
    Do()
```

---

### Reading Responses

**`Scan` — the fastest path for JSON APIs:**

```go
var user User
// Executes the request AND decodes the JSON body in one call.
err := client.Get(ctx, "/users/1").Scan(&user)
```

**`Do` — when you need the response itself:**

```go
resp, err := client.Get(ctx, "/users/1").Do()
if err != nil {
    return err
}

// Decode options (call exactly one — each closes the body)
var user User
err = resp.JSON(&user)      // JSON → struct
text, err := resp.Text()    // body as string
raw, err := resp.Bytes()    // body as []byte

// Status helpers
resp.IsOK()           // 2xx
resp.IsRedirect()     // 3xx
resp.IsClientError()  // 4xx
resp.IsServerError()  // 5xx

// Underlying *http.Response (body not yet read)
resp.StatusCode   // int
resp.Status       // "200 OK"
resp.Header       // http.Header
resp.Raw          // *http.Response
```

---

### Error Handling

By default, any non-2xx response is returned as `*FetchError`. This means
you never need to check `resp.IsOK()` manually.

```go
var user User
err := client.Get(ctx, "/users/99").Scan(&user)
if err != nil {
    if fe, ok := fetch.AsFetchError(err); ok {
        // HTTP-level error
        fmt.Println(fe.StatusCode) // e.g. 404
        fmt.Println(fe.Status)     // e.g. "404 Not Found"
        fmt.Println(string(fe.Body)) // raw response body

        switch {
        case fe.IsNotFound():
            // handle 404
        case fe.IsUnauthorized():
            // handle 401 — refresh token, redirect to login, etc.
        case fe.IsForbidden():
            // handle 403
        case fe.IsServerError():
            // handle 5xx
        }
    }
    // Network / timeout / interceptor error
    return err
}
```

**Opt out of automatic error wrapping** (handle every status yourself):

```go
client, _ := fetch.New("https://api.example.com",
    fetch.WithoutErrorOnStatus(),
)

resp, err := client.Get(ctx, "/maybe-404").Do()
if err != nil {
    return err // only network errors reach here
}
if !resp.IsOK() {
    // decide what to do with 4xx / 5xx
}
```

---

### Interceptors

Interceptors run on every request/response made by a client — ideal for
cross-cutting concerns like logging, auth token injection, and metrics.

```go
// ── Request interceptor ───────────────────────────────────────────────────
// Injects a correlation ID into every outgoing request.
client.UseRequest(func(req *http.Request) (*http.Request, error) {
    req.Header.Set("X-Request-ID", uuid.New().String())
    return req, nil
})

// Refreshes an expired bearer token transparently.
client.UseRequest(func(req *http.Request) (*http.Request, error) {
    token, err := tokenStore.Valid()
    if err != nil {
        return nil, err // aborts the request
    }
    req.Header.Set("Authorization", "Bearer "+token)
    return req, nil
})

// ── Response interceptor ──────────────────────────────────────────────────
// Structured request logging.
client.UseResponse(func(resp *http.Response) (*http.Response, error) {
    log.Printf("[fetch] %s %s → %s",
        resp.Request.Method,
        resp.Request.URL.Path,
        resp.Status,
    )
    return resp, nil
})

// Multiple interceptors are executed in registration order.
client.UseRequest(interceptorA, interceptorB, interceptorC)
```

---

### Package-level API

For scripts and one-off requests where creating a `Client` is overkill, use
the package-level functions. They work without a base URL.

```go
// No client needed — just pass a full URL.
var post Post
err := fetch.Get(ctx, "https://jsonplaceholder.typicode.com/posts/1").
    Scan(&post)

err = fetch.Post(ctx, "https://example.com/events",
    map[string]string{"event": "signup"},
).Do()

// Adjust the default timeout globally.
fetch.SetDefaultTimeout(5 * time.Second)
```

---

## API Reference

### `Client`

| Method | Description |
|---|---|
| `fetch.New(baseURL, ...Option)` | Create a new client |
| `fetch.MustNew(baseURL, ...Option)` | Create a client, panic on error |
| `client.UseRequest(...fn)` | Register request interceptor(s) |
| `client.UseResponse(...fn)` | Register response interceptor(s) |

### Options

| Option | Description | Default |
|---|---|---|
| `WithTimeout(d)` | Request timeout | `30s` |
| `WithHTTPClient(c)` | Custom `*http.Client` | built-in |
| `WithDefaultHeaders(m)` | Headers sent with every request | — |
| `WithoutErrorOnStatus()` | Disable `*FetchError` for non-2xx | enabled |

### `*Request` (builder)

| Method | Description |
|---|---|
| `WithParam(key, value)` | Set a single query parameter |
| `WithParams(map)` | Set multiple query parameters |
| `WithHeader(key, value)` | Set a single header |
| `WithHeaders(map)` | Set multiple headers |
| `WithBearerToken(token)` | Set `Authorization: Bearer <token>` |
| `WithBasicAuth(user, pass)` | Set HTTP Basic Auth header |
| `WithJSONBody(v)` | Set a JSON-encoded body |
| `WithFormBody(map)` | Set a URL-encoded form body |
| `WithMultipartBody(map)` | Set a multipart/form-data body |
| `WithTextBody(s)` | Set a plain-text body |
| `WithRawBody(r, contentType)` | Set an arbitrary `io.Reader` body |
| `Do()` | Execute → `(*Response, error)` |
| `Scan(v)` | Execute + JSON-decode → `error` |

### `*Response`

| Method | Description |
|---|---|
| `JSON(v)` | Decode body as JSON into v |
| `XML(v)` | Decode body as XML into v |
| `Text()` | Read body as string |
| `Bytes()` | Read body as `[]byte` |
| `IsOK()` | `true` for 2xx |
| `IsRedirect()` | `true` for 3xx |
| `IsClientError()` | `true` for 4xx |
| `IsServerError()` | `true` for 5xx |

### `*FetchError`

| Field / Method | Description |
|---|---|
| `StatusCode int` | HTTP status code |
| `Status string` | HTTP status string (e.g. `"404 Not Found"`) |
| `Body []byte` | Raw response body |
| `Header http.Header` | Response headers |
| `IsNotFound()` | `true` for 404 |
| `IsUnauthorized()` | `true` for 401 |
| `IsForbidden()` | `true` for 403 |
| `IsServerError()` | `true` for 5xx |
| `fetch.AsFetchError(err)` | Unwrap any error to `*FetchError` |

---

## Contributing

Contributions are welcome and appreciated. Please follow these steps.

### 1. Fork & clone

```bash
git clone https://github.com/rixotech/fetch-go.git
cd fetch
```

### 2. Create a feature branch

```bash
git checkout -b feat/your-feature-name
# or for bug fixes:
git checkout -b fix/what-was-broken
```

### 3. Make your changes

- Keep the public API **backward compatible** unless the change is intentional and documented.
- Add or update tests for every changed behaviour.
- Ensure all exported symbols have Go doc comments.

### 4. Run tests and checks

```bash
# All tests must pass
go test ./... -race -count=1

# No vet warnings
go vet ./...

# Format your code
gofmt -w .
```

### 5. Commit with a clear message

We follow [Conventional Commits](https://www.conventionalcommits.org):

```bash
git commit -m "feat: add WithRetry option for automatic retries"
git commit -m "fix: nil pointer when base URL has trailing slash"
git commit -m "docs: add multipart upload example to README"
git commit -m "test: cover 401 interceptor refresh scenario"
```

### 6. Open a Pull Request

Push your branch and open a PR against `main`. Include:
- **What** the change does
- **Why** it is needed
- Any **breaking changes** (if applicable)

### Reporting bugs

Open an issue at [github.com/rixotech/fetch-go/issues](https://github.com/rixotech/fetch-go/issues) and include:
- Go version (`go version`)
- OS and architecture
- A minimal code snippet that reproduces the bug
- Expected vs actual behaviour

### Suggesting features

Open a [GitHub Discussion](https://github.com/rixotech/fetch-go/discussions) before opening a PR for large changes, so the design can be agreed on first.

---

## Changelog

### v0.1.0 — Initial Release
- Fluent request builder with full HTTP verb support
- JSON, form, multipart, text, and raw body types
- Typed `*FetchError` with status helpers
- Request and response interceptor chains
- `WithBearerToken` and `WithBasicAuth` auth helpers
- `Scan(v)` one-liner for JSON decode
- Package-level convenience API

---

## License

Released under the [MIT License](LICENSE).
Copyright (c) 2026 RixoTech.

---

<div align="center">

Made with ☕ and Go · [Documentation](https://pkg.go.dev/github.com/rixotech/fetch-go) · [Report a Bug](https://github.com/rixotech/fetch-go/issues)

</div>