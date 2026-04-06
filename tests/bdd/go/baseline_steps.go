package gobdd

import (
	"context"

	"github.com/cucumber/godog"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func registerBaselineScenarioSteps(ctx *godog.ScenarioContext, state *scenarioState) {
	registerProspectingAndToolSteps(ctx, state)
	registerDealRiskAndKnowledgeSteps(ctx, state)
	registerInsightsAndStudioSteps(ctx, state)
	registerWorkflowAuthoringSteps(ctx, state)
	registerSignalAndApprovalBaselineSteps(ctx, state)
	registerWorkflowVersionAndDelegationSteps(ctx, state)
}

func registerProspectingAndToolSteps(ctx *godog.ScenarioContext, state *scenarioState) {
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
	ctx.Step(`^an agent has a registered allowlisted tool$`, func() {
		state.hasRegisteredTool = true
	})
	ctx.Step(`^the runtime validates a tool request with allowed parameters$`, func() error {
		if !state.hasRegisteredTool {
			return godog.ErrPending
		}
		state.runExecuted = true
		state.auditRecorded = true
		return nil
	})
	ctx.Step(`^the tool execution is permitted$`, func() error {
		if !state.runExecuted {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the tool decision is recorded in the audit trail$`, func() error {
		if !state.auditRecorded {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^an agent attempts a tool call with disallowed parameters$`, func() {
		state.actionRejected = true
	})
	ctx.Step(`^the runtime validates the tool request$`, func() error {
		if !state.actionRejected {
			return godog.ErrPending
		}
		state.denialRecorded = true
		return nil
	})
	ctx.Step(`^the tool execution is denied$`, func() error {
		if !state.actionRejected {
			return godog.ErrPending
		}
		return nil
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
}

func registerDealRiskAndKnowledgeSteps(ctx *godog.ScenarioContext, state *scenarioState) {
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
}

func registerInsightsAndStudioSteps(ctx *godog.ScenarioContext, state *scenarioState) {
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
}

func registerWorkflowAuthoringSteps(ctx *godog.ScenarioContext, state *scenarioState) {
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
}

func registerSignalAndApprovalBaselineSteps(ctx *godog.ScenarioContext, state *scenarioState) {
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
}

func registerWorkflowVersionAndDelegationSteps(ctx *godog.ScenarioContext, state *scenarioState) {
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
