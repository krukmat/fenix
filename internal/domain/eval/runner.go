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

const statusFailed = "failed"

// Run — Task 4.7: FR-242 execution record for an eval suite.
type Run struct {
	ID              string            `json:"id"`
	WorkspaceID     string            `json:"workspaceId"`
	EvalSuiteID     string            `json:"evalSuiteId"`
	PromptVersionID *string           `json:"promptVersionId,omitempty"`
	Status          string            `json:"status"` // "running" | "passed" | "failed"
	Provenance      *ReplayProvenance `json:"provenance,omitempty"`
	ReplayArtifact  *ReplayArtifact   `json:"replay_artifact,omitempty"` // task-C2.4: populated when replay provenance is present
	ScoredResult    *ScoredResult     `json:"scored_result,omitempty"`
	Scores          Scores            `json:"scores"`
	Details         []TestCaseResult  `json:"details"`
	TriggeredBy     *string           `json:"triggeredBy,omitempty"`
	StartedAt       time.Time         `json:"startedAt"`
	CompletedAt     *time.Time        `json:"completedAt,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
}

// Scores — computed metric scores (0.0 to 1.0).
type Scores struct {
	Groundedness            float64       `json:"groundedness"`
	Exactitude              float64       `json:"exactitude"`
	Abstention              float64       `json:"abstention"`
	PolicyAdherence         float64       `json:"policy_adherence"`
	OutcomeAccuracy         float64       `json:"outcome_accuracy"`
	ToolCallPrecision       float64       `json:"tool_call_precision"`
	ToolCallRecall          float64       `json:"tool_call_recall"`
	ToolCallF1              float64       `json:"tool_call_f1"`
	ForbiddenToolViolations int           `json:"forbidden_tool_violations"`
	PolicyCompliance        float64       `json:"policy_compliance"`
	ApprovalAccuracy        float64       `json:"approval_accuracy"`
	EvidenceCoverage        float64       `json:"evidence_coverage"`
	ForbiddenEvidenceCount  int           `json:"forbidden_evidence_count"`
	StateMutationAccuracy   float64       `json:"state_mutation_accuracy"`
	AuditCompleteness       float64       `json:"audit_completeness"`
	ContractValidity        float64       `json:"contract_validity"`
	AbstentionAccuracy      float64       `json:"abstention_accuracy"`
	LatencyCompliance       float64       `json:"latency_compliance"`
	ToolBudgetCompliance    float64       `json:"tool_budget_compliance"`
	ScorecardScore          float64       `json:"scorecard_score"`
	ScorecardVerdict        Verdict       `json:"scorecard_verdict"`
	ScoredResult            *ScoredResult `json:"scored_result,omitempty"`
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
	Provenance      *ReplayProvenance
	Scenario        *GoldenScenario
}

// RunnerService — Task 4.7: executes eval suites and calculates scores.
type RunnerService struct {
	db           *sql.DB
	querier      sqlcgen.Querier
	suiteService *SuiteService
	replayEngine ReplayEngine // task-C2.4: nil means legacy-only path
	scoring      *ScoringService
}

// NewRunnerService constructs a RunnerService without replay support.
// Task 4.7: FR-242
func NewRunnerService(db *sql.DB) *RunnerService {
	return &RunnerService{
		db:           db,
		querier:      sqlcgen.New(db),
		suiteService: NewSuiteService(db),
		scoring:      NewScoringService(),
	}
}

// NewRunnerServiceWithReplay constructs a RunnerService with replay engine support.
// task-C2.4: when RunInput.Provenance contains source refs, BuildReplay is invoked.
func NewRunnerServiceWithReplay(db *sql.DB, engine ReplayEngine) *RunnerService {
	return &RunnerService{
		db:           db,
		querier:      sqlcgen.New(db),
		suiteService: NewSuiteService(db),
		replayEngine: engine,
		scoring:      NewScoringService(),
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
	normalized := normalizeReplayProvenance(in.Provenance)
	if provenanceErr := s.persistRunProvenance(ctx, in.WorkspaceID, runID, normalized); provenanceErr != nil {
		return nil, provenanceErr
	}

	// task-C2.4: isolated replay path when engine is wired and source refs are present.
	artifact, err := s.buildReplayArtifact(ctx, in.WorkspaceID, runID, normalized)
	if err != nil {
		return nil, err
	}
	scoredResult := s.buildScoredResult(artifact, in.Scenario)

	run, err := s.scoreAndFinalize(ctx, row, suite.TestCases, suite.Thresholds, runID, in.WorkspaceID, normalized, scoredResult)
	if err != nil {
		return nil, err
	}
	run.ReplayArtifact = artifact
	return run, nil
}

// scoreAndFinalize runs keyword scoring, persists the result, and builds the Run value.
func (s *RunnerService) scoreAndFinalize(
	ctx context.Context,
	row sqlcgen.EvalRun,
	cases []TestCase,
	thresholds Thresholds,
	runID, workspaceID string,
	provenance *ReplayProvenance,
	scoredResult *ScoredResult,
) (*Run, error) {
	results, scores := scoreTestCases(cases)
	if scoredResult != nil {
		scores = mergeScoredResult(scores, *scoredResult)
	}
	status := evalStatus(scores, thresholds)

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
		WorkspaceID: workspaceID,
	}); updateErr != nil {
		return nil, fmt.Errorf("update eval run: %w", updateErr)
	}
	return rowToRun(row, scores, results, status, provenance)
}

// buildReplayArtifact invokes the replay engine when source refs are present.
// Returns nil artifact and nil error when replay is not applicable. (task-C2.4)
func (s *RunnerService) buildReplayArtifact(
	ctx context.Context,
	workspaceID, runID string,
	provenance *ReplayProvenance,
) (*ReplayArtifact, error) {
	if s.replayEngine == nil || !hasReplaySourceReferences(provenance) {
		return nil, nil
	}
	artifact, err := s.replayEngine.BuildReplay(ctx, ReplayRequest{
		EvalRunID:   runID,
		WorkspaceID: workspaceID,
		Provenance:  provenance,
	})
	if err != nil {
		return nil, fmt.Errorf("replay-backed eval run: %w", err)
	}
	return artifact, nil
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
	run, err := parseRun(row)
	if err != nil {
		return nil, err
	}
	if provenanceErr := s.attachRunProvenance(ctx, run); provenanceErr != nil {
		return nil, provenanceErr
	}
	return run, nil
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
		if provenanceErr := s.attachRunProvenance(ctx, run); provenanceErr != nil {
			return nil, provenanceErr
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
	return statusFailed
}

// rowToRun builds a Run from freshly-created row + computed values.
func rowToRun(
	row sqlcgen.EvalRun,
	scores Scores,
	details []TestCaseResult,
	status string,
	provenance *ReplayProvenance,
) (*Run, error) {
	return &Run{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		EvalSuiteID:     row.EvalSuiteID,
		PromptVersionID: row.PromptVersionID,
		Status:          status,
		Provenance:      provenance,
		ScoredResult:    scores.ScoredResult,
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
		ScoredResult:    scores.ScoredResult,
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

func (s *RunnerService) buildScoredResult(artifact *ReplayArtifact, scenario *GoldenScenario) *ScoredResult {
	if s.scoring == nil || artifact == nil || scenario == nil {
		return nil
	}
	scoredResult := s.scoring.Score(*artifact, *scenario)
	return &scoredResult
}

func mergeScoredResult(scores Scores, scoredResult ScoredResult) Scores {
	scores.OutcomeAccuracy = scoredResult.Scorecard.Metrics.OutcomeAccuracy
	scores.ToolCallPrecision = scoredResult.Scorecard.Metrics.ToolCallPrecision
	scores.ToolCallRecall = scoredResult.Scorecard.Metrics.ToolCallRecall
	scores.ToolCallF1 = scoredResult.Scorecard.Metrics.ToolCallF1
	scores.ForbiddenToolViolations = scoredResult.Scorecard.Metrics.ForbiddenToolViolations
	scores.PolicyCompliance = scoredResult.Scorecard.Metrics.PolicyCompliance
	scores.ApprovalAccuracy = scoredResult.Scorecard.Metrics.ApprovalAccuracy
	scores.EvidenceCoverage = scoredResult.Scorecard.Metrics.EvidenceCoverage
	scores.ForbiddenEvidenceCount = scoredResult.Scorecard.Metrics.ForbiddenEvidenceCount
	scores.StateMutationAccuracy = scoredResult.Scorecard.Metrics.StateMutationAccuracy
	scores.AuditCompleteness = scoredResult.Scorecard.Metrics.AuditCompleteness
	scores.ContractValidity = scoredResult.Scorecard.Metrics.ContractValidity
	scores.AbstentionAccuracy = scoredResult.Scorecard.Metrics.AbstentionAccuracy
	scores.LatencyCompliance = scoredResult.Scorecard.Metrics.LatencyCompliance
	scores.ToolBudgetCompliance = scoredResult.Scorecard.Metrics.ToolBudgetCompliance
	scores.ScorecardScore = scoredResult.Scorecard.TotalScore
	scores.ScorecardVerdict = scoredResult.HardGateAssessment.FinalVerdict
	scores.ScoredResult = &scoredResult
	return scores
}

func normalizeReplayProvenance(in *ReplayProvenance) *ReplayProvenance {
	if in == nil {
		return nil
	}
	out := *in
	if out.Mode == "" {
		switch {
		case out.BenchmarkCaseID != nil:
			out.Mode = ReplayModeBenchmark
		case out.SourceAgentRunID != nil || out.SourceCognitiveWorkspaceID != nil || out.SourceTraceID != nil:
			out.Mode = ReplayModeReplay
		default:
			out.Mode = ReplayModeAdHoc
		}
	}
	return &out
}

func (s *RunnerService) persistRunProvenance(
	ctx context.Context,
	workspaceID, runID string,
	provenance *ReplayProvenance,
) error {
	if provenance == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE eval_run
		SET benchmark_case_id = ?,
		    synthetic_org_id = ?,
		    source_agent_run_id = ?,
		    source_cognitive_workspace_id = ?,
		    source_trace_id = ?,
		    replay_mode = ?
		WHERE id = ? AND workspace_id = ?`,
		provenance.BenchmarkCaseID,
		provenance.SyntheticOrgID,
		provenance.SourceAgentRunID,
		provenance.SourceCognitiveWorkspaceID,
		provenance.SourceTraceID,
		provenance.Mode,
		runID,
		workspaceID,
	)
	if err != nil {
		return fmt.Errorf("persist eval run provenance: %w", err)
	}
	return nil
}

