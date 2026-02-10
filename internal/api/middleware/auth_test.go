// Task 1.6.9: TDD tests for Bearer JWT AuthMiddleware
// Covers: token absent, invalid, expired, valid — and context injection.
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/api/middleware"
	pkgauth "github.com/matiasleandrokruk/fenix/pkg/auth"
)

// TestMain sets JWT_SECRET before any test runs.
// Task 1.6.14: pkgauth.GenerateJWT panics if JWT_SECRET is not set.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
}

// ===== HELPER =====

// nextHandler returns an http.Handler that sets called=true and records the context.
func nextHandler(called *bool, capturedCtx *context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		if capturedCtx != nil {
			ctx := r.Context()
			*capturedCtx = ctx
		}
		w.WriteHeader(http.StatusOK)
	})
}

// makeRequest creates a GET request with an optional Authorization header.
func makeRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// ===== TESTS: TOKEN ABSENT =====

// TestAuthMiddleware_NoToken verifies that missing Authorization header returns 401.
func TestAuthMiddleware_NoToken(t *testing.T) {
	t.Parallel()

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(""))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}

	if called {
		t.Error("next handler should NOT be called when token is missing")
	}
}

// TestAuthMiddleware_EmptyBearerValue verifies that "Bearer " with empty token returns 401.
func TestAuthMiddleware_EmptyBearerValue(t *testing.T) {
	t.Parallel()

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}

	if called {
		t.Error("next handler should NOT be called for empty Bearer token")
	}
}

// TestAuthMiddleware_WrongScheme verifies that non-Bearer scheme returns 401.
func TestAuthMiddleware_WrongScheme(t *testing.T) {
	t.Parallel()

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}

	if called {
		t.Error("next handler should NOT be called for non-Bearer scheme")
	}
}

// ===== TESTS: INVALID TOKEN =====

// TestAuthMiddleware_InvalidToken verifies that a garbage token returns 401.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	t.Parallel()

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest("not.a.real.jwt"))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}

	if called {
		t.Error("next handler should NOT be called for invalid token")
	}
}

// TestAuthMiddleware_TamperedToken verifies that a token with modified payload returns 401.
func TestAuthMiddleware_TamperedToken(t *testing.T) {
	t.Parallel()

	// Generate a valid token then truncate it (simulates tampering)
	validToken, _ := pkgauth.GenerateJWT("user-1", "ws-1")
	tampered := validToken[:len(validToken)-10] + "TAMPERED!!"

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(tampered))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}

	if called {
		t.Error("next handler should NOT be called for tampered token")
	}
}

// TestAuthMiddleware_ExpiredToken verifies that an expired token returns 401.
// Note: Cannot use t.Parallel() — buildExpiredToken calls t.Setenv.
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Build an expired token manually (exp = 1 second ago)
	expiredToken := buildExpiredToken(t, "user-1", "ws-1")

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(expiredToken))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}

	if called {
		t.Error("next handler should NOT be called for expired token")
	}
}

// ===== TESTS: VALID TOKEN =====

// TestAuthMiddleware_ValidToken verifies that a valid token passes through to next handler.
func TestAuthMiddleware_ValidToken(t *testing.T) {
	t.Parallel()

	token, err := pkgauth.GenerateJWT("user-abc", "ws-xyz")
	if err != nil {
		t.Fatalf("GenerateJWT error = %v", err)
	}

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(token))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", rr.Code, http.StatusOK)
	}

	if !called {
		t.Error("next handler SHOULD be called for valid token")
	}
}

// TestAuthMiddleware_InjectsUserIDInContext verifies that UserID is in context after valid token.
func TestAuthMiddleware_InjectsUserIDInContext(t *testing.T) {
	t.Parallel()

	userID := "user-abc-123"
	token, _ := pkgauth.GenerateJWT(userID, "ws-xyz")

	var capturedCtx context.Context
	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, &capturedCtx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(token))

	if !called {
		t.Fatal("next handler was not called")
	}

	gotUserID, ok := capturedCtx.Value(ctxkeys.UserID).(string)
	if !ok || gotUserID == "" {
		t.Error("UserID not injected in context")
	}

	if gotUserID != userID {
		t.Errorf("context UserID = %q; want %q", gotUserID, userID)
	}
}

// TestAuthMiddleware_InjectsWorkspaceIDInContext verifies that WorkspaceID is in context.
func TestAuthMiddleware_InjectsWorkspaceIDInContext(t *testing.T) {
	t.Parallel()

	workspaceID := "ws-xyz-789"
	token, _ := pkgauth.GenerateJWT("user-abc", workspaceID)

	var capturedCtx context.Context
	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, &capturedCtx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(token))

	if !called {
		t.Fatal("next handler was not called")
	}

	gotWsID, ok := capturedCtx.Value(ctxkeys.WorkspaceID).(string)
	if !ok || gotWsID == "" {
		t.Error("WorkspaceID not injected in context")
	}

	if gotWsID != workspaceID {
		t.Errorf("context WorkspaceID = %q; want %q", gotWsID, workspaceID)
	}
}

// TestAuthMiddleware_BothClaimsInjected verifies UserID and WorkspaceID both present.
func TestAuthMiddleware_BothClaimsInjected(t *testing.T) {
	t.Parallel()

	userID := "user-full-1"
	workspaceID := "ws-full-1"
	token, _ := pkgauth.GenerateJWT(userID, workspaceID)

	var capturedCtx context.Context
	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, &capturedCtx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(token))

	if !called {
		t.Fatal("next handler was not called")
	}

	gotUserID, _ := capturedCtx.Value(ctxkeys.UserID).(string)
	gotWsID, _ := capturedCtx.Value(ctxkeys.WorkspaceID).(string)

	if gotUserID != userID {
		t.Errorf("context UserID = %q; want %q", gotUserID, userID)
	}

	if gotWsID != workspaceID {
		t.Errorf("context WorkspaceID = %q; want %q", gotWsID, workspaceID)
	}
}

// TestAuthMiddleware_ErrorResponseIsJSON verifies that 401 response is JSON.
func TestAuthMiddleware_ErrorResponseIsJSON(t *testing.T) {
	t.Parallel()

	called := false
	handler := middleware.AuthMiddleware(nextHandler(&called, nil))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest(""))

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q; want %q", contentType, "application/json")
	}
}

// ===== HELPER: build expired token =====

// buildExpiredToken creates a JWT that is already expired (exp = now - 1s).
// Uses JWT_SECRET from env to sign it so ParseJWT can validate the signature,
// then reject it due to expiry.
func buildExpiredToken(t *testing.T, userID, workspaceID string) string {
	t.Helper()

	secret := []byte("test-secret-key-32-chars-min!!!")
	t.Setenv("JWT_SECRET", string(secret))

	now := time.Now()
	claims := &pkgauth.Claims{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Second)), // already expired
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("buildExpiredToken: failed to sign: %v", err)
	}

	return signed
}
