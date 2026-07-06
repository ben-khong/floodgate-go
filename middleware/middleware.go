// This file implements the HTTP middleware that intercepts requests,
// checks the rate limit, and either passes the request through or
// rejects it with a 429 response.
package middleware

import (
	"distributed-rate-limiter/limiter"
	"fmt"
	"net"
	"net/http"
	"time"
)

// RateLimitMiddleware wraps a handler and rejects requests that exceed the
// rate limit. It sets standard rate limit headers on every response.
func RateLimitMiddleware(l limiter.Limiter, next http.Handler) http.Handler {
	// http.HandlerFunc converts a plain function into an http.Handler.
	// next is the actual handler we call if the request is allowed.
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
		// Set headers on every response so clients know their current rate limit status.
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		if !result.Allowed {
			// Retry-After tells the client how many seconds to wait before retrying.
			// Only sent on 429 responses.
			retryAfter := time.Until(result.ResetAt).Seconds()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter)))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractIP pulls the client IP from the request's RemoteAddr field.
// RemoteAddr is formatted as "host:port" so SplitHostPort is needed.
func extractIP(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("Error: Unable to extract IP: %w", err)
	}
	return host, nil
}
