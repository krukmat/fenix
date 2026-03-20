// ratelimit_test.go: unit tests for RateLimitMiddleware (C4).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// rateLimitOKHandler returns 200 always.
var rateLimitOKHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// newIPRequest builds a request that appears to come from the given remote IP.
func newIPRequest(ip string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = ip + ":54321"
	return req
}

// TestRateLimit_UnderLimit_Passes verifies requests within the limit pass through.
func TestRateLimit_UnderLimit_Passes(t *testing.T) {
	t.Parallel()

	handler := RateLimitMiddleware(3, time.Minute)(rateLimitOKHandler)

	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newIPRequest("10.0.0.1"))
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: status = %d; want %d", i+1, rr.Code, http.StatusOK)
		}
	}
}

// TestRateLimit_ExceedLimit_Returns429 verifies the (limit+1)th request returns 429.
func TestRateLimit_ExceedLimit_Returns429(t *testing.T) {
	t.Parallel()

	limit := 2
	handler := RateLimitMiddleware(limit, time.Minute)(rateLimitOKHandler)
	ip := "10.0.0.2"

	for i := 0; i < limit; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newIPRequest(ip))
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newIPRequest(ip))
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d; want %d (should be rate-limited)", rr.Code, http.StatusTooManyRequests)
	}
}

// TestRateLimit_DifferentIPs_IndependentCounters verifies that two different IPs
// have independent rate-limit buckets.
func TestRateLimit_DifferentIPs_IndependentCounters(t *testing.T) {
	t.Parallel()

	limit := 1
	handler := RateLimitMiddleware(limit, time.Minute)(rateLimitOKHandler)

	// IP A uses its single allowed request.
	rrA := httptest.NewRecorder()
	handler.ServeHTTP(rrA, newIPRequest("10.0.1.1"))
	if rrA.Code != http.StatusOK {
		t.Errorf("IP A first request: status = %d; want %d", rrA.Code, http.StatusOK)
	}

	// IP B still gets its full quota.
	rrB := httptest.NewRecorder()
	handler.ServeHTTP(rrB, newIPRequest("10.0.1.2"))
	if rrB.Code != http.StatusOK {
		t.Errorf("IP B first request: status = %d; want %d (should not be limited)", rrB.Code, http.StatusOK)
	}
}

// TestRateLimit_WindowReset_AllowsAgain verifies that a new window resets the counter.
func TestRateLimit_WindowReset_AllowsAgain(t *testing.T) {
	t.Parallel()

	// Use a very short window (1ms) so we can observe the reset quickly.
	limit := 1
	window := time.Millisecond
	handler := RateLimitMiddleware(limit, window)(rateLimitOKHandler)
	ip := "10.0.0.3"

	// Exhaust the window.
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, newIPRequest(ip))
	if rr1.Code != http.StatusOK {
		t.Errorf("first request: status = %d; want %d", rr1.Code, http.StatusOK)
	}

	// Wait for window to expire.
	time.Sleep(5 * time.Millisecond)

	// After reset should be allowed again.
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, newIPRequest(ip))
	if rr2.Code != http.StatusOK {
		t.Errorf("after window reset: status = %d; want %d", rr2.Code, http.StatusOK)
	}
}

// TestRateLimit_XRealIP_UsedOverRemoteAddr verifies that X-Real-IP header is used
// for bucket keying (RealIP middleware sets this).
func TestRateLimit_XRealIP_UsedOverRemoteAddr(t *testing.T) {
	t.Parallel()

	limit := 1
	handler := RateLimitMiddleware(limit, time.Minute)(rateLimitOKHandler)

	realIP := "192.168.1.50"
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.99:12345" // different from real IP
	req.Header.Set("X-Real-IP", realIP)

	// First request from realIP — should pass.
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req)
	if rr1.Code != http.StatusOK {
		t.Errorf("first request: status = %d; want %d", rr1.Code, http.StatusOK)
	}

	// Second request with same X-Real-IP — should be limited.
	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req2.RemoteAddr = "10.0.0.100:12345" // different RemoteAddr
	req2.Header.Set("X-Real-IP", realIP)

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("second request same X-Real-IP: status = %d; want %d", rr2.Code, http.StatusTooManyRequests)
	}
}
