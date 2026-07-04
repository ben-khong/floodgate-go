package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() (*redis.Client, error) {
	// This connects to Redis and establises a client connection
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// This sends a PING command to Redis and it responds with PONG if alive.
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to redis: %w", err)
	}

	// REMINDER: Context is Go's way of carrying cancellation signals and
	// deadlines across function calls. context.Background() is the root
	// context, no deadline, never cancelled.

	return client, nil
}
