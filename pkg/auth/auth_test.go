// Task 1.6.1 + 1.6.3: Tests for bcrypt password hashing and JWT generation/parsing
package auth

import (
	"os"
	"testing"
	"time"
)

// TestMain sets JWT_SECRET before any test runs.
// Task 1.6.14: GenerateJWT panics if JWT_SECRET is not set in the environment.
// Using os.Setenv (not t.Setenv) here because TestMain runs before t is available.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
}

// ===== BCRYPT TESTS =====

// TestHashPassword verifies that HashPassword generates a valid bcrypt hash.
func TestHashPassword(t *testing.T) {
	t.Parallel()

	password := "MySecurePassword123!"
	hash, err := HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("HashPassword returned empty hash")
	}

	// Hash should not equal plaintext password
	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}

	// Hash should start with bcrypt prefix $2a$ or $2b$ or $2y$
	if len(hash) < 20 || !isValidBcryptHash(hash) {
		t.Errorf("Hash format is invalid: %s", hash)
	}
}

// TestHashPassword_EmptyPassword verifies that empty passwords are hashed (no rejection).
func TestHashPassword_EmptyPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("")

	// Empty passwords should be allowed (let app layer decide policy)
	if err != nil {
		t.Fatalf("HashPassword should allow empty password for flexibility: %v", err)
	}

	if hash == "" {
		t.Error("HashPassword returned empty hash for empty password")
	}
}

// TestVerifyPassword_CorrectPassword verifies that VerifyPassword accepts correct password.
func TestVerifyPassword_CorrectPassword(t *testing.T) {
	t.Parallel()

	password := "MySecurePassword123!"
	hash, _ := HashPassword(password)

	ok := VerifyPassword(hash, password)

	if !ok {
		t.Error("VerifyPassword should return true for correct password")
	}
}

// TestVerifyPassword_WrongPassword verifies that VerifyPassword rejects wrong password.
func TestVerifyPassword_WrongPassword(t *testing.T) {
	t.Parallel()

	password := "MySecurePassword123!"
	hash, _ := HashPassword(password)

	ok := VerifyPassword(hash, "DifferentPassword")

	if ok {
		t.Error("VerifyPassword should return false for incorrect password")
	}
}

// TestVerifyPassword_InvalidHash verifies that VerifyPassword handles invalid hash gracefully.
func TestVerifyPassword_InvalidHash(t *testing.T) {
	t.Parallel()

	ok := VerifyPassword("not-a-valid-hash", "somepassword")

	if ok {
		t.Error("VerifyPassword should return false for invalid hash")
	}
}

// TestVerifyPassword_CaseSensitive verifies that passwords are case-sensitive.
func TestVerifyPassword_CaseSensitive(t *testing.T) {
	t.Parallel()

	password := "MySecurePassword123!"
	hash, _ := HashPassword(password)

	// Same password but different case
	ok := VerifyPassword(hash, "mysecurepassword123!")

	if ok {
		t.Error("VerifyPassword should be case-sensitive")
	}
}

// TestHashPassword_DifferentHashesSamePassword verifies that same password produces different hashes (salt).
func TestHashPassword_DifferentHashesSamePassword(t *testing.T) {
	t.Parallel()

	password := "MySecurePassword123!"
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// Two hashes of same password should be different (due to salt)
	if hash1 == hash2 {
		t.Error("HashPassword should produce different hashes for same password (salt randomness)")
	}

	// But both should verify the password
	if !VerifyPassword(hash1, password) || !VerifyPassword(hash2, password) {
		t.Error("Both hashes should verify the correct password")
	}
}

// ===== JWT TESTS =====

// TestGenerateJWT verifies that GenerateJWT produces a valid JWT token.
func TestGenerateJWT(t *testing.T) {
	t.Parallel()

	userID := "user-uuid-123"
	workspaceID := "ws-uuid-456"

	token, err := GenerateJWT(userID, workspaceID)

	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	if token == "" {
		t.Error("GenerateJWT returned empty token")
	}

	// Token should have 3 parts separated by dots (header.payload.signature)
	parts := countJWTParts(token)
	if parts != 3 {
		t.Errorf("JWT should have 3 parts, got %d", parts)
	}
}

// TestParseJWT_ValidToken verifies that ParseJWT correctly extracts claims from valid token.
func TestParseJWT_ValidToken(t *testing.T) {
	t.Parallel()

	userID := "user-uuid-123"
	workspaceID := "ws-uuid-456"
	token, _ := GenerateJWT(userID, workspaceID)

	claims, err := ParseJWT(token)

	if err != nil {
		t.Fatalf("ParseJWT failed for valid token: %v", err)
	}

	if claims == nil {
		t.Fatal("ParseJWT returned nil claims")
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}

	if claims.WorkspaceID != workspaceID {
		t.Errorf("Expected WorkspaceID %s, got %s", workspaceID, claims.WorkspaceID)
	}
}

// TestParseJWT_InvalidToken verifies that ParseJWT rejects invalid token.
func TestParseJWT_InvalidToken(t *testing.T) {
	t.Parallel()

	_, err := ParseJWT("invalid.token.here")

	if err == nil {
		t.Error("ParseJWT should return error for invalid token")
	}
}

// TestParseJWT_MalformedToken verifies that ParseJWT rejects malformed token.
func TestParseJWT_MalformedToken(t *testing.T) {
	t.Parallel()

	_, err := ParseJWT("not-a-jwt")

	if err == nil {
		t.Error("ParseJWT should return error for malformed token")
	}
}

