package examples

import (
	"context"
	"time"

	"github.com/rixotech/fetch-go"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	client := fetch.New("https://api.example.com",
		fetch.WithTimeout(10*time.Second),
		fetch.WithDefaultHeaders(map[string]string{
			"Accept": "application/json",
		}),
	)

	var users []User

	err := client.Get(context.Background(), "/users").
		WithParam("page", "1").
		Scan(&users)

	if err != nil {
		// handle error
	}
}
