// ratelimit.go: in-memory, per-IP rate limiting middleware (C4).
// Suitable for MVP / single-instance deployments. Not cluster-safe.
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)


// bucket tracks request count and the start of the current window for one IP.
type bucket struct {
	mu        sync.Mutex
	count     int
	windowEnd time.Time
}

// ipLimiter is a map of IP address to its rate-limit bucket.
type ipLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

func newIPLimiter(limit int, window time.Duration) *ipLimiter {
	return &ipLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}
}

// allow returns true if the request from ip is within the rate limit.
func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{}
		l.buckets[ip] = b
	}
	l.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.After(b.windowEnd) {
		// Start a new window.
		b.count = 0
		b.windowEnd = now.Add(l.window)
	}

	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// remoteIP returns only a client IP established by trusted middleware.
// When none is present, it falls back to the TCP peer address and never
// trusts forwarding headers supplied directly by the request.
func remoteIP(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimitMiddleware returns a middleware that limits requests per IP.
//   - limit: maximum number of requests allowed in the window
//   - window: duration of each rate-limit window
//
// Responds with 429 Too Many Requests when the limit is exceeded.
func RateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := newIPLimiter(limit, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := remoteIP(r)
			if !limiter.allow(ip) {
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
