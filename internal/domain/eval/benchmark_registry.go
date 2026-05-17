package eval

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// CreateBenchmarkCaseInput is the validated input for persisting a benchmark case.
type CreateBenchmarkCaseInput struct {
	WorkspaceID     string
	SyntheticOrgID  *string
	Slug            string
	Name            string
	Domain          string
	Version         int
	InputPayload    json.RawMessage
	ExpectedOutcome json.RawMessage
	Tags            []string
}

// RunBenchmarkCaseInput identifies which suite should execute a persisted benchmark case.
type RunBenchmarkCaseInput struct {
	WorkspaceID     string
	EvalSuiteID     string
	PromptVersionID *string
	TriggeredBy     *string
}

// BenchmarkRegistryService orchestrates persisted benchmark cases and benchmark-backed runs.
type BenchmarkRegistryService struct {
	db     *sql.DB
	runner *RunnerService
}

// NewBenchmarkRegistryService constructs a benchmark registry service.
func NewBenchmarkRegistryService(db *sql.DB, runner *RunnerService) *BenchmarkRegistryService {
	return &BenchmarkRegistryService{db: db, runner: runner}
}

// benchmarkInsertArgs holds the normalized, serialized values ready for INSERT.
type benchmarkInsertArgs struct {
	id              string
	syntheticOrgID  *string
	version         int
	inputPayload    string
	expectedOutcome string
	tagsJSON        string
}

// prepareBenchmarkInsertArgs normalizes and serializes all fields that require
// transformation before the INSERT — keeping Create below the complexity gate.
func prepareBenchmarkInsertArgs(in CreateBenchmarkCaseInput) (benchmarkInsertArgs, error) {
	orgID := in.SyntheticOrgID
	// Empty string must become NULL so SQLite does not reject the FK reference.
	if orgID != nil && *orgID == "" {
		orgID = nil
	}

	version := in.Version
	if version <= 0 {
		version = 1
	}

	tagsRaw, err := json.Marshal(normalizeStringSlice(in.Tags))
	if err != nil {
		return benchmarkInsertArgs{}, fmt.Errorf("marshal benchmark tags: %w", err)
	}

	return benchmarkInsertArgs{
		id:              uuid.NewV7().String(),
		syntheticOrgID:  orgID,
		version:         version,
		inputPayload:    string(normalizeRawMessage(in.InputPayload)),
		expectedOutcome: string(normalizeRawMessage(in.ExpectedOutcome)),
		tagsJSON:        string(tagsRaw),
	}, nil
}

