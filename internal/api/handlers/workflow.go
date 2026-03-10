package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	tooldomain "github.com/matiasleandrokruk/fenix/internal/domain/tool"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type WorkflowService interface {
	Create(ctx context.Context, input workflowdomain.CreateWorkflowInput) (*workflowdomain.Workflow, error)
	Get(ctx context.Context, workspaceID, workflowID string) (*workflowdomain.Workflow, error)
	List(ctx context.Context, workspaceID string, input workflowdomain.ListWorkflowsInput) ([]*workflowdomain.Workflow, error)
	Update(ctx context.Context, workspaceID, workflowID string, input workflowdomain.UpdateWorkflowInput) (*workflowdomain.Workflow, error)
	DeleteDraft(ctx context.Context, workspaceID, workflowID string) error
}

type WorkflowHandler struct {
	service WorkflowService
	authz   ActionAuthorizer
	db      *sql.DB
	runtime *workflowRuntime
}

type workflowCacheInvalidator interface {
	InvalidateCache(workflowID string)
}

type workflowRuntime struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tooldomain.ToolRegistry
	policyEngine    *policy.PolicyEngine
	approvalService *policy.ApprovalService
	cacheInvalidator workflowCacheInvalidator
}

type CreateWorkflowRequest struct {
	AgentDefinitionID *string `json:"agent_definition_id,omitempty"`
	Name              string  `json:"name"`
	Description       string  `json:"description,omitempty"`
	DSLSource         string  `json:"dsl_source"`
	SpecSource        string  `json:"spec_source,omitempty"`
}

type UpdateWorkflowRequest struct {
	AgentDefinitionID *string `json:"agent_definition_id,omitempty"`
	Description       string  `json:"description,omitempty"`
	DSLSource         string  `json:"dsl_source"`
	SpecSource        string  `json:"spec_source,omitempty"`
}

