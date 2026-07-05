package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type FixedWindowLimiter struct {
	client *redis.Client
	limit  int
	// time.Duration represents a length of time. We use it for ambiguity.
	window time.Duration
}

// Constructors are functions that conveniently create and return a struct.
func NewFixedWindow(client *redis.Client, limit int, window time.Duration) *FixedWindowLimiter {
	// Remember that '*' means that this type is a pointer while & gives the address of it.
	// Without '*' in the return type, FixedWindowLimiter gets copied everytime NewFixedWindow is called.
	return &FixedWindowLimiter{
		client: client,
		limit:  limit,
		// window is the time when that window will expire
		window: window,
	}
}

// The (fw *FixedWindowLimiter) part is called a reciever. It helps us access a struct's fields.
func (fw *FixedWindowLimiter) Allow(ctx context.Context, key string) (Result, error) {
	// .Truncate() rounds the time to the nearest value you pass in it
	windowStart := time.Now().Truncate(fw.window)
	resetAt := windowStart.Add(fw.window)

	// If we were to pass the key instead of a window key, we would be relying on Redis TTL.
	// Not relying on TTL is important because it can lag.
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())

	redisClient := fw.client

	// If INCR is called on a key that doesn't exist yet, it returns 1
	count, err := redisClient.Incr(ctx, windowKey).Result()
	if err != nil {
		return Result{}, fmt.Errorf("error: could not increment redis counter: %w", err)
	}

	// If first request in that window, set an expiry
	if count == 1 {
		_, err = redisClient.Expire(ctx, windowKey, fw.window).Result()
		if err != nil {
			return Result{}, fmt.Errorf("error: could not set expiry: %w", err)
		}
	}

	if count <= int64(fw.limit) {
		return Result{
			Allowed:   true,
			Remaining: fw.limit - int(count),
			ResetAt:   resetAt,
		}, nil
	} else {
		return Result{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   resetAt,
		}, nil
	}

}
