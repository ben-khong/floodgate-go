package limiter

import (
	"context"
	"time"
)

// REMINDER: Anything that declares an interface's methods is that type
type Limiter interface {
	Allow(ctx context.Context, key string) (Result, error)
}

type Result struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}
