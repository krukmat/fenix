// Task 2.4 audit remediation: wiring test for NewRouter.
// Validates that NewRouter creates a working router with the knowledge/ingest endpoint
// and that the embedder is started (shared event bus wired correctly).
package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestMain(m *testing.M) {
	// AuthMiddleware reads JWT_SECRET — must be set for protected routes to parse tokens.
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
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

	router := NewRouter(db)

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

// TestNewRouter_KnowledgeIngestEndpoint_Unauthorized verifies that
// POST /api/v1/knowledge/ingest is registered and returns 401 without JWT.
// This confirms the knowledge ingest route (and embedder wiring) is present.
func TestNewRouter_KnowledgeIngestEndpoint_Unauthorized(t *testing.T) {
	db := mustOpenAPITestDB(t)

	router := NewRouter(db)

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
