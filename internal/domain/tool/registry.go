package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrToolExecutorAlreadyRegistered = errors.New("tool executor already registered")
	ErrToolExecutorNotRegistered     = errors.New("tool executor not registered")
	ErrToolDefinitionNotFound        = errors.New("tool definition not found")
	ErrToolValidationFailed          = errors.New("tool params validation failed")
	ErrToolDefinitionInvalid         = errors.New("tool definition invalid")
	ErrToolInactive                  = errors.New("tool is inactive")
	ErrToolPermissionDenied          = errors.New("tool permission denied")
	ErrToolUserContextMissing        = errors.New("tool user context missing")
)

//nolint:revive // tipo público persistido/serializado y usado transversalmente
type ToolDefinition struct {
	ID                  string
	WorkspaceID         string
	Name                string
	Description         *string
	InputSchema         json.RawMessage
	RequiredPermissions []string
	IsActive            bool
	CreatedBy           *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

//nolint:revive // tipos públicos consolidados en módulo tool
type CreateToolDefinitionInput struct {
	WorkspaceID         string
	Name                string
	Description         *string
	InputSchema         json.RawMessage
	RequiredPermissions []string
	CreatedBy           *string
}

//nolint:revive // tipos publicos consolidados en modulo tool
type UpdateToolDefinitionInput struct {
	ID                  string
	WorkspaceID         string
	Name                string
	Description         *string
	InputSchema         json.RawMessage
	RequiredPermissions []string
}

//nolint:revive // interfaz publica usada por runtime para enforcement de tools
type ToolAuthorizer interface {
	CheckToolPermission(ctx context.Context, userID, toolID string) (bool, error)
}

type UsageRecorder interface {
	RecordEvent(ctx context.Context, input usage.RecordEventInput) (*usage.Event, error)
}

//nolint:revive // registro principal usado transversalmente en app/api/tests
type ToolRegistry struct {
	db        *sql.DB
	executors map[string]ToolExecutor
	authz     ToolAuthorizer
	audit     AuditLogger
	usage     UsageRecorder
}

func NewToolRegistry(db *sql.DB) *ToolRegistry {
	return NewToolRegistryWithRuntimeAndUsage(db, nil, nil, nil)
}

func NewToolRegistryWithAuthorizer(db *sql.DB, authz ToolAuthorizer) *ToolRegistry {
	return NewToolRegistryWithRuntimeAndUsage(db, authz, nil, nil)
}

func NewToolRegistryWithRuntime(db *sql.DB, authz ToolAuthorizer, audit AuditLogger) *ToolRegistry {
	return NewToolRegistryWithRuntimeAndUsage(db, authz, audit, nil)
}

func NewToolRegistryWithRuntimeAndUsage(db *sql.DB, authz ToolAuthorizer, audit AuditLogger, usage UsageRecorder) *ToolRegistry {
	return &ToolRegistry{db: db, executors: make(map[string]ToolExecutor), authz: authz, audit: audit, usage: usage}
}

func (r *ToolRegistry) Register(name string, executor ToolExecutor) error {
	name = strings.TrimSpace(name)
	if name == "" || executor == nil {
		return ErrToolExecutorNotRegistered
	}
	if _, exists := r.executors[name]; exists {
		return ErrToolExecutorAlreadyRegistered
	}
	r.executors[name] = executor
	return nil
}

func (r *ToolRegistry) Get(name string) (ToolExecutor, error) {
	executor, ok := r.executors[name]
	if !ok {
		return nil, ErrToolExecutorNotRegistered
	}
	return executor, nil
}

func (r *ToolRegistry) CreateToolDefinition(ctx context.Context, in CreateToolDefinitionInput) (*ToolDefinition, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := validateToolSchema(in.InputSchema); err != nil {
		return nil, err
	}

	requiredPermsRaw, err := json.Marshal(in.RequiredPermissions)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	item := &ToolDefinition{
		ID:                  uuid.NewV7().String(),
		WorkspaceID:         in.WorkspaceID,
		Name:                strings.TrimSpace(in.Name),
		Description:         in.Description,
		InputSchema:         in.InputSchema,
		RequiredPermissions: in.RequiredPermissions,
		IsActive:            true,
		CreatedBy:           in.CreatedBy,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tool_definition (
			id, workspace_id, name, description, input_schema,
			required_permissions, is_active, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.WorkspaceID,
		item.Name,
		item.Description,
		[]byte(item.InputSchema),
		[]byte(requiredPermsRaw),
		1,
		item.CreatedBy,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (r *ToolRegistry) UpdateToolDefinition(ctx context.Context, in UpdateToolDefinitionInput) (*ToolDefinition, error) {
	if err := validateUpdateInput(in); err != nil {
		return nil, err
	}

	requiredPermsRaw, err := marshalRequiredPermissions(in.RequiredPermissions)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE tool_definition
		SET name = ?, description = ?, input_schema = ?, required_permissions = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`,
		strings.TrimSpace(in.Name),
		in.Description,
		[]byte(in.InputSchema),
		[]byte(requiredPermsRaw),
		now,
		in.ID,
		in.WorkspaceID,
	)
	if err != nil {
		return nil, err
	}
	if errRows := ensureRowsAffected(res, ErrToolDefinitionNotFound); errRows != nil {
		return nil, errRows
	}

	return r.GetToolDefinitionByID(ctx, in.WorkspaceID, in.ID)
}

func (r *ToolRegistry) ListToolDefinitions(ctx context.Context, workspaceID string) ([]*ToolDefinition, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, description, input_schema,
		       required_permissions, is_active, created_by, created_at, updated_at
		FROM tool_definition
		WHERE workspace_id = ?
		ORDER BY created_at ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*ToolDefinition, 0)
	for rows.Next() {
		item, scanErr := scanToolDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	return out, nil
}

func (r *ToolRegistry) GetToolDefinitionByID(ctx context.Context, workspaceID, id string) (*ToolDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, input_schema,
		       required_permissions, is_active, created_by, created_at, updated_at
		FROM tool_definition
		WHERE workspace_id = ? AND id = ?
		LIMIT 1
	`, workspaceID, id)

	item, scanErr := scanToolDefinition(row)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return nil, ErrToolDefinitionNotFound
	}
	if scanErr != nil {
		return nil, scanErr
	}
	return item, nil
}

func (r *ToolRegistry) SetToolDefinitionActive(ctx context.Context, workspaceID, id string, isActive bool) (*ToolDefinition, error) {
	now := time.Now().UTC()
	activeRaw := 0
	if isActive {
		activeRaw = 1
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE tool_definition
		SET is_active = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, activeRaw, now, id, workspaceID)
	if err != nil {
		return nil, err
	}

	if errRows := ensureRowsAffected(res, ErrToolDefinitionNotFound); errRows != nil {
		return nil, errRows
	}

	return r.GetToolDefinitionByID(ctx, workspaceID, id)
}

func (r *ToolRegistry) DeleteToolDefinition(ctx context.Context, workspaceID, id string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM tool_definition
		WHERE id = ? AND workspace_id = ?
	`, id, workspaceID)
	if err != nil {
		return err
	}

	return ensureRowsAffected(res, ErrToolDefinitionNotFound)
}

func (r *ToolRegistry) ValidateParams(ctx context.Context, workspaceID, toolName string, params json.RawMessage) error {
	def, defErr := r.getToolDefinitionByName(ctx, workspaceID, toolName)
	if defErr != nil {
		return defErr
	}

	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}

	var input map[string]any
	if unmarshalErr := json.Unmarshal(params, &input); unmarshalErr != nil {
		return fmt.Errorf("%w: params must be a json object", ErrToolValidationFailed)
	}

	var schema map[string]any
	if unmarshalErr := json.Unmarshal(def.InputSchema, &schema); unmarshalErr != nil {
		return fmt.Errorf("%w: invalid persisted schema", ErrToolValidationFailed)
	}

	if valErr := validateAgainstMinimalSchema(input, schema); valErr != nil {
		return valErr
	}

	return nil
}

func (r *ToolRegistry) getToolDefinitionByName(ctx context.Context, workspaceID, toolName string) (*ToolDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, input_schema,
		       required_permissions, is_active, created_by, created_at, updated_at
		FROM tool_definition
		WHERE workspace_id = ? AND name = ?
		LIMIT 1
	`, workspaceID, toolName)

	item, scanErr := scanToolDefinition(row)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return nil, ErrToolDefinitionNotFound
	}
	if scanErr != nil {
		return nil, scanErr
	}
	return item, nil
}