func (s *RunnerService) attachRunProvenance(ctx context.Context, run *Run) error {
	provenance, err := s.loadRunProvenance(ctx, run.ID, run.WorkspaceID)
	if err != nil {
		return err
	}
	run.Provenance = provenance
	return nil
}

func (s *RunnerService) loadRunProvenance(
	ctx context.Context,
	runID, workspaceID string,
) (*ReplayProvenance, error) {
	var raw runProvenanceRow
	err := s.db.QueryRowContext(ctx, `
		SELECT replay_mode, benchmark_case_id, synthetic_org_id, source_agent_run_id,
		       source_cognitive_workspace_id, source_trace_id
		FROM eval_run
		WHERE id = ? AND workspace_id = ?`,
		runID,
		workspaceID,
	).Scan(
		&raw.Mode,
		&raw.BenchmarkCaseID,
		&raw.SyntheticOrgID,
		&raw.SourceAgentRunID,
		&raw.SourceCognitiveWorkspaceID,
		&raw.SourceTraceID,
	)
	if err != nil {
		return nil, fmt.Errorf("load eval run provenance: %w", err)
	}
	if raw.isEmptyAdHoc() {
		return nil, nil
	}
	return raw.toDomain(), nil
}

func nullStringPtr(in sql.NullString) *string {
	if !in.Valid {
		return nil
	}
	value := in.String
	return &value
}

