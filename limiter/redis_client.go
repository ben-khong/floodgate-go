package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisClient interface {
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}
