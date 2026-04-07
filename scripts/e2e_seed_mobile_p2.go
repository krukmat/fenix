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
)

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

	auth, err := loginOrRegister(ctx, apiURL)
	if err != nil {
		fail(err)
	}

	db, err := sqlite.NewDB(databaseURL)
	if err != nil {
		fail(err)
	}
	defer db.Close()

	seeded, err := seedFixtures(ctx, db, auth)
	if err != nil {
		fail(err)
	}

	seeded.Credentials.Email = testEmail
	seeded.Credentials.Password = testPassword

	if err := json.NewEncoder(os.Stdout).Encode(seeded); err != nil {
		fail(err)
	}
}

func seedFixtures(ctx context.Context, db *sql.DB, auth authResponse) (*seedOutput, error) {
	suffix := time.Now().UTC().Format("20060102T150405")

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

	caseID, caseSubject, err := seedCase(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, fmt.Errorf("seedCase: %w", err)
	}

	// W6-T3: wedge runs — completed, handed-off, denied-by-policy
	completedRunID, err := seedRun(ctx, db, auth, runParams{
		entityType: "case",
		entityID:   caseID,
		suffix:     suffix + "_completed",
		status:     "completed",
		agentType:  "support",
		agentName:  "Support Agent",
		latencyMs:  1200,
		cost:       0.05,
	})
	if err != nil {
		return nil, fmt.Errorf("seedCompletedRun: %w", err)
	}

	handoffRunID, err := seedRun(ctx, db, auth, runParams{
		entityType:    "case",
		entityID:      caseID,
		suffix:        suffix + "_handoff",
		status:        "handed_off",
		agentType:     "support",
		agentName:     "Support Agent",
		latencyMs:     800,
		cost:          0.03,
		abstentionReason: "Escalated to human — insufficient confidence",
	})
	if err != nil {
		return nil, fmt.Errorf("seedHandoffRun: %w", err)
	}

	deniedRunID, err := seedRun(ctx, db, auth, runParams{
		entityType:       "case",
		entityID:         caseID,
		suffix:           suffix + "_denied",
		status:           "denied_by_policy",
		agentType:        "support",
		agentName:        "Support Agent",
		latencyMs:        300,
		cost:             0.01,
		rejectionReason:  "Policy: external email requires manager approval",
	})
	if err != nil {
		return nil, fmt.Errorf("seedDeniedRun: %w", err)
	}

	// W6-T3: usage events for governance screen
	if err := seedUsageEvents(ctx, db, auth, completedRunID); err != nil {
		return nil, fmt.Errorf("seedUsageEvents: %w", err)
	}

	// W6-T3: quota policy + state for governance screen
	if err := seedQuotaPolicy(ctx, db, auth); err != nil {
		return nil, fmt.Errorf("seedQuotaPolicy: %w", err)
	}

	// W6-T3: pending approval for inbox
	approvalID, err := seedApproval(ctx, db, auth, caseID, suffix)
	if err != nil {
		return nil, fmt.Errorf("seedApproval: %w", err)
	}

	out := &seedOutput{}
	out.Account.ID = accountID
	out.Contact.ID = contactID
	out.Contact.Email = contactEmail
	out.Deal.ID = dealID
	out.Case.ID = caseID
	out.Case.Subject = caseSubject
	out.AgentRuns.CompletedID = completedRunID
	out.AgentRuns.HandoffID = handoffRunID
	out.AgentRuns.DeniedByPolicyID = deniedRunID
	out.Inbox.ApprovalID = approvalID
	return out, nil
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
}

func seedRun(ctx context.Context, db *sql.DB, auth authResponse, p runParams) (string, error) {
	agentDefID := uuid.NewV7().String()
	runID := uuid.NewV7().String()
	now := time.Now().UTC()

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
		metricName string
		value      float64
	}{
		{"tokens", 512},
		{"cost_euros", 0.05},
	}
	for _, e := range events {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO usage_event (
				id, workspace_id, actor_id, actor_type, run_id,
				metric_name, value, recorded_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.NewV7().String(), auth.WorkspaceID,
			auth.UserID, "user", runID,
			e.metricName, e.value,
			now.Format(time.RFC3339), now.Format(time.RFC3339),
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
		INSERT INTO governance_policy (
			id, workspace_id, policy_type, metric_name, limit_value,
			reset_period, enforcement_mode, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		policyID, auth.WorkspaceID,
		"token_budget", "tokens", 100000,
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
			id, workspace_id, policy_id, current_value,
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

func seedApproval(ctx context.Context, db *sql.DB, auth authResponse, caseID, suffix string) (string, error) {
	now := time.Now().UTC()
	approvalID := uuid.NewV7().String()
	expiresAt := now.Add(24 * time.Hour)

	payload := fmt.Sprintf(`{"entity_type":"case","entity_id":%q,"action":"close_case"}`, caseID)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO approval_request (
			id, workspace_id, requested_by, approver_id,
			action, resource_type, resource_id, payload,
			reason, status, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		approvalID, auth.WorkspaceID,
		auth.UserID, auth.UserID,
		"close_case", "case", caseID, payload,
		"E2E wedge seed approval "+suffix,
		"pending",
		expiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}
	return approvalID, nil
}

// ─── Auth helpers ─────────────────────────────────────────────────────────────

func loginOrRegister(ctx context.Context, apiURL string) (authResponse, error) {
	auth, err := requestAuth(ctx, apiURL, "/auth/login", map[string]string{
		"email":    testEmail,
		"password": testPassword,
	})
	if err == nil {
		return auth, nil
	}

	reqErr := &requestError{}
	if !asRequestError(err, reqErr) || reqErr.Status != http.StatusUnauthorized {
		return authResponse{}, err
	}

	return requestAuth(ctx, apiURL, "/auth/register", map[string]string{
		"email":         testEmail,
		"password":      testPassword,
		"displayName":   testDisplayName,
		"workspaceName": testWorkspaceName,
	})
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
	if err := json.Unmarshal(raw, &auth); err != nil {
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
