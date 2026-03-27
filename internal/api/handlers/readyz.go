package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

const readyStatusReady = "ready"

type readyzResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Chat     string `json:"chat"`
	Embed    string `json:"embed"`
}

// NewReadyzHandler checks DB, chat provider and embed provider readiness.
func NewReadyzHandler(db *sql.DB, chat, embed llm.LLMProvider) http.HandlerFunc {
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
			resp.Database = "error"
		}
		if err := checkProviderReady(chat); err != nil {
			resp.Status = healthStatusDegraded
			resp.Chat = "error"
		}
		if err := checkProviderReady(embed); err != nil {
			resp.Status = healthStatusDegraded
			resp.Embed = "error"
		}

		code := http.StatusOK
		if resp.Status != readyStatusReady {
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

func checkProviderReady(provider llm.LLMProvider) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return provider.HealthCheck(ctx)
}