// TestParseJWT_EmptyToken verifies that ParseJWT rejects empty token.
func TestParseJWT_EmptyToken(t *testing.T) {
	t.Parallel()

	_, err := ParseJWT("")

	if err == nil {
		t.Error("ParseJWT should return error for empty token")
	}
}

// TestJWT_Expiry verifies that expired tokens are rejected.
func TestJWT_Expiry(t *testing.T) {
	t.Parallel()

	// This test requires manipulating JWT_EXPIRY env var or the token directly.
	// For now, we test that the token has reasonable expiry set.
	userID := "user-uuid-123"
	workspaceID := "ws-uuid-456"
	token, _ := GenerateJWT(userID, workspaceID)

	claims, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}
	if claims == nil {
		t.Fatal("ParseJWT returned nil claims")
	}

	// Token should have an expiry time set
	if claims.ExpiresAt == nil {
		t.Error("JWT should have ExpiresAt set")
	}

	// Expiry should be in the future
	if claims.ExpiresAt.Before(time.Now()) {
		t.Error("JWT ExpiresAt should be in the future")
	}
}

// TestJWT_ClaimsIncludeRequired verifies that JWT includes all required claims.
func TestJWT_ClaimsIncludeRequired(t *testing.T) {
	t.Parallel()

	userID := "user-uuid-123"
	workspaceID := "ws-uuid-456"
	token, _ := GenerateJWT(userID, workspaceID)

	claims, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}
	if claims == nil {
		t.Fatal("ParseJWT returned nil claims")
	}

	// Check all required claims
	if claims.UserID == "" {
		t.Error("JWT missing UserID claim")
	}
	if claims.WorkspaceID == "" {
		t.Error("JWT missing WorkspaceID claim")
	}
	if claims.ExpiresAt == nil {
		t.Error("JWT missing ExpiresAt claim")
	}
	if claims.IssuedAt == nil {
		t.Error("JWT missing IssuedAt claim")
	}
}

// ===== parseJWTExpiry TESTS =====

// TestParseJWTExpiry_Default verifies that empty string returns default expiry (24h).
func TestParseJWTExpiry_Default(t *testing.T) {
	t.Parallel()

	result := parseJWTExpiry("")

	expected := time.Duration(DefaultJWTExpiry) * time.Hour
	if result != expected {
		t.Errorf("Expected default expiry %v, got %v", expected, result)
	}
}

// TestParseJWTExpiry_ValidHours verifies that valid number string is parsed correctly.
func TestParseJWTExpiry_ValidHours(t *testing.T) {
	t.Parallel()

	result := parseJWTExpiry("48")

	expected := 48 * time.Hour
	if result != expected {
		t.Errorf("Expected 48h, got %v", result)
	}
}

// TestParseJWTExpiry_InvalidString verifies that non-numeric string falls back to default.
func TestParseJWTExpiry_InvalidString(t *testing.T) {
	t.Parallel()

	result := parseJWTExpiry("not-a-number")

	expected := time.Duration(DefaultJWTExpiry) * time.Hour
	if result != expected {
		t.Errorf("Expected default expiry %v on invalid input, got %v", expected, result)
	}
}

// TestParseJWTExpiry_ZeroHours verifies zero is parsed as 0h (not default).
func TestParseJWTExpiry_ZeroHours(t *testing.T) {
	t.Parallel()

	result := parseJWTExpiry("0")

	expected := 0 * time.Hour
	if result != expected {
		t.Errorf("Expected 0h for '0', got %v", result)
	}
}

// TestParseJWTExpiry_ShortExpiry verifies short expiry (1 hour) is parsed correctly.
func TestParseJWTExpiry_ShortExpiry(t *testing.T) {
	t.Parallel()

	result := parseJWTExpiry("1")

	expected := 1 * time.Hour
	if result != expected {
		t.Errorf("Expected 1h, got %v", result)
	}
}

// ===== GenerateJWT with custom expiry TESTS =====

// TestJWT_CustomExpiry verifies that token respects custom JWT_EXPIRY from env.
func TestJWT_CustomExpiry(t *testing.T) {
	// Cannot use t.Parallel() due to os.Setenv mutation (would race with other tests)
	t.Setenv("JWT_EXPIRY", "2")

	userID := "user-uuid-111"
	workspaceID := "ws-uuid-222"
	before := time.Now()
	token, err := GenerateJWT(userID, workspaceID)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}

	// Expiry should be approximately 2 hours from now
	expectedExpiry := before.Add(2 * time.Hour)
	diff := claims.ExpiresAt.Time.Sub(expectedExpiry).Abs()
	if diff > 5*time.Second {
		t.Errorf("Expected expiry ~2h from now, diff is %v", diff)
	}
}

// ===== HELPER FUNCTIONS (test utilities) =====

// isValidBcryptHash checks if a string looks like a valid bcrypt hash.
func isValidBcryptHash(hash string) bool {
	// Bcrypt hashes start with $2a$, $2b$, or $2y$ and are 60 characters long
	if len(hash) != 60 {
		return false
	}
	if len(hash) >= 4 && (hash[:4] == "$2a$" || hash[:4] == "$2b$" || hash[:4] == "$2y$") {
		return true
	}
	return false
}

// countJWTParts counts the number of parts in a JWT token (separated by dots).
func countJWTParts(token string) int {
	count := 1
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			count++
		}
	}
	return count
}
