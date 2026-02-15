// Task 1.6.8: AuthService — Register and Login business logic
// Handles workspace creation, user creation, password hashing, and JWT issuance.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
	pkgauth "github.com/matiasleandrokruk/fenix/pkg/auth"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// ErrInvalidCredentials is returned by Login when email or password is incorrect.
// Using a single error for both cases avoids leaking whether an email exists (security).
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrEmailAlreadyExists is returned by Register when the email is already taken.
var ErrEmailAlreadyExists = errors.New("email already registered")

// RegisterInput holds the data needed to create a new workspace and user.
// Task 1.6: WorkspaceName creates the tenant; Email is the unique login identifier.
type RegisterInput struct {
	Email         string
	Password      string
	DisplayName   string
	WorkspaceName string
}

// LoginInput holds the credentials for authentication.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult is returned after successful Register or Login.
// Token is a signed JWT containing UserID and WorkspaceID claims.
//nolint:revive // API de dominio estable; renombrar rompe referencias amplias
type AuthResult struct {
	Token       string
	UserID      string
	WorkspaceID string
}

// AuthService defines the authentication business operations.
//nolint:revive // interfaz pública estable en el módulo auth
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
}

// authService is the concrete implementation backed by SQLite.
type authService struct {
	db          *sql.DB
	auditLogger auditLogger
}

type auditLogger interface {
	LogWithDetails(
		ctx context.Context,
		workspaceID string,
		actorID string,
		actorType domainaudit.ActorType,
		action string,
		entityType *string,
		entityID *string,
		details *domainaudit.EventDetails,
		outcome domainaudit.Outcome,
	) error
}

// NewAuthService creates a new AuthService backed by the provided DB.
func NewAuthService(db *sql.DB) AuthService {
	return &authService{db: db}
}

// NewAuthServiceWithAudit creates a new AuthService with audit logging.
func NewAuthServiceWithAudit(db *sql.DB, logger auditLogger) AuthService {
	return &authService{db: db, auditLogger: logger}
}

// Register creates a new workspace and user, then returns a JWT.
// Task 1.6.8: Workspace + user creation is atomic via SQLite transaction.
// Password is hashed with bcrypt before storage; plaintext is never stored.
func (s *authService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	hash, err := pkgauth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	workspaceID := uuid.NewV7().String()
	userID := uuid.NewV7().String()

	if err := s.insertWorkspaceAndUser(ctx, insertParams{
		workspaceID:   workspaceID,
		userID:        userID,
		workspaceName: input.WorkspaceName,
		email:         input.Email,
		passwordHash:  hash,
		displayName:   input.DisplayName,
	}); err != nil {
		return nil, err
	}

	token, err := pkgauth.GenerateJWT(userID, workspaceID)
	if err != nil {
		s.logAuthFailure(ctx, workspaceID, userID, "register", "jwt_generation_failed")
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	s.logAuthSuccess(ctx, workspaceID, userID, "register")

	return &AuthResult{Token: token, UserID: userID, WorkspaceID: workspaceID}, nil
}

// insertParams bundles the data needed for atomic workspace + user creation.
type insertParams struct {
	workspaceID   string
	userID        string
	workspaceName string
	email         string
	passwordHash  string
	displayName   string
}

// insertWorkspaceAndUser executes workspace + user creation in a single transaction.
// Task 1.6.8: Extracted from Register to reduce cyclomatic complexity below threshold.
func (s *authService) insertWorkspaceAndUser(ctx context.Context, p insertParams) error {
	now := time.Now().UTC().Format(time.RFC3339)
	slug := generateSlug(p.workspaceName, p.workspaceID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, p.workspaceID, p.workspaceName, slug, now, now)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_account (id, workspace_id, email, password_hash, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, ?)
	`, p.userID, p.workspaceID, p.email, p.passwordHash, p.displayName, now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return tx.Commit()
}

// Login verifies credentials and returns a JWT.
// Task 1.6.8: Always returns ErrInvalidCredentials for any failure (email not found OR wrong password)
// to avoid revealing whether the email exists (security).
func (s *authService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	var userID, workspaceID string
	var passwordHash sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, password_hash
		FROM user_account
		WHERE email = ? AND status = 'active'
		LIMIT 1
	`, input.Email).Scan(&userID, &workspaceID, &passwordHash)

	if err != nil {
		// Whether the user doesn't exist or there's a DB error, return generic message
		s.logAuthFailure(ctx, "unknown", "unknown", "login", "user_not_found_or_query_error")
		return nil, ErrInvalidCredentials
	}

	// User found but has no password hash (OIDC-only account)
	if !passwordHash.Valid || passwordHash.String == "" {
		s.logAuthFailure(ctx, workspaceID, userID, "login", "missing_password_hash")
		return nil, ErrInvalidCredentials
	}

	// Verify password (constant-time comparison via bcrypt)
	if !pkgauth.VerifyPassword(passwordHash.String, input.Password) {
		s.logAuthFailure(ctx, workspaceID, userID, "login", "invalid_password")
		return nil, ErrInvalidCredentials
	}

	// Credentials valid — issue JWT
	token, err := pkgauth.GenerateJWT(userID, workspaceID)
	if err != nil {
		s.logAuthFailure(ctx, workspaceID, userID, "login", "jwt_generation_failed")
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	s.logAuthSuccess(ctx, workspaceID, userID, "login")

	return &AuthResult{
		Token:       token,
		UserID:      userID,
		WorkspaceID: workspaceID,
	}, nil
}

// generateSlug creates a URL-safe workspace slug from the name + a short ID suffix.
// generateSlug creates a URL-safe workspace slug from the name + the full workspace ID.
// Task 1.6.8: Full ID used as suffix to guarantee uniqueness even for identical names.
// Task 1.6.11 fix: Replaced first-8-chars truncation — UUID v7 timestamps are identical
// for workspaces created within the same millisecond, causing UNIQUE constraint failures.
// slugChar maps a single rune to its slug representation.
// Returns the lowercase char for letters, digit as-is, '-' for spaces/dashes, or -1 to skip.
// Extracted from generateSlug to reduce cyclomatic complexity (each case is 1 branch).
func slugChar(c rune) rune {
	switch {
	case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return c
	case c >= 'A' && c <= 'Z':
		return c + 32 // to lower
	case c == ' ', c == '-':
		return '-'
	default:
		return -1 // skip
	}
}

func generateSlug(name, id string) string {
	// strings.Map calls slugChar for each rune; -1 means drop the character.
	slug := strings.Map(slugChar, name)
	// Use full UUID as suffix — guarantees uniqueness regardless of timing
	return slug + "-" + id
}

// isUniqueViolation checks if an SQLite error is a UNIQUE constraint violation.
// Task 1.6.8: SQLite surfaces this as error message containing "UNIQUE constraint failed".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *authService) logAuthSuccess(ctx context.Context, workspaceID, userID, action string) {
	if s.auditLogger == nil {
		return
	}
	_ = s.auditLogger.LogWithDetails(
		ctx,
		workspaceID,
		userID,
		domainaudit.ActorTypeUser,
		action,
		nil,
		nil,
		nil,
		domainaudit.OutcomeSuccess,
	)
}

func (s *authService) logAuthFailure(ctx context.Context, workspaceID, userID, action, reason string) {
	if s.auditLogger == nil {
		return
	}
	_ = s.auditLogger.LogWithDetails(
		ctx,
		workspaceID,
		userID,
		domainaudit.ActorTypeUser,
		action,
		nil,
		nil,
		&domainaudit.EventDetails{Metadata: map[string]any{"reason": reason}},
		domainaudit.OutcomeError,
	)
}
