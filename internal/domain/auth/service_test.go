// Task 1.6.7: TDD tests for AuthService (Register and Login business logic)
// Tests run against in-memory SQLite with real migrations.
package auth_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/auth"
)

// TestMain sets JWT_SECRET before any test runs.
// Task 1.6.14: pkgauth.GenerateJWT panics if JWT_SECRET is not set.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
}

// ===== REGISTER TESTS =====

// TestAuthService_Register_Success verifies that registering creates workspace, user, and returns JWT.
func TestAuthService_Register_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, err := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "alice@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Alice",
		WorkspaceName: "Acme Corp",
	})

	if err != nil {
		t.Fatalf("Register() error = %v; want nil", err)
	}

	if result.Token == "" {
		t.Error("Register() Token is empty; want JWT token")
	}

	if result.UserID == "" {
		t.Error("Register() UserID is empty; want non-empty ID")
	}

	if result.WorkspaceID == "" {
		t.Error("Register() WorkspaceID is empty; want non-empty ID")
	}
}

// TestAuthService_Register_TokenIsValid verifies that the returned token has valid claims.
func TestAuthService_Register_TokenIsValid(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "bob@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Bob",
		WorkspaceName: "Acme Corp",
	})

	// Parse and verify JWT claims
	claims, err := auth.ParseJWT(result.Token)
	if err != nil {
		t.Fatalf("Returned token is not a valid JWT: %v", err)
	}

	if claims.UserID != result.UserID {
		t.Errorf("JWT UserID = %q; want %q", claims.UserID, result.UserID)
	}

	if claims.WorkspaceID != result.WorkspaceID {
		t.Errorf("JWT WorkspaceID = %q; want %q", claims.WorkspaceID, result.WorkspaceID)
	}
}

// TestAuthService_Register_UserPersistedInDB verifies the user is stored in the database.
func TestAuthService_Register_UserPersistedInDB(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "carol@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Carol",
		WorkspaceName: "Acme Corp",
	})

	// Verify user exists in DB with correct fields
	var email, displayName, status string
	var passwordHash sql.NullString
	err := db.QueryRow(`
		SELECT email, display_name, status, password_hash
		FROM user_account WHERE id = ?
	`, result.UserID).Scan(&email, &displayName, &status, &passwordHash)

	if err != nil {
		t.Fatalf("User not found in DB after Register: %v", err)
	}

	if email != "carol@acme.com" {
		t.Errorf("email = %q; want %q", email, "carol@acme.com")
	}

	if displayName != "Carol" {
		t.Errorf("display_name = %q; want %q", displayName, "Carol")
	}

	if status != "active" {
		t.Errorf("status = %q; want %q", status, "active")
	}

	// Password should be stored as a bcrypt hash, not plaintext
	if !passwordHash.Valid || passwordHash.String == "" {
		t.Error("password_hash is NULL or empty; want bcrypt hash")
	}

	if passwordHash.String == "SecurePass123!" {
		t.Error("password_hash should not equal plaintext password")
	}
}

// TestAuthService_Register_WorkspaceCreated verifies the workspace is created.
func TestAuthService_Register_WorkspaceCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "dave@example.com",
		Password:      "SecurePass123!",
		DisplayName:   "Dave",
		WorkspaceName: "Example LLC",
	})

	var name string
	err := db.QueryRow(`SELECT name FROM workspace WHERE id = ?`, result.WorkspaceID).Scan(&name)
	if err != nil {
		t.Fatalf("Workspace not found in DB after Register: %v", err)
	}

	if name != "Example LLC" {
		t.Errorf("workspace.name = %q; want %q", name, "Example LLC")
	}
}

// TestAuthService_Register_DuplicateEmail verifies that duplicate email returns error.
func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	input := domainauth.RegisterInput{
		Email:         "dup@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Dup",
		WorkspaceName: "Acme Corp",
	}

	// First registration should succeed
	_, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("First Register() error = %v; want nil", err)
	}

	// Second registration with same email should fail
	_, err = svc.Register(context.Background(), input)
	if err == nil {
		t.Error("Register() with duplicate email should return error; got nil")
	}
}

// ===== LOGIN TESTS =====

