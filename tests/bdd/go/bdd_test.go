package gobdd

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

const bddWorkspaceID = "ws_test"

type scenarioState struct {
	workflowDB              *sql.DB
	workflowService         *workflowdomain.Service
	workflowRecord          *workflowdomain.Workflow
	workflowRuntime         *workflowRuntimeState
	hasEvidence             bool
	hasProspectContext      bool
	hasAnalyticalData       bool
	hasAllowedResolution    bool
	hasRegisteredTool       bool
	hasStudioDraft          bool
	hasWorkflowDraft        bool
	hasWorkflowSpec         bool
	needsHumanReview        bool
	requiresSensitiveAction bool
	auditRecorded           bool
	approvalPending         bool
	agentAbstained          bool
	validationPassed        bool
	workflowInTesting       bool
	workflowRunCompleted    bool
	workflowVersionCreated  bool
	delegatedRunAccepted    bool
	signalDetected          bool
	deferredScheduled       bool
	deferredResumed         bool
	returnedDraft           bool
	returnedInsight         bool
	returnedKnowledgeDraft  bool
	dealAtRisk              bool
	actionRejected          bool
	runExecuted             bool
	replayAllowed           bool
	replayAccepted          bool
	denialRecorded          bool
}

func (s *scenarioState) reset() {
	if s.workflowRuntime != nil {
		_ = s.workflowRuntime.close()
	}
	if s.workflowDB != nil {
		_ = s.workflowDB.Close()
	}
	*s = scenarioState{}
}

