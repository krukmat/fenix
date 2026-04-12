// W6-T3: Wedge-first seed — removed workflow fixtures, added approval, handoff,
// denied_by_policy, completed runs, usage events, and quota policy.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/config"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	defaultAPIURL      = "http://localhost:8080"
	defaultDatabaseURL = "./data/fenixcrm.db"
	testEmail          = "e2e@fenixcrm.test"
	testPassword       = "e2eTestPass123!"
	testDisplayName    = "E2E Test User"
	testWorkspaceName  = "E2E Test Workspace"
	emptyJSONArray     = "[]"
	cleanupTableDeal   = "deal"
)

var cleanupWorkspaceQueries = map[string]string{
	"approval_request":   "DELETE FROM approval_request WHERE workspace_id = ?",
	"quota_state":        "DELETE FROM quota_state WHERE workspace_id = ?",
	"quota_policy":       "DELETE FROM quota_policy WHERE workspace_id = ?",
	"usage_event":        "DELETE FROM usage_event WHERE workspace_id = ?",
	"vec_embedding":      "DELETE FROM vec_embedding WHERE workspace_id = ?",
	"embedding_document": "DELETE FROM embedding_document WHERE workspace_id = ?",
	"agent_run_step":     "DELETE FROM agent_run_step WHERE workspace_id = ?",
	"agent_run":          "DELETE FROM agent_run WHERE workspace_id = ?",
	"signal":             "DELETE FROM signal WHERE workspace_id = ?",
	"attachment":         "DELETE FROM attachment WHERE workspace_id = ?",
	"note":               "DELETE FROM note WHERE workspace_id = ?",
	"timeline_event":     "DELETE FROM timeline_event WHERE workspace_id = ?",
	"activity":           "DELETE FROM activity WHERE workspace_id = ?",
	"evidence":           "DELETE FROM evidence WHERE workspace_id = ?",
	"knowledge_item":     "DELETE FROM knowledge_item WHERE workspace_id = ?",
	"case_ticket":        "DELETE FROM case_ticket WHERE workspace_id = ?",
	cleanupTableDeal:     "DELETE FROM deal WHERE workspace_id = ?",
	"contact":            "DELETE FROM contact WHERE workspace_id = ?",
	"account":            "DELETE FROM account WHERE workspace_id = ?",
	"agent_definition":   "DELETE FROM agent_definition WHERE workspace_id = ?",
}

