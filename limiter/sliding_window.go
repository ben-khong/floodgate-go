package limiter

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SlidingWindowLimiter struct {
	client    *redis.Client
	limit     int
	window    time.Duration
	subWindow time.Duration
}

func NewSlidingWindow(client *redis.Client, limit int, window time.Duration, numSubWindows int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		client:    client,
		limit:     limit,
		window:    window,
		subWindow: window / (time.Duration(numSubWindows)),
	}
}

func (sw *SlidingWindowLimiter) Allow(ctx context.Context, key string) (Result, error) {
	curSW := time.Now().Truncate(sw.subWindow)
	prevSW := curSW.Add(-sw.subWindow)

	curKey := fmt.Sprintf("%s:%d", key, curSW.Unix())
	prevKey := fmt.Sprintf("%s:%d", key, prevSW.Unix())

	vals, err := sw.client.MGet(ctx, prevKey, curKey).Result()
	if err != nil {
		return Result{}, fmt.Errorf("error: could not retrieve values of specified keys: %w", err)
	}

	prevEnd := prevSW.Add(sw.subWindow)
	swStart := time.Now().Add(-sw.window)
	swSize := sw.subWindow

	remainingRatio := math.Min(1.0, float64((prevEnd.Sub(swStart))/swSize))

	var prevCount float64
	if vals[0] != nil {
		prevCount, err = strconv.ParseFloat(vals[0].(string), 64)
		if err != nil {
			return Result{}, fmt.Errorf("error parsing prevCount: %w", err)
		}
	}

	var currCount float64
	if vals[1] != nil {
		currCount, err = strconv.ParseFloat(vals[1].(string), 64)
		if err != nil {
			return Result{}, fmt.Errorf("error parsing currCount: %w", err)
		}
	}

	estimate := (float64(prevCount) * remainingRatio) + float64(currCount)

	if estimate < float64(sw.limit) {
		currCount, err := sw.client.Incr(ctx, curKey).Result()
		if err != nil {
			return Result{}, fmt.Errorf("error: could not increment current sub window counter: %w", err)
		}

		if currCount == 1 {
			sw.client.Expire(ctx, curKey, sw.window)
		}

		return Result{
			Allowed:   true,
			Remaining: sw.limit - int(estimate) - 1,
			ResetAt:   time.Now().Add(sw.window),
		}, nil
	}
	return Result{
		Allowed:   false,
		Remaining: 0,
		ResetAt:   time.Now().Add(sw.window),
	}, nil
}