type runProvenanceRow struct {
	Mode                       string
	BenchmarkCaseID            sql.NullString
	SyntheticOrgID             sql.NullString
	SourceAgentRunID           sql.NullString
	SourceCognitiveWorkspaceID sql.NullString
	SourceTraceID              sql.NullString
}

func (row runProvenanceRow) isEmptyAdHoc() bool {
	return row.Mode == string(ReplayModeAdHoc) &&
		!row.BenchmarkCaseID.Valid &&
		!row.SyntheticOrgID.Valid &&
		!row.SourceAgentRunID.Valid &&
		!row.SourceCognitiveWorkspaceID.Valid &&
		!row.SourceTraceID.Valid
}

func (row runProvenanceRow) toDomain() *ReplayProvenance {
	return &ReplayProvenance{
		Mode:                       ReplayMode(row.Mode),
		BenchmarkCaseID:            nullStringPtr(row.BenchmarkCaseID),
		SyntheticOrgID:             nullStringPtr(row.SyntheticOrgID),
		SourceAgentRunID:           nullStringPtr(row.SourceAgentRunID),
		SourceCognitiveWorkspaceID: nullStringPtr(row.SourceCognitiveWorkspaceID),
		SourceTraceID:              nullStringPtr(row.SourceTraceID),
	}
}
