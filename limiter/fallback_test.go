package limiter

import (
	"context"
	"testing"
	"time"
)

func TestFallback(t *testing.T) {
	t.Run("allows request under limit", func(t *testing.T) {
		fb := newFallback(5, time.Minute)
		result, err := fb.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.Allowed != true {
			t.Errorf("expected allowed=true, got allowed=false")
		}
	})

	t.Run("deny request over limit", func(t *testing.T) {
		fb := newFallback(3, time.Minute)

		for i := 0; i < 3; i++ {
			fb.Allow(context.Background(), "test-key")
		}
		result, err := fb.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.Allowed != false {
			t.Errorf("expected allowed=false, got allowed=true")
		}
	})

	t.Run("allow request exactly up to limit", func(t *testing.T) {
		fb := newFallback(10, time.Minute)

		for i := 0; i < 9; i++ {
			fb.Allow(context.Background(), "test-key")
		}
		result, err := fb.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.Allowed != true {
			t.Errorf("expected allowed=true, got allowed=false")
		}
	})

	t.Run("new window resets counter", func(t *testing.T) {
		fb := newFallback(2, time.Second)
		for i := 0; i < 2; i++ {
			fb.Allow(context.Background(), "test-key")
		}
		time.Sleep(1100 * time.Millisecond)
		result, err := fb.Allow(context.Background(), "test-key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected allowed=true after window reset, got allowed=false")
		}
	})
}