// TestAuthService_Login_Success verifies successful login returns JWT.
func TestAuthService_Login_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	// Register first
	regResult, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "eve@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Eve",
		WorkspaceName: "Acme Corp",
	})

	// Login
	loginResult, err := svc.Login(context.Background(), domainauth.LoginInput{
		Email:    "eve@acme.com",
		Password: "SecurePass123!",
	})

	if err != nil {
		t.Fatalf("Login() error = %v; want nil", err)
	}

	if loginResult.Token == "" {
		t.Error("Login() Token is empty; want JWT token")
	}

	if loginResult.UserID != regResult.UserID {
		t.Errorf("Login() UserID = %q; want %q", loginResult.UserID, regResult.UserID)
	}

	if loginResult.WorkspaceID != regResult.WorkspaceID {
		t.Errorf("Login() WorkspaceID = %q; want %q", loginResult.WorkspaceID, regResult.WorkspaceID)
	}
}

// TestAuthService_Login_TokenIsValid verifies the login token has valid JWT claims.
func TestAuthService_Login_TokenIsValid(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	svc.Register(context.Background(), domainauth.RegisterInput{ //nolint:errcheck
		Email: "frank@acme.com", Password: "SecurePass123!", DisplayName: "Frank", WorkspaceName: "Acme Corp",
	})

	result, _ := svc.Login(context.Background(), domainauth.LoginInput{
		Email: "frank@acme.com", Password: "SecurePass123!",
	})

	claims, err := auth.ParseJWT(result.Token)
	if err != nil {
		t.Fatalf("Login() token is not valid JWT: %v", err)
	}

	if claims.UserID == "" || claims.WorkspaceID == "" {
		t.Error("Login() JWT claims missing UserID or WorkspaceID")
	}
}

// TestAuthService_Login_WrongPassword verifies that wrong password returns error.
func TestAuthService_Login_WrongPassword(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	svc.Register(context.Background(), domainauth.RegisterInput{ //nolint:errcheck
		Email: "grace@acme.com", Password: "SecurePass123!", DisplayName: "Grace", WorkspaceName: "Acme Corp",
	})

	_, err := svc.Login(context.Background(), domainauth.LoginInput{
		Email:    "grace@acme.com",
		Password: "WrongPassword!",
	})

	if err == nil {
		t.Error("Login() with wrong password should return error; got nil")
	}
}

// TestAuthService_Login_NonExistentEmail verifies that unknown email returns error.
func TestAuthService_Login_NonExistentEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	_, err := svc.Login(context.Background(), domainauth.LoginInput{
		Email:    "nobody@acme.com",
		Password: "SomePassword!",
	})

	if err == nil {
		t.Error("Login() with non-existent email should return error; got nil")
	}
}

// TestAuthService_Login_ErrorMessageGeneric verifies error message doesn't reveal whether email exists.
func TestAuthService_Login_ErrorMessageGeneric(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	svc.Register(context.Background(), domainauth.RegisterInput{ //nolint:errcheck
		Email: "hank@acme.com", Password: "SecurePass123!", DisplayName: "Hank", WorkspaceName: "Acme Corp",
	})

	// Wrong password — should say "invalid credentials", not "password incorrect"
	_, errWrongPw := svc.Login(context.Background(), domainauth.LoginInput{
		Email: "hank@acme.com", Password: "WrongPassword!",
	})

	// Non-existent email — should give the same generic error
	_, errNoUser := svc.Login(context.Background(), domainauth.LoginInput{
		Email: "nosuchuser@acme.com", Password: "SecurePass123!",
	})

	// Both should return the same error type (ErrInvalidCredentials)
	if errWrongPw == nil || errNoUser == nil {
		t.Fatal("Both login attempts should fail")
	}

	if errWrongPw.Error() != errNoUser.Error() {
		t.Errorf("Error messages should be identical for security: got %q vs %q",
			errWrongPw.Error(), errNoUser.Error())
	}
}

// ===== TEST HELPERS =====

// mustOpenDB opens an in-memory SQLite DB with all migrations applied.
func mustOpenDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("sqlite.NewDB error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp error = %v", err)
	}

	return db
}

// randID generates a unique random ID string for test isolation.
var counter int64

func randID() string {
	counter++
	return time.Now().Format("20060102150405") + fmt.Sprintf("%04d", counter)
}
