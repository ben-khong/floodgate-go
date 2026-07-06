package middleware

import (
	"context"
	"distributed-rate-limiter/limiter"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeLimiter struct {
	result limiter.Result
	err    error
}

func (f *fakeLimiter) Allow(ctx context.Context, key string) (limiter.Result, error) {
	return f.result, f.err
}

func TestMiddleware(t *testing.T) {
	t.Run("allows request and sets headers", func(t *testing.T) {
		fl := &fakeLimiter{result: limiter.Result{Allowed: true, Remaining: 5, ResetAt: time.Now().Add(time.Minute)}}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rr := httptest.NewRecorder()

		RateLimitMiddleware(fl, next).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if rr.Header().Get("X-RateLimit-Remaining") == "" {
			t.Errorf("expected X-RateLimit-Remaining header to be set")
		}
		if rr.Header().Get("X-RateLimit-Reset") == "" {
			t.Errorf("expected X-RateLimit-Reset header to be set")
		}
	})

	t.Run("denies request with 429", func(t *testing.T) {
		fl := &fakeLimiter{result: limiter.Result{Allowed: false, Remaining: 0, ResetAt: time.Now().Add(time.Minute)}}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rr := httptest.NewRecorder()

		RateLimitMiddleware(fl, next).ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", rr.Code)
		}
		if rr.Header().Get("Retry-After") == "" {
			t.Errorf("expected Retry-After header to be set")
		}
	})

	t.Run("returns 500 when limiter errors", func(t *testing.T) {
		fl := &fakeLimiter{err: errors.New("redis down")}
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rr := httptest.NewRecorder()

		RateLimitMiddleware(fl, next).ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})
}
