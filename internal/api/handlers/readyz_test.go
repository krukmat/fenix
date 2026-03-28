package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

type readyzStubProvider struct {
	healthErr error
}

func (s *readyzStubProvider) ChatCompletion(context.Context, llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, nil
}

func (s *readyzStubProvider) Embed(context.Context, llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return nil, nil
}

func (s *readyzStubProvider) ModelInfo() llm.ModelMeta {
	return llm.ModelMeta{}
}

func (s *readyzStubProvider) HealthCheck(context.Context) error {
	return s.healthErr
}

func TestReadyzHandler_AllOK(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewReadyzHandler(db, &readyzStubProvider{}, &readyzStubProvider{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusOK)
	}
	if !contains(w.Body.String(), `"status":"ready"`) {
		t.Fatalf("body missing ready status: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"database":"ok"`) || !contains(w.Body.String(), `"chat":"ok"`) || !contains(w.Body.String(), `"embed":"ok"`) {
		t.Fatalf("body missing ready component states: %s", w.Body.String())
	}
}

func TestReadyzHandler_DBDown(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	db.Close()
	handler := NewReadyzHandler(db, &readyzStubProvider{}, &readyzStubProvider{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusServiceUnavailable)
	}
	if !contains(w.Body.String(), `"database":"error"`) {
		t.Fatalf("body missing database error: %s", w.Body.String())
	}
}

func TestReadyzHandler_ChatDown(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewReadyzHandler(db, &readyzStubProvider{healthErr: errors.New("chat down")}, &readyzStubProvider{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	// Chat is an optional provider — system stays operable, so 200 with degraded status.
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusOK)
	}
	if !contains(w.Body.String(), `"chat":"error"`) {
		t.Fatalf("body missing chat error: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"status":"degraded"`) {
		t.Fatalf("body missing degraded status: %s", w.Body.String())
	}
}

func TestReadyzHandler_EmbedDown(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewReadyzHandler(db, &readyzStubProvider{}, &readyzStubProvider{healthErr: errors.New("embed down")})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	// Embed is an optional provider — system stays operable, so 200 with degraded status.
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusOK)
	}
	if !contains(w.Body.String(), `"embed":"error"`) {
		t.Fatalf("body missing embed error: %s", w.Body.String())
	}
	if !contains(w.Body.String(), `"status":"degraded"`) {
		t.Fatalf("body missing degraded status: %s", w.Body.String())
	}
}
