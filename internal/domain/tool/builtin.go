package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

const (
	BuiltinCreateTask          = "create_task"
	BuiltinUpdateCase          = "update_case"
	BuiltinSendReply           = "send_reply"
	BuiltinGetLead             = "get_lead"
	BuiltinGetAccount          = "get_account"
	BuiltinCreateKnowledgeItem = "create_knowledge_item"
	BuiltinUpdateKnowledgeItem = "update_knowledge_item"
	BuiltinQueryMetrics        = "query_metrics"
)

type BuiltinServices struct {
	DB      *sql.DB
	Case    *crm.CaseService
	Lead    *crm.LeadService
	Account *crm.AccountService
	Ingest  knowledgeIngestor
}

type knowledgeIngestor interface {
	Ingest(ctx context.Context, input knowledge.CreateKnowledgeItemInput) (*knowledge.KnowledgeItem, error)
}

type builtinDefinition struct {
	Name                string
	Description         string
	InputSchema         json.RawMessage
	RequiredPermissions []string
}

func builtinDefinitions() []builtinDefinition {
	return []builtinDefinition{
		{
			Name:                BuiltinCreateTask,
			Description:         "Create a CRM task activity linked to an entity",
			InputSchema:         json.RawMessage(`{"type":"object","required":["owner_id","title","entity_type","entity_id"],"properties":{"owner_id":{"type":"string"},"title":{"type":"string"},"due_date":{"type":"string"},"entity_type":{"type":"string"},"entity_id":{"type":"string"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:create_task"},
		},
		{
			Name:                BuiltinUpdateCase,
			Description:         "Update case status/priority and emit record.updated",
			InputSchema:         json.RawMessage(`{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"},"status":{"type":"string"},"priority":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:update_case"},
		},
		{
			Name:                BuiltinSendReply,
			Description:         "Create a case reply note",
			InputSchema:         json.RawMessage(`{"type":"object","required":["case_id","body"],"properties":{"case_id":{"type":"string"},"body":{"type":"string"},"is_internal":{"type":"boolean"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:send_reply"},
		},
		{
			Name:                BuiltinGetLead,
			Description:         "Fetch a lead by id in current workspace",
			InputSchema:         json.RawMessage(`{"type":"object","required":["lead_id"],"properties":{"lead_id":{"type":"string"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:get_lead"},
		},
		{
			Name:                BuiltinGetAccount,
			Description:         "Fetch an account by id in current workspace",
			InputSchema:         json.RawMessage(`{"type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:get_account"},
		},
		{
			Name:                BuiltinCreateKnowledgeItem,
			Description:         "Create knowledge item from title/content/source",
			InputSchema:         json.RawMessage(`{"type":"object","required":["title","content","source_type","workspace_id"],"properties":{"title":{"type":"string"},"content":{"type":"string"},"source_type":{"type":"string"},"workspace_id":{"type":"string"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:create_knowledge_item"},
		},
		{
			Name:                BuiltinUpdateKnowledgeItem,
			Description:         "Update an existing knowledge item title/content",
			InputSchema:         json.RawMessage(`{"type":"object","required":["id"],"properties":{"id":{"type":"string"},"title":{"type":"string"},"content":{"type":"string"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:update_knowledge_item"},
		},
		{
			Name:                BuiltinQueryMetrics,
			Description:         "Query aggregated CRM metrics",
			InputSchema:         json.RawMessage(`{"type":"object","required":["metric","workspace_id"],"properties":{"metric":{"type":"string","enum":["sales_funnel","deal_aging","case_volume","case_backlog","mttr"]},"workspace_id":{"type":"string"},"from":{"type":"string"},"to":{"type":"string"}},"additionalProperties":false}`),
			RequiredPermissions: []string{"tools:query_metrics"},
		},
	}
}

func (r *ToolRegistry) EnsureBuiltInToolDefinitions(ctx context.Context, workspaceID string) error {
	for _, def := range builtinDefinitions() {
		if _, err := r.getToolDefinitionByName(ctx, workspaceID, def.Name); err == nil {
			continue
		} else if err != ErrToolDefinitionNotFound {
			return err
		}

		description := def.Description
		if _, err := r.CreateToolDefinition(ctx, CreateToolDefinitionInput{
			WorkspaceID:         workspaceID,
			Name:                def.Name,
			Description:         &description,
			InputSchema:         def.InputSchema,
			RequiredPermissions: def.RequiredPermissions,
		}); err != nil {
			if !isUniqueConstraintError(err) {
				return err
			}
		}
	}

	return nil
}

func (r *ToolRegistry) EnsureBuiltInToolDefinitionsForAllWorkspaces(ctx context.Context) error {
	workspaceIDs, err := r.listWorkspaceIDs(ctx)
	if err != nil {
		return err
	}
	return r.ensureBuiltInsForWorkspaces(ctx, workspaceIDs)
}

func (r *ToolRegistry) listWorkspaceIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM workspace`)
	if err != nil {
		return nil, err
	}

	workspaceIDs := make([]string, 0, 8)

	for rows.Next() {
		var workspaceID string
		if scanErr := rows.Scan(&workspaceID); scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		workspaceIDs = append(workspaceIDs, workspaceID)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		_ = rows.Close()
		return nil, rowsErr
	}
	if closeErr := rows.Close(); closeErr != nil {
		return nil, closeErr
	}
	return workspaceIDs, nil
}

func (r *ToolRegistry) ensureBuiltInsForWorkspaces(ctx context.Context, workspaceIDs []string) error {
	for _, workspaceID := range workspaceIDs {
		if err := r.EnsureBuiltInToolDefinitions(ctx, workspaceID); err != nil {
			return err
		}
	}
	return nil
}

func RegisterBuiltInExecutors(registry *ToolRegistry, services BuiltinServices) error {
	registrations := []struct {
		name     string
		executor ToolExecutor
	}{
		{name: BuiltinCreateTask, executor: NewCreateTaskExecutor(services.DB)},
		{name: BuiltinUpdateCase, executor: NewUpdateCaseExecutor(services.Case)},
		{name: BuiltinSendReply, executor: NewSendReplyExecutor(services.DB, services.Case)},
		{name: BuiltinGetLead, executor: NewGetLeadExecutor(services.Lead)},
		{name: BuiltinGetAccount, executor: NewGetAccountExecutor(services.Account)},
		{name: BuiltinCreateKnowledgeItem, executor: NewCreateKnowledgeItemExecutor(services.Ingest)},
		{name: BuiltinUpdateKnowledgeItem, executor: NewUpdateKnowledgeItemExecutor(services.DB)},
		{name: BuiltinQueryMetrics, executor: NewQueryMetricsExecutor(services.DB)},
	}

	for _, registration := range registrations {
		if err := registerBuiltinExecutor(registry, registration.name, registration.executor); err != nil {
			return err
		}
	}

	return nil
}

func registerBuiltinExecutor(registry *ToolRegistry, name string, executor ToolExecutor) error {
	if err := registry.Register(name, executor); err != nil && err != ErrToolExecutorAlreadyRegistered {
		return err
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil || err == sql.ErrNoRows {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
