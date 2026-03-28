package signal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrSignalNotFound = errors.New("signal not found")
)

type Status string

const (
	StatusActive    Status = "active"
	StatusDismissed Status = "dismissed"
	StatusExpired   Status = "expired"
)

type Signal struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspaceId"`
	EntityType  string          `json:"entityType"`
	EntityID    string          `json:"entityId"`
	SignalType  string          `json:"signalType"`
	Confidence  float64         `json:"confidence"`
	EvidenceIDs []string        `json:"evidenceIds"`
	SourceType  string          `json:"sourceType"`
	SourceID    string          `json:"sourceId"`
	Metadata    json.RawMessage `json:"metadata"`
	Status      Status          `json:"status"`
	DismissedBy *string         `json:"dismissedBy,omitempty"`
	DismissedAt *time.Time      `json:"dismissedAt,omitempty"`
	ExpiresAt   *time.Time      `json:"expiresAt,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type Filters struct {
	Status     *Status
	EntityType string
	EntityID   string
}

type CreateInput struct {
	ID          string
	WorkspaceID string
	EntityType  string
	EntityID    string
	SignalType  string
	Confidence  float64
	EvidenceIDs []string
	SourceType  string
	SourceID    string
	Metadata    json.RawMessage
	Status      Status
	ExpiresAt   *time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (*Signal, error) {
	now := nowRFC3339()
	evidenceIDs, err := json.Marshal(input.EvidenceIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal evidence ids: %w", err)
	}
	metadata := normalizeJSON(input.Metadata, []byte("{}"))

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO signal (
			id, workspace_id, entity_type, entity_id, signal_type, confidence,
			evidence_ids, source_type, source_id, metadata, status, dismissed_by,
			dismissed_at, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?)
		RETURNING
			id, workspace_id, entity_type, entity_id, signal_type, confidence,
			evidence_ids, source_type, source_id, metadata, status, dismissed_by,
			dismissed_at, expires_at, created_at, updated_at
	`,
		input.ID,
		input.WorkspaceID,
		input.EntityType,
		input.EntityID,
		input.SignalType,
		input.Confidence,
		evidenceIDs,
		input.SourceType,
		input.SourceID,
		metadata,
		string(input.Status),
		formatOptionalTime(input.ExpiresAt),
		now,
		now,
	)

	out, err := scanSignal(row)
	if err != nil {
		return nil, fmt.Errorf("create signal: %w", err)
	}
	return out, nil
}

func (r *Repository) GetByID(ctx context.Context, workspaceID, signalID string) (*Signal, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, entity_type, entity_id, signal_type, confidence,
			evidence_ids, source_type, source_id, metadata, status, dismissed_by,
			dismissed_at, expires_at, created_at, updated_at
		FROM signal
		WHERE id = ? AND workspace_id = ?
		LIMIT 1
	`, signalID, workspaceID)

	out, err := scanSignal(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSignalNotFound
		}
		return nil, fmt.Errorf("get signal by id: %w", err)
	}
	return out, nil
}

func (r *Repository) List(ctx context.Context, workspaceID string, filters Filters) ([]*Signal, error) {
	switch {
	case filters.EntityType != "" && filters.EntityID != "":
		return r.GetByEntity(ctx, workspaceID, filters.EntityType, filters.EntityID)
	case filters.Status != nil:
		rows, err := r.db.QueryContext(ctx, `
			SELECT
				id, workspace_id, entity_type, entity_id, signal_type, confidence,
				evidence_ids, source_type, source_id, metadata, status, dismissed_by,
				dismissed_at, expires_at, created_at, updated_at
			FROM signal
			WHERE workspace_id = ? AND status = ?
			ORDER BY created_at DESC
		`, workspaceID, string(*filters.Status))
		if err != nil {
			return nil, fmt.Errorf("list signals by status: %w", err)
		}
		defer rows.Close()
		return scanSignalRows(rows)
	default:
		rows, err := r.db.QueryContext(ctx, `
			SELECT
				id, workspace_id, entity_type, entity_id, signal_type, confidence,
				evidence_ids, source_type, source_id, metadata, status, dismissed_by,
				dismissed_at, expires_at, created_at, updated_at
			FROM signal
			WHERE workspace_id = ?
			ORDER BY created_at DESC
		`, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("list signals by workspace: %w", err)
		}
		defer rows.Close()
		return scanSignalRows(rows)
	}
}

func (r *Repository) GetByEntity(ctx context.Context, workspaceID, entityType, entityID string) ([]*Signal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, workspace_id, entity_type, entity_id, signal_type, confidence,
			evidence_ids, source_type, source_id, metadata, status, dismissed_by,
			dismissed_at, expires_at, created_at, updated_at
		FROM signal
		WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?
		ORDER BY created_at DESC
	`, workspaceID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("list signals by entity: %w", err)
	}
	defer rows.Close()
	return scanSignalRows(rows)
}

func (r *Repository) CountActiveByEntities(ctx context.Context, workspaceID, entityType string, entityIDs []string) (map[string]int, error) {
	counts := make(map[string]int, len(entityIDs))
	if len(entityIDs) == 0 {
		return counts, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(entityIDs)), ",")
	args := make([]any, 0, len(entityIDs)+3)
	args = append(args, workspaceID, entityType, string(StatusActive))
	for _, entityID := range entityIDs {
		args = append(args, entityID)
	}
	query := fmt.Sprintf(`
		SELECT entity_id, COUNT(*)
		FROM signal
		WHERE workspace_id = ? AND entity_type = ? AND status = ? AND entity_id IN (%s)
		GROUP BY entity_id
	`, placeholders)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("count active signals by entities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var entityID string
		var count int
		if err := rows.Scan(&entityID, &count); err != nil {
			return nil, fmt.Errorf("scan active signal count: %w", err)
		}
		counts[entityID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active signal counts: %w", err)
	}
	return counts, nil
}

func (r *Repository) Dismiss(ctx context.Context, workspaceID, signalID, actorID string) (*Signal, error) {
	now := nowRFC3339()
	row := r.db.QueryRowContext(ctx, `
		UPDATE signal
		SET status = 'dismissed', dismissed_by = ?, dismissed_at = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
		RETURNING
			id, workspace_id, entity_type, entity_id, signal_type, confidence,
			evidence_ids, source_type, source_id, metadata, status, dismissed_by,
			dismissed_at, expires_at, created_at, updated_at
	`, actorID, now, now, signalID, workspaceID)

	out, err := scanSignal(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSignalNotFound
		}
		return nil, fmt.Errorf("dismiss signal: %w", err)
	}
	return out, nil
}
