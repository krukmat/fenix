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
	"github.com/matiasleandrokruk/fenix/internal/domain/workflow"
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

type seedOutput struct {
	Credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	} `json:"credentials"`
	Account struct {
		ID string `json:"id"`
	} `json:"account"`
	Deal struct {
		ID string `json:"id"`
	} `json:"deal"`
	Case struct {
		ID string `json:"id"`
	} `json:"case"`
	Workflows struct {
		ActiveID   string `json:"activeId"`
		ArchivedID string `json:"archivedId"`
	} `json:"workflows"`
	AgentRuns struct {
		RejectedID     string `json:"rejectedId"`
		DealRejectedID string `json:"dealRejectedId"`
		CaseRejectedID string `json:"caseRejectedId"`
	} `json:"agentRuns"`
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

	encodeErr := json.NewEncoder(os.Stdout).Encode(seeded)
	if encodeErr != nil {
		fail(encodeErr)
	}
}

func seedFixtures(ctx context.Context, db *sql.DB, auth authResponse) (*seedOutput, error) {
	suffix := time.Now().UTC().Format("20060102T150405")
	accountID, err := seedAccount(ctx, db, auth, suffix)
	if err != nil {
		return nil, err
	}

	dealID, err := seedDeal(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, err
	}

	caseID, err := seedCase(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, err
	}

	activeWorkflowID, err := seedActiveWorkflow(ctx, db, auth, suffix)
	if err != nil {
		return nil, err
	}

	archivedWorkflowID, err := seedArchivedWorkflow(ctx, db, auth, suffix)
	if err != nil {
		return nil, err
	}

	rejectedRunID, err := seedRejectedRun(ctx, db, auth, accountID, suffix)
	if err != nil {
		return nil, err
	}

	dealRejectedRunID, err := seedEntityRejectedRun(ctx, db, auth, "deal", dealID, suffix+"_deal")
	if err != nil {
		return nil, err
	}

	caseRejectedRunID, err := seedEntityRejectedRun(ctx, db, auth, "case", caseID, suffix+"_case")
	if err != nil {
		return nil, err
	}

	out := &seedOutput{}
	out.Account.ID = accountID
	out.Deal.ID = dealID
	out.Case.ID = caseID
	out.Workflows.ActiveID = activeWorkflowID
	out.Workflows.ArchivedID = archivedWorkflowID
	out.AgentRuns.RejectedID = rejectedRunID
	out.AgentRuns.DealRejectedID = dealRejectedRunID
	out.AgentRuns.CaseRejectedID = caseRejectedRunID
	return out, nil
}

func seedAccount(ctx context.Context, db *sql.DB, auth authResponse, suffix string) (string, error) {
	accountSvc := crm.NewAccountService(db)
	account, err := accountSvc.Create(ctx, crm.CreateAccountInput{
		WorkspaceID: auth.WorkspaceID,
		Name:        "E2E P2 Account " + suffix,
		Industry:    "Technology",
		OwnerID:     auth.UserID,
	})
	if err != nil {
		return "", err
	}
	return account.ID, nil
}

func seedActiveWorkflow(ctx context.Context, db *sql.DB, auth authResponse, suffix string) (string, error) {
	svc := workflow.NewService(db)
	createdBy := auth.UserID
	wf, err := svc.Create(ctx, workflow.CreateWorkflowInput{
		WorkspaceID:     auth.WorkspaceID,
		Name:            "e2e_p2_active_" + suffix,
		Description:     "Seeded active workflow for Detox smoke",
		DSLSource:       fmt.Sprintf("WORKFLOW e2e_p2_active_%s\nON case.created\nSET case.status = \"open\"", suffix),
		CreatedByUserID: &createdBy,
	})
	if err != nil {
		return "", err
	}
	_, err = svc.MarkTesting(ctx, auth.WorkspaceID, wf.ID)
	if err != nil {
		return "", err
	}
	_, err = svc.Activate(ctx, auth.WorkspaceID, wf.ID)
	if err != nil {
		return "", err
	}
	return wf.ID, nil
}

func seedArchivedWorkflow(ctx context.Context, db *sql.DB, auth authResponse, suffix string) (string, error) {
	svc := workflow.NewService(db)
	createdBy := auth.UserID
	wf, err := svc.Create(ctx, workflow.CreateWorkflowInput{
		WorkspaceID:     auth.WorkspaceID,
		Name:            "e2e_p2_archived_" + suffix,
		Description:     "Seeded archived workflow for Detox rollback smoke",
		DSLSource:       fmt.Sprintf("WORKFLOW e2e_p2_archived_%s\nON case.created\nSET case.status = \"resolved\"", suffix),
		CreatedByUserID: &createdBy,
	})
	if err != nil {
		return "", err
	}
	_, err = svc.MarkTesting(ctx, auth.WorkspaceID, wf.ID)
	if err != nil {
		return "", err
	}
	_, err = svc.Activate(ctx, auth.WorkspaceID, wf.ID)
	if err != nil {
		return "", err
	}
	_, err = svc.MarkArchived(ctx, auth.WorkspaceID, wf.ID)
	if err != nil {
		return "", err
	}
	return wf.ID, nil
}

