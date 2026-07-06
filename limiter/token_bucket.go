package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBucketLimiter maintains a bucket of tokens that refills at a constant
// rate. Each request consumes one token. Bursts are allowed up to the bucket
// capacity, and requests are denied when the bucket is empty.
type TokenBucketLimiter struct {
	client     *redis.Client
	capacity   int
	refillRate float64
	fallback   *fallback
}

func NewTokenBucket(client *redis.Client, capacity int, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		fallback:   newFallback(capacity/2, time.Minute),
	}
}

// tokenBucketScript is defined at package level to avoid recompiling the script on every request.
var tokenBucketScript = `
	local keys = KEYS[1]
	local capacity = tonumber(ARGV[1])
	local refill_rate = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])

	local result = redis.call("HMGET", keys, "tokens", "lastRefill")
	local tokens = result[1] -- Lua arrays start at 1, not 0 
	local lastRefill = result[2]

	if tokens == false then
		tokens = capacity 
	end 
	if lastRefill == false then
		lastRefill = now 
	end

	local elapsedTime = now - lastRefill
	local tokensToAdd = elapsedTime * refill_rate

	tokens = math.min(capacity, tokens+tokensToAdd)

	local allowed = 0
	if tokens >= 1 then 
		tokens = tokens - 1
		allowed = 1
	else 
		allowed = 0
	end

	local remaining = tokens
	redis.call("HSET", keys, "tokens", tokens, "lastRefill", now)
	redis.call("EXPIRE", keys, 3600)

	return {allowed, remaining}
`

func (tb *TokenBucketLimiter) Allow(ctx context.Context, key string) (Result, error) {
	result, err := tb.client.Eval(ctx, tokenBucketScript,
		[]string{key},
		tb.capacity, tb.refillRate, time.Now().Unix(),
	).Result()
	if err != nil {
		return tb.fallback.Allow(ctx, key)
	}

	vals := result.([]interface{})
	allowed := vals[0].(int64)
	remaining := vals[1].(int64)

	return Result{
		Allowed:   allowed == 1,
		Remaining: int(remaining),
		ResetAt:   time.Time{},
	}, nil
}
