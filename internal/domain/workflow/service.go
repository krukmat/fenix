package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	maxSourceSizeBytes = 64 * 1024
)

var (
	ErrInvalidWorkflowInput    = errors.New("invalid workflow input")
	ErrWorkflowNotEditable     = errors.New("workflow is not editable")
	ErrWorkflowNameConflict    = errors.New("workflow name/version conflict")
	ErrWorkflowActiveConflict  = errors.New("workflow active version conflict")
	ErrInvalidStatusTransition = errors.New("invalid workflow status transition")
	ErrWorkflowVersionInvalid  = errors.New("invalid workflow version operation")
	ErrWorkflowDeleteInvalid   = errors.New("invalid workflow delete operation")
)

type CreateWorkflowInput struct {
	WorkspaceID       string
	AgentDefinitionID *string
	Name              string
	Description       string
	DSLSource         string
	SpecSource        string
	CreatedByUserID   *string
}

type UpdateWorkflowInput struct {
	AgentDefinitionID *string
	Description       string
	DSLSource         string
	SpecSource        string
}

type ListWorkflowsInput struct {
	Status *Status
	Name   string
}

type Service struct {
	repo                      *Repository
	scheduler                 workflowScheduler
	cartaBudgetLimitsResolver func(string) (map[string]any, error)
	cartaInvariantRulesResolver func(string) ([]map[string]any, error)
}

type workflowScheduler interface {
	CancelBySource(ctx context.Context, workspaceID, sourceID string) (int64, error)
}

func defaultBudgetLimitsResolver(string) (map[string]any, error)   { return nil, nil }
func defaultInvariantRulesResolver(string) ([]map[string]any, error) { return nil, nil }

func RegisterCartaBudgetLimitsResolver(resolver func(string) (map[string]any, error)) {
	globalCartaBudgetLimitsResolver = resolver
}

func RegisterCartaInvariantRulesResolver(resolver func(string) ([]map[string]any, error)) {
	globalCartaInvariantRulesResolver = resolver
}

// globalCartaBudgetLimitsResolver and globalCartaInvariantRulesResolver are used by
// NewService/NewServiceWithRepository to set per-instance resolvers at startup.
// Tests should use NewServiceWithResolvers to avoid shared state between parallel tests.
var globalCartaBudgetLimitsResolver func(string) (map[string]any, error)
var globalCartaInvariantRulesResolver func(string) ([]map[string]any, error)

func newService(repo *Repository, scheduler workflowScheduler, budgetResolver func(string) (map[string]any, error), invariantResolver func(string) ([]map[string]any, error)) *Service {
	if budgetResolver == nil {
		budgetResolver = defaultBudgetLimitsResolver
	}
	if invariantResolver == nil {
		invariantResolver = defaultInvariantRulesResolver
	}
	return &Service{
		repo:                        repo,
		scheduler:                   scheduler,
		cartaBudgetLimitsResolver:   budgetResolver,
		cartaInvariantRulesResolver: invariantResolver,
	}
}

func NewService(db *sql.DB) *Service {
	return newService(NewRepository(db), nil, globalCartaBudgetLimitsResolver, globalCartaInvariantRulesResolver)
}

func NewServiceWithRepository(repo *Repository) *Service {
	return newService(repo, nil, globalCartaBudgetLimitsResolver, globalCartaInvariantRulesResolver)
}

func NewServiceWithDependencies(repo *Repository, scheduler workflowScheduler) *Service {
	return newService(repo, scheduler, globalCartaBudgetLimitsResolver, globalCartaInvariantRulesResolver)
}

// NewServiceWithResolvers creates a Service with explicit resolvers — use in tests to avoid
// shared global state between parallel tests.
func NewServiceWithResolvers(repo *Repository, budgetResolver func(string) (map[string]any, error), invariantResolver func(string) ([]map[string]any, error)) *Service {
	return newService(repo, nil, budgetResolver, invariantResolver)
}

