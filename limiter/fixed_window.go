package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// FixedWindowLimiter divides time into fixed intervals and counts requests
// in each interval. When the count exceeds the limit, requests are denied
// until the next window starts.
type FixedWindowLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	// fallback uses a 1 minute window as a reasonable default when Redis is unavailable
	fallback *fallback
}

func NewFixedWindow(client *redis.Client, limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		client:   client,
		limit:    limit,
		window:   window,
		fallback: newFallback(limit/2, window),
	}
}

func (fw *FixedWindowLimiter) Allow(ctx context.Context, key string) (Result, error) {
	windowStart := time.Now().Truncate(fw.window)
	resetAt := windowStart.Add(fw.window)

	// Include window timestamp in key to avoid relying on Redis TTL, which can lag.
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())

	redisClient := fw.client

	// If INCR is called on a key that doesn't exist yet, it returns 1
	count, err := redisClient.Incr(ctx, windowKey).Result()
	if err != nil {
		return fw.fallback.Allow(ctx, key)
	}
	// If first request in that window, set an expiry
	if count == 1 {
		_, err = redisClient.Expire(ctx, windowKey, fw.window).Result()
		if err != nil {
			return fw.fallback.Allow(ctx, key)
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
