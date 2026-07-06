// Package limiter provides rate limiting algorithms and a shared interface
// for controlling request rates using Redis as a distributed counter store.
package limiter

import (
	"context"
	"time"
)

// Limiter is implemented by all rate limiting algorithms.
type Limiter interface {
	Allow(ctx context.Context, key string) (Result, error)
}

// Result holds the outcome of a rate limit check.
type Result struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}