func (s *Service) Create(ctx context.Context, input CreateWorkflowInput) (*Workflow, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	desc := trimOptionalString(input.Description)
	spec := trimOptionalString(input.SpecSource)

	workflow, err := s.repo.Create(ctx, CreateInput{
		ID:                uuid.NewV7().String(),
		WorkspaceID:       input.WorkspaceID,
		AgentDefinitionID: input.AgentDefinitionID,
		Name:              strings.TrimSpace(input.Name),
		Description:       desc,
		DSLSource:         input.DSLSource,
		SpecSource:        spec,
		Version:           1,
		Status:            StatusDraft,
		CreatedByUserID:   input.CreatedByUserID,
	})
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrWorkflowNameConflict
		}
		return nil, err
	}
	return workflow, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.repo.GetByID(ctx, workspaceID, workflowID)
}

func (s *Service) GetActiveByAgentDefinition(ctx context.Context, workspaceID, agentDefinitionID string) (*Workflow, error) {
	return s.repo.GetActiveByAgentDefinition(ctx, workspaceID, agentDefinitionID)
}

func (s *Service) List(ctx context.Context, workspaceID string, input ListWorkflowsInput) ([]*Workflow, error) {
	workflows, err := s.listBase(ctx, workspaceID, input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return workflows, nil
	}

	name := strings.TrimSpace(input.Name)
	out := make([]*Workflow, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow.Name == name {
			out = append(out, workflow)
		}
	}
	return out, nil
}

func (s *Service) ListVersions(ctx context.Context, workspaceID, workflowID string) ([]*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListVersionsByName(ctx, workspaceID, existing.Name)
}

func (s *Service) Update(ctx context.Context, workspaceID, workflowID string, input UpdateWorkflowInput) (*Workflow, error) {
	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusDraft {
		return nil, ErrWorkflowNotEditable
	}

	desc := trimOptionalString(input.Description)
	spec := trimOptionalString(input.SpecSource)

	updated, err := s.repo.Update(ctx, workspaceID, workflowID, UpdateInput{
		AgentDefinitionID: input.AgentDefinitionID,
		Description:       desc,
		DSLSource:         input.DSLSource,
		SpecSource:        spec,
		Status:            existing.Status,
		ArchivedAt:        existing.ArchivedAt,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) SetStatus(ctx context.Context, workspaceID, workflowID string, next Status) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}

	if validateErr := s.validateStatusChange(ctx, existing, next); validateErr != nil {
		return nil, validateErr
	}

	updated, err := s.persistStatusChange(ctx, existing, next)
	if err != nil {
		return nil, err
	}
	if postErr := s.afterStatusChange(ctx, updated, next); postErr != nil {
		return nil, postErr
	}
	return updated, nil
}

func (s *Service) validateStatusChange(ctx context.Context, existing *Workflow, next Status) error {
	if !isValidStatusTransition(existing.Status, next) {
		return ErrInvalidStatusTransition
	}
	if next != StatusActive {
		return nil
	}
	return s.ensureNoOtherActiveWorkflow(ctx, existing)
}

func (s *Service) persistStatusChange(ctx context.Context, workflow *Workflow, next Status) (*Workflow, error) {
	return s.repo.Update(ctx, workflow.WorkspaceID, workflow.ID, UpdateInput{
		AgentDefinitionID: workflow.AgentDefinitionID,
		Description:       workflow.Description,
		DSLSource:         workflow.DSLSource,
		SpecSource:        workflow.SpecSource,
		Status:            next,
		ArchivedAt:        archivedAtForStatus(next, workflow.ArchivedAt),
	})
}

func (s *Service) afterStatusChange(ctx context.Context, workflow *Workflow, next Status) error {
	if next != StatusArchived {
		return nil
	}
	return s.cancelScheduledJobsForWorkflow(ctx, workflow)
}

func (s *Service) MarkTesting(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.SetStatus(ctx, workspaceID, workflowID, StatusTesting)
}

func (s *Service) MarkActive(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.SetStatus(ctx, workspaceID, workflowID, StatusActive)
}

