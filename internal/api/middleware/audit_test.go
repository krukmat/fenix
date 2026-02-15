// Traces: FR-070, NFR-031
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
)

type fakeAuditLogger struct {
	called     int
	workspace  string
	actorID    string
	actorType  domainaudit.ActorType
	action     string
	entityType *string
	entityID   *string
	outcome    domainaudit.Outcome
	details    *domainaudit.EventDetails
}

func (f *fakeAuditLogger) LogWithDetails(
	_ context.Context,
	workspaceID string,
	actorID string,
	actorType domainaudit.ActorType,
	action string,
	entityType *string,
	entityID *string,
	details *domainaudit.EventDetails,
	outcome domainaudit.Outcome,
) error {
	f.called++
	f.workspace = workspaceID
	f.actorID = actorID
	f.actorType = actorType
	f.action = action
	f.entityType = entityType
	f.entityID = entityID
	f.details = details
	f.outcome = outcome
	return nil
}

func TestAuditMiddleware_NoLogger_PassesThrough(t *testing.T) {
	t.Parallel()

	nextCalled := false
	h := AuditMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil))

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestAuditMiddleware_MissingWorkspace_PassesWithoutAudit(t *testing.T) {
	t.Parallel()

	logger := &fakeAuditLogger{}
	nextCalled := false
	h := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req = req.WithContext(ctxkeys.WithValue(req.Context(), ctxkeys.UserID, "user-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if logger.called != 0 {
		t.Fatalf("expected no audit log calls, got %d", logger.called)
	}
}

func TestAuditMiddleware_MissingUser_PassesWithoutAudit(t *testing.T) {
	t.Parallel()

	logger := &fakeAuditLogger{}
	nextCalled := false
	h := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req = req.WithContext(ctxkeys.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if logger.called != 0 {
		t.Fatalf("expected no audit log calls, got %d", logger.called)
	}
}

func TestAuditMiddleware_LogsActionAndOutcome(t *testing.T) {
	t.Parallel()

	logger := &fakeAuditLogger{}
	h := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", nil)
	ctx := ctxkeys.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws-1")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "user-1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if logger.called != 1 {
		t.Fatalf("expected 1 audit log call, got %d", logger.called)
	}
	if logger.workspace != "ws-1" || logger.actorID != "user-1" {
		t.Fatalf("unexpected workspace/user: %q/%q", logger.workspace, logger.actorID)
	}
	if logger.actorType != domainaudit.ActorTypeUser {
		t.Fatalf("unexpected actor type: %q", logger.actorType)
	}
	if logger.action != "create_account" {
		t.Fatalf("unexpected action: %q", logger.action)
	}
	if logger.entityType == nil || *logger.entityType != "account" {
		t.Fatalf("unexpected entityType: %v", logger.entityType)
	}
	if logger.entityID != nil {
		t.Fatalf("expected nil entityID for collection, got %v", *logger.entityID)
	}
	if logger.outcome != domainaudit.OutcomeSuccess {
		t.Fatalf("unexpected outcome: %q", logger.outcome)
	}
	if logger.details == nil || logger.details.Metadata == nil {
		t.Fatal("expected metadata in details")
	}
}

func TestStatusRecorder_WriteHeader(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rr, statusCode: http.StatusOK}
	sr.WriteHeader(http.StatusTeapot)

	if sr.statusCode != http.StatusTeapot {
		t.Fatalf("expected statusCode %d, got %d", http.StatusTeapot, sr.statusCode)
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected response %d, got %d", http.StatusTeapot, rr.Code)
	}
}

func TestGetStringContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if _, ok := getStringContext(ctx, ctxkeys.UserID); ok {
		t.Fatal("expected false when key missing")
	}

	ctx = context.WithValue(ctx, ctxkeys.UserID, 123)
	if _, ok := getStringContext(ctx, ctxkeys.UserID); ok {
		t.Fatal("expected false when value is not string")
	}

	ctx = context.WithValue(ctx, ctxkeys.UserID, "")
	if _, ok := getStringContext(ctx, ctxkeys.UserID); ok {
		t.Fatal("expected false for empty string")
	}

	ctx = context.WithValue(ctx, ctxkeys.UserID, "user-1")
	if got, ok := getStringContext(ctx, ctxkeys.UserID); !ok || got != "user-1" {
		t.Fatalf("expected user-1/true, got %q/%v", got, ok)
	}
}

func TestOutcomeFromStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		want   domainaudit.Outcome
	}{
		{http.StatusOK, domainaudit.OutcomeSuccess},
		{http.StatusNoContent, domainaudit.OutcomeSuccess},
		{http.StatusUnauthorized, domainaudit.OutcomeDenied},
		{http.StatusForbidden, domainaudit.OutcomeDenied},
		{http.StatusBadRequest, domainaudit.OutcomeError},
		{http.StatusInternalServerError, domainaudit.OutcomeError},
	}

	for _, tt := range tests {
		if got := outcomeFromStatus(tt.status); got != tt.want {
			t.Fatalf("status=%d got=%q want=%q", tt.status, got, tt.want)
		}
	}
}

func TestActionFromRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantAction string
		wantType   *string
		wantID     *string
	}{
		{"fallback invalid path", http.MethodGet, "/health", "get_request", nil, nil},
		{"unknown entity", http.MethodGet, "/api/v1/unknown", "get_request", nil, nil},
		{"collection post", http.MethodPost, "/api/v1/accounts", "create_account", strPtr("account"), nil},
		{"collection get", http.MethodGet, "/api/v1/notes", "list_note", strPtr("note"), nil},
		{"entity get", http.MethodGet, "/api/v1/accounts/a1", "get_account", strPtr("account"), strPtr("a1")},
		{"entity put", http.MethodPut, "/api/v1/deals/d1", "update_deal", strPtr("deal"), strPtr("d1")},
		{"entity patch", http.MethodPatch, "/api/v1/cases/c1", "update_case", strPtr("case"), strPtr("c1")},
		{"entity delete", http.MethodDelete, "/api/v1/contacts/c1", "delete_contact", strPtr("contact"), strPtr("c1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, typ, id := actionFromRequest(tt.method, tt.path)
			if action != tt.wantAction {
				t.Fatalf("action got=%q want=%q", action, tt.wantAction)
			}

			if (typ == nil) != (tt.wantType == nil) {
				t.Fatalf("entityType nil mismatch: got=%v want=%v", typ == nil, tt.wantType == nil)
			}
			if typ != nil && *typ != *tt.wantType {
				t.Fatalf("entityType got=%q want=%q", *typ, *tt.wantType)
			}

			if (id == nil) != (tt.wantID == nil) {
				t.Fatalf("entityID nil mismatch: got=%v want=%v", id == nil, tt.wantID == nil)
			}
			if id != nil && *id != *tt.wantID {
				t.Fatalf("entityID got=%q want=%q", *id, *tt.wantID)
			}
		})
	}
}

func TestSingularEntity(t *testing.T) {
	t.Parallel()

	if got := singularEntity("accounts"); got != "account" {
		t.Fatalf("expected account, got %q", got)
	}
	if got := singularEntity("does-not-exist"); got != "" {
		t.Fatalf("expected empty for unknown entity, got %q", got)
	}
}

func TestActionHelpers(t *testing.T) {
	t.Parallel()

	if got := actionForCollection(http.MethodPost, "account"); got != "create_account" {
		t.Fatalf("unexpected collection post action: %q", got)
	}
	if got := actionForCollection(http.MethodGet, "account"); got != "list_account" {
		t.Fatalf("unexpected collection get action: %q", got)
	}
	if got := actionForCollection(http.MethodPut, "account"); got != "put_account" {
		t.Fatalf("unexpected collection fallback action: %q", got)
	}

	if got := actionForEntity(http.MethodGet, "account"); got != "get_account" {
		t.Fatalf("unexpected entity get action: %q", got)
	}
	if got := actionForEntity(http.MethodPut, "account"); got != "update_account" {
		t.Fatalf("unexpected entity put action: %q", got)
	}
	if got := actionForEntity(http.MethodPatch, "account"); got != "update_account" {
		t.Fatalf("unexpected entity patch action: %q", got)
	}
	if got := actionForEntity(http.MethodDelete, "account"); got != "delete_account" {
		t.Fatalf("unexpected entity delete action: %q", got)
	}
	if got := actionForEntity(http.MethodPost, "account"); got != "create_account" {
		t.Fatalf("unexpected entity post action: %q", got)
	}
	if got := actionForEntity(http.MethodOptions, "account"); got != "options_account" {
		t.Fatalf("unexpected entity fallback action: %q", got)
	}
}

func TestStrPtr(t *testing.T) {
	t.Parallel()

	if got := strPtr("x"); got == nil || *got != "x" {
		t.Fatalf("unexpected ptr result: %v", got)
	}
}
