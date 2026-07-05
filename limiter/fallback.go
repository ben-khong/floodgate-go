package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type fallback struct {
	mu       sync.Mutex
	counters map[string]int
	limit    int
	window   time.Duration
}

func newFallback(limit int, window time.Duration) *fallback {
	return &fallback{
		counters: make(map[string]int),
		limit:    limit,
		window:   window,
	}
}

func (fb *fallback) Allow(ctx context.Context, key string) (Result, error) {
	windowStart := time.Now().Truncate(fb.window)
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())

	fb.mu.Lock()
	defer fb.mu.Unlock()
	fb.counters[windowKey]++

	if fb.counters[windowKey] <= fb.limit {
		return Result{
			Allowed:   true,
			Remaining: fb.limit - fb.counters[windowKey],
			ResetAt:   windowStart.Add(fb.window),
		}, nil
	}
	return Result{
		Allowed:   false,
		Remaining: 0,
		ResetAt:   windowStart.Add(fb.window),
	}, nil
}
