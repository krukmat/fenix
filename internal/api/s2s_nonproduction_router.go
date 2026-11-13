//go:build s2s_nonproduction

package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	apmiddleware "github.com/matiasleandrokruk/fenix/internal/api/middleware"
)

// S2SNonProductionRouteConfig is deliberately separate from the production
// server and router configuration. Neither NewRouter nor NewServer reads it.
type S2SNonProductionRouteConfig struct {
	Environment string
	Enabled     bool
}

// NewNonProductionS2SRouter constructs an isolated, default-OFF test router.
// It is compiled only with -tags=s2s_nonproduction; no production router
// calls it. A real nonproduction runtime must additionally obtain explicit
// operator approval before enabling the route, issuing service credentials or
// provisioning a diagnostic capability.
func NewNonProductionS2SRouter(db *sql.DB, cfg S2SNonProductionRouteConfig) http.Handler {
	router := chi.NewRouter()
	if !cfg.Enabled || cfg.Environment != "integration-test" || db == nil {
		return router
	}

	// The persisted provider checks a dedicated Fenix service principal,
	// bound tenant/workspace, operator policy, input schema and operation ID.
	// The existing human JWT middleware must never guard this independent path.
	admission := NewDormantS2SAdmissionFromPolicies(db)
	router.With(apmiddleware.RateLimitMiddleware(5, time.Minute)).
		Post(DormantS2SAdmissionPath, admission.ServeHTTP)
	return router
}
