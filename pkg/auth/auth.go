// Task 1.6.2 + 1.6.4: Authentication package — bcrypt password hashing and JWT generation/parsing
// This is a leaf package with no domain dependencies. Used by internal/domain/auth and internal/api/middleware.
package auth

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ===== CONSTANTS =====

// BCryptCost is the work factor for bcrypt. 12 is a good balance for MVP (security vs performance).
const BCryptCost = 12

// DefaultJWTExpiry is the default JWT expiration time in hours if not set via env.
const DefaultJWTExpiry = 24

const (
	envJWTSecret = "JWT_SECRET"
	envJWTExpiry = "JWT_EXPIRY"
)

// ===== ENVIRONMENT VARIABLES =====

// getJWTSecret reads JWT_SECRET from environment. Panics if not set.
// This ensures auth cannot be initialized without a secret configured.
func getJWTSecret() []byte {
	secret := os.Getenv(envJWTSecret)
	if secret == "" {
		panic(envJWTSecret + " environment variable not set — cannot initialize auth")
	}
	return []byte(secret)
}

// parseJWTExpiry parses an expiry string (hours) into a Duration.
// Task 1.6.4: Extracted for testability — getJWTExpiry is the env-reading wrapper.
// Returns DefaultJWTExpiry if empty string or invalid number (graceful degradation).
func parseJWTExpiry(expiryStr string) time.Duration {
	if expiryStr == "" {
		return time.Duration(DefaultJWTExpiry) * time.Hour
	}

	hours, err := strconv.Atoi(expiryStr)
	if err != nil {
		return time.Duration(DefaultJWTExpiry) * time.Hour
	}

	return time.Duration(hours) * time.Hour
}

// getJWTExpiry reads JWT_EXPIRY from environment in hours. Defaults to DefaultJWTExpiry.
func getJWTExpiry() time.Duration {
	return parseJWTExpiry(os.Getenv(envJWTExpiry))
}

// ===== BCRYPT FUNCTIONS =====

// HashPassword hashes a plaintext password using bcrypt.
// Task 1.6.2: Generates a secure bcrypt hash with work factor 12.
// Returns error if bcrypt fails (unlikely in practice, but handle it).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BCryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a plaintext password against a bcrypt hash.
// Task 1.6.2: Returns true if password matches hash, false otherwise.
// Returns false (not error) for invalid hashes to avoid leaking hash format info in responses.
func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ===== JWT FUNCTIONS =====

// Claims represents the JWT claims for FenixCRM.
// Task 1.6.4: Minimal claims per architecture.md Section 8.
// UserID and WorkspaceID are custom claims; the rest are standard JWT claims.
type Claims struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed JWT token with user and workspace claims.
// Task 1.6.4: Uses JWT_SECRET from env and JWT_EXPIRY (default 24 hours).
// Panics if JWT_SECRET is not set (fail-fast for configuration errors).
func GenerateJWT(userID, workspaceID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(getJWTExpiry())

	claims := &Claims{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(getJWTSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

// ParseJWT validates and parses a JWT token, extracting claims.
// Task 1.6.4: Returns error if token is invalid, expired, or malformed.
// Does NOT return error for missing JWT_SECRET — that's a startup failure.
func ParseJWT(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is HMAC-SHA256 (prevent algorithm substitution attacks)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return getJWTSecret(), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid JWT claims or signature")
	}

	return claims, nil
}
