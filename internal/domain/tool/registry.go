package tool

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

var (
	ErrToolExecutorAlreadyRegistered = errors.New("tool executor already registered")
	ErrToolExecutorNotRegistered     = errors.New("tool executor not registered")
	ErrToolDefinitionNotFound        = errors.New("tool definition not found")
	ErrToolValidationFailed          = errors.New("tool params validation failed")
)

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

type CreateToolDefinitionInput struct {
	WorkspaceID         string
	Name                string
	Description         *string
	InputSchema         json.RawMessage
	RequiredPermissions []string
	CreatedBy           *string
}

type ToolRegistry struct {
	db        *sql.DB
	executors map[string]ToolExecutor
}

func NewToolRegistry(db *sql.DB) *ToolRegistry {
	return &ToolRegistry{db: db, executors: make(map[string]ToolExecutor)}
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

	if len(in.InputSchema) == 0 {
		in.InputSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{}}`)
	}
	if !json.Valid(in.InputSchema) {
		return nil, fmt.Errorf("input schema must be valid json")
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ToolRegistry) ValidateParams(ctx context.Context, workspaceID, toolName string, params json.RawMessage) error {
	def, err := r.getToolDefinitionByName(ctx, workspaceID, toolName)
	if err != nil {
		return err
	}

	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}

	var input map[string]any
	if err := json.Unmarshal(params, &input); err != nil {
		return fmt.Errorf("%w: params must be a json object", ErrToolValidationFailed)
	}

	var schema map[string]any
	if err := json.Unmarshal(def.InputSchema, &schema); err != nil {
		return fmt.Errorf("%w: invalid persisted schema", ErrToolValidationFailed)
	}

	if err := validateAgainstMinimalSchema(input, schema); err != nil {
		return err
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

	item, err := scanToolDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrToolDefinitionNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func validateAgainstMinimalSchema(input, schema map[string]any) error {
	requiredKeys := extractStringSlice(schema["required"])
	for _, key := range requiredKeys {
		if _, ok := input[key]; !ok {
			return fmt.Errorf("%w: missing required field %q", ErrToolValidationFailed, key)
		}
	}

	allowAdditional := true
	if v, ok := schema["additionalProperties"].(bool); ok {
		allowAdditional = v
	}

	allowedProps := map[string]struct{}{}
	if props, ok := schema["properties"].(map[string]any); ok {
		for key := range props {
			allowedProps[key] = struct{}{}
		}
	}

	if !allowAdditional {
		for key := range input {
			if _, ok := allowedProps[key]; !ok {
				return fmt.Errorf("%w: unknown field %q", ErrToolValidationFailed, key)
			}
		}
	}

	return nil
}

func extractStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
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
