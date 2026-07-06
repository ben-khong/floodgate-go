// Package store provides the Redis client connection for the rate limiter.
// It exposes a single constructor that connects and validates the connection
// before returning the client.

package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// context.Background() is the root context, no deadline, never cancelled.
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to redis: %w", err)
	}

	return client, nil
}
