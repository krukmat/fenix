// Package middleware provides HTTP middleware for the FenixCRM API.
// cors.go: strict origin allowlist CORS middleware (C2 — no external dep).
package middleware

import (
	"net/http"
)

const (
	headerOrigin        = "Origin"
	headerACAllowOrigin = "Access-Control-Allow-Origin"
)

// CORSMiddleware returns a middleware that enforces a strict CORS allowlist.
// Only requests whose Origin header exactly matches an allowed origin receive CORS
// response headers. Pre-flight OPTIONS requests are terminated with 204.
// Requests from other origins are passed through without CORS headers, which
// causes the browser to block cross-origin access.
//
// Usage:
//
//	r.Use(middleware.CORSMiddleware("http://localhost:3000", "http://localhost:3001"))
func CORSMiddleware(allowedOrigins ...string) func(http.Handler) http.Handler {
	allowed := originSet(allowedOrigins)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get(headerOrigin)
			if _, ok := allowed[origin]; ok {
				w.Header().Set(headerACAllowOrigin, origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", headerOrigin)
			}

			// Respond to pre-flight without calling downstream handlers.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originSet(origins []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return allowed
}
