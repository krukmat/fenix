package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// Task 4.9 — NFR-030: health status values
const (
	healthStatusOK       = "ok"
	healthStatusDegraded = "degraded"
	healthStatusError    = "error"
)

// NewHealthHandler checks DB connectivity and returns structured JSON.
// Task 4.9 — NFR-030: Enriched health check (was: inline {"status":"ok"})
func NewHealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, mimeJSON)

		dbStatus := healthStatusOK
		if err := db.Ping(); err != nil {
			dbStatus = healthStatusError
		}

		status := healthStatusOK
		code := http.StatusOK
		if dbStatus != healthStatusOK {
			status = healthStatusDegraded
			code = http.StatusServiceUnavailable
		}

		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"status":   status,
			"database": dbStatus,
		})
	}
}
