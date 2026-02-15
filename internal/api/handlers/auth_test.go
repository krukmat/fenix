// Task 1.6.11: TDD tests for Auth HTTP handlers (register + login)
// Tests run against a real in-memory SQLite DB — no mocking.
// Covers: success paths, error paths, response shape, status codes.
// Traces: FR-060
package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// TestMain sets package-level environment variables needed by auth tests.
// Task 1.6.11: JWT_SECRET must be set before GenerateJWT is called (it panics otherwise).
// Using TestMain (instead of t.Setenv) allows t.Parallel() across all auth tests.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
}

// ===== TEST HELPERS (auth-specific) =====

// mustOpenAuthDB opens in-memory SQLite with all migrations applied.
// Separate helper so auth tests are self-contained (don't rely on account_test.go helpers).
func mustOpenAuthDB(t *testing.T) *sql.DB {
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

// newAuthHandler creates an AuthHandler wired to a real AuthService.
func newAuthHandler(db *sql.DB) *AuthHandler {
	return NewAuthHandler(domainauth.NewAuthService(db))
}

// registerPayload is the JSON body for POST /auth/register.
type registerPayload struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"displayName"`
	WorkspaceName string `json:"workspaceName"`
}

// loginPayload is the JSON body for POST /auth/login.
type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse is the expected success body returned by both endpoints.
type authResponse struct {
	Token       string `json:"token"`
	UserID      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

// postRequest builds a POST request with JSON body.
func postRequest(t *testing.T, path string, body interface{}) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ===== REGISTER TESTS =====

// TestAuthHandler_Register_Success verifies 201 + token returned on valid input.
func TestAuthHandler_Register_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/register", registerPayload{
		Email:         "alice@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Alice",
		WorkspaceName: "Acme Corp",
	})
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Register status = %d; want %d. body: %s", rr.Code, http.StatusCreated, rr.Body.String())
	}
}

// TestAuthHandler_Register_ResponseShape verifies response body has expected fields.
func TestAuthHandler_Register_ResponseShape(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/register", registerPayload{
		Email:         "bob@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Bob",
		WorkspaceName: "Acme Corp",
	})
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	var resp authResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error = %v", err)
	}

	if resp.Token == "" {
		t.Error("response Token is empty; want JWT string")
	}
	if resp.UserID == "" {
		t.Error("response UserID is empty; want non-empty ID")
	}
	if resp.WorkspaceID == "" {
		t.Error("response WorkspaceID is empty; want non-empty ID")
	}
}

// TestAuthHandler_Register_ContentTypeJSON verifies Content-Type header on success.
func TestAuthHandler_Register_ContentTypeJSON(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/register", registerPayload{
		Email:         "carol@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Carol",
		WorkspaceName: "Acme Corp",
	})
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q; want %q", ct, "application/json")
	}
}

// TestAuthHandler_Register_DuplicateEmail verifies 409 when email is already registered.
func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	payload := registerPayload{
		Email:         "dup@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Dup",
		WorkspaceName: "Acme Corp",
	}

	// First registration should succeed
	h.Register(httptest.NewRecorder(), postRequest(t, "/auth/register", payload))

	// Second registration with same email
	rr := httptest.NewRecorder()
	h.Register(rr, postRequest(t, "/auth/register", payload))

	if rr.Code != http.StatusConflict {
		t.Errorf("Register duplicate status = %d; want %d", rr.Code, http.StatusConflict)
	}
}

// TestAuthHandler_Register_MissingEmail verifies 400 when email is absent.
func TestAuthHandler_Register_MissingEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/register", registerPayload{
		Email:         "",
		Password:      "SecurePass123!",
		DisplayName:   "Dave",
		WorkspaceName: "Acme Corp",
	})
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Register missing email status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestAuthHandler_Register_MissingPassword verifies 400 when password is absent.
func TestAuthHandler_Register_MissingPassword(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/register", registerPayload{
		Email:         "eve@acme.com",
		Password:      "",
		DisplayName:   "Eve",
		WorkspaceName: "Acme Corp",
	})
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Register missing password status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestAuthHandler_Register_MissingWorkspaceName verifies 400 when workspace name is absent.
func TestAuthHandler_Register_MissingWorkspaceName(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/register", registerPayload{
		Email:         "frank@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Frank",
		WorkspaceName: "",
	})
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Register missing workspace name status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestAuthHandler_Register_InvalidJSON verifies 400 on malformed request body.
func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Register invalid JSON status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// ===== LOGIN TESTS =====

// TestAuthHandler_Login_Success verifies 200 + token returned on valid credentials.
func TestAuthHandler_Login_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	// Register first
	h.Register(httptest.NewRecorder(), postRequest(t, "/auth/register", registerPayload{
		Email:         "grace@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Grace",
		WorkspaceName: "Acme Corp",
	}))

	req := postRequest(t, "/auth/login", loginPayload{
		Email:    "grace@acme.com",
		Password: "SecurePass123!",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Login status = %d; want %d. body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

// TestAuthHandler_Login_ResponseShape verifies response body has expected fields.
func TestAuthHandler_Login_ResponseShape(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	h.Register(httptest.NewRecorder(), postRequest(t, "/auth/register", registerPayload{
		Email:         "hank@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Hank",
		WorkspaceName: "Acme Corp",
	}))

	req := postRequest(t, "/auth/login", loginPayload{
		Email: "hank@acme.com", Password: "SecurePass123!",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	var resp authResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error = %v", err)
	}

	if resp.Token == "" {
		t.Error("Login response Token is empty; want JWT string")
	}
	if resp.UserID == "" {
		t.Error("Login response UserID is empty; want non-empty ID")
	}
	if resp.WorkspaceID == "" {
		t.Error("Login response WorkspaceID is empty; want non-empty ID")
	}
}

// TestAuthHandler_Login_WrongPassword verifies 401 on wrong password.
func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	h.Register(httptest.NewRecorder(), postRequest(t, "/auth/register", registerPayload{
		Email:         "ivan@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Ivan",
		WorkspaceName: "Acme Corp",
	}))

	req := postRequest(t, "/auth/login", loginPayload{
		Email: "ivan@acme.com", Password: "WrongPassword!",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Login wrong password status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}
}

// TestAuthHandler_Login_NonExistentEmail verifies 401 on unknown email.
func TestAuthHandler_Login_NonExistentEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/login", loginPayload{
		Email: "nobody@acme.com", Password: "SomePass!",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Login non-existent email status = %d; want %d", rr.Code, http.StatusUnauthorized)
	}
}

// TestAuthHandler_Login_MissingEmail verifies 400 when email is absent.
func TestAuthHandler_Login_MissingEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/login", loginPayload{
		Email: "", Password: "SecurePass123!",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Login missing email status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestAuthHandler_Login_MissingPassword verifies 400 when password is absent.
func TestAuthHandler_Login_MissingPassword(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := postRequest(t, "/auth/login", loginPayload{
		Email: "judy@acme.com", Password: "",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Login missing password status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestAuthHandler_Login_InvalidJSON verifies 400 on malformed request body.
func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Login invalid JSON status = %d; want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestAuthHandler_Login_ContentTypeJSON verifies Content-Type header on success.
func TestAuthHandler_Login_ContentTypeJSON(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	h.Register(httptest.NewRecorder(), postRequest(t, "/auth/register", registerPayload{
		Email:         "kate@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Kate",
		WorkspaceName: "Acme Corp",
	}))

	req := postRequest(t, "/auth/login", loginPayload{
		Email: "kate@acme.com", Password: "SecurePass123!",
	})
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q; want %q", ct, "application/json")
	}
}

// TestAuthHandler_Login_IDsMatchRegistration verifies UserID+WorkspaceID match registration.
func TestAuthHandler_Login_IDsMatchRegistration(t *testing.T) {
	t.Parallel()

	db := mustOpenAuthDB(t)
	h := newAuthHandler(db)

	// Register and capture the IDs
	regRR := httptest.NewRecorder()
	h.Register(regRR, postRequest(t, "/auth/register", registerPayload{
		Email:         "leo@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Leo",
		WorkspaceName: "Acme Corp",
	}))

	var regResp authResponse
	if err := json.NewDecoder(regRR.Body).Decode(&regResp); err != nil {
		t.Fatalf("decode register response error = %v", err)
	}

	// Login and compare IDs
	loginRR := httptest.NewRecorder()
	h.Login(loginRR, postRequest(t, "/auth/login", loginPayload{
		Email: "leo@acme.com", Password: "SecurePass123!",
	}))

	var loginResp authResponse
	if err := json.NewDecoder(loginRR.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response error = %v", err)
	}

	if loginResp.UserID != regResp.UserID {
		t.Errorf("Login UserID = %q; want %q", loginResp.UserID, regResp.UserID)
	}
	if loginResp.WorkspaceID != regResp.WorkspaceID {
		t.Errorf("Login WorkspaceID = %q; want %q", loginResp.WorkspaceID, regResp.WorkspaceID)
	}
}