func seedRejectedRun(ctx context.Context, db *sql.DB, auth authResponse, accountID, suffix string) (string, error) {
	agentDefinitionID := uuid.NewV7().String()
	runID := uuid.NewV7().String()
	now := time.Now().UTC()
	agentName := "e2e_p2_agent_" + suffix

	if _, err := db.ExecContext(ctx, `
		INSERT INTO agent_definition (
			id, workspace_id, name, description, agent_type, objective,
			allowed_tools, limits, trigger_config, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, '[]', '{}', '{}', 'active', ?, ?)
	`,
		agentDefinitionID,
		auth.WorkspaceID,
		agentName,
		"Seeded agent definition for Mobile P2 Detox smoke",
		"support",
		`{"goal":"validate mobile p2 smoke"}`,
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	triggerContext := fmt.Sprintf(`{"entity_type":"account","entity_id":"%s","account":{"id":"%s"}}`, accountID, accountID)
	output := fmt.Sprintf(`{"workflow_id":"wf-seeded-rejected","entity_type":"account","entity_id":"%s","rejection_reason":"Policy threshold not met","reason":"Policy threshold not met"}`, accountID)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, triggered_by_user_id,
			trigger_type, trigger_context, status, inputs,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace,
			tool_calls, output, abstention_reason,
			total_tokens, total_cost, latency_ms, trace_id,
			started_at, completed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		runID,
		auth.WorkspaceID,
		agentDefinitionID,
		auth.UserID,
		"manual",
		triggerContext,
		"rejected",
		`{"source":"mobile-e2e"}`,
		emptyJSONArray,
		emptyJSONArray,
		emptyJSONArray,
		emptyJSONArray,
		output,
		"Policy threshold not met",
		128,
		0.01,
		420,
		uuid.NewV7().String(),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	return runID, nil
}

func seedDeal(ctx context.Context, db *sql.DB, auth authResponse, accountID, suffix string) (string, error) {
	svc := crm.NewDealService(db)
	title := "E2E P2 Deal " + suffix
	deal, err := svc.Create(ctx, crm.CreateDealInput{
		WorkspaceID: auth.WorkspaceID,
		AccountID:   accountID,
		OwnerID:     auth.UserID,
		Title:       title,
		Status:      "open",
	})
	if err != nil {
		return "", err
	}
	return deal.ID, nil
}

func seedCase(ctx context.Context, db *sql.DB, auth authResponse, accountID, suffix string) (string, error) {
	svc := crm.NewCaseService(db)
	subject := "E2E P2 Case " + suffix
	ct, err := svc.Create(ctx, crm.CreateCaseInput{
		WorkspaceID: auth.WorkspaceID,
		AccountID:   accountID,
		OwnerID:     auth.UserID,
		Subject:     subject,
		Priority:    "medium",
		Status:      "open",
	})
	if err != nil {
		return "", err
	}
	return ct.ID, nil
}

func seedEntityRejectedRun(ctx context.Context, db *sql.DB, auth authResponse, entityType, entityID, suffix string) (string, error) {
	agentDefinitionID := uuid.NewV7().String()
	runID := uuid.NewV7().String()
	now := time.Now().UTC()
	agentName := "e2e_p2_agent_" + suffix

	if _, err := db.ExecContext(ctx, `
		INSERT INTO agent_definition (
			id, workspace_id, name, description, agent_type, objective,
			allowed_tools, limits, trigger_config, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, '[]', '{}', '{}', 'active', ?, ?)
	`,
		agentDefinitionID,
		auth.WorkspaceID,
		agentName,
		"Seeded agent definition for Mobile P2 Detox smoke ("+entityType+")",
		"support",
		`{"goal":"validate mobile p2 smoke `+entityType+`"}`,
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	triggerContext := fmt.Sprintf(`{"entity_type":%q,"entity_id":%q}`, entityType, entityID)
	output := fmt.Sprintf(`{"entity_type":%q,"entity_id":%q,"rejection_reason":"Policy threshold not met","reason":"Policy threshold not met"}`, entityType, entityID)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, triggered_by_user_id,
			trigger_type, trigger_context, status, inputs,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace,
			tool_calls, output, abstention_reason,
			total_tokens, total_cost, latency_ms, trace_id,
			started_at, completed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		runID,
		auth.WorkspaceID,
		agentDefinitionID,
		auth.UserID,
		"manual",
		triggerContext,
		"rejected",
		`{"source":"mobile-e2e"}`,
		emptyJSONArray,
		emptyJSONArray,
		emptyJSONArray,
		emptyJSONArray,
		output,
		"Policy threshold not met",
		128,
		0.01,
		420,
		uuid.NewV7().String(),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	); err != nil {
		return "", err
	}

	return runID, nil
}

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
	unmarshalErr := json.Unmarshal(raw, &auth)
	if unmarshalErr != nil {
		return authResponse{}, unmarshalErr
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
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}
