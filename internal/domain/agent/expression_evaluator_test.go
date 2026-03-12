package agent

import (
	"errors"
	"testing"
)

func TestExpressionEvaluatorEvaluateIdentifierAndLiteral(t *testing.T) {
	t.Parallel()

	evaluator := NewExpressionEvaluator()
	ctx := map[string]any{
		"case": map[string]any{
			"priority": "high",
		},
	}

	value, err := evaluator.Evaluate(&IdentifierExpr{
		Name:     "case.priority",
		Position: Position{Line: 1, Column: 1},
	}, ctx)
	if err != nil {
		t.Fatalf("Evaluate(identifier) error = %v", err)
	}
	if value != "high" {
		t.Fatalf("identifier value = %#v, want high", value)
	}

	value, err = evaluator.Evaluate(&LiteralExpr{
		Value:    "resolved",
		Position: Position{Line: 1, Column: 5},
	}, ctx)
	if err != nil {
		t.Fatalf("Evaluate(literal) error = %v", err)
	}
	if value != "resolved" {
		t.Fatalf("literal value = %#v, want resolved", value)
	}
}

func TestExpressionEvaluatorEvaluateComparison(t *testing.T) {
	t.Parallel()

	evaluator := NewExpressionEvaluator()
	ctx := map[string]any{
		"lead": map[string]any{
			"score": 0.9,
		},
	}

	result, err := evaluator.EvaluateCondition(&ComparisonExpr{
		Left:     &IdentifierExpr{Name: "lead.score", Position: Position{Line: 1, Column: 4}},
		Operator: TokenGTE,
		Right:    &LiteralExpr{Value: 0.8, Position: Position{Line: 1, Column: 18}},
		Position: Position{Line: 1, Column: 1},
	}, ctx)
	if err != nil {
		t.Fatalf("EvaluateCondition() error = %v", err)
	}
	if !result {
		t.Fatal("expected condition to be true")
	}
}

func TestExpressionEvaluatorEvaluateInOperator(t *testing.T) {
	t.Parallel()

	evaluator := NewExpressionEvaluator()
	ctx := map[string]any{
		"case": map[string]any{
			"priority": "urgent",
		},
	}

	result, err := evaluator.EvaluateCondition(&ComparisonExpr{
		Left:     &IdentifierExpr{Name: "case.priority", Position: Position{Line: 1, Column: 4}},
		Operator: TokenIn,
		Right: &ArrayLiteralExpr{
			Elements: []Expression{
				&LiteralExpr{Value: "high", Position: Position{Line: 1, Column: 22}},
				&LiteralExpr{Value: "urgent", Position: Position{Line: 1, Column: 30}},
			},
			Position: Position{Line: 1, Column: 21},
		},
		Position: Position{Line: 1, Column: 1},
	}, ctx)
	if err != nil {
		t.Fatalf("EvaluateCondition() error = %v", err)
	}
	if !result {
		t.Fatal("expected IN condition to be true")
	}
}

func TestExpressionEvaluatorRejectsNumericTypeMismatch(t *testing.T) {
	t.Parallel()

	evaluator := NewExpressionEvaluator()
	ctx := map[string]any{
		"lead": map[string]any{
			"score": "high",
		},
	}

	_, err := evaluator.EvaluateCondition(&ComparisonExpr{
		Left:     &IdentifierExpr{Name: "lead.score", Position: Position{Line: 3, Column: 4}},
		Operator: TokenGTE,
		Right:    &LiteralExpr{Value: 0.8, Position: Position{Line: 3, Column: 18}},
		Position: Position{Line: 3, Column: 1},
	}, ctx)
	if err == nil {
		t.Fatal("expected error")
	}

	var evalErr *ExpressionEvalError
	if !errors.As(err, &evalErr) {
		t.Fatalf("expected ExpressionEvalError, got %T", err)
	}
	if evalErr.Position.Line != 3 {
		t.Fatalf("evalErr.Position = %+v, want line 3", evalErr.Position)
	}
}

func TestExpressionEvaluatorRejectsNonBooleanCondition(t *testing.T) {
	t.Parallel()

	evaluator := NewExpressionEvaluator()
	_, err := evaluator.EvaluateCondition(&LiteralExpr{
		Value:    "resolved",
		Position: Position{Line: 2, Column: 1},
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExpressionEvalErrorMessage(t *testing.T) {
	t.Parallel()

	e := &ExpressionEvalError{Position: Position{Line: 4, Column: 8}, Reason: "unsupported expression type"}
	want := "expression evaluation error at line 4, column 8: unsupported expression type"
	if e.Error() != want {
		t.Fatalf("Error() = %q, want %q", e.Error(), want)
	}
}
