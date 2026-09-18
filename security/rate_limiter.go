package security

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]rateLimitEntry
	max     int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter with the given
// maximum requests allowed within the time window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]rateLimitEntry),
		max:     max,
		window:  window,
	}
}

// Middleware limits requests based on the client IP address.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		now := time.Now()

		rl.mu.Lock()

		entry, exists := rl.clients[clientIP]

		if !exists || now.Sub(entry.windowStart) >= rl.window {
			entry = rateLimitEntry{
				count:       0,
				windowStart: now,
			}
		}

		entry.count++
		rl.clients[clientIP] = entry

		count := entry.count
		windowStart := entry.windowStart

		rl.mu.Unlock()

		w.Header().Set(
			"X-RateLimit-Limit",
			strconv.Itoa(rl.max),
		)

		if count > rl.max {
			retryAfter := int(
				time.Until(windowStart.Add(rl.window)).Seconds(),
			)

			if retryAfter < 1 {
				retryAfter = 1
			}

			w.Header().Set(
				"Retry-After",
				strconv.Itoa(retryAfter),
			)

			http.Error(
				w,
				"Too Many Requests",
				http.StatusTooManyRequests,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP returns the direct client IP address.
func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(
		strings.TrimSpace(r.RemoteAddr),
	)

	if err == nil {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}
