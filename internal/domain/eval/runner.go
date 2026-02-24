package eval

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// Run — Task 4.7: FR-242 execution record for an eval suite.
type Run struct {
	ID              string           `json:"id"`
	WorkspaceID     string           `json:"workspaceId"`
	EvalSuiteID     string           `json:"evalSuiteId"`
	PromptVersionID *string          `json:"promptVersionId,omitempty"`
	Status          string           `json:"status"` // "running" | "passed" | "failed"
	Scores          Scores           `json:"scores"`
	Details         []TestCaseResult `json:"details"`
	TriggeredBy     *string          `json:"triggeredBy,omitempty"`
	StartedAt       time.Time        `json:"startedAt"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
}

// Scores — computed metric scores (0.0 to 1.0).
type Scores struct {
	Groundedness    float64 `json:"groundedness"`
	Exactitude      float64 `json:"exactitude"`
	Abstention      float64 `json:"abstention"`
	PolicyAdherence float64 `json:"policy_adherence"`
}

// TestCaseResult — per-case evaluation result.
type TestCaseResult struct {
	Input              string   `json:"input"`
	Output             string   `json:"output"`
	Passed             bool     `json:"passed"`
	MatchedKeywords    []string `json:"matched_keywords"`
	AbstainedCorrectly bool     `json:"abstained_correctly"`
}

// RunInput — input to trigger a new eval run.
type RunInput struct {
	WorkspaceID     string
	EvalSuiteID     string
	PromptVersionID *string // optional
	TriggeredBy     *string // user_id, optional
}

// RunnerService — Task 4.7: executes eval suites and calculates scores.
type RunnerService struct {
	querier      sqlcgen.Querier
	suiteService *SuiteService
}

// NewRunnerService constructs a RunnerService.
// Task 4.7: FR-242
func NewRunnerService(db *sql.DB) *RunnerService {
	return &RunnerService{
		querier:      sqlcgen.New(db),
		suiteService: NewSuiteService(db),
	}
}

// Run executes an eval suite and persists the result.
// Task 4.7: FR-242 — scoring is keyword-based (no LLM required for MVP).
func (s *RunnerService) Run(ctx context.Context, in RunInput) (*Run, error) {
	suite, err := s.suiteService.GetByID(ctx, in.WorkspaceID, in.EvalSuiteID)
	if err != nil {
		return nil, fmt.Errorf("fetch eval suite: %w", err)
	}

	runID := uuid.NewV7().String()
	row, err := s.querier.CreateEvalRun(ctx, sqlcgen.CreateEvalRunParams{
		ID:              runID,
		WorkspaceID:     in.WorkspaceID,
		EvalSuiteID:     in.EvalSuiteID,
		PromptVersionID: in.PromptVersionID,
		TriggeredBy:     in.TriggeredBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create eval run: %w", err)
	}

	results, scores := scoreTestCases(suite.TestCases)
	status := evalStatus(scores, suite.Thresholds)

	scoresJSON, err := json.Marshal(scores)
	if err != nil {
		return nil, fmt.Errorf("marshal scores: %w", err)
	}
	detailsJSON, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("marshal details: %w", err)
	}
	if updateErr := s.querier.UpdateEvalRunResult(ctx, sqlcgen.UpdateEvalRunResultParams{
		Status:      status,
		Scores:      string(scoresJSON),
		Details:     string(detailsJSON),
		ID:          runID,
		WorkspaceID: in.WorkspaceID,
	}); updateErr != nil {
		return nil, fmt.Errorf("update eval run: %w", updateErr)
	}

	return rowToRun(row, scores, results, status)
}

// GetRun returns a single eval run.
// Task 4.7: FR-242
func (s *RunnerService) GetRun(ctx context.Context, workspaceID, id string) (*Run, error) {
	row, err := s.querier.GetEvalRunByID(ctx, sqlcgen.GetEvalRunByIDParams{
		ID: id, WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get eval run: %w", err)
	}
	return parseRun(row)
}

// ListRuns returns paginated eval runs for a workspace.
// Task 4.7: FR-242
func (s *RunnerService) ListRuns(ctx context.Context, workspaceID string, limit, offset int) ([]*Run, error) {
	rows, err := s.querier.ListEvalRuns(ctx, sqlcgen.ListEvalRunsParams{
		WorkspaceID: workspaceID,
		Limit:       int64(limit),
		Offset:      int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list eval runs: %w", err)
	}
	runs := make([]*Run, 0, len(rows))
	for _, row := range rows {
		run, parseErr := parseRun(row)
		if parseErr != nil {
			return nil, parseErr
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// scoreTestCases executes keyword-based scoring for each test case.
// MVP: no LLM required. Simulated output = echo of input.
// Task 4.7: Groundedness=1.0 (stub always outputs), Exactitude=keyword hit rate,
// Abstention=0.0 (cannot simulate without LLM), Policy=1.0 (not enforced in MVP).
func scoreTestCases(cases []TestCase) ([]TestCaseResult, Scores) {
	if len(cases) == 0 {
		return nil, Scores{Groundedness: 1.0, Exactitude: 1.0, Abstention: 1.0, PolicyAdherence: 1.0}
	}

	results := make([]TestCaseResult, 0, len(cases))
	exactHits, abstainCases := 0, 0

	for _, tc := range cases {
		result := scoreSingleTestCase(tc)
		results = append(results, result)
		if result.Passed {
			exactHits++
		}
		if tc.ShouldAbstain {
			abstainCases++
		}
	}

	return results, computeScores(len(cases), exactHits, abstainCases)
}

// scoreSingleTestCase scores a single test case.
func scoreSingleTestCase(tc TestCase) TestCaseResult {
	simulatedOutput := tc.Input // MVP stub: echo input
	matched := matchKeywords(simulatedOutput, tc.ExpectedKeywords)
	return TestCaseResult{
		Input:              tc.Input,
		Output:             simulatedOutput,
		Passed:             !tc.ShouldAbstain && len(matched) > 0,
		MatchedKeywords:    matched,
		AbstainedCorrectly: false, // MVP: requires LLM
	}
}

// computeScores calculates final scores from hit counts.
func computeScores(totalCases, exactHits, abstainCases int) Scores {
	nonAbstainCases := totalCases - abstainCases
	exactitude := 1.0
	if nonAbstainCases > 0 {
		exactitude = float64(exactHits) / float64(nonAbstainCases)
	}
	abstention := 1.0
	// MVP: cannot simulate abstention without LLM — abstention always 1.0 for no abstain cases
	return Scores{
		Groundedness:    1.0,
		Exactitude:      exactitude,
		Abstention:      abstention,
		PolicyAdherence: 1.0,
	}
}

// matchKeywords returns which expected keywords appear in output (case-insensitive).
func matchKeywords(output string, keywords []string) []string {
	lower := strings.ToLower(output)
	matched := make([]string, 0)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			matched = append(matched, kw)
		}
	}
	return matched
}

// evalStatus compares scores against thresholds, returns "passed" or "failed".
func evalStatus(scores Scores, thr Thresholds) string {
	if scores.Groundedness >= thr.Groundedness &&
		scores.Exactitude >= thr.Exactitude &&
		scores.Abstention >= thr.Abstention &&
		scores.PolicyAdherence >= thr.Policy {
		return "passed"
	}
	return "failed"
}

// rowToRun builds a Run from freshly-created row + computed values.
func rowToRun(row sqlcgen.EvalRun, scores Scores, details []TestCaseResult, status string) (*Run, error) {
	return &Run{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		EvalSuiteID:     row.EvalSuiteID,
		PromptVersionID: row.PromptVersionID,
		Status:          status,
		Scores:          scores,
		Details:         details,
		TriggeredBy:     row.TriggeredBy,
		StartedAt:       row.StartedAt,
		CreatedAt:       row.CreatedAt,
	}, nil
}

// parseRun builds a Run from a persisted row (JSON deserialization).
func parseRun(row sqlcgen.EvalRun) (*Run, error) {
	var scores Scores
	if err := json.Unmarshal([]byte(row.Scores), &scores); err != nil {
		return nil, fmt.Errorf("parse scores: %w", err)
	}
	var details []TestCaseResult
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		return nil, fmt.Errorf("parse details: %w", err)
	}
	run := &Run{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		EvalSuiteID:     row.EvalSuiteID,
		PromptVersionID: row.PromptVersionID,
		Status:          row.Status,
		Scores:          scores,
		Details:         details,
		TriggeredBy:     row.TriggeredBy,
		StartedAt:       row.StartedAt,
		CreatedAt:       row.CreatedAt,
	}
	// CompletedAt is nullable — sqlc generates *sql.NullTime with emit_pointers_for_null_types: true
	if row.CompletedAt != nil {
		run.CompletedAt = row.CompletedAt
	}
	return run, nil
}
