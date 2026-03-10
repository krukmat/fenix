package agent

import (
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

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
