// Task 2.4 audit remediation: wiring test for NewRouter.
// Validates that NewRouter creates a working router with the knowledge/ingest endpoint
// and that the embedder is started (shared event bus wired correctly).
// C2/C3/C4: integration tests for CORS, password validation, and rate limiting.
package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/infra/config"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestMain(m *testing.M) {
	// AuthMiddleware reads JWT_SECRET — must be set for protected routes to parse tokens.
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
}

// mustNewRouter wraps NewRouter and fails the test if construction fails.
func mustNewRouter(t *testing.T, db *sql.DB) *chi.Mux {
	t.Helper()
	r, err := NewRouter(db)
	if err != nil {
		t.Fatalf("mustNewRouter: %v", err)
	}
	return r
}

// mustNewRouterWithConfig wraps newRouterWithConfig and fails the test if construction fails.
func mustNewRouterWithConfig(t *testing.T, db *sql.DB, cfg config.Config) *chi.Mux {
	t.Helper()
	r, err := newRouterWithConfig(db, cfg)
	if err != nil {
		t.Fatalf("mustNewRouterWithConfig: %v", err)
	}
	return r
}

// mustOpenAPITestDB opens an in-memory SQLite DB with all migrations applied.
func mustOpenAPITestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("mustOpenAPITestDB: NewDB: %v", err)
	}
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("mustOpenAPITestDB: MigrateUp: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestNewRouter_HealthEndpoint verifies that NewRouter registers the /health route.
func TestNewRouter_HealthEndpoint(t *testing.T) {
	db := mustOpenAPITestDB(t)

	router := mustNewRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 from /health, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ok") {
		t.Errorf("expected body to contain 'ok', got %q", w.Body.String())
	}
}

func TestNewRouter_ReadyzEndpoint(t *testing.T) {
	db := mustOpenAPITestDB(t)
	cfg := testCfg()
	cfg.OllamaBaseURL = "http://127.0.0.1:1"

	router := mustNewRouterWithConfig(t, db, cfg)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// LLM providers are optional — degraded but DB is up, so 200 not 503.
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 from /readyz when only LLM is down, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"degraded"`) {
		t.Errorf("expected degraded body, got %q", w.Body.String())
	}
}

// TestNewRouter_KnowledgeIngestEndpoint_Unauthorized verifies that
// POST /api/v1/knowledge/ingest is registered and returns 401 without JWT.
// This confirms the knowledge ingest route (and embedder wiring) is present.
func TestNewRouter_KnowledgeIngestEndpoint_Unauthorized(t *testing.T) {
	db := mustOpenAPITestDB(t)

	router := mustNewRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest",
		strings.NewReader(`{"title":"test","raw_content":"hello","source_type":"document"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Without JWT, AuthMiddleware must reject with 401.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated /api/v1/knowledge/ingest, got %d", w.Code)
	}
}

// ===== C2: CORS integration tests =====

// testCfg returns a Config with a known BFFOrigin for router-level tests.
func testCfg() config.Config {
	return config.Config{
		LLMProvider:        "ollama",
		OllamaBaseURL:      "http://localhost:11434",
		OllamaModel:        "nomic-embed-text",
		OllamaChatModel:    "llama3.2:3b",
		BFFOrigin:          "http://localhost:3000",
		CORSAllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
	}
}

// TestRouter_CORS_AllowedOrigin_ReceivesHeaders verifies ACAO header for BFF origin.
func TestRouter_CORS_AllowedOrigin_ReceivesHeaders(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := mustNewRouterWithConfig(t, db, testCfg())

	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q; want %q", got, "http://localhost:3000")
	}
}

func TestRouter_CORS_LocalDevOrigin_ReceivesHeaders(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := mustNewRouterWithConfig(t, db, testCfg())

	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q; want %q", got, "http://localhost:5173")
	}
}

// TestRouter_CORS_BlockedOrigin_NoHeaders verifies no ACAO header for unlisted origin.
func TestRouter_CORS_BlockedOrigin_NoHeaders(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := mustNewRouterWithConfig(t, db, testCfg())

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://attacker.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty for blocked origin", got)
	}
}

// ===== C3: Password minimum length integration tests =====

// registerBody encodes a register JSON payload.
func registerBody(email, password, displayName, workspaceName string) *bytes.Reader {
	b, _ := json.Marshal(map[string]string{
		"email":         email,
		"password":      password,
		"displayName":   displayName,
		"workspaceName": workspaceName,
	})
	return bytes.NewReader(b)
}

// TestRouter_Register_ShortPassword_Returns400 verifies 400 for passwords < 12 chars.
func TestRouter_Register_ShortPassword_Returns400(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := mustNewRouterWithConfig(t, db, testCfg())

	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		registerBody("short@test.com", "Short1!", "Short", "TestCo"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("short password: status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRouter_Register_ValidPassword_Returns201 verifies 201 for passwords >= 12 chars.
func TestRouter_Register_ValidPassword_Returns201(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := mustNewRouterWithConfig(t, db, testCfg())

	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		registerBody("valid@test.com", "ValidPassword1!", "Valid", "TestCo"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("valid password: status = %d; want %d. body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

// ===== C4: Rate limiting integration tests =====

// TestRouter_Login_RateLimit_Returns429 verifies 429 after exceeding login rate limit.
// The login limiter allows 5 req/min per IP; we send 6 from the same IP.
func TestRouter_Login_RateLimit_Returns429(t *testing.T) {
	db := mustOpenAPITestDB(t)
	// Use a custom config so this test is isolated from other test router instances.
	cfg := testCfg()
	router := mustNewRouterWithConfig(t, db, cfg)

	loginJSON := strings.NewReader(`{"email":"x@x.com","password":"pass"}`)

	// Register first so login requests get proper 401 (not 500).
	// (We only care about 429, so any non-429 for first 5 is acceptable.)
	const loginLimit = 5
	ip := "10.1.2.3"

	for i := 0; i < loginLimit; i++ {
		loginJSON = strings.NewReader(`{"email":"x@x.com","password":"pass"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", loginJSON)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":9999"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d unexpectedly rate-limited before limit reached", i+1)
		}
	}

	// The (limit+1)th request should be rate-limited.
	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"x@x.com","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":9999"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("6th login request: status = %d; want %d", w.Code, http.StatusTooManyRequests)
	}
}

// TestRouter_Register_RateLimit_Returns429 verifies 429 after exceeding register rate limit.
// The register limiter allows 3 req/hour per IP; we send 4 from the same IP.
func TestRouter_Register_RateLimit_Returns429(t *testing.T) {
	db := mustOpenAPITestDB(t)
	cfg := testCfg()
	router := mustNewRouterWithConfig(t, db, cfg)

	const registerLimit = 3
	ip := "10.2.3.4"

	for i := 0; i < registerLimit; i++ {
		body := registerBody(
			"user"+string(rune('a'+i))+"@test.com",
			"ValidPassword1!",
			"User",
			"TestCo"+string(rune('a'+i)),
		)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":9999"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d unexpectedly rate-limited before limit reached", i+1)
		}
	}

	// The (limit+1)th request should be rate-limited.
	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		registerBody("extra@test.com", "ValidPassword1!", "Extra", "ExtraCo"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":9999"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th register request: status = %d; want %d", w.Code, http.StatusTooManyRequests)
	}
}