func (s *Service) Activate(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	existing, err := s.loadWorkflowForActivation(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if err = s.syncCartaBudgetLimits(ctx, existing); err != nil {
		return nil, err
	}
	if err = s.syncCartaInvariantRules(ctx, existing); err != nil {
		return nil, err
	}
	archiveErr := s.archivePreviousActiveWorkflow(ctx, existing)
	if archiveErr != nil {
		return nil, archiveErr
	}
	return s.promoteWorkflowToActive(ctx, existing)
}

func (s *Service) MarkArchived(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.SetStatus(ctx, workspaceID, workflowID, StatusArchived)
}

func (s *Service) loadWorkflowForActivation(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusTesting {
		return nil, ErrInvalidStatusTransition
	}
	return existing, nil
}

func (s *Service) archivePreviousActiveWorkflow(ctx context.Context, workflow *Workflow) error {
	active, err := s.repo.GetActiveByName(ctx, workflow.WorkspaceID, workflow.Name)
	if errors.Is(err, ErrWorkflowNotFound) || active == nil || active.ID == workflow.ID {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.repo.Update(ctx, active.WorkspaceID, active.ID, UpdateInput{
		AgentDefinitionID: active.AgentDefinitionID,
		Description:       active.Description,
		DSLSource:         active.DSLSource,
		SpecSource:        active.SpecSource,
		Status:            StatusArchived,
		ArchivedAt:        archivedAtForStatus(StatusArchived, active.ArchivedAt),
	})
	return err
}

func (s *Service) syncCartaBudgetLimits(ctx context.Context, workflow *Workflow) error {
	if workflow == nil || workflow.AgentDefinitionID == nil || workflow.SpecSource == nil || !isCartaSource(*workflow.SpecSource) {
		return nil
	}
	merged, err := s.resolveMergedBudgetLimits(ctx, workflow.WorkspaceID, *workflow.AgentDefinitionID, *workflow.SpecSource)
	if err != nil || merged == nil {
		return err
	}
	return s.updateAgentDefinitionLimits(ctx, workflow.WorkspaceID, *workflow.AgentDefinitionID, merged)
}

func (s *Service) resolveMergedBudgetLimits(ctx context.Context, workspaceID, agentDefinitionID, specSource string) (map[string]any, error) {
	limits, err := s.cartaBudgetLimitsResolver(specSource)
	if err != nil {
		return nil, fmt.Errorf("resolve carta budget limits: %w", err)
	}
	if limits == nil {
		return nil, nil
	}
	current, err := s.loadAgentDefinitionLimits(ctx, workspaceID, agentDefinitionID)
	if err != nil {
		return nil, err
	}
	return mergeAgentDefinitionLimits(current, limits), nil
}

func (s *Service) syncCartaInvariantRules(ctx context.Context, workflow *Workflow) error {
	if workflow == nil || workflow.AgentDefinitionID == nil || workflow.SpecSource == nil || !isCartaSource(*workflow.SpecSource) {
		return nil
	}

	policySetID, ok, err := s.loadAgentDefinitionPolicySetID(ctx, workflow.WorkspaceID, *workflow.AgentDefinitionID)
	if err != nil || !ok {
		return err
	}

	return s.loadAndMergeInvariantRules(ctx, workflow.WorkspaceID, policySetID, *workflow.SpecSource)
}

func (s *Service) loadAndMergeInvariantRules(ctx context.Context, workspaceID, policySetID, specSource string) error {
	rules, err := s.cartaInvariantRulesResolver(specSource)
	if err != nil {
		return fmt.Errorf("resolve carta invariant rules: %w", err)
	}
	if len(rules) == 0 {
		return nil
	}

	current, versionID, err := s.loadActivePolicyRules(ctx, workspaceID, policySetID)
	if err != nil {
		return err
	}
	merged := mergePolicyRulesByAction(current, rules)
	return s.updatePolicyVersionRules(ctx, versionID, merged)
}

func (s *Service) loadAgentDefinitionLimits(ctx context.Context, workspaceID, agentDefinitionID string) (map[string]any, error) {
	var raw sql.NullString
	err := s.repo.db.QueryRowContext(ctx, `
		SELECT limits
		FROM agent_definition
		WHERE id = ? AND workspace_id = ?
	`, agentDefinitionID, workspaceID).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("load agent definition limits: %w", err)
	}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return map[string]any{}, nil
	}

	limits := make(map[string]any)
	if err = json.Unmarshal([]byte(raw.String), &limits); err != nil {
		return nil, fmt.Errorf("decode agent definition limits: %w", err)
	}
	return limits, nil
}

