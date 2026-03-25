package agent

import (
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func TestToolNameForStatement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		stmt Statement
		want string
	}{
		{
			name: "set case status maps to update_case",
			stmt: &SetStatement{Target: &IdentifierExpr{Name: "case.status"}},
			want: tool.BuiltinUpdateCase,
		},
		{
			name: "notify salesperson maps to create_task",
			stmt: &NotifyStatement{Target: &IdentifierExpr{Name: "salesperson"}},
			want: tool.BuiltinCreateTask,
		},
		{
			name: "notify contact maps to send_reply",
			stmt: &NotifyStatement{Target: &IdentifierExpr{Name: "contact"}},
			want: tool.BuiltinSendReply,
		},
		{
			name: "if statement has no tool",
			stmt: &IfStatement{},
			want: "",
		},
		{
			name: "wait statement has no tool",
			stmt: &WaitStatement{},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToolNameForStatement(tt.stmt); got != tt.want {
				t.Fatalf("ToolNameForStatement() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVerbMapperMapStatementSet(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()
	stmt := &SetStatement{
		Target: &IdentifierExpr{Name: "case.status"},
		Value:  &LiteralExpr{Value: "resolved"},
	}

	op, err := mapper.MapStatement(stmt, map[string]any{
		"case": map[string]any{"id": "case-1"},
	})
	if err != nil {
		t.Fatalf("MapStatement returned error: %v", err)
	}
	if op.Kind != RuntimeOperationTool {
		t.Fatalf("unexpected kind: %s", op.Kind)
	}
	if op.ToolName != tool.BuiltinUpdateCase {
		t.Fatalf("unexpected tool: %s", op.ToolName)
	}
	if op.Params["status"] != "resolved" {
		t.Fatalf("unexpected status param: %v", op.Params["status"])
	}
	if op.Params["case_id"] != "case-1" {
		t.Fatalf("unexpected case_id: %v", op.Params["case_id"])
	}
}

func TestVerbMapperMapStatementNotify(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()
	stmt := &NotifyStatement{
		Target: &IdentifierExpr{Name: "salesperson"},
		Value:  &LiteralExpr{Value: "review this case"},
	}

	op, err := mapper.MapStatement(stmt, map[string]any{
		"case":           map[string]any{"id": "case-1"},
		"owner_id":       "owner-1",
		"salesperson_id": "owner-2",
	})
	if err != nil {
		t.Fatalf("MapStatement returned error: %v", err)
	}
	if op.ToolName != tool.BuiltinCreateTask {
		t.Fatalf("unexpected tool: %s", op.ToolName)
	}
	if op.Params["owner_id"] != "owner-1" {
		t.Fatalf("unexpected owner_id: %v", op.Params["owner_id"])
	}
	if op.Params["entity_type"] != bridgeEntityCase {
		t.Fatalf("unexpected entity_type: %v", op.Params["entity_type"])
	}
}

func TestVerbMapperMapStatementAgent(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()
	stmt := &AgentStatement{
		Name: &IdentifierExpr{Name: "support"},
		Input: &ObjectLiteralExpr{
			Fields: []ObjectField{
				{Key: "priority", Value: &LiteralExpr{Value: "high"}},
			},
		},
	}

	op, err := mapper.MapStatement(stmt, nil)
	if err != nil {
		t.Fatalf("MapStatement returned error: %v", err)
	}
	if op.Kind != RuntimeOperationAgent {
		t.Fatalf("unexpected kind: %s", op.Kind)
	}
	if op.AgentName != "support" {
		t.Fatalf("unexpected agent name: %s", op.AgentName)
	}
	if op.Params["priority"] != "high" {
		t.Fatalf("unexpected params: %#v", op.Params)
	}
}

func TestVerbMapperMapBridgeStepReusesMappings(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()
	step := BridgeStep{
		ID: "step-1",
		Action: BridgeAction{
			Verb:   BridgeVerbNotify,
			Target: "contact.reply",
			Args:   map[string]any{"message": "done"},
		},
	}

	op, err := mapper.MapBridgeStep(step, map[string]any{
		"case": map[string]any{"id": "case-1"},
	})
	if err != nil {
		t.Fatalf("MapBridgeStep returned error: %v", err)
	}
	if op.ToolName != tool.BuiltinSendReply {
		t.Fatalf("unexpected tool: %s", op.ToolName)
	}
	if op.Params["body"] != "done" {
		t.Fatalf("unexpected params: %#v", op.Params)
	}
}

func TestVerbMapperMapStatementFailsForUnsupportedTarget(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()
	_, err := mapper.MapStatement(&SetStatement{
		Target: &IdentifierExpr{Name: "deal.stage"},
		Value:  &LiteralExpr{Value: "won"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported target")
	}
	if !errors.Is(err, ErrVerbMappingFailed) {
		t.Fatalf("expected ErrVerbMappingFailed, got %v", err)
	}
}

func TestVerbMapperDispatchAndSurfaceHelpers(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()

	dispatchStmt := &DispatchStatement{
		Target: &IdentifierExpr{Name: "remote-agent"},
		Payload: &ObjectLiteralExpr{
			Fields: []ObjectField{
				{Key: "workflow_name", Value: &LiteralExpr{Value: "wf_remote"}},
			},
		},
	}
	dispatchOp, err := mapper.MapStatement(dispatchStmt, nil)
	if err != nil {
		t.Fatalf("MapStatement(dispatch) error = %v", err)
	}
	if dispatchOp.Kind != RuntimeOperationDispatch || dispatchOp.AgentName != "remote-agent" {
		t.Fatalf("unexpected dispatch op = %#v", dispatchOp)
	}

	surfaceStmt := &SurfaceStatement{
		Entity:  &IdentifierExpr{Name: "case"},
		View:    &IdentifierExpr{Name: "salesperson.view"},
		Payload: &LiteralExpr{Value: "review"},
	}
	surfaceOp, err := mapper.MapStatement(surfaceStmt, nil)
	if err != nil {
		t.Fatalf("MapStatement(surface) error = %v", err)
	}
	if surfaceOp.Kind != RuntimeOperationSurface || surfaceOp.Target != "case" {
		t.Fatalf("unexpected surface op = %#v", surfaceOp)
	}
	if surfaceOp.Params["view"] != "salesperson.view" || surfaceOp.Params["value"] != "review" {
		t.Fatalf("unexpected surface params = %#v", surfaceOp.Params)
	}

	notifyOp, err := mapNotifyOperation("contact.reply", "done", map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("mapNotifyOperation(contact.reply) error = %v", err)
	}
	if notifyOp.ToolName != tool.BuiltinSendReply || notifyOp.Params["body"] != "done" {
		t.Fatalf("unexpected notify op = %#v", notifyOp)
	}

	agentOp := mapAgentOperation("support", nil)
	if agentOp.Kind != RuntimeOperationAgent || agentOp.AgentName != "support" {
		t.Fatalf("unexpected agent op = %#v", agentOp)
	}

	dispatchMapped := mapDispatchOperation("router", nil)
	if dispatchMapped.Kind != RuntimeOperationDispatch || dispatchMapped.AgentName != "router" {
		t.Fatalf("unexpected mapped dispatch op = %#v", dispatchMapped)
	}

	surfaceMapped := mapSurfaceOperation("deal", "sales.team", nil)
	if surfaceMapped.Kind != RuntimeOperationSurface || surfaceMapped.Params["entity"] != "deal" {
		t.Fatalf("unexpected mapped surface op = %#v", surfaceMapped)
	}

	if value, ok := actionArg(map[string]any{"x": 1}, "x"); !ok || value.(int) != 1 {
		t.Fatalf("actionArg() = %#v, %v", value, ok)
	}
	normalizedInput, err := normalizeAgentInput("hello")
	if err != nil {
		t.Fatalf("normalizeAgentInput() error = %v", err)
	}
	if normalizedInput["input"] != "hello" {
		t.Fatalf("normalizeAgentInput() = %#v", normalizedInput)
	}
	normalizedSurface, err := normalizeSurfacePayload("hello")
	if err != nil {
		t.Fatalf("normalizeSurfacePayload() error = %v", err)
	}
	if normalizedSurface["value"] != "hello" {
		t.Fatalf("normalizeSurfacePayload() = %#v", normalizedSurface)
	}
}

func TestVerbMapperSurfaceAndDispatchValidationFailures(t *testing.T) {
	t.Parallel()

	mapper := NewVerbMapper()

	if _, err := mapper.MapStatement(&SurfaceStatement{
		Entity:  nil,
		View:    &IdentifierExpr{Name: "salesperson"},
		Payload: &LiteralExpr{Value: "review"},
	}, nil); !errors.Is(err, ErrVerbMappingFailed) {
		t.Fatalf("nil entity: expected ErrVerbMappingFailed, got %v", err)
	}

	if _, err := mapper.MapStatement(&SurfaceStatement{
		Entity:  &IdentifierExpr{Name: "case"},
		View:    nil,
		Payload: &LiteralExpr{Value: "review"},
	}, nil); !errors.Is(err, ErrVerbMappingFailed) {
		t.Fatalf("nil view: expected ErrVerbMappingFailed, got %v", err)
	}

	if _, err := mapper.MapStatement(&SurfaceStatement{
		Entity:  &IdentifierExpr{Name: "case"},
		View:    &IdentifierExpr{Name: "salesperson"},
		Payload: nil,
	}, nil); !errors.Is(err, ErrVerbMappingFailed) {
		t.Fatalf("nil payload: expected ErrVerbMappingFailed, got %v", err)
	}

	if _, err := mapper.MapStatement(&DispatchStatement{
		Target:  &IdentifierExpr{Name: "router"},
		Payload: nil,
	}, nil); err == nil {
		t.Fatal("expected dispatch validation error")
	}
}