// Create persists a benchmark case and returns the stored domain value.
func (s *BenchmarkRegistryService) Create(ctx context.Context, in CreateBenchmarkCaseInput) (*BenchmarkCase, error) {
	if err := validateCreateBenchmarkCaseInput(in); err != nil {
		return nil, err
	}
	args, err := prepareBenchmarkInsertArgs(in)
	if err != nil {
		return nil, err
	}
	if validateErr := s.validateSyntheticOrgReference(ctx, in.WorkspaceID, args.syntheticOrgID); validateErr != nil {
		return nil, validateErr
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO benchmark_case (
			id, workspace_id, synthetic_org_id, slug, name, domain, version, input_payload, expected_outcome, tags
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		args.id, in.WorkspaceID, args.syntheticOrgID,
		in.Slug, in.Name, in.Domain, args.version,
		args.inputPayload, args.expectedOutcome, args.tagsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("create benchmark case: %w", err)
	}

	return s.GetByID(ctx, in.WorkspaceID, args.id)
}

func (s *BenchmarkRegistryService) validateSyntheticOrgReference(
	ctx context.Context,
	workspaceID string,
	syntheticOrgID *string,
) error {
	if syntheticOrgID == nil || *syntheticOrgID == "" {
		return nil
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM synthetic_org
		WHERE id = ? AND workspace_id = ?`,
		*syntheticOrgID, workspaceID,
	)

	var id string
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("synthetic org not found: %w", err)
		}
		return fmt.Errorf("validate synthetic org reference: %w", err)
	}
	return nil
}

// GetByID returns one persisted benchmark case or sql.ErrNoRows.
func (s *BenchmarkRegistryService) GetByID(ctx context.Context, workspaceID, id string) (*BenchmarkCase, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, synthetic_org_id, slug, name, domain, version,
		       input_payload, expected_outcome, tags, created_at, updated_at
		FROM benchmark_case
		WHERE id = ? AND workspace_id = ?`,
		id, workspaceID,
	)

	var raw benchmarkCaseRow
	if err := row.Scan(
		&raw.ID,
		&raw.WorkspaceID,
		&raw.SyntheticOrgID,
		&raw.Slug,
		&raw.Name,
		&raw.Domain,
		&raw.Version,
		&raw.InputPayload,
		&raw.ExpectedOutcome,
		&raw.Tags,
		&raw.CreatedAt,
		&raw.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get benchmark case: %w", err)
	}
	return raw.toDomain()
}

// RunBenchmarkCase resolves a persisted benchmark case and triggers a benchmark-backed eval run.
func (s *BenchmarkRegistryService) RunBenchmarkCase(ctx context.Context, benchmarkCaseID string, in RunBenchmarkCaseInput) (*Run, error) {
	if s.runner == nil {
		return nil, fmt.Errorf("run benchmark case: runner service is required")
	}
	benchmarkCase, err := s.GetByID(ctx, in.WorkspaceID, benchmarkCaseID)
	if err != nil {
		return nil, err
	}

	run, err := s.runner.Run(ctx, RunInput{
		WorkspaceID:     in.WorkspaceID,
		EvalSuiteID:     in.EvalSuiteID,
		PromptVersionID: in.PromptVersionID,
		TriggeredBy:     in.TriggeredBy,
		Provenance: &ReplayProvenance{
			Mode:            ReplayModeBenchmark,
			BenchmarkCaseID: &benchmarkCase.ID,
			SyntheticOrgID:  benchmarkCase.SyntheticOrgID,
		},
	})
	if err != nil {
		return nil, err
	}
	return run, nil
}

type benchmarkCaseRow struct {
	ID              string
	WorkspaceID     string
	SyntheticOrgID  sql.NullString
	Slug            string
	Name            string
	Domain          string
	Version         int
	InputPayload    string
	ExpectedOutcome string
	Tags            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (row benchmarkCaseRow) toDomain() (*BenchmarkCase, error) {
	tags, err := parseBenchmarkTags(row.Tags)
	if err != nil {
		return nil, err
	}

	return &BenchmarkCase{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		SyntheticOrgID:  nullStringPtr(row.SyntheticOrgID),
		Slug:            row.Slug,
		Name:            row.Name,
		Domain:          row.Domain,
		Version:         row.Version,
		InputPayload:    json.RawMessage(row.InputPayload),
		ExpectedOutcome: json.RawMessage(row.ExpectedOutcome),
		Tags:            tags,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func validateCreateBenchmarkCaseInput(in CreateBenchmarkCaseInput) error {
	if in.WorkspaceID == "" {
		return fmt.Errorf("benchmark workspace_id is required")
	}
	if in.Slug == "" {
		return fmt.Errorf("benchmark slug is required")
	}
	if in.Name == "" {
		return fmt.Errorf("benchmark name is required")
	}
	if _, ok := validDomains[in.Domain]; !ok {
		return fmt.Errorf("invalid benchmark domain %q", in.Domain)
	}
	return nil
}

func normalizeRawMessage(in json.RawMessage) json.RawMessage {
	if len(in) == 0 {
		return json.RawMessage(`{}`)
	}
	out := make(json.RawMessage, len(in))
	copy(out, in)
	return out
}

func normalizeStringSlice(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func parseBenchmarkTags(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil, fmt.Errorf("parse benchmark tags: %w", err)
	}
	return tags, nil
}
