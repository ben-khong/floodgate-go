package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type fallback struct {
	mu       sync.Mutex
	counters map[string]int // old keys are never deleted - acceptable for short fallback usage
	limit    int
	window   time.Duration
}

// fallback is an in-memory rate limiter used when Redis is unavailable.
// It uses a fixed window counter protected by a mutex for concurrent access.
// Limits are set lower than the main limiter since counters are not shared
// across servers.
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

	// Lock protects the counters map from concurrent writes across goroutines.
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
