package main

import (
	"distributed-rate-limiter/store"
	// "distributed-rate-limiter/limiter"
	// "distributed-rate-limiter/middleware"
	// "net/http"
)

func main() {
	client, err := store.NewRedisClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()
}
