package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

const readyStatusReady = "ready"

type readyzResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Chat     string `json:"chat"`
	Embed    string `json:"embed"`
}

type readinessChecker interface {
	HealthCheck(context.Context) error
}

// NewReadyzHandler checks DB, chat provider and embed provider readiness.
func NewReadyzHandler(db *sql.DB, chat, embed readinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, mimeJSON)

		resp := readyzResponse{
			Status:   readyStatusReady,
			Database: healthStatusOK,
			Chat:     healthStatusOK,
			Embed:    healthStatusOK,
		}

		if err := checkDBReady(db); err != nil {
			resp.Status = healthStatusDegraded
			resp.Database = healthStatusError
		}
		if err := checkProviderReady(chat); err != nil {
			resp.Status = healthStatusDegraded
			resp.Chat = healthStatusError
		}
		if err := checkProviderReady(embed); err != nil {
			resp.Status = healthStatusDegraded
			resp.Embed = healthStatusError
		}

		// 503 only when the database is unavailable — the system cannot serve requests.
		// Chat/embed provider failures degrade capability but the API remains operable.
		code := http.StatusOK
		if resp.Database == healthStatusError {
			code = http.StatusServiceUnavailable
		}
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

func checkDBReady(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func checkProviderReady(provider readinessChecker) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return provider.HealthCheck(ctx)
}
