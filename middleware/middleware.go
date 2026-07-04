package middleware

import (
	"distributed-rate-limiter/limiter"
	"fmt"
	"net"
	"net/http"
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
		if !result.Allowed {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
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