func (s *Service) updateAgentDefinitionLimits(ctx context.Context, workspaceID, agentDefinitionID string, limits map[string]any) error {
	payload, err := json.Marshal(limits)
	if err != nil {
		return fmt.Errorf("encode agent definition limits: %w", err)
	}
	_, err = s.repo.db.ExecContext(ctx, `
		UPDATE agent_definition
		SET limits = ?, updated_at = datetime('now')
		WHERE id = ? AND workspace_id = ?
	`, string(payload), agentDefinitionID, workspaceID)
	if err != nil {
		return fmt.Errorf("sync agent definition limits: %w", err)
	}
	return nil
}

func (s *Service) loadAgentDefinitionPolicySetID(ctx context.Context, workspaceID, agentDefinitionID string) (string, bool, error) {
	var raw sql.NullString
	err := s.repo.db.QueryRowContext(ctx, `
		SELECT policy_set_id
		FROM agent_definition
		WHERE id = ? AND workspace_id = ?
	`, agentDefinitionID, workspaceID).Scan(&raw)
	if err != nil {
		return "", false, fmt.Errorf("load agent definition policy set: %w", err)
	}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return "", false, nil
	}
	return raw.String, true, nil
}

func (s *Service) loadActivePolicyRules(ctx context.Context, workspaceID, policySetID string) ([]map[string]any, string, error) {
	var versionID string
	var raw string
	err := s.repo.db.QueryRowContext(ctx, `
		SELECT id, policy_json
		FROM policy_version
		WHERE workspace_id = ?
		  AND policy_set_id = ?
		  AND status = 'active'
		ORDER BY version_number DESC, created_at DESC
		LIMIT 1
	`, workspaceID, policySetID).Scan(&versionID, &raw)
	if err != nil {
		return nil, "", fmt.Errorf("load active policy rules: %w", err)
	}
	rules, err := decodePolicyRules(raw)
	if err != nil {
		return nil, "", err
	}
	return rules, versionID, nil
}

func (s *Service) updatePolicyVersionRules(ctx context.Context, versionID string, rules []map[string]any) error {
	payload, err := json.Marshal(map[string]any{"rules": rules})
	if err != nil {
		return fmt.Errorf("encode policy rules: %w", err)
	}
	_, err = s.repo.db.ExecContext(ctx, `
		UPDATE policy_version
		SET policy_json = ?
		WHERE id = ?
	`, string(payload), versionID)
	if err != nil {
		return fmt.Errorf("sync policy version rules: %w", err)
	}
	return nil
}

func mergeAgentDefinitionLimits(current, carta map[string]any) map[string]any {
	merged := make(map[string]any, len(current)+len(carta))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range carta {
		merged[key] = value
	}
	return merged
}

func mergePolicyRulesByAction(current, carta []map[string]any) []map[string]any {
	merged := make([]map[string]any, 0, len(current)+len(carta))
	indexByAction := make(map[string]int, len(current)+len(carta))

	for _, rule := range current {
		cloned := clonePolicyRule(rule)
		action, _ := cloned["action"].(string)
		indexByAction[action] = len(merged)
		merged = append(merged, cloned)
	}
	for _, rule := range carta {
		cloned := clonePolicyRule(rule)
		action, _ := cloned["action"].(string)
		if idx, ok := indexByAction[action]; ok {
			merged[idx] = cloned
			continue
		}
		indexByAction[action] = len(merged)
		merged = append(merged, cloned)
	}

	return merged
}

func decodePolicyRules(raw string) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []map[string]any{}, nil
	}

	var doc struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal([]byte(trimmed), &doc); err == nil && doc.Rules != nil {
		return doc.Rules, nil
	}

	var rules []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rules); err != nil {
		return nil, fmt.Errorf("decode policy rules: %w", err)
	}
	return rules, nil
}

func clonePolicyRule(rule map[string]any) map[string]any {
	cloned := make(map[string]any, len(rule))
	for key, value := range rule {
		cloned[key] = value
	}
	return cloned
}

func isCartaSource(source string) bool {
	trimmed := strings.TrimSpace(source)
	return trimmed == "CARTA" || strings.HasPrefix(trimmed, "CARTA ")
}

func (s *Service) promoteWorkflowToActive(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	return s.repo.Update(ctx, workflow.WorkspaceID, workflow.ID, UpdateInput{
		AgentDefinitionID: workflow.AgentDefinitionID,
		Description:       workflow.Description,
		DSLSource:         workflow.DSLSource,
		SpecSource:        workflow.SpecSource,
		Status:            StatusActive,
		ArchivedAt:        archivedAtForStatus(StatusActive, workflow.ArchivedAt),
	})
}

