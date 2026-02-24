package eval

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// Suite — Task 4.7: FR-242 dataset definition for evaluating prompts/policies.
type Suite struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	Name        string     `json:"name"`
	Domain      string     `json:"domain"`     // "support" | "sales" | "general"
	TestCases   []TestCase `json:"testCases"`  // parsed from JSON
	Thresholds  Thresholds `json:"thresholds"` // parsed from JSON
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// TestCase — single evaluation input/expected pair.
type TestCase struct {
	Input            string   `json:"input"`
	ExpectedKeywords []string `json:"expected_keywords"` // keywords that should appear in output
	ShouldAbstain    bool     `json:"should_abstain"`    // true = agent should refuse to answer
}

// Thresholds — minimum passing scores per metric (0.0 to 1.0).
type Thresholds struct {
	Groundedness float64 `json:"groundedness"` // default: 0.8
	Exactitude   float64 `json:"exactitude"`   // default: 0.85
	Abstention   float64 `json:"abstention"`   // default: 0.95
	Policy       float64 `json:"policy"`       // default: 1.0
}

// CreateSuiteInput — input for creating a new eval suite.
type CreateSuiteInput struct {
	WorkspaceID string
	Name        string
	Domain      string
	TestCases   []TestCase
	Thresholds  Thresholds
}

// UpdateSuiteInput — input for updating an eval suite.
type UpdateSuiteInput struct {
	ID          string
	WorkspaceID string
	Name        string
	Domain      string
	TestCases   []TestCase
	Thresholds  Thresholds
}

// SuiteService — Task 4.7: CRUD for eval suites.
type SuiteService struct {
	querier sqlcgen.Querier
}

// NewSuiteService constructs a new SuiteService.
// Task 4.7: FR-242
func NewSuiteService(db *sql.DB) *SuiteService {
	return &SuiteService{querier: sqlcgen.New(db)}
}

// Create persists a new eval suite and returns it.
// Task 4.7: FR-242
func (s *SuiteService) Create(ctx context.Context, in CreateSuiteInput) (*Suite, error) {
	id := uuid.NewV7().String()

	tcJSON, err := json.Marshal(in.TestCases)
	if err != nil {
		return nil, fmt.Errorf("marshal test_cases: %w", err)
	}
	thrJSON, err := json.Marshal(in.Thresholds)
	if err != nil {
		return nil, fmt.Errorf("marshal thresholds: %w", err)
	}

	row, err := s.querier.CreateEvalSuite(ctx, sqlcgen.CreateEvalSuiteParams{
		ID:          id,
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Domain:      in.Domain,
		TestCases:   string(tcJSON),
		Thresholds:  string(thrJSON),
	})
	if err != nil {
		return nil, fmt.Errorf("create eval suite: %w", err)
	}

	return rowToSuite(row)
}

// GetByID returns a single suite or sql.ErrNoRows.
// Task 4.7: FR-242
func (s *SuiteService) GetByID(ctx context.Context, workspaceID, id string) (*Suite, error) {
	row, err := s.querier.GetEvalSuiteByID(ctx, sqlcgen.GetEvalSuiteByIDParams{
		ID: id, WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get eval suite: %w", err)
	}
	return rowToSuite(row)
}

// List returns all suites for a workspace.
// Task 4.7: FR-242
func (s *SuiteService) List(ctx context.Context, workspaceID string) ([]*Suite, error) {
	rows, err := s.querier.ListEvalSuites(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list eval suites: %w", err)
	}
	suites := make([]*Suite, 0, len(rows))
	for _, row := range rows {
		suite, parseErr := rowToSuite(row)
		if parseErr != nil {
			return nil, parseErr
		}
		suites = append(suites, suite)
	}
	return suites, nil
}

// Update replaces suite fields.
// Task 4.7: FR-242
func (s *SuiteService) Update(ctx context.Context, in UpdateSuiteInput) error {
	tcJSON, err := json.Marshal(in.TestCases)
	if err != nil {
		return fmt.Errorf("marshal test_cases: %w", err)
	}
	thrJSON, err := json.Marshal(in.Thresholds)
	if err != nil {
		return fmt.Errorf("marshal thresholds: %w", err)
	}
	return s.querier.UpdateEvalSuite(ctx, sqlcgen.UpdateEvalSuiteParams{
		Name:        in.Name,
		Domain:      in.Domain,
		TestCases:   string(tcJSON),
		Thresholds:  string(thrJSON),
		ID:          in.ID,
		WorkspaceID: in.WorkspaceID,
	})
}

// Delete removes a suite.
// Task 4.7: FR-242
func (s *SuiteService) Delete(ctx context.Context, workspaceID, id string) error {
	return s.querier.DeleteEvalSuite(ctx, sqlcgen.DeleteEvalSuiteParams{
		ID: id, WorkspaceID: workspaceID,
	})
}

// rowToSuite converts a sqlcgen.EvalSuite to domain Suite.
func rowToSuite(row sqlcgen.EvalSuite) (*Suite, error) {
	var tc []TestCase
	if err := json.Unmarshal([]byte(row.TestCases), &tc); err != nil {
		return nil, fmt.Errorf("parse test_cases: %w", err)
	}
	var thr Thresholds
	if err := json.Unmarshal([]byte(row.Thresholds), &thr); err != nil {
		return nil, fmt.Errorf("parse thresholds: %w", err)
	}
	return &Suite{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Name:        row.Name,
		Domain:      row.Domain,
		TestCases:   tc,
		Thresholds:  thr,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}
