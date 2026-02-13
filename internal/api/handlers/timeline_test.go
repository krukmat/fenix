package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestTimelineHandler_ListTimeline_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewTimelineService(db)
	handler := NewTimelineHandler(svc)

	for i := 0; i < 2; i++ {
		_, err := svc.Create(t.Context(), crm.CreateTimelineEventInput{
			WorkspaceID: wsID,
			EntityType:  "account",
			EntityID:    accountID,
			ActorID:     ownerID,
			EventType:   "created",
		})
		if err != nil {
			t.Fatalf("seed timeline event failed: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timeline?limit=10&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.ListTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Fatalf("expected data field")
	}
}

func TestTimelineHandler_ListTimeline_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewTimelineHandler(crm.NewTimelineService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timeline", nil)
	rr := httptest.NewRecorder()
	handler.ListTimeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestTimelineHandler_ListTimelineByEntity_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewTimelineService(db)
	handler := NewTimelineHandler(svc)

	_, err := svc.Create(t.Context(), crm.CreateTimelineEventInput{
		WorkspaceID: wsID,
		EntityType:  "account",
		EntityID:    accountID,
		ActorID:     ownerID,
		EventType:   "created",
	})
	if err != nil {
		t.Fatalf("seed timeline event failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timeline/account/"+accountID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("entity_type", "account")
	rctx.URLParams.Add("entity_id", accountID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ListTimelineByEntity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
