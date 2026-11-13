//go:build s2s_nonproduction

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestB4W1S2SRouterFailsClosedUnlessExplicitlyOptedIn(t *testing.T) {
	db := mustOpenAPITestDB(t)
	for _, tc := range []struct {
		name string
		db   bool
		cfg  S2SNonProductionRouteConfig
	}{
		{"zero value", true, S2SNonProductionRouteConfig{}},
		{"environment without enablement", true, S2SNonProductionRouteConfig{Environment: "integration-test"}},
		{"enabled outside test environment", true, S2SNonProductionRouteConfig{Enabled: true, Environment: "production"}},
		{"enabled without database", false, S2SNonProductionRouteConfig{Enabled: true, Environment: "integration-test"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var routerDB = db
			if !tc.db {
				routerDB = nil
			}
			router := NewNonProductionS2SRouter(routerDB, tc.cfg)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, testS2SRequest(t, testS2SToken(t)))
			if w.Code != http.StatusNotFound {
				t.Fatalf("unexpected S2S route activation: status=%d", w.Code)
			}
		})
	}
}

func TestB4W1S2SRouterExplicitTestRouteRetainsServiceAuth(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := NewNonProductionS2SRouter(db, S2SNonProductionRouteConfig{
		Environment: "integration-test",
		Enabled:     true,
	})

	t.Run("missing bearer is denied", func(t *testing.T) {
		req := testS2SRequest(t, testS2SToken(t))
		req.Header.Del("Authorization")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("unprotected test S2S route: %d", w.Code)
		}
	})

	t.Run("valid but unprovisioned bearer never accepts operation", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, testS2SRequest(t, testS2SToken(t)))
		if w.Code == http.StatusNotFound || w.Code == http.StatusAccepted || w.Code == http.StatusOK {
			t.Fatalf("unprovisioned principal accepted or route missing: %d", w.Code)
		}
	})

	t.Run("no status GET extension", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, DormantS2SAdmissionPath, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusOK || w.Code == http.StatusAccepted {
			t.Fatalf("unexpected status endpoint: %d", w.Code)
		}
	})
}