func (s *Service) NewVersion(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusActive {
		return nil, ErrWorkflowVersionInvalid
	}

	parentID := existing.ID
	next, err := s.repo.Create(ctx, CreateInput{
		ID:                uuid.NewV7().String(),
		WorkspaceID:       existing.WorkspaceID,
		AgentDefinitionID: existing.AgentDefinitionID,
		ParentVersionID:   &parentID,
		Name:              existing.Name,
		Description:       cloneOptionalString(existing.Description),
		DSLSource:         existing.DSLSource,
		SpecSource:        cloneOptionalString(existing.SpecSource),
		Version:           existing.Version + 1,
		Status:            StatusDraft,
		CreatedByUserID:   existing.CreatedByUserID,
	})
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrWorkflowNameConflict
		}
		return nil, err
	}
	return next, nil
}

func (s *Service) Rollback(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusArchived {
		return nil, ErrWorkflowVersionInvalid
	}
	return s.MarkActive(ctx, workspaceID, workflowID)
}

func (s *Service) DeleteDraft(ctx context.Context, workspaceID, workflowID string) error {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return ErrWorkflowDeleteInvalid
	}
	return s.repo.Delete(ctx, workspaceID, workflowID)
}

func (s *Service) listBase(ctx context.Context, workspaceID string, input ListWorkflowsInput) ([]*Workflow, error) {
	if input.Status != nil {
		return s.repo.ListByStatus(ctx, workspaceID, *input.Status)
	}
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

func validateCreateInput(input CreateWorkflowInput) error {
	if strings.TrimSpace(input.WorkspaceID) == "" {
		return invalidWorkflowInput("workspace_id is required", nil)
	}
	if strings.TrimSpace(input.Name) == "" {
		return invalidWorkflowInput("name is required", nil)
	}
	if err := validateDSLSource(input.DSLSource); err != nil {
		return err
	}
	if err := validateOptionalSourceSize("spec_source", input.SpecSource); err != nil {
		return err
	}
	return nil
}

func validateUpdateInput(input UpdateWorkflowInput) error {
	if err := validateDSLSource(input.DSLSource); err != nil {
		return err
	}
	if err := validateOptionalSourceSize("spec_source", input.SpecSource); err != nil {
		return err
	}
	return nil
}

func validateDSLSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return invalidWorkflowInput("dsl_source is required", nil)
	}
	if err := validateOptionalSourceSize("dsl_source", source); err != nil {
		return err
	}
	return nil
}

func validateOptionalSourceSize(field, source string) error {
	if len(source) > maxSourceSizeBytes {
		return invalidWorkflowInput(fmt.Sprintf("%s exceeds %d bytes", field, maxSourceSizeBytes), nil)
	}
	return nil
}

func invalidWorkflowInput(reason string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalidWorkflowInput, reason)
	}
	return fmt.Errorf("%w: %s: %w", ErrInvalidWorkflowInput, reason, err)
}

func trimOptionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (s *Service) ensureNoOtherActiveWorkflow(ctx context.Context, workflow *Workflow) error {
	active, err := s.repo.GetActiveByName(ctx, workflow.WorkspaceID, workflow.Name)
	if err != nil {
		if errors.Is(err, ErrWorkflowNotFound) {
			return nil
		}
		return err
	}
	if active.ID != workflow.ID {
		return ErrWorkflowActiveConflict
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isValidStatusTransition(current, next Status) bool {
	if current == next {
		return true
	}

	switch current {
	case StatusDraft:
		return next == StatusTesting
	case StatusTesting:
		return next == StatusDraft || next == StatusActive
	case StatusActive:
		return next == StatusArchived
	case StatusArchived:
		return next == StatusActive
	default:
		return false
	}
}

func archivedAtForStatus(next Status, existing *time.Time) *time.Time {
	if next == StatusArchived {
		now := time.Now().UTC()
		return &now
	}
	return existing
}

func (s *Service) cancelScheduledJobsForWorkflow(ctx context.Context, workflow *Workflow) error {
	if s == nil || s.scheduler == nil || workflow == nil {
		return nil
	}
	_, err := s.scheduler.CancelBySource(ctx, workflow.WorkspaceID, workflow.ID)
	return err
}