func (r *ToolRegistry) getDefinitionForExecution(ctx context.Context, workspaceID, toolName string) (*ToolDefinition, error) {
	def, err := r.getToolDefinitionByName(ctx, workspaceID, toolName)
	if err == nil {
		return def, nil
	}
	if !errors.Is(err, ErrToolDefinitionNotFound) {
		return nil, err
	}
	if ensureErr := r.EnsureBuiltInToolDefinitions(ctx, workspaceID); ensureErr != nil {
		return nil, ensureErr
	}
	return r.getToolDefinitionByName(ctx, workspaceID, toolName)
}

func validateToolSchema(raw json.RawMessage) error {
	schema, err := parseToolSchema(raw)
	if err != nil {
		return err
	}
	if errType := validateSchemaObjectType(schema); errType != nil {
		return errType
	}
	props, err := validateSchemaProperties(schema)
	if err != nil {
		return err
	}
	if errAdd := validateSchemaAdditionalProperties(schema); errAdd != nil {
		return errAdd
	}
	return validateSchemaRequiredKeys(schema, props)
}

func validateAgainstMinimalSchema(input, schema map[string]any) error {
	if valErr := validateRequiredFields(input, extractStringSlice(schema["required"])); valErr != nil {
		return valErr
	}

	if resolveAdditionalProperties(schema) {
		return nil
	}

	return validateUnknownFields(input, buildAllowedPropsSet(schema))
}

