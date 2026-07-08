package main

import (
	"distributed-rate-limiter/limiter"
	"distributed-rate-limiter/middleware"
	"distributed-rate-limiter/store"
	"fmt"
	"net/http"
	"time"
)

func main() {
	client, err := store.NewRedisClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("connected to redis!")

	l := limiter.NewFixedWindow(client, 100, time.Minute)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	wrapped := middleware.RateLimitMiddleware(l, handler)

	http.Handle("/", wrapped)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
