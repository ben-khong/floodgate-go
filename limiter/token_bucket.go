package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucketLimiter struct {
	client     *redis.Client
	capacity   int
	refillRate float64
}

func NewTokenBucket(client *redis.Client, capacity int, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
	}
}

func (tb *TokenBucketLimiter) Allow(ctx context.Context, key string) (Result, error) {
	redisClient := tb.client
	// HGETALL returns all fields and values of the hash stored at key (map[string]string).
	hGetAllResult, err := redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return Result{}, fmt.Errorf("error: HGetAll() command not working: %w", err)
	}

	// Default to a full bucket if this key has never been seen
	tokens := float64(tb.capacity)
	lastRefill := float64(time.Now().Unix())

	// If the key already exists
	if len(hGetAllResult) != 0 {
		tokens, err = strconv.ParseFloat(hGetAllResult["tokens"], 64)
		if err != nil {
			return Result{}, fmt.Errorf("error: ParseFloat() cannot convert tokens to float64: %w", err)
		}

		lastRefill, err = strconv.ParseFloat(hGetAllResult["lastRefill"], 64)
		if err != nil {
			return Result{}, fmt.Errorf("error: ParseFloat() cannot convert lastRefill to float64: %w", err)
		}

		timeNow := time.Now().Unix()
		elapsedTime := float64(timeNow) - lastRefill

		tokensToAdd := elapsedTime * tb.refillRate
		// Cap at capacity — bucket cannot overflow
		tokens = min(float64(tb.capacity), tokens+tokensToAdd)
	}

	allowed := tokens >= 1
	if allowed {
		tokens--
	}

	_, err = redisClient.HSet(ctx, key, "tokens", tokens, "lastRefill", lastRefill).Result()
	if err != nil {
		return Result{}, fmt.Errorf("error: HSet() command not working: %w", err)
	}

	if allowed {
		return Result{Allowed: true, Remaining: int(tokens), ResetAt: time.Time{}}, nil
	} else {
		return Result{Allowed: false, Remaining: 0, ResetAt: time.Time{}}, nil
	}

}
