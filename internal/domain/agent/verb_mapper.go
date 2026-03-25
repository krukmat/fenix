package agent

import (
	"fmt"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

var ErrVerbMappingFailed = fmt.Errorf("verb mapping failed")

type RuntimeOperationKind string

const (
	RuntimeOperationTool     RuntimeOperationKind = "tool"
	RuntimeOperationAgent    RuntimeOperationKind = "agent"
	RuntimeOperationDispatch RuntimeOperationKind = "dispatch"
	RuntimeOperationSurface  RuntimeOperationKind = "surface"
)

const notifyTargetContactReply = "contact.reply"

type RuntimeOperation struct {
	Kind      RuntimeOperationKind `json:"kind"`
	Verb      string               `json:"verb"`
	Target    string               `json:"target,omitempty"`
	ToolName  string               `json:"tool_name,omitempty"`
	AgentName string               `json:"agent_name,omitempty"`
	Params    map[string]any       `json:"params,omitempty"`
}

type VerbMapper struct {
	evaluator *ExpressionEvaluator
}

func NewVerbMapper() *VerbMapper {
	return &VerbMapper{evaluator: NewExpressionEvaluator()}
}

func ToolNameForStatement(stmt Statement) string {
	switch node := stmt.(type) {
	case *SetStatement:
		if node == nil || node.Target == nil {
			return ""
		}
		return toolNameForSetTarget(node.Target.Name)
	case *NotifyStatement:
		if node == nil || node.Target == nil {
			return ""
		}
		return toolNameForNotifyTarget(node.Target.Name)
	default:
		return ""
	}
}

func (m *VerbMapper) MapStatement(stmt Statement, evalCtx map[string]any) (*RuntimeOperation, error) {
	if handled, op, err := m.mapLeafStatement(stmt, evalCtx); handled {
		return op, err
	}
	return nil, fmt.Errorf("%w: unsupported statement type", ErrVerbMappingFailed)
}

func (m *VerbMapper) mapLeafStatement(stmt Statement, evalCtx map[string]any) (bool, *RuntimeOperation, error) {
	switch node := stmt.(type) {
	case *SetStatement:
		op, err := m.mapSetNode(node, evalCtx)
		return true, op, err
	case *NotifyStatement:
		op, err := m.mapNotifyNode(node, evalCtx)
		return true, op, err
	case *AgentStatement:
		op, err := m.mapAgentStatement(node, evalCtx)
		return true, op, err
	case *DispatchStatement:
		op, err := m.mapDispatchStatement(node, evalCtx)
		return true, op, err
	case *SurfaceStatement:
		op, err := m.mapSurfaceStatement(node, evalCtx)
		return true, op, err
	default:
		return false, nil, nil
	}
}

func (m *VerbMapper) mapSetNode(node *SetStatement, evalCtx map[string]any) (*RuntimeOperation, error) {
	value, err := m.evaluator.Evaluate(node.Value, evalCtx)
	if err != nil {
		return nil, err
	}
	return mapSetOperation(node.Target.Name, value, evalCtx)
}

func (m *VerbMapper) mapNotifyNode(node *NotifyStatement, evalCtx map[string]any) (*RuntimeOperation, error) {
	value, err := m.evaluator.Evaluate(node.Value, evalCtx)
	if err != nil {
		return nil, err
	}
	return mapNotifyOperation(node.Target.Name, value, evalCtx)
}

func (m *VerbMapper) MapBridgeStep(step BridgeStep, evalCtx map[string]any) (*RuntimeOperation, error) {
	verb := strings.ToUpper(strings.TrimSpace(step.Action.Verb))
	switch verb {
	case BridgeVerbSet:
		value, ok := actionArg(step.Action.Args, "value")
		if !ok {
			return nil, fmt.Errorf("%w: step %s: SET requires args.value", ErrVerbMappingFailed, step.ID)
		}
		return mapSetOperation(step.Action.Target, value, evalCtx)
	case BridgeVerbNotify:
		message, ok := actionArg(step.Action.Args, "message")
		if !ok {
			return nil, fmt.Errorf("%w: step %s: NOTIFY requires args.message", ErrVerbMappingFailed, step.ID)
		}
		return mapNotifyOperation(step.Action.Target, message, evalCtx)
	case BridgeVerbAgent:
		return mapAgentOperation(step.Action.Target, step.Action.Args), nil
	default:
		return nil, fmt.Errorf("%w: step %s: unsupported verb %s", ErrVerbMappingFailed, step.ID, verb)
	}
}

func (m *VerbMapper) mapAgentStatement(stmt *AgentStatement, evalCtx map[string]any) (*RuntimeOperation, error) {
	if stmt == nil || stmt.Name == nil || strings.TrimSpace(stmt.Name.Name) == "" {
		return nil, fmt.Errorf("%w: AGENT requires target", ErrVerbMappingFailed)
	}
	if stmt.Input == nil {
		return mapAgentOperation(stmt.Name.Name, nil), nil
	}
	value, err := m.evaluator.Evaluate(stmt.Input, evalCtx)
	if err != nil {
		return nil, err
	}
	params, err := normalizeAgentInput(value)
	if err != nil {
		return nil, err
	}
	return mapAgentOperation(stmt.Name.Name, params), nil
}

func (m *VerbMapper) mapDispatchStatement(stmt *DispatchStatement, evalCtx map[string]any) (*RuntimeOperation, error) {
	if stmt == nil || stmt.Target == nil || strings.TrimSpace(stmt.Target.Name) == "" {
		return nil, fmt.Errorf("%w: DISPATCH requires target", ErrVerbMappingFailed)
	}
	if stmt.Payload == nil {
		return nil, fmt.Errorf("%w: DISPATCH requires payload", ErrVerbMappingFailed)
	}
	value, err := m.evaluator.Evaluate(stmt.Payload, evalCtx)
	if err != nil {
		return nil, err
	}
	params, err := normalizeAgentInput(value)
	if err != nil {
		return nil, err
	}
	return mapDispatchOperation(stmt.Target.Name, params), nil
}

func (m *VerbMapper) mapSurfaceStatement(stmt *SurfaceStatement, evalCtx map[string]any) (*RuntimeOperation, error) {
	if err := validateSurfaceMapping(stmt); err != nil {
		return nil, err
	}
	value, err := m.evaluator.Evaluate(stmt.Payload, evalCtx)
	if err != nil {
		return nil, err
	}
	params, err := normalizeSurfacePayload(value)
	if err != nil {
		return nil, err
	}
	return mapSurfaceOperation(stmt.Entity.Name, stmt.View.Name, params), nil
}

func validateSurfaceMapping(stmt *SurfaceStatement) error {
	switch {
	case stmt == nil || stmt.Entity == nil || strings.TrimSpace(stmt.Entity.Name) == "":
		return fmt.Errorf("%w: SURFACE requires entity", ErrVerbMappingFailed)
	case stmt.View == nil || strings.TrimSpace(stmt.View.Name) == "":
		return fmt.Errorf("%w: SURFACE requires target view", ErrVerbMappingFailed)
	case stmt.Payload == nil:
		return fmt.Errorf("%w: SURFACE requires payload", ErrVerbMappingFailed)
	default:
		return nil
	}
}

func mapSetOperation(target string, value any, evalCtx map[string]any) (*RuntimeOperation, error) {
	toolName := toolNameForSetTarget(target)
	switch toolName {
	case tool.BuiltinUpdateCase:
	default:
		return nil, fmt.Errorf("%w: unsupported SET target %s", ErrVerbMappingFailed, target)
	}
	switch strings.TrimSpace(target) {
	case "case.status":
		return &RuntimeOperation{
			Kind:     RuntimeOperationTool,
			Verb:     "SET",
			Target:   target,
			ToolName: toolName,
			Params: map[string]any{
				"case_id": resolveEntityID(evalCtx, bridgeEntityCase),
				"status":  value,
			},
		}, nil
	case "case.priority":
		return &RuntimeOperation{
			Kind:     RuntimeOperationTool,
			Verb:     "SET",
			Target:   target,
			ToolName: toolName,
			Params: map[string]any{
				"case_id":  resolveEntityID(evalCtx, bridgeEntityCase),
				"priority": value,
			},
		}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported SET target %s", ErrVerbMappingFailed, target)
	}
}

func mapNotifyOperation(target string, value any, evalCtx map[string]any) (*RuntimeOperation, error) {
	toolName := toolNameForNotifyTarget(target)
	if toolName == "" {
		return nil, fmt.Errorf("%w: unsupported NOTIFY target %s", ErrVerbMappingFailed, target)
	}
	switch strings.TrimSpace(target) {
	case surfaceEntityContact, notifyTargetContactReply:
		return &RuntimeOperation{
			Kind:     RuntimeOperationTool,
			Verb:     "NOTIFY",
			Target:   target,
			ToolName: toolName,
			Params: map[string]any{
				"case_id": resolveEntityID(evalCtx, bridgeEntityCase),
				"body":    value,
			},
		}, nil
	case "salesperson", "salesperson.task":
		entityType, entityID := resolvePrimaryEntity(evalCtx)
		return &RuntimeOperation{
			Kind:     RuntimeOperationTool,
			Verb:     "NOTIFY",
			Target:   target,
			ToolName: toolName,
			Params: map[string]any{
				"owner_id":    resolveOwnerID(evalCtx),
				"title":       value,
				"entity_type": entityType,
				"entity_id":   entityID,
			},
		}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported NOTIFY target %s", ErrVerbMappingFailed, target)
	}
}

func toolNameForSetTarget(target string) string {
	switch strings.TrimSpace(target) {
	case "case.status", "case.priority":
		return tool.BuiltinUpdateCase
	default:
		return ""
	}
}

func toolNameForNotifyTarget(target string) string {
	switch strings.TrimSpace(target) {
	case surfaceEntityContact, notifyTargetContactReply:
		return tool.BuiltinSendReply
	case "salesperson", "salesperson.task":
		return tool.BuiltinCreateTask
	default:
		return ""
	}
}

func mapAgentOperation(target string, params map[string]any) *RuntimeOperation {
	if params == nil {
		params = map[string]any{}
	}
	return &RuntimeOperation{
		Kind:      RuntimeOperationAgent,
		Verb:      "AGENT",
		Target:    strings.TrimSpace(target),
		AgentName: strings.TrimSpace(target),
		Params:    params,
	}
}

func mapDispatchOperation(target string, params map[string]any) *RuntimeOperation {
	if params == nil {
		params = map[string]any{}
	}
	target = strings.TrimSpace(target)
	return &RuntimeOperation{
		Kind:      RuntimeOperationDispatch,
		Verb:      "DISPATCH",
		Target:    target,
		AgentName: target,
		Params:    params,
	}
}

func mapSurfaceOperation(entity, view string, params map[string]any) *RuntimeOperation {
	if params == nil {
		params = map[string]any{}
	}
	entity = strings.TrimSpace(entity)
	view = strings.TrimSpace(view)
	params["entity"] = entity
	params["view"] = view
	return &RuntimeOperation{
		Kind:   RuntimeOperationSurface,
		Verb:   "SURFACE",
		Target: entity,
		Params: params,
	}
}

func actionArg(args map[string]any, key string) (any, bool) {
	if len(args) == 0 {
		return nil, false
	}
	value, ok := args[key]
	return value, ok
}

func normalizeAgentInput(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	if params, ok := value.(map[string]any); ok {
		return params, nil
	}
	return map[string]any{"input": value}, nil
}

func normalizeSurfacePayload(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	if params, ok := value.(map[string]any); ok {
		return params, nil
	}
	return map[string]any{"value": value}, nil
}
