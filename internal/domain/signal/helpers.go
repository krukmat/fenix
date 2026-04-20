package signal

import (
	"database/sql"
	"encoding/json"
	"time"
)

type signalRow struct {
	ID          string
	WorkspaceID string
	EntityType  string
	EntityID    string
	SignalType  string
	Confidence  float64
	EvidenceIDs []byte
	SourceType  string
	SourceID    string
	Metadata    []byte
	Status      string
	DismissedBy *string
	DismissedAt *string
	ExpiresAt   *string
	CreatedAt   string
	UpdatedAt   string
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func parseRFC3339Time(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

func parseOptionalRFC3339(value *string) *time.Time {
	if value == nil {
		return nil
	}
	t := parseRFC3339Time(*value)
	return &t
}

func normalizeJSON(raw json.RawMessage, fallback []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}

func scanSignal(scanner interface {
	Scan(dest ...any) error
}) (*Signal, error) {
	var row signalRow
	if err := scanner.Scan(
		&row.ID,
		&row.WorkspaceID,
		&row.EntityType,
		&row.EntityID,
		&row.SignalType,
		&row.Confidence,
		&row.EvidenceIDs,
		&row.SourceType,
		&row.SourceID,
		&row.Metadata,
		&row.Status,
		&row.DismissedBy,
		&row.DismissedAt,
		&row.ExpiresAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return rowToSignal(row), nil
}

func scanSignalRows(rows *sql.Rows) ([]*Signal, error) {
	out := make([]*Signal, 0)
	for rows.Next() {
		item, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func rowToSignal(row signalRow) *Signal {
	var evidenceIDs []string
	if len(row.EvidenceIDs) > 0 {
		_ = json.Unmarshal(row.EvidenceIDs, &evidenceIDs)
	}

	metadata := json.RawMessage(`{}`)
	if len(row.Metadata) > 0 {
		metadata = json.RawMessage(row.Metadata)
	}

	return &Signal{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		EntityType:  row.EntityType,
		EntityID:    row.EntityID,
		SignalType:  row.SignalType,
		Confidence:  row.Confidence,
		EvidenceIDs: evidenceIDs,
		SourceType:  row.SourceType,
		SourceID:    row.SourceID,
		Metadata:    metadata,
		Status:      Status(row.Status),
		DismissedBy: row.DismissedBy,
		DismissedAt: parseOptionalRFC3339(row.DismissedAt),
		ExpiresAt:   parseOptionalRFC3339(row.ExpiresAt),
		CreatedAt:   parseRFC3339Time(row.CreatedAt),
		UpdatedAt:   parseRFC3339Time(row.UpdatedAt),
	}
}
