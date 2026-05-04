package fetch_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rixotech/fetch-go"
)

// ExampleNew demonstrates creating a reusable client with common defaults.
func ExampleNew() {
	client := fetch.New("https://api.example.com",
		fetch.WithTimeout(10*time.Second),
		fetch.WithDefaultHeaders(map[string]string{
			"Accept":     "application/json",
			"User-Agent": "my-app/1.0",
		}),
	)

	// Attach a request interceptor that injects an auth token on every call.
	client.UseRequest(func(req *http.Request) (*http.Request, error) {
		req.Header.Set("Authorization", "Bearer "+getToken())
		return req, nil
	})

	// Attach a response interceptor for structured logging.
	client.UseResponse(func(resp *http.Response) (*http.Response, error) {
		fmt.Printf("[fetch] %s %s → %s\n", resp.Request.Method, resp.Request.URL.Path, resp.Status)
		return resp, nil
	})

	ctx := context.Background()

	// ── GET with query params ─────────────────────────────────────────────
	var users []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	client.Get(ctx, "/users").
		WithParam("page", "1").
		WithParam("limit", "20").
		Scan(&users)
	if err := client.Error(); err != nil {
		// Non-2xx responses surface as *fetch.FetchError.
		if fe, ok := fetch.AsFetchError(err); ok {
			fmt.Printf("HTTP %d: %s\n", fe.StatusCode, fe.Body)
		}
		log.Fatal(err)
	}

	// ── POST with JSON body ───────────────────────────────────────────────
	type CreateUserReq struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var created User
	client.Post(ctx, "/users", CreateUserReq{
		Name:  "Alice",
		Email: "alice@example.com",
	}).Scan(&created)
	if err := client.Error(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created user id=%d\n", created.ID)

	// ── PUT ───────────────────────────────────────────────────────────────
	var updated User
	client.Put(ctx, "/users/1", CreateUserReq{Name: "Alice Updated"}).
		Scan(&updated)
	if err := client.Error(); err != nil {
		log.Fatal(err)
	}

	// ── PATCH ─────────────────────────────────────────────────────────────
	client.Patch(ctx, "/users/1", map[string]string{"name": "Alice V2"}).
		Scan(&updated)
	if err := client.Error(); err != nil {
		log.Fatal(err)
	}

	// ── DELETE ────────────────────────────────────────────────────────────
	resp, err := client.Delete(ctx, "/users/1", nil).Do()
	if err := client.Error(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("delete status: %s\n", resp.Status)

	// ── Form-encoded body ─────────────────────────────────────────────────
	_, err = client.Post(ctx, "/login", nil).
		WithFormBody(map[string]string{
			"username": "alice",
			"password": "s3cr3t",
		}).Do()
	if err != nil {
		log.Fatal(err)
	}

	// ── Raw text body ──────────────────────────────────────────────────────
	_, err = client.Post(ctx, "/logs", nil).
		WithTextBody("something happened").
		Do()
	if err != nil {
		log.Fatal(err)
	}

	// ── Read response as plain text ────────────────────────────────────────
	r, err := client.Get(ctx, "/health").Do()
	if err != nil {
		log.Fatal(err)
	}
	text, _ := r.Text()
	fmt.Println(text)
}

func getToken() string { return "secret-token" }
