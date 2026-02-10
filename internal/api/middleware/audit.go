// Task 1.7: HTTP audit middleware for protected routes.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
)

// AuditLogger is the minimal contract used by AuditMiddleware.
// domainaudit.AuditService satisfies this interface.
type AuditLogger interface {
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

// AuditMiddleware logs protected HTTP requests into audit_event.
// Expected order in router: AuthMiddleware -> AuditMiddleware -> handlers.
func AuditMiddleware(logger AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if logger == nil {
				next.ServeHTTP(w, r)
				return
			}

			workspaceID, ok := getStringContext(r.Context(), ctxkeys.WorkspaceID)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			userID, ok := getStringContext(r.Context(), ctxkeys.UserID)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(recorder, r)

			action, entityType, entityID := actionFromRequest(r.Method, r.URL.Path)
			_ = logger.LogWithDetails(
				r.Context(),
				workspaceID,
				userID,
				domainaudit.ActorTypeUser,
				action,
				entityType,
				entityID,
				&domainaudit.EventDetails{Metadata: map[string]any{
					"method":      r.Method,
					"path":        r.URL.Path,
					"status_code": recorder.statusCode,
					"duration_ms": time.Since(start).Milliseconds(),
				}},
				outcomeFromStatus(recorder.statusCode),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusRecorder) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func getStringContext(ctx context.Context, key ctxkeys.Key) (string, bool) {
	v, ok := ctx.Value(key).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

func outcomeFromStatus(statusCode int) domainaudit.Outcome {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return domainaudit.OutcomeSuccess
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return domainaudit.OutcomeDenied
	default:
		return domainaudit.OutcomeError
	}
}

func actionFromRequest(method, path string) (string, *string, *string) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 3 || segments[0] != "api" || segments[1] != "v1" {
		action := strings.ToLower(method) + "_request"
		return action, nil, nil
	}

	entityType := singularEntity(segments[2])
	if entityType == "" {
		action := strings.ToLower(method) + "_request"
		return action, nil, nil
	}

	if len(segments) == 3 {
		action := actionForCollection(method, entityType)
		return action, strPtr(entityType), nil
	}

	entityID := segments[3]
	action := actionForEntity(method, entityType)
	return action, strPtr(entityType), strPtr(entityID)
}

func singularEntity(entity string) string {
	entityMap := map[string]string{
		"accounts":    "account",
		"contacts":    "contact",
		"leads":       "lead",
		"deals":       "deal",
		"cases":       "case",
		"pipelines":   "pipeline",
		"activities":  "activity",
		"notes":       "note",
		"attachments": "attachment",
		"timeline":    "timeline",
	}

	if value, ok := entityMap[entity]; ok {
		return value
	}
	return ""
}

func actionForCollection(method, entity string) string {
	if method == http.MethodPost {
		return "create_" + entity
	}
	if method == http.MethodGet {
		return "list_" + entity
	}
	return strings.ToLower(method) + "_" + entity
}

func actionForEntity(method, entity string) string {
	if method == http.MethodGet {
		return "get_" + entity
	}
	if method == http.MethodPut || method == http.MethodPatch {
		return "update_" + entity
	}
	if method == http.MethodDelete {
		return "delete_" + entity
	}
	if method == http.MethodPost {
		return "create_" + entity
	}
	return strings.ToLower(method) + "_" + entity
}

func strPtr(v string) *string {
	return &v
}