type authResponse struct {
	Token       string `json:"token"`
	UserID      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

// seedOutput is the JSON written to stdout — consumed by seed-and-run.sh.
type seedOutput struct {
	Credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	} `json:"credentials"`
	// Auth exposes the runtime session from loginOrRegister so the screenshot
	// runner can bootstrap an authenticated session via the e2e-bootstrap deep
	// link instead of driving the login UI. See
	// docs/plans/maestro-screenshot-auth-bypass-plan.md.
	Auth struct {
		Token       string `json:"token"`
		UserID      string `json:"userId"`
		WorkspaceID string `json:"workspaceId"`
	} `json:"auth"`
	Account struct {
		ID string `json:"id"`
	} `json:"account"`
	Contact struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"contact"`
	Deal struct {
		ID string `json:"id"`
	} `json:"deal"`
	Case struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	} `json:"case"`
	AgentRuns struct {
		// W6-T3: wedge-relevant run statuses
		CompletedID      string `json:"completedId"`
		HandoffID        string `json:"handoffId"`
		DeniedByPolicyID string `json:"deniedByPolicyId"`
	} `json:"agentRuns"`
	Inbox struct {
		ApprovalID string `json:"approvalId"`
		SignalID   string `json:"signalId"`
	} `json:"inbox"`
}

type requestError struct {
	Status int
	Body   string
}

func (e *requestError) Error() string {
	return fmt.Sprintf("request failed with status %d: %s", e.Status, e.Body)
}

func main() {
	ctx := context.Background()
	apiURL := envOr("API_URL", defaultAPIURL)
	databaseURL := envOr("DATABASE_URL", defaultDatabaseURL)

	db, err := sqlite.NewDB(databaseURL)
	if err != nil {
		fail(err)
	}
	defer db.Close()

	auth, err := loginOrRegister(ctx, apiURL, db)
	if err != nil {
		fail(err)
	}

	seeded, err := seedFixtures(ctx, db, auth)
	if err != nil {
		fail(err)
	}

	seeded.Credentials.Email = testEmail
	seeded.Credentials.Password = testPassword
	// Expose auth session for the screenshot runner's e2e-bootstrap deep link.
	seeded.Auth.Token = auth.Token
	seeded.Auth.UserID = auth.UserID
	seeded.Auth.WorkspaceID = auth.WorkspaceID

	err = json.NewEncoder(os.Stdout).Encode(seeded)
	if err != nil {
		fail(err)
	}
}

type wedgeRunIDs struct {
	completedID string
	handoffIDs  []string
	deniedIDs   []string
}

func seedFixtures(ctx context.Context, db *sql.DB, auth authResponse) (*seedOutput, error) {
	err := cleanupExistingFixtures(ctx, db, auth.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("cleanupExistingFixtures: %w", err)
	}

	baseNow := time.Now().UTC().Truncate(time.Second)
	suffix := baseNow.Format("20060102T150405")

	accountID, err := seedAccount(ctx, db, auth, suffix)
	if err != nil {
		return nil, fmt.Errorf("seedAccount: %w", err)
	}

	contactID, contactEmail, err := seedContact(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, fmt.Errorf("seedContact: %w", err)
	}

	dealID, err := seedDeal(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, fmt.Errorf("seedDeal: %w", err)
	}
	if seedKnowledgeErr := seedDealKnowledge(ctx, db, auth, dealID, suffix); seedKnowledgeErr != nil {
		return nil, fmt.Errorf("seedDealKnowledge: %w", seedKnowledgeErr)
	}

	caseID, caseSubject, err := seedCase(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, fmt.Errorf("seedCase: %w", err)
	}

	// W6-T3: wedge runs — completed, handed-off, denied-by-policy
	runs, err := seedWedgeRuns(ctx, db, auth, caseID, suffix, baseNow)
	if err != nil {
		return nil, err
	}

	approvalID, signalID, err := seedGovernanceAndApproval(ctx, db, auth, runs.completedID, dealID, caseID, suffix, baseNow)
	if err != nil {
		return nil, fmt.Errorf("seedApproval: %w", err)
	}

	return buildSeedOutput(accountID, contactID, contactEmail, dealID, caseID, caseSubject, runs, approvalID, signalID), nil
}

func cleanupExistingFixtures(ctx context.Context, db *sql.DB, workspaceID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = setForeignKeys(ctx, tx, false); err != nil {
		return err
	}

	tables := []string{
		"approval_request",
		"quota_state",
		"quota_policy",
		"usage_event",
		"vec_embedding",
		"embedding_document",
		"agent_run_step",
		"agent_run",
		"signal",
		"attachment",
		"note",
		"timeline_event",
		"activity",
		"evidence",
		"knowledge_item",
		"case_ticket",
		cleanupTableDeal,
		"contact",
		"account",
		"agent_definition",
	}

	if err = cleanupWorkspaceTables(ctx, tx, workspaceID, tables); err != nil {
		return err
	}

	if err = setForeignKeys(ctx, tx, true); err != nil {
		return err
	}

	return tx.Commit()
}

func setForeignKeys(ctx context.Context, tx *sql.Tx, enabled bool) error {
	value := "OFF"
	if enabled {
		value = "ON"
	}
	_, err := tx.ExecContext(ctx, "PRAGMA foreign_keys = "+value)
	return err
}

func cleanupWorkspaceTables(ctx context.Context, tx *sql.Tx, workspaceID string, tables []string) error {
	for _, table := range tables {
		query, ok := cleanupWorkspaceQueries[table]
		if !ok {
			return fmt.Errorf("unsupported cleanup table: %s", table)
		}
		if _, err := tx.ExecContext(ctx, query, workspaceID); err != nil {
			return err
		}
	}
	return nil
}

// seedWedgeRuns seeds the three wedge-relevant run statuses (completed, handed-off, denied).
// W6-T3: extracted to keep seedFixtures within funlen limit.
func seedWedgeRuns(ctx context.Context, db *sql.DB, auth authResponse, caseID, suffix string, baseNow time.Time) (wedgeRunIDs, error) {
	completedID, err := seedRun(ctx, db, auth, runParams{
		entityType: "case",
		entityID:   caseID,
		suffix:     suffix + "_completed",
		status:     "completed",
		agentType:  "support",
		agentName:  "Support Agent",
		latencyMs:  1200,
		cost:       0.05,
		occurredAt: baseNow.Add(-55 * time.Minute),
	})
	if err != nil {
		return wedgeRunIDs{}, fmt.Errorf("seedCompletedRun: %w", err)
	}

	handoffOlderID, err := seedRun(ctx, db, auth, runParams{
		entityType:       "case",
		entityID:         caseID,
		suffix:           suffix + "_handoff_old",
		status:           "handed_off",
		agentType:        "support",
		agentName:        "Support Agent",
		latencyMs:        800,
		cost:             0.03,
		abstentionReason: "Escalated to billing lead after refund policy mismatch",
		occurredAt:       baseNow.Add(-20 * time.Minute),
	})
	if err != nil {
		return wedgeRunIDs{}, fmt.Errorf("seedHandoffRun: %w", err)
	}

	handoffLatestID, err := seedRun(ctx, db, auth, runParams{
		entityType:       "case",
		entityID:         caseID,
		suffix:           suffix + "_handoff_new",
		status:           "handed_off",
		agentType:        "support",
		agentName:        "Support Agent",
		latencyMs:        730,
		cost:             0.03,
		abstentionReason: "Escalated to operations owner for contract exception",
		occurredAt:       baseNow.Add(-10 * time.Minute),
	})
	if err != nil {
		return wedgeRunIDs{}, fmt.Errorf("seedLatestHandoffRun: %w", err)
	}

	deniedOlderID, err := seedRun(ctx, db, auth, runParams{
		entityType:      "case",
		entityID:        caseID,
		suffix:          suffix + "_denied_old",
		status:          "denied_by_policy",
		agentType:       "support",
		agentName:       "Support Agent",
		latencyMs:       300,
		cost:            0.01,
		rejectionReason: "Policy blocked refund promise without finance approval",
		occurredAt:      baseNow.Add(-18 * time.Minute),
	})
	if err != nil {
		return wedgeRunIDs{}, fmt.Errorf("seedDeniedRun: %w", err)
	}

	deniedLatestID, err := seedRun(ctx, db, auth, runParams{
		entityType:      "case",
		entityID:        caseID,
		suffix:          suffix + "_denied_new",
		status:          "denied_by_policy",
		agentType:       "support",
		agentName:       "Support Agent",
		latencyMs:       280,
		cost:            0.01,
		rejectionReason: "Policy blocked outbound message with unverified pricing",
		occurredAt:      baseNow.Add(-8 * time.Minute),
	})
	if err != nil {
		return wedgeRunIDs{}, fmt.Errorf("seedLatestDeniedRun: %w", err)
	}

	return wedgeRunIDs{
		completedID: completedID,
		handoffIDs:  []string{handoffLatestID, handoffOlderID},
		deniedIDs:   []string{deniedLatestID, deniedOlderID},
	}, nil
}

func buildSeedOutput(accountID, contactID, contactEmail, dealID, caseID, caseSubject string, runs wedgeRunIDs, approvalID, signalID string) *seedOutput {
	out := &seedOutput{}
	out.Account.ID = accountID
	out.Contact.ID = contactID
	out.Contact.Email = contactEmail
	out.Deal.ID = dealID
	out.Case.ID = caseID
	out.Case.Subject = caseSubject
	out.AgentRuns.CompletedID = runs.completedID
	if len(runs.handoffIDs) > 0 {
		out.AgentRuns.HandoffID = runs.handoffIDs[0]
	}
	if len(runs.deniedIDs) > 0 {
		out.AgentRuns.DeniedByPolicyID = runs.deniedIDs[0]
	}
	out.Inbox.ApprovalID = approvalID
	out.Inbox.SignalID = signalID
	return out
}

// ─── CRM fixtures ────────────────────────────────────────────────────────────

func seedAccount(ctx context.Context, db *sql.DB, auth authResponse, suffix string) (string, error) {
	svc := crm.NewAccountService(db)
	account, err := svc.Create(ctx, crm.CreateAccountInput{
		WorkspaceID: auth.WorkspaceID,
		Name:        "E2E Wedge Account " + suffix,
		Industry:    "Technology",
		OwnerID:     auth.UserID,
	})
	if err != nil {
		return "", err
	}
	return account.ID, nil
}

func seedContact(ctx context.Context, db *sql.DB, auth authResponse, accountID, suffix string) (string, string, error) {
	svc := crm.NewContactService(db)
	email := "e2e.wedge.contact." + suffix + "@fenixcrm.test"
	contact, err := svc.Create(ctx, crm.CreateContactInput{
		WorkspaceID: auth.WorkspaceID,
		AccountID:   accountID,
		FirstName:   "Wedge",
		LastName:    "Contact " + suffix,
		Email:       email,
		OwnerID:     auth.UserID,
	})
	if err != nil {
		return "", "", err
	}
	return contact.ID, email, nil
}

func seedDeal(ctx context.Context, db *sql.DB, auth authResponse, accountID, suffix string) (string, error) {
	pipelineSvc := crm.NewPipelineService(db)
	pipeline, err := pipelineSvc.Create(ctx, crm.CreatePipelineInput{
		WorkspaceID: auth.WorkspaceID,
		Name:        "E2E Wedge Sales " + suffix,
		EntityType:  "deal",
	})
	if err != nil {
		return "", err
	}

	stage, err := pipelineSvc.CreateStage(ctx, crm.CreatePipelineStageInput{
		PipelineID: pipeline.ID,
		Name:       "Discovery",
		Position:   1,
	})
	if err != nil {
		return "", err
	}

	svc := crm.NewDealService(db)
	deal, err := svc.Create(ctx, crm.CreateDealInput{
		WorkspaceID: auth.WorkspaceID,
		AccountID:   accountID,
		PipelineID:  pipeline.ID,
		StageID:     stage.ID,
		OwnerID:     auth.UserID,
		Title:       "E2E Wedge Deal " + suffix,
		Status:      "open",
	})
	if err != nil {
		return "", err
	}
	return deal.ID, nil
}

func seedDealKnowledge(ctx context.Context, db *sql.DB, auth authResponse, dealID, suffix string) error {
	llmProvider, err := llm.NewEmbedProvider(config.Load())
	if err != nil {
		return err
	}

	bus := eventbus.New()
	ingest := knowledge.NewIngestService(db, bus)
	embedder := knowledge.NewEmbedderService(db, llmProvider)

	entityType := "deal"
	entityID := dealID
	sourceSystem := "e2e-seed"
	permissionContext := fmt.Sprintf(`{"workspace_id":%q}`, auth.WorkspaceID)

	item, ingestErr := ingest.Ingest(ctx, knowledge.CreateKnowledgeItemInput{
		WorkspaceID:       auth.WorkspaceID,
		SourceSystem:      &sourceSystem,
		SourceType:        knowledge.SourceTypeDocument,
		PermissionContext: &permissionContext,
		Title:             "Deal follow-up brief source " + suffix,
		RawContent: `entity_type:deal
entity_id:` + dealID + `
Latest updates timeline:
- Champion confirmed budget approval for the expansion deal.
- Procurement requested the security addendum this week.
- Decision call is scheduled for Friday.
Risks:
- Legal review could slip by three business days.
- Procurement needs revised pricing language.
Next steps:
- Send the security addendum today.
- Follow up with procurement tomorrow.
- Update the deal after the Friday decision call.`,
		EntityType: &entityType,
		EntityID:   &entityID,
	})
	if ingestErr != nil {
		return ingestErr
	}
	if embedErr := embedder.EmbedChunks(ctx, item.ID, auth.WorkspaceID); embedErr != nil {
		return embedErr
	}

	return nil
}

func seedCase(ctx context.Context, db *sql.DB, auth authResponse, accountID, suffix string) (string, string, error) {
	svc := crm.NewCaseService(db)
	subject := "E2E Wedge Case " + suffix
	ct, err := svc.Create(ctx, crm.CreateCaseInput{
		WorkspaceID: auth.WorkspaceID,
		AccountID:   accountID,
		OwnerID:     auth.UserID,
		Subject:     subject,
		Priority:    "medium",
		Status:      "open",
	})
	if err != nil {
		return "", "", err
	}
	return ct.ID, subject, nil
}

// seedGovernanceAndApproval seeds usage events, quota policy, and inbox approval in one call.
// W6-T3: extracted to reduce seedFixtures cognitive complexity below gocognit threshold.
func seedGovernanceAndApproval(ctx context.Context, db *sql.DB, auth authResponse, runID, dealID, caseID, suffix string, baseNow time.Time) (string, string, error) {
	if err := seedUsageEvents(ctx, db, auth, runID); err != nil {
		return "", "", fmt.Errorf("seedUsageEvents: %w", err)
	}
	if err := seedQuotaPolicy(ctx, db, auth); err != nil {
		return "", "", fmt.Errorf("seedQuotaPolicy: %w", err)
	}
	if err := ensureSignalAccessRole(ctx, db, auth); err != nil {
		return "", "", fmt.Errorf("ensureSignalAccessRole: %w", err)
	}
	signalIDs, err := seedInboxSignals(ctx, db, auth, dealID, caseID, runID, suffix, baseNow)
	if err != nil {
		return "", "", fmt.Errorf("seedInboxSignals: %w", err)
	}
	approvalIDs, err := seedInboxApprovals(ctx, db, auth, caseID, suffix, baseNow)
	if err != nil {
		return "", "", fmt.Errorf("seedApproval: %w", err)
	}
	if len(approvalIDs) == 0 {
		return "", "", errors.New("seedApproval: expected at least one approval")
	}
	if len(signalIDs) == 0 {
		return "", "", errors.New("seedInboxSignals: expected at least one signal")
	}
	return approvalIDs[0], signalIDs[0], nil
}

// ─── Agent run fixtures ───────────────────────────────────────────────────────

type runParams struct {
	entityType       string
	entityID         string
	suffix           string
	status           string
	agentType        string
	agentName        string
	latencyMs        int
	cost             float64
	abstentionReason string
	rejectionReason  string
	occurredAt       time.Time
}

func seedRun(ctx context.Context, db *sql.DB, auth authResponse, p runParams) (string, error) {
	agentDefID := uuid.NewV7().String()
	runID := uuid.NewV7().String()
	now := p.occurredAt.UTC().Truncate(time.Second)
	if now.IsZero() {
		now = time.Now().UTC().Truncate(time.Second)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO agent_definition (
			id, workspace_id, name, description, agent_type, objective,
			allowed_tools, limits, trigger_config, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, '[]', '{}', '{}', 'active', ?, ?)`,
		agentDefID, auth.WorkspaceID,
		"e2e_wedge_"+p.suffix, "Seeded for Maestro wedge audit",
		p.agentType, `{"goal":"wedge smoke"}`,
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	triggerContext := fmt.Sprintf(`{"entity_type":%q,"entity_id":%q}`, p.entityType, p.entityID)
	output := fmt.Sprintf(`{"agent_name":%q,"entity_type":%q,"entity_id":%q,"rejection_reason":%q}`,
		p.agentName, p.entityType, p.entityID, p.rejectionReason)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, triggered_by_user_id,
			trigger_type, trigger_context, status, inputs,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace,
			tool_calls, output, abstention_reason,
			total_tokens, total_cost, latency_ms, trace_id,
			started_at, completed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, auth.WorkspaceID, agentDefID, auth.UserID,
		"manual", triggerContext, p.status,
		`{"source":"maestro-e2e"}`,
		emptyJSONArray, emptyJSONArray, emptyJSONArray, emptyJSONArray,
		output, p.abstentionReason,
		512, p.cost, p.latencyMs,
		uuid.NewV7().String(),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	return runID, nil
}

