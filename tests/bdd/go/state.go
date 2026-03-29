package gobdd

import (
	"database/sql"

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
