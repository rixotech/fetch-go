# fetch

An ergonomic HTTP client for Go, inspired by axios.

## Install
​```bash
go get github.com/YOUR_USERNAME/fetch
​```

## Quick start
​```go
client, err := fetch.New("https://api.example.com",
    fetch.WithTimeout(10*time.Second),
)

var users []User
err = client.Get(ctx, "/users").
    WithParam("page", "1").
    Scan(&users)
​```