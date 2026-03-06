package handlers

import (
	"context"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

type ActionAuthorizer interface {
	CheckActionPermission(
		ctx context.Context,
		userID, resource, action string,
		attrs map[string]string,
	) (bool, error)
}

func checkActionAuthorization(
	w http.ResponseWriter,
	r *http.Request,
	authz ActionAuthorizer,
	resource, action string,
) bool {
	if authz == nil {
		return true
	}

	userID, ok := r.Context().Value(ctxkeys.UserID).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user_id in context")
		return false
	}

	allowed, err := authz.CheckActionPermission(r.Context(), userID, resource, action, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authorization failed")
		return false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}

	return true
}