func validateUpdateInput(in UpdateToolDefinitionInput) error {
	if strings.TrimSpace(in.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	return validateToolSchema(in.InputSchema)
}

func marshalRequiredPermissions(perms []string) ([]byte, error) {
	return json.Marshal(perms)
}

func ensureRowsAffected(res sql.Result, notFoundErr error) error {
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return notFoundErr
	}
	return nil
}

func parseToolSchema(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: input schema is required", ErrToolDefinitionInvalid)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%w: input schema must be valid json", ErrToolDefinitionInvalid)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("%w: input schema must be a json object", ErrToolDefinitionInvalid)
	}
	return schema, nil
}

func validateSchemaObjectType(schema map[string]any) error {
	if schemaType, _ := schema["type"].(string); schemaType != "object" {
		return fmt.Errorf("%w: input schema type must be object", ErrToolDefinitionInvalid)
	}
	return nil
}

func validateSchemaProperties(schema map[string]any) (map[string]any, error) {
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) == 0 {
		return nil, fmt.Errorf("%w: input schema properties must be a non-empty object", ErrToolDefinitionInvalid)
	}
	return props, nil
}

func validateSchemaAdditionalProperties(schema map[string]any) error {
	if _, ok := schema["additionalProperties"].(bool); !ok {
		return fmt.Errorf("%w: input schema additionalProperties must be explicit", ErrToolDefinitionInvalid)
	}
	return nil
}

func validateSchemaRequiredKeys(schema map[string]any, props map[string]any) error {
	required, err := parseRequiredKeys(schema["required"])
	if err != nil {
		return err
	}
	for _, key := range required {
		if _, exists := props[key]; !exists {
			return fmt.Errorf("%w: required field %q must be declared in properties", ErrToolDefinitionInvalid, key)
		}
	}
	return nil
}

func parseRequiredKeys(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}

	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: required must be an array of strings", ErrToolDefinitionInvalid)
	}

	out := make([]string, 0, len(arr))
	for _, item := range arr {
		key, isString := item.(string)
		if !isString || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%w: required must be an array of non-empty strings", ErrToolDefinitionInvalid)
		}
		out = append(out, key)
	}
	return out, nil
}

func validateRequiredFields(input map[string]any, requiredKeys []string) error {
	for _, key := range requiredKeys {
		if _, keyExists := input[key]; !keyExists {
			return fmt.Errorf("%w: missing required field %q", ErrToolValidationFailed, key)
		}
	}
	return nil
}

func resolveAdditionalProperties(schema map[string]any) bool {
	allowAdditional := true
	if addProp, hasProp := schema["additionalProperties"].(bool); hasProp {
		allowAdditional = addProp
	}
	return allowAdditional
}

func buildAllowedPropsSet(schema map[string]any) map[string]struct{} {
	allowedProps := map[string]struct{}{}
	if props, hasProps := schema["properties"].(map[string]any); hasProps {
		for key := range props {
			allowedProps[key] = struct{}{}
		}
	}
	return allowedProps
}

func validateUnknownFields(input map[string]any, allowedProps map[string]struct{}) error {
	for key := range input {
		if _, allowed := allowedProps[key]; !allowed {
			return fmt.Errorf("%w: unknown field %q", ErrToolValidationFailed, key)
		}
	}
	return nil
}

func extractStringSlice(v any) []string {
	if arr, isArr := v.([]any); isArr {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, isStr := item.(string); isStr && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

type toolScanner interface {
	Scan(dest ...any) error
}

func scanToolDefinition(scan toolScanner) (*ToolDefinition, error) {
	var (
		item             ToolDefinition
		descriptionRaw   sql.NullString
		requiredPermsRaw []byte
		createdByRaw     sql.NullString
		isActiveRaw      int
	)

	if err := scan.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.Name,
		&descriptionRaw,
		&item.InputSchema,
		&requiredPermsRaw,
		&isActiveRaw,
		&createdByRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.IsActive = isActiveRaw == 1
	if descriptionRaw.Valid {
		v := descriptionRaw.String
		item.Description = &v
	}
	if createdByRaw.Valid {
		v := createdByRaw.String
		item.CreatedBy = &v
	}

	var perms []string
	if len(requiredPermsRaw) > 0 {
		_ = json.Unmarshal(requiredPermsRaw, &perms)
	}
	item.RequiredPermissions = perms

	return &item, nil
}
