package limiter

import (
	"context"
	"time"
)

// REMINDER: Anything that uses an interface method is its type
type Limiter interface {
	// Allow determines whether or not a request goes through
	Allow(ctx context.Context, key string) (Result, error)
}

type Result struct {
	Allowed   bool
	Remaining int       // How many requests left in the current window
	ResetAt   time.Time // When the current window expires
}
