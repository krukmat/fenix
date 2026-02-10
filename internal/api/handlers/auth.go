// Task 1.6.12: HTTP handlers for register + login (public endpoints — no AuthMiddleware)
// Translates HTTP requests into domain/auth.AuthService calls and maps domain errors to HTTP codes.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

// AuthHandler handles authentication HTTP requests (register and login).
// Task 1.6.12: Public endpoints — no workspace or JWT context required.
type AuthHandler struct {
	authService domainauth.AuthService
}

// NewAuthHandler creates a new AuthHandler backed by the provided AuthService.
func NewAuthHandler(authService domainauth.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRequest is the request body for POST /auth/register.
// Task 1.6.12: WorkspaceName creates the tenant; Email is unique login identifier.
type RegisterRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"displayName"`
	WorkspaceName string `json:"workspaceName"`
}

// LoginRequest is the request body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is the response body returned after successful register or login.
// Task 1.6.12: camelCase JSON to match frontend conventions (userId, workspaceId).
type AuthResponse struct {
	Token       string `json:"token"`
	UserID      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

// Register handles POST /auth/register.
// Task 1.6.12: Creates a new workspace + user, returns JWT token.
//
// Response codes:
//   - 201 Created: registration successful
//   - 400 Bad Request: invalid JSON or missing required fields
//   - 409 Conflict: email already registered
//   - 500 Internal Server Error: unexpected failure
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateRegisterRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.authService.Register(r.Context(), domainauth.RegisterInput{
		Email:         req.Email,
		Password:      req.Password,
		DisplayName:   req.DisplayName,
		WorkspaceName: req.WorkspaceName,
	})
	if err != nil {
		if errors.Is(err, domainauth.ErrEmailAlreadyExists) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{ //nolint:errcheck
		Token:       result.Token,
		UserID:      result.UserID,
		WorkspaceID: result.WorkspaceID,
	})
}

// Login handles POST /auth/login.
// Task 1.6.12: Verifies credentials, returns JWT token.
//
// Response codes:
//   - 200 OK: login successful
//   - 400 Bad Request: invalid JSON or missing required fields
//   - 401 Unauthorized: invalid credentials (generic — doesn't reveal if email exists)
//   - 500 Internal Server Error: unexpected failure
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateLoginRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.authService.Login(r.Context(), domainauth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domainauth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AuthResponse{ //nolint:errcheck
		Token:       result.Token,
		UserID:      result.UserID,
		WorkspaceID: result.WorkspaceID,
	})
}

// validateRegisterRequest checks required fields for the register endpoint.
// Task 1.6.12: Extracted to reduce cyclomatic complexity of Register handler.
func validateRegisterRequest(req RegisterRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if req.WorkspaceName == "" {
		return errors.New("workspaceName is required")
	}
	return nil
}

// validateLoginRequest checks required fields for the login endpoint.
// Task 1.6.12: Extracted to reduce cyclomatic complexity of Login handler.
func validateLoginRequest(req LoginRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	return nil
}
