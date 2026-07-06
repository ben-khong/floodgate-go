package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedis struct {
	counts map[string]int64
}

func (f *fakeRedis) Incr(ctx context.Context, key string) *redis.IntCmd {
	f.counts[key]++
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(f.counts[key])
	return cmd
}

func (f *fakeRedis) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(true)
	return cmd
}

func TestFixedWindow(t *testing.T) {
	t.Run("allow requests under the limit", func(t *testing.T) {
		fb := &fakeRedis{counts: make(map[string]int64)}
		fw := NewFixedWindow(fb, 5, time.Minute)
		result, err := fw.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.Allowed != true {
			t.Errorf("expected allowed=true, got allowed=false")
		}
	})

	t.Run("deny request over the limit", func(t *testing.T) {
		fb := &fakeRedis{counts: make(map[string]int64)}
		fw := NewFixedWindow(fb, 3, time.Minute)
		for i := 0; i < 3; i++ {
			fw.Allow(context.Background(), "test-key")
		}
		result, err := fw.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.Allowed != false {
			t.Errorf("expected allowed=false, got allowed=true")
		}
	})

	t.Run("allow request exactly up to limit", func(t *testing.T) {
		fb := &fakeRedis{counts: make(map[string]int64)}
		fw := NewFixedWindow(fb, 7, time.Minute)

		for i := 0; i < 6; i++ {
			fw.Allow(context.Background(), "test-key")
		}
		result, err := fw.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.Allowed != true {
			t.Errorf("expected allowed=true, got allowed=false")
		}
	})
}
