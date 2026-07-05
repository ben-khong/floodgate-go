package middleware

import (
	"distributed-rate-limiter/limiter"
	"fmt"
	"net"
	"net/http"
	"time"
)

func RateLimitMiddleware(l limiter.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, err := extractIP(r)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		result, err := l.Allow(r.Context(), host)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		if !result.Allowed {
			retryAfter := time.Until(result.ResetAt).Seconds()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter)))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("Error: Unable to extract IP: %w", err)
	}
	return host, nil
}
