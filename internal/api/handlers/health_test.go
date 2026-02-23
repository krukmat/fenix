// Task 4.9 — NFR-030: Health check tests
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_DBOk(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	handler := NewHealthHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}
	if !contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("body missing status ok: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"database":"ok"`) {
		t.Errorf("body missing database ok: %s", w.Body.String())
	}
}

func TestHealthHandler_DBError(t *testing.T) {
	t.Parallel()
	// NOTE: modernc.org/sqlite is pure Go and can open files even if they don't exist.
	// To properly test the error case, we use a closed DB connection.
	db := mustOpenDBWithMigrations(t)
	db.Close() // Close the DB to force Ping to fail

	handler := NewHealthHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d; want %d", w.Code, http.StatusServiceUnavailable)
	}
	if !contains(w.Body.String(), `"status":"degraded"`) {
		t.Errorf("body missing status degraded: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"database":"error"`) {
		t.Errorf("body missing database error: %s", w.Body.String())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
