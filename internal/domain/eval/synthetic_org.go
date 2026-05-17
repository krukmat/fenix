package eval

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// CreateSyntheticOrgInput is the validated input for persisting one synthetic org fixture.
type CreateSyntheticOrgInput struct {
	WorkspaceID string
	Slug        string
	Name        string
	Version     int
	Seed        int64
	FixtureData json.RawMessage
}

// SyntheticOrgFixtureSnapshot is the deterministic generated output for one synthetic org.
type SyntheticOrgFixtureSnapshot struct {
	SyntheticOrgID string          `json:"syntheticOrgId"`
	WorkspaceID    string          `json:"workspaceId"`
	Slug           string          `json:"slug"`
	Name           string          `json:"name"`
	Version        int             `json:"version"`
	Seed           int64           `json:"seed"`
	FixtureData    json.RawMessage `json:"fixtureData"`
}

// SyntheticOrgService persists and generates deterministic synthetic org fixtures.
type SyntheticOrgService struct {
	db *sql.DB
}

// NewSyntheticOrgService constructs a synthetic org service.
func NewSyntheticOrgService(db *sql.DB) *SyntheticOrgService {
	return &SyntheticOrgService{db: db}
}

// Create persists a synthetic org and returns the stored domain value.
func (s *SyntheticOrgService) Create(ctx context.Context, in CreateSyntheticOrgInput) (*SyntheticOrg, error) {
	if err := validateCreateSyntheticOrgInput(in); err != nil {
		return nil, err
	}

	id := uuid.NewV7().String()
	version := in.Version
	if version <= 0 {
		version = 1
	}
	fixtureData, err := normalizeJSONObject(in.FixtureData)
	if err != nil {
		return nil, fmt.Errorf("normalize synthetic org fixture data: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO synthetic_org (
			id, workspace_id, slug, name, version, seed, fixture_data
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		in.WorkspaceID,
		in.Slug,
		in.Name,
		version,
		in.Seed,
		string(fixtureData),
	)
	if err != nil {
		return nil, fmt.Errorf("create synthetic org: %w", err)
	}

	return s.GetByID(ctx, in.WorkspaceID, id)
}

// GetByID returns one persisted synthetic org or sql.ErrNoRows.
func (s *SyntheticOrgService) GetByID(ctx context.Context, workspaceID, id string) (*SyntheticOrg, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, slug, name, version, seed, fixture_data, created_at, updated_at
		FROM synthetic_org
		WHERE id = ? AND workspace_id = ?`,
		id, workspaceID,
	)

	var raw syntheticOrgRow
	if err := row.Scan(
		&raw.ID,
		&raw.WorkspaceID,
		&raw.Slug,
		&raw.Name,
		&raw.Version,
		&raw.Seed,
		&raw.FixtureData,
		&raw.CreatedAt,
		&raw.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get synthetic org: %w", err)
	}
	return raw.toDomain()
}

// Generate returns a deterministic fixture snapshot for one stored synthetic org.
func (s *SyntheticOrgService) Generate(ctx context.Context, workspaceID, id string) (json.RawMessage, error) {
	org, err := s.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}

	snapshot := SyntheticOrgFixtureSnapshot{
		SyntheticOrgID: org.ID,
		WorkspaceID:    org.WorkspaceID,
		Slug:           org.Slug,
		Name:           org.Name,
		Version:        org.Version,
		Seed:           org.Seed,
		FixtureData:    normalizeRawMessage(org.FixtureData),
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal synthetic org fixture snapshot: %w", err)
	}
	return json.RawMessage(raw), nil
}

type syntheticOrgRow struct {
	ID          string
	WorkspaceID string
	Slug        string
	Name        string
	Version     int
	Seed        int64
	FixtureData string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (row syntheticOrgRow) toDomain() (*SyntheticOrg, error) {
	fixtureData, err := normalizeJSONObject(json.RawMessage(row.FixtureData))
	if err != nil {
		return nil, fmt.Errorf("parse synthetic org fixture data: %w", err)
	}

	return &SyntheticOrg{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Slug:        row.Slug,
		Name:        row.Name,
		Version:     row.Version,
		Seed:        row.Seed,
		FixtureData: fixtureData,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func validateCreateSyntheticOrgInput(in CreateSyntheticOrgInput) error {
	if in.WorkspaceID == "" {
		return fmt.Errorf("synthetic org workspace_id is required")
	}
	if in.Slug == "" {
		return fmt.Errorf("synthetic org slug is required")
	}
	if in.Name == "" {
		return fmt.Errorf("synthetic org name is required")
	}
	return nil
}

func normalizeJSONObject(in json.RawMessage) (json.RawMessage, error) {
	if len(in) == 0 {
		return json.RawMessage(`{}`), nil
	}

	var decoded any
	if err := json.Unmarshal(in, &decoded); err != nil {
		return nil, err
	}

	out, err := json.Marshal(decoded)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
