// Task 1.6.10: Bearer JWT AuthMiddleware
// Reads Authorization: Bearer <token>, validates it, injects user_id + workspace_id into context.
package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	pkgauth "github.com/matiasleandrokruk/fenix/pkg/auth"
)

// AuthMiddleware validates the Bearer JWT token and injects claims into context.
// Task 1.6.10: Used on all /api/v1/* routes except /auth/register and /auth/login.
//
// Flow:
//  1. Read "Authorization: Bearer <token>" header
//  2. Reject if missing or not Bearer scheme → 401
//  3. Parse + validate JWT → 401 on invalid/expired
//  4. Inject ctxkeys.UserID and ctxkeys.WorkspaceID into context
//  5. Call next handler
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractBearerToken(r)
		if tokenString == "" {
			writeUnauthorized(w, "missing or invalid Authorization header")
			return
		}

		claims, err := pkgauth.ParseJWT(tokenString)
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}

		// Inject claims into context using typed keys (prevents collision — Task 1.3 TD-1 lesson)
		ctx := r.Context()
		ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, claims.UserID)
		ctx = ctxkeys.WithValue(ctx, ctxkeys.WorkspaceID, claims.WorkspaceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractBearerToken extracts the token from "Authorization: Bearer <token>".
// Returns empty string if header is missing, wrong scheme, or token is empty.
// Extracted for testability and to reduce cyclomatic complexity of AuthMiddleware.
func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}

	// Must start with "Bearer " (case-sensitive per RFC 7235)
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	token := strings.TrimPrefix(header, prefix)
	token = strings.TrimSpace(token)
	return token
}

// writeUnauthorized writes a 401 JSON response.
// Uses consistent format with writeError in handlers package.
func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": message}) //nolint:errcheck
}