// ─── Governance fixtures ─────────────────────────────────────────────────────

func seedUsageEvents(ctx context.Context, db *sql.DB, auth authResponse, runID string) error {
	now := time.Now().UTC()
	events := []struct {
		toolName      string
		modelName     string
		inputUnits    int
		outputUnits   int
		estimatedCost float64
		latencyMs     int
	}{
		{"crm_lookup", "n/a", 256, 0, 0.01, 120},
		{"support_agent", "gpt-5", 256, 128, 0.05, 1200},
	}
	for _, e := range events {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO usage_event (
				id, workspace_id, actor_id, actor_type, run_id,
				tool_name, model_name, input_units, output_units,
				estimated_cost, latency_ms, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.NewV7().String(), auth.WorkspaceID,
			auth.UserID, "user", runID,
			e.toolName, e.modelName, e.inputUnits, e.outputUnits,
			e.estimatedCost, e.latencyMs, now.Format(time.RFC3339),
		); err != nil {
			return err
		}
	}
	return nil
}

func seedQuotaPolicy(ctx context.Context, db *sql.DB, auth authResponse) error {
	now := time.Now().UTC()
	policyID := uuid.NewV7().String()

	// Insert policy
	if _, err := db.ExecContext(ctx, `
		INSERT INTO quota_policy (
			id, workspace_id, policy_type, scope_type, scope_id, metric_name,
			limit_value, reset_period, enforcement_mode, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		policyID, auth.WorkspaceID,
		"token_budget", "workspace", nil, "tokens", 100000,
		"daily", "soft",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		// Policy table may not exist or policy already inserted — non-fatal for demo
		return nil
	}

	// Insert quota state for current period
	periodStart := now.UTC().Truncate(24 * time.Hour)
	periodEnd := periodStart.Add(24 * time.Hour).Add(-time.Second)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO quota_state (
			id, workspace_id, quota_policy_id, current_value,
			period_start, period_end, last_event_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewV7().String(), auth.WorkspaceID, policyID,
		1500,
		periodStart.Format(time.RFC3339), periodEnd.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return nil
}

// ─── Inbox fixtures ──────────────────────────────────────────────────────────

func seedInboxApprovals(ctx context.Context, db *sql.DB, auth authResponse, caseID, suffix string, baseNow time.Time) ([]string, error) {
	approvals := []struct {
		action    string
		reason    string
		createdAt time.Time
		expiresIn time.Duration
	}{
		{
			action:    "close_case",
			reason:    "Customer requested a billing exception before closure",
			createdAt: baseNow.Add(-30 * time.Minute),
			expiresIn: 2 * time.Hour,
		},
		{
			action:    "send_external_email",
			reason:    "Manager confirmation required before sending pricing terms",
			createdAt: baseNow.Add(-6 * time.Minute),
			expiresIn: 6 * time.Hour,
		},
	}

	ids := make([]string, 0, len(approvals))
	for index, approval := range approvals {
		approvalID, err := seedApproval(ctx, db, auth, caseID, suffix, index, approval.action, approval.reason, approval.createdAt, approval.expiresIn)
		if err != nil {
			return nil, err
		}
		ids = append(ids, approvalID)
	}

	return ids, nil
}

func seedApproval(
	ctx context.Context,
	db *sql.DB,
	auth authResponse,
	caseID, suffix string,
	index int,
	action, reason string,
	createdAt time.Time,
	expiresIn time.Duration,
) (string, error) {
	now := createdAt.UTC().Truncate(time.Second)
	approvalID := uuid.NewV7().String()
	expiresAt := now.Add(expiresIn)

	payload := fmt.Sprintf(`{"entity_type":"case","entity_id":%q,"action":%q}`, caseID, action)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO approval_request (
			id, workspace_id, requested_by, approver_id,
			action, resource_type, resource_id, payload,
			reason, status, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		approvalID, auth.WorkspaceID,
		auth.UserID, auth.UserID,
		action, "case", caseID, payload,
		fmt.Sprintf("E2E wedge seed approval %s #%d: %s", suffix, index+1, reason),
		"pending",
		expiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}
	return approvalID, nil
}

func seedInboxSignals(
	ctx context.Context,
	db *sql.DB,
	auth authResponse,
	dealID, caseID, runID, suffix string,
	baseNow time.Time,
) ([]string, error) {
	signals := []struct {
		entityType string
		entityID   string
		signalType string
		confidence float64
		summary    string
		createdAt  time.Time
	}{
		{
			entityType: "deal",
			entityID:   dealID,
			signalType: "expansion_intent",
			confidence: 0.94,
			summary:    "Procurement asked for the security addendum and budget is already confirmed.",
			createdAt:  baseNow.Add(-16 * time.Minute),
		},
		{
			entityType: "case",
			entityID:   caseID,
			signalType: "escalation_risk",
			confidence: 0.73,
			summary:    "The customer has asked twice for a manual exception in the last hour.",
			createdAt:  baseNow.Add(-4 * time.Minute),
		},
	}

	ids := make([]string, 0, len(signals))
	for index, signal := range signals {
		signalID, err := seedSignal(ctx, db, auth, runID, suffix, index, signal.entityType, signal.entityID, signal.signalType, signal.summary, signal.confidence, signal.createdAt)
		if err != nil {
			return nil, err
		}
		ids = append(ids, signalID)
	}

	return ids, nil
}

func seedSignal(
	ctx context.Context,
	db *sql.DB,
	auth authResponse,
	runID, suffix string,
	index int,
	entityType, entityID, signalType, summary string,
	confidence float64,
	createdAt time.Time,
) (string, error) {
	now := createdAt.UTC().Truncate(time.Second)
	signalID := uuid.NewV7().String()
	evidenceIDs := fmt.Sprintf(`["e2e-signal-%s-%d"]`, suffix, index+1)
	metadata := fmt.Sprintf(`{"summary":%q,"label":"E2E inbox signal %d"}`, summary, index+1)

	_, err := db.ExecContext(ctx, `
		INSERT INTO signal (
			id, workspace_id, entity_type, entity_id, signal_type, confidence,
			evidence_ids, source_type, source_id, metadata, status,
			dismissed_by, dismissed_at, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, ?, ?)`,
		signalID, auth.WorkspaceID, entityType, entityID, signalType, confidence,
		evidenceIDs, "agent_run", runID, metadata, "active",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return "", err
	}
	return signalID, nil
}

func ensureSignalAccessRole(ctx context.Context, db *sql.DB, auth authResponse) error {
	const roleName = "E2E Signal Access"
	const permissions = `{"api":["signals.list","signals.dismiss"]}`

	now := time.Now().UTC().Truncate(time.Second)
	var roleID string
	err := db.QueryRowContext(ctx, `
		SELECT id
		FROM role
		WHERE workspace_id = ? AND name = ?
		LIMIT 1
	`, auth.WorkspaceID, roleName).Scan(&roleID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		roleID = uuid.NewV7().String()
		if _, execErr := db.ExecContext(ctx, `
			INSERT INTO role (
				id, workspace_id, name, description, permissions, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			roleID, auth.WorkspaceID, roleName,
			"Grants signal list/dismiss for deterministic screenshot fixtures",
			permissions,
			now.Format(time.RFC3339), now.Format(time.RFC3339),
		); execErr != nil {
			return execErr
		}
	}

	_, err = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO user_role (id, user_id, role_id, created_at)
		VALUES (?, ?, ?, ?)
	`, uuid.NewV7().String(), auth.UserID, roleID, now.Format(time.RFC3339))
	return err
}

// ─── Auth helpers ─────────────────────────────────────────────────────────────

func loginOrRegister(ctx context.Context, apiURL string, db *sql.DB) (authResponse, error) {
	_, err := lookupExistingAuth(ctx, db, testEmail)
	if err == nil {
		return requestAuth(ctx, apiURL, "/auth/login", map[string]string{
			"email":    testEmail,
			"password": testPassword,
		})
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return authResponse{}, err
	}

	auth, err := requestAuth(ctx, apiURL, "/auth/register", map[string]string{
		"email":         testEmail,
		"password":      testPassword,
		"displayName":   testDisplayName,
		"workspaceName": testWorkspaceName,
	})
	if err == nil {
		return auth, nil
	}

	reqErr := &requestError{}
	if asRequestError(err, reqErr) && (reqErr.Status == http.StatusConflict || reqErr.Status == http.StatusTooManyRequests) {
		return lookupExistingAuth(ctx, db, testEmail)
	}

	return authResponse{}, err
}

func lookupExistingAuth(ctx context.Context, db *sql.DB, email string) (authResponse, error) {
	var auth authResponse
	err := db.QueryRowContext(ctx, `
		SELECT id, workspace_id
		FROM user_account
		WHERE email = ? AND status = 'active'
		LIMIT 1
	`, email).Scan(&auth.UserID, &auth.WorkspaceID)
	if err != nil {
		return authResponse{}, err
	}
	return auth, nil
}

func requestAuth(ctx context.Context, apiURL, path string, payload map[string]string) (authResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return authResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+path, bytes.NewReader(body))
	if err != nil {
		return authResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return authResponse{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return authResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return authResponse{}, &requestError{Status: resp.StatusCode, Body: string(raw)}
	}

	var auth authResponse
	err = json.Unmarshal(raw, &auth)
	if err != nil {
		return authResponse{}, err
	}
	return auth, nil
}

func asRequestError(err error, target *requestError) bool {
	if err == nil {
		return false
	}
	var reqErr *requestError
	if !errors.As(err, &reqErr) || reqErr == nil {
		return false
	}
	*target = *reqErr
	return true
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}