type WorkflowResponse struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspace_id"`
	AgentDefinitionID *string `json:"agent_definition_id,omitempty"`
	ParentVersionID   *string `json:"parent_version_id,omitempty"`
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	DSLSource         string  `json:"dsl_source"`
	SpecSource        *string `json:"spec_source,omitempty"`
	Version           int     `json:"version"`
	Status            string  `json:"status"`
	CreatedByUserID   *string `json:"created_by_user_id,omitempty"`
	ArchivedAt        *string `json:"archived_at,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type ExecuteWorkflowRequest struct {
	TriggerContext json.RawMessage `json:"trigger_context,omitempty"`
	Inputs         json.RawMessage `json:"inputs,omitempty"`
}

func NewWorkflowHandler(service WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{service: service}
}

func NewWorkflowHandlerWithAuthorizer(service WorkflowService, authz ActionAuthorizer) *WorkflowHandler {
	return &WorkflowHandler{service: service, authz: authz}
}

func NewWorkflowHandlerWithRuntime(service WorkflowService, authz ActionAuthorizer, db *sql.DB, orchestrator *agent.Orchestrator, toolRegistry *tooldomain.ToolRegistry, policyEngine *policy.PolicyEngine, approvalService *policy.ApprovalService, cacheInvalidator workflowCacheInvalidator) *WorkflowHandler {
	return &WorkflowHandler{
		service: service,
		authz:   authz,
		db:      db,
		runtime: &workflowRuntime{
			orchestrator:    orchestrator,
			toolRegistry:    toolRegistry,
			policyEngine:    policyEngine,
			approvalService: approvalService,
			cacheInvalidator: cacheInvalidator,
		},
	}
}

func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "workflows.create") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	var req CreateWorkflowRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}

	userID, _ := r.Context().Value(ctxkeys.UserID).(string)
	var createdBy *string
	if userID != "" {
		createdBy = &userID
	}

	out, err := h.service.Create(r.Context(), workflowdomain.CreateWorkflowInput{
		WorkspaceID:       workspaceID,
		AgentDefinitionID: req.AgentDefinitionID,
		Name:              req.Name,
		Description:       req.Description,
		DSLSource:         req.DSLSource,
		SpecSource:        req.SpecSource,
		CreatedByUserID:   createdBy,
	})
	if err != nil {
		writeWorkflowError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = writeJSONOr500(w, map[string]any{"data": workflowToResponse(out)})
}

func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "workflows.get") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errWorkflowIDRequired)
		return
	}

	out, err := h.service.Get(r.Context(), workspaceID, id)
	if err != nil {
		writeWorkflowError(w, err)
		return
	}

	_ = writeJSONOr500(w, map[string]any{"data": workflowToResponse(out)})
}

func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "workflows.list") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	input, err := decodeWorkflowListInput(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	out, err := h.service.List(r.Context(), workspaceID, input)
	if err != nil {
		writeWorkflowError(w, err)
		return
	}

	response := make([]*WorkflowResponse, 0, len(out))
	for _, workflow := range out {
		response = append(response, workflowToResponse(workflow))
	}
	_ = writeJSONOr500(w, map[string]any{"data": response})
}

func (h *WorkflowHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "workflows.update") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errWorkflowIDRequired)
		return
	}

	var req UpdateWorkflowRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}

	out, err := h.service.Update(r.Context(), workspaceID, id, workflowdomain.UpdateWorkflowInput{
		AgentDefinitionID: req.AgentDefinitionID,
		Description:       req.Description,
		DSLSource:         req.DSLSource,
		SpecSource:        req.SpecSource,
	})
	if err != nil {
		writeWorkflowError(w, err)
		return
	}
	if h.runtime != nil && h.runtime.cacheInvalidator != nil {
		h.runtime.cacheInvalidator.InvalidateCache(id)
	}

	_ = writeJSONOr500(w, map[string]any{"data": workflowToResponse(out)})
}

func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "workflows.delete") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errWorkflowIDRequired)
		return
	}

	if err := h.service.DeleteDraft(r.Context(), workspaceID, id); err != nil {
		writeWorkflowError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkflowHandler) Execute(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "workflows.execute") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	if h.runtime == nil || h.runtime.orchestrator == nil || h.db == nil {
		writeError(w, http.StatusInternalServerError, "workflow execute runtime is not configured")
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errWorkflowIDRequired)
		return
	}

	var req ExecuteWorkflowRequest
	if !decodeOptionalWorkflowExecuteBody(w, r, &req) {
		return
	}

	item, err := h.service.Get(r.Context(), workspaceID, id)
	if err != nil {
		writeWorkflowError(w, err)
		return
	}
	if item.AgentDefinitionID == nil || *item.AgentDefinitionID == "" {
		writeError(w, http.StatusConflict, "workflow must be linked to an agent definition")
		return
	}
	if item.Status != workflowdomain.StatusActive {
		writeError(w, http.StatusConflict, "workflow must be active to execute")
		return
	}

	runner := agent.NewDSLRunnerWithDependencies(staticWorkflowResolver{workflow: item}, agent.NewDSLRuntime(), nil)
	userID, _ := r.Context().Value(ctxkeys.UserID).(string)
	var triggeredBy *string
	if userID != "" {
		triggeredBy = &userID
	}

	run, err := runner.Run(r.Context(), &agent.RunContext{
		Orchestrator:    h.runtime.orchestrator,
		ToolRegistry:    h.runtime.toolRegistry,
		PolicyEngine:    h.runtime.policyEngine,
		ApprovalService: h.runtime.approvalService,
		DB:              h.db,
	}, agent.TriggerAgentInput{
		AgentID:        *item.AgentDefinitionID,
		WorkspaceID:    workspaceID,
		TriggeredBy:    triggeredBy,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: normalizeOptionalJSONObject(req.TriggerContext),
		Inputs:         normalizeOptionalJSONObject(req.Inputs),
	})
	if err != nil {
		writeWorkflowExecuteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = writeJSONOr500(w, map[string]any{
		"data": map[string]any{
			"workflow_id": id,
			"run":         agentRunToResponse(run),
		},
	})
}

func decodeWorkflowListInput(r *http.Request) (workflowdomain.ListWorkflowsInput, error) {
	var input workflowdomain.ListWorkflowsInput
	if name := r.URL.Query().Get("name"); name != "" {
		input.Name = name
	}
	if status := r.URL.Query().Get(queryStatus); status != "" {
		parsed := workflowdomain.Status(status)
		switch parsed {
		case workflowdomain.StatusDraft, workflowdomain.StatusTesting, workflowdomain.StatusActive, workflowdomain.StatusArchived:
			input.Status = &parsed
		default:
			return input, errors.New("invalid workflow status")
		}
	}
	return input, nil
}

func writeWorkflowError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workflowdomain.ErrWorkflowNotFound):
		writeError(w, http.StatusNotFound, "workflow not found")
	case errors.Is(err, workflowdomain.ErrInvalidWorkflowInput):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, workflowdomain.ErrWorkflowNameConflict),
		errors.Is(err, workflowdomain.ErrWorkflowNotEditable),
		errors.Is(err, workflowdomain.ErrInvalidStatusTransition),
		errors.Is(err, workflowdomain.ErrWorkflowVersionInvalid),
		errors.Is(err, workflowdomain.ErrWorkflowDeleteInvalid),
		errors.Is(err, workflowdomain.ErrWorkflowActiveConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeWorkflowExecuteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workflowdomain.ErrWorkflowNotFound), errors.Is(err, agent.ErrDSLWorkflowNotFound):
		writeError(w, http.StatusNotFound, "workflow not found")
	case errors.Is(err, workflowdomain.ErrInvalidWorkflowInput),
		errors.Is(err, agent.ErrInvalidTriggerType):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeOptionalWorkflowExecuteBody(w http.ResponseWriter, r *http.Request, dst *ExecuteWorkflowRequest) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return true
	}
	return decodeBodyJSON(w, r, dst)
}

func normalizeOptionalJSONObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

type staticWorkflowResolver struct {
	workflow *workflowdomain.Workflow
}

func (s staticWorkflowResolver) GetActiveByAgentDefinition(_ context.Context, workspaceID, agentDefinitionID string) (*workflowdomain.Workflow, error) {
	if s.workflow == nil || s.workflow.WorkspaceID != workspaceID || s.workflow.Status != workflowdomain.StatusActive || s.workflow.AgentDefinitionID == nil || *s.workflow.AgentDefinitionID != agentDefinitionID {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	return s.workflow, nil
}

func workflowToResponse(in *workflowdomain.Workflow) *WorkflowResponse {
	if in == nil {
		return nil
	}

	return &WorkflowResponse{
		ID:                in.ID,
		WorkspaceID:       in.WorkspaceID,
		AgentDefinitionID: in.AgentDefinitionID,
		ParentVersionID:   in.ParentVersionID,
		Name:              in.Name,
		Description:       in.Description,
		DSLSource:         in.DSLSource,
		SpecSource:        in.SpecSource,
		Version:           in.Version,
		Status:            string(in.Status),
		CreatedByUserID:   in.CreatedByUserID,
		ArchivedAt:        formatOptionalWorkflowTime(in.ArchivedAt),
		CreatedAt:         in.CreatedAt.Format(timeFormatISO),
		UpdatedAt:         in.UpdatedAt.Format(timeFormatISO),
	}
}

func formatOptionalWorkflowTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(timeFormatISO)
	return &formatted
}