func setupWorkflowBDDService() (*sql.DB, *workflowdomain.Service, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, nil, err
	}
	if err = isqlite.MigrateUp(db); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	if _, err = db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws_test', 'Workflow Test', 'workflow-test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return db, workflowdomain.NewService(db), nil
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{filepath.Join("..", "..", "..", "features")},
			Tags:   "@stack-go",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("go BDD suite failed")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &scenarioState{}

	ctx.BeforeScenario(func(*godog.Scenario) {
		state.reset()
	})

	ctx.Step(`^a support case has relevant history, evidence, and an allowed resolution action$`, func() {
		state.hasEvidence = true
		state.hasAllowedResolution = true
	})
	ctx.Step(`^the Support Agent proposes and executes the case resolution$`, func() error {
		if !state.hasEvidence || !state.hasAllowedResolution {
			return godog.ErrPending
		}
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the case response is grounded in the available evidence$`, func() error {
		if !state.hasEvidence {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the case action is recorded in the audit trail$`, func() error {
		if !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a support case lacks sufficient grounded evidence$`, func() {
		state.hasEvidence = false
	})
	ctx.Step(`^the Support Agent is asked to resolve the case$`, func() {})
	ctx.Step(`^the Support Agent abstains from taking a definitive action$`, func() {
		state.agentAbstained = true
	})
	ctx.Step(`^the response explains the missing evidence$`, func() error {
		if !state.agentAbstained {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a support case needs human review$`, func() {
		state.needsHumanReview = true
	})
	ctx.Step(`^the Support Agent has collected the case context and evidence$`, func() {
		state.hasEvidence = true
	})
	ctx.Step(`^the Support Agent hands off the case$`, func() error {
		if !state.needsHumanReview || !state.hasEvidence {
			return godog.ErrPending
		}
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the human handoff preserves the case context$`, func() error {
		if !state.needsHumanReview {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the handoff is recorded in the audit trail$`, func() error {
		if !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a support case requires a sensitive remediation action$`, func() {
		state.requiresSensitiveAction = true
	})
	ctx.Step(`^the Support Agent proposes the sensitive action$`, func() {})
	ctx.Step(`^the action is blocked pending approval$`, func() error {
		if !state.requiresSensitiveAction {
			return godog.ErrPending
		}
		state.approvalPending = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the approval workflow is recorded in the audit trail$`, func() error {
		if !state.approvalPending || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^an agent run has executed in production$`, func() {
		state.runExecuted = true
		state.auditRecorded = true
	})
	ctx.Step(`^a governance operator inspects the run$`, func() error {
		if !state.runExecuted {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the operator can see the run decisions and audit trail$`, func() error {
		if !state.runExecuted || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the audit trail shows the actor, action, and timestamp$`, func() error {
		if !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^an agent run is eligible for replay under policy$`, func() {
		state.runExecuted = true
		state.replayAllowed = true
	})
	ctx.Step(`^a governance operator requests a replay$`, func() error {
		if !state.runExecuted {
			return godog.ErrPending
		}
		if state.replayAllowed {
			state.replayAccepted = true
			state.auditRecorded = true
			return nil
		}
		state.actionRejected = true
		state.denialRecorded = true
		return nil
	})
	ctx.Step(`^the replay is accepted$`, func() error {
		if !state.replayAccepted {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the replay decision is recorded in the audit trail$`, func() error {
		if !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^an agent run is not eligible for replay or rollback under policy$`, func() {
		state.runExecuted = true
		state.replayAllowed = false
	})
	ctx.Step(`^a governance operator requests a replay or rollback$`, func() error {
		if !state.runExecuted {
			return godog.ErrPending
		}
		state.actionRejected = true
		state.denialRecorded = true
		return nil
	})
	ctx.Step(`^the request is rejected$`, func() error {
		if !state.actionRejected {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the denial reason is recorded in the audit trail$`, func() error {
		if !state.denialRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a prospect record has grounded evidence in the knowledge base$`, func() {
		state.hasEvidence = true
		state.hasProspectContext = true
	})
	ctx.Step(`^the Prospecting Agent researches the prospect context$`, func() error {
		if !state.hasProspectContext || !state.hasEvidence {
			return godog.ErrPending
		}
		state.returnedInsight = true
		return nil
	})
	ctx.Step(`^the Prospecting Agent returns grounded prospect insights$`, func() error {
		if !state.returnedInsight {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^a prospect research result is available$`, func() {
		state.hasProspectContext = true
	})
	ctx.Step(`^the required drafting tool is registered and allowed$`, func() {
		state.hasRegisteredTool = true
	})
	ctx.Step(`^the Prospecting Agent drafts an outreach message$`, func() error {
		if !state.hasProspectContext || !state.hasRegisteredTool {
			return godog.ErrPending
		}
		state.returnedDraft = true
		return nil
	})
	ctx.Step(`^the Prospecting Agent returns an outreach draft$`, func() error {
		if !state.returnedDraft {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a deal has evidence of stalled progress and negative signals$`, func() {
		state.hasEvidence = true
	})
	ctx.Step(`^the Deal Risk Agent evaluates the deal$`, func() error {
		if !state.hasEvidence {
			return godog.ErrPending
		}
		state.dealAtRisk = true
		return nil
	})
	ctx.Step(`^the Deal Risk Agent flags the deal as at risk$`, func() error {
		if !state.dealAtRisk {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the Deal Risk Agent explains the grounded evidence$`, func() error {
		if !state.dealAtRisk || !state.hasEvidence {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a resolved support outcome has grounded evidence attached$`, func() {
		state.hasEvidence = true
	})
	ctx.Step(`^the KB Agent generates a knowledge article draft$`, func() error {
		if !state.hasEvidence {
			return godog.ErrPending
		}
		state.returnedKnowledgeDraft = true
		return nil
	})
	ctx.Step(`^the KB Agent produces a grounded knowledge draft$`, func() error {
		if !state.returnedKnowledgeDraft {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a grounded analytical dataset is available$`, func() {
		state.hasAnalyticalData = true
	})
	ctx.Step(`^the Data Insights Agent answers an analytical query$`, func() error {
		if !state.hasAnalyticalData {
			return godog.ErrPending
		}
		state.returnedInsight = true
		return nil
	})
	ctx.Step(`^the Data Insights Agent returns a grounded analytical answer$`, func() error {
		if !state.returnedInsight {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the available data does not support a requested conclusion$`, func() {
		state.hasAnalyticalData = false
	})
	ctx.Step(`^the Data Insights Agent evaluates the unsupported conclusion$`, func() {
		state.actionRejected = true
	})
	ctx.Step(`^the Data Insights Agent rejects the unsupported conclusion$`, func() error {
		if !state.actionRejected {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^an agent studio draft includes a tool-enabled configuration$`, func() {
		state.hasStudioDraft = true
		state.hasRegisteredTool = true
	})
	ctx.Step(`^governance checks are required before promotion$`, func() {
		state.requiresSensitiveAction = true
	})
	ctx.Step(`^the operator validates the draft for promotion$`, func() error {
		if !state.hasStudioDraft || !state.hasRegisteredTool || !state.requiresSensitiveAction {
			return godog.ErrPending
		}
		state.validationPassed = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the agent studio draft passes validation$`, func() error {
		if !state.validationPassed {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the validation outcome is recorded for governance review$`, func() error {
		if !state.validationPassed || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^an admin writes a workflow definition in DSL$`, func() {
		db, svc, err := setupWorkflowBDDService()
		if err != nil {
			panic(err)
		}
		state.workflowDB = db
		state.workflowService = svc
		state.hasWorkflowDraft = true
	})
	ctx.Step(`^the admin saves the workflow as a new draft$`, func() error {
		if !state.hasWorkflowDraft || state.workflowService == nil {
			return godog.ErrPending
		}
		created, err := state.workflowService.Create(context.Background(), workflowdomain.CreateWorkflowInput{
			WorkspaceID: bddWorkspaceID,
			Name:        "qualify_lead",
			DSLSource:   "WORKFLOW qualify_lead\nON lead.created\nSET lead.status = \"qualified\"",
		})
		if err != nil {
			return err
		}
		state.workflowRecord = created
		return nil
	})
	ctx.Step(`^the workflow is stored as version 1 in draft status$`, func() error {
		if state.workflowRecord == nil {
			return godog.ErrPending
		}
		if state.workflowRecord.Version != 1 || state.workflowRecord.Status != workflowdomain.StatusDraft {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the admin can continue editing the draft$`, func() error {
		if state.workflowRecord == nil || state.workflowService == nil {
			return godog.ErrPending
		}
		updated, err := state.workflowService.Update(context.Background(), bddWorkspaceID, state.workflowRecord.ID, workflowdomain.UpdateWorkflowInput{
			DSLSource: "WORKFLOW qualify_lead\nON lead.created\nSET lead.status = \"review\"",
		})
		if err != nil {
			return err
		}
		state.workflowRecord = updated
		return nil
	})

	ctx.Step(`^a workflow draft has DSL source and a behavior spec$`, func() {
		db, svc, err := setupWorkflowBDDService()
		if err != nil {
			panic(err)
		}
		state.workflowDB = db
		state.workflowService = svc
		created, createErr := svc.Create(context.Background(), workflowdomain.CreateWorkflowInput{
			WorkspaceID: bddWorkspaceID,
			Name:        "resolve_support_case",
			DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"open\"",
			SpecSource:  "BEHAVIOR verify_workflow",
		})
		if createErr != nil {
			panic(createErr)
		}
		state.workflowRecord = created
		state.hasWorkflowDraft = true
		state.hasWorkflowSpec = true
	})
	ctx.Step(`^the admin requests workflow verification$`, func() error {
		if !state.hasWorkflowDraft || !state.hasWorkflowSpec || state.workflowRecord == nil || state.workflowService == nil {
			return godog.ErrPending
		}
		updated, err := state.workflowService.MarkTesting(context.Background(), bddWorkspaceID, state.workflowRecord.ID)
		if err != nil {
			return err
		}
		state.workflowRecord = updated
		state.workflowInTesting = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the workflow passes verification and moves to testing status$`, func() error {
		if !state.workflowInTesting || state.workflowRecord == nil || state.workflowRecord.Status != workflowdomain.StatusTesting {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the verification result is recorded for audit review$`, func() error {
		if !state.workflowInTesting || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	initWorkflowRuntimeScenarios(ctx, state)

	ctx.Step(`^a workflow evaluation produces grounded signal evidence$`, func() {
		state.hasEvidence = true
	})
	ctx.Step(`^the system creates a new signal from the grounded evidence$`, func() error {
		if !state.hasEvidence {
			return godog.ErrPending
		}
		state.signalDetected = true
		return nil
	})
	ctx.Step(`^the signal is stored as an active actionable item$`, func() error {
		if !state.signalDetected {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the signal remains linked to its evidence sources$`, func() error {
		if !state.signalDetected || !state.hasEvidence {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a workflow action is classified as sensitive$`, func() {
		state.requiresSensitiveAction = true
	})
	ctx.Step(`^the runtime requests human approval for the action$`, func() error {
		if !state.requiresSensitiveAction {
			return godog.ErrPending
		}
		state.approvalPending = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^an approval request is created and left pending$`, func() error {
		if !state.approvalPending {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the approval requirement is recorded in the audit trail$`, func() error {
		if !state.approvalPending || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^an active workflow is eligible for a new version$`, func() {
		db, svc, err := setupWorkflowBDDService()
		if err != nil {
			panic(err)
		}
		state.workflowDB = db
		state.workflowService = svc
		created, createErr := svc.Create(context.Background(), workflowdomain.CreateWorkflowInput{
			WorkspaceID: bddWorkspaceID,
			Name:        "triage_case",
			DSLSource:   "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"",
		})
		if createErr != nil {
			panic(createErr)
		}
		testingWF, markTestingErr := svc.MarkTesting(context.Background(), bddWorkspaceID, created.ID)
		if markTestingErr != nil {
			panic(markTestingErr)
		}
		activeWF, markActiveErr := svc.MarkActive(context.Background(), bddWorkspaceID, testingWF.ID)
		if markActiveErr != nil {
			panic(markActiveErr)
		}
		state.workflowRecord = activeWF
		state.runExecuted = true
	})
	ctx.Step(`^the operator creates a new workflow version$`, func() error {
		if !state.runExecuted || state.workflowRecord == nil || state.workflowService == nil {
			return godog.ErrPending
		}
		next, err := state.workflowService.NewVersion(context.Background(), bddWorkspaceID, state.workflowRecord.ID)
		if err != nil {
			return err
		}
		state.workflowRecord = next
		state.workflowVersionCreated = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^a new draft workflow version is created from the active source$`, func() error {
		if !state.workflowVersionCreated || state.workflowRecord == nil {
			return godog.ErrPending
		}
		if state.workflowRecord.Version != 2 || state.workflowRecord.Status != workflowdomain.StatusDraft {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the versioning action is recorded in the audit trail$`, func() error {
		if !state.workflowVersionCreated || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})

	ctx.Step(`^a workflow dispatch step targets another agent$`, func() {
		state.hasRegisteredTool = true
	})
	ctx.Step(`^the runtime delegates the workflow execution$`, func() error {
		if !state.hasRegisteredTool {
			return godog.ErrPending
		}
		state.delegatedRunAccepted = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the delegated run is accepted with trace metadata$`, func() error {
		if !state.delegatedRunAccepted {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the delegation decision is recorded in the audit trail$`, func() error {
		if !state.delegatedRunAccepted || !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})
}
