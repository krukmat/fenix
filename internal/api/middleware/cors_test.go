// cors_test.go: unit tests for CORSMiddleware (C2).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAllowedOrigin = "http://localhost:3000"

// okHandler is a trivial downstream handler for middleware tests.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// newCORSRequest builds a request with the given method and origin header.
func newCORSRequest(method, origin string) *http.Request {
	req := httptest.NewRequest(method, "/auth/login", nil)
	if origin != "" {
		req.Header.Set(headerOrigin, origin)
	}
	return req
}

// TestCORSMiddleware_AllowedOrigin_ReceivesHeaders verifies that a request from
// the allowed origin gets the Access-Control-Allow-Origin header set.
func TestCORSMiddleware_AllowedOrigin_ReceivesHeaders(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodPost, testAllowedOrigin)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	got := rr.Header().Get(headerACAllowOrigin)
	if got != testAllowedOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q; want %q", got, testAllowedOrigin)
	}
}

// TestCORSMiddleware_AllowedOrigin_PassesThrough verifies that the downstream
// handler is called and returns 200 for allowed origins.
func TestCORSMiddleware_AllowedOrigin_PassesThrough(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodPost, testAllowedOrigin)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusOK)
	}
}

// TestCORSMiddleware_BlockedOrigin_NoHeaders verifies that a request from a
// disallowed origin does NOT receive CORS headers.
func TestCORSMiddleware_BlockedOrigin_NoHeaders(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodPost, "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	got := rr.Header().Get(headerACAllowOrigin)
	if got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty (blocked origin)", got)
	}
}

// TestCORSMiddleware_BlockedOrigin_DownstreamStillCalled verifies that the
// downstream handler is still called for blocked origins (browser handles the
// block, not the server).
func TestCORSMiddleware_BlockedOrigin_DownstreamStillCalled(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodPost, "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d; want %d (downstream should still run)", rr.Code, http.StatusOK)
	}
}

// TestCORSMiddleware_NoOriginHeader_NoHeaders verifies that requests without an
// Origin header (e.g. curl, server-to-server) do not get CORS headers.
func TestCORSMiddleware_NoOriginHeader_NoHeaders(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodGet, "") // no origin
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	got := rr.Header().Get(headerACAllowOrigin)
	if got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty (no origin header)", got)
	}
}

// TestCORSMiddleware_Preflight_AllowedOrigin_Returns204 verifies that OPTIONS
// pre-flight from the allowed origin returns 204 and is not passed downstream.
func TestCORSMiddleware_Preflight_AllowedOrigin_Returns204(t *testing.T) {
	t.Parallel()

	// Downstream would return 500 if invoked — ensures OPTIONS is handled by middleware.
	failHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := CORSMiddleware(testAllowedOrigin)(failHandler)
	req := newCORSRequest(http.MethodOptions, testAllowedOrigin)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d; want %d", rr.Code, http.StatusNoContent)
	}
}

// TestCORSMiddleware_Preflight_BlockedOrigin_Returns204 verifies that OPTIONS
// pre-flight from a blocked origin still returns 204 (no CORS headers added).
func TestCORSMiddleware_Preflight_BlockedOrigin_Returns204(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodOptions, "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d; want %d", rr.Code, http.StatusNoContent)
	}

	got := rr.Header().Get(headerACAllowOrigin)
	if got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty (blocked origin)", got)
	}
}

// TestCORSMiddleware_AllowedOrigin_AllowMethodsHeader verifies Access-Control-Allow-Methods is set.
func TestCORSMiddleware_AllowedOrigin_AllowMethodsHeader(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(testAllowedOrigin)(okHandler)
	req := newCORSRequest(http.MethodOptions, testAllowedOrigin)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	got := rr.Header().Get("Access-Control-Allow-Methods")
	if got == "" {
		t.Error("Access-Control-Allow-Methods is empty; want non-empty")
	}
}
