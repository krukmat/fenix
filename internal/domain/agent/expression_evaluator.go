package agent

import "fmt"

type ExpressionEvalError struct {
	Position Position
	Reason   string
}

func (e *ExpressionEvalError) Error() string {
	return fmt.Sprintf("expression evaluation error at line %d, column %d: %s", e.Position.Line, e.Position.Column, e.Reason)
}

type ExpressionEvaluator struct{}

func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{}
}

func (e *ExpressionEvaluator) Evaluate(expr Expression, evalCtx map[string]any) (any, error) {
	if expr == nil {
		return nil, &ExpressionEvalError{Reason: "expression is required"}
	}

	switch node := expr.(type) {
	case *IdentifierExpr:
		return resolveBridgeValue(evalCtx, node.Name), nil
	case *LiteralExpr:
		return node.Value, nil
	case *ArrayLiteralExpr:
		return e.evaluateArray(node, evalCtx)
	case *ObjectLiteralExpr:
		return e.evaluateObject(node, evalCtx)
	case *ComparisonExpr:
		return e.evaluateComparison(node, evalCtx)
	default:
		return nil, &ExpressionEvalError{
			Position: expr.Pos(),
			Reason:   "unsupported expression node",
		}
	}
}

func (e *ExpressionEvaluator) EvaluateCondition(expr Expression, evalCtx map[string]any) (bool, error) {
	value, err := e.Evaluate(expr, evalCtx)
	if err != nil {
		return false, err
	}
	boolean, ok := value.(bool)
	if !ok {
		return false, &ExpressionEvalError{
			Position: expr.Pos(),
			Reason:   "condition must evaluate to boolean",
		}
	}
	return boolean, nil
}

func (e *ExpressionEvaluator) evaluateArray(expr *ArrayLiteralExpr, evalCtx map[string]any) ([]any, error) {
	values := make([]any, 0, len(expr.Elements))
	for _, element := range expr.Elements {
		value, err := e.Evaluate(element, evalCtx)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (e *ExpressionEvaluator) evaluateObject(expr *ObjectLiteralExpr, evalCtx map[string]any) (map[string]any, error) {
	values := make(map[string]any, len(expr.Fields))
	for _, field := range expr.Fields {
		value, err := e.Evaluate(field.Value, evalCtx)
		if err != nil {
			return nil, err
		}
		values[field.Key] = value
	}
	return values, nil
}

func (e *ExpressionEvaluator) evaluateComparison(expr *ComparisonExpr, evalCtx map[string]any) (bool, error) {
	left, err := e.Evaluate(expr.Left, evalCtx)
	if err != nil {
		return false, err
	}
	right, err := e.Evaluate(expr.Right, evalCtx)
	if err != nil {
		return false, err
	}

	switch expr.Operator {
	case TokenEqual:
		return compareEquality(left, right), nil
	case TokenNotEqual:
		return !compareEquality(left, right), nil
	case TokenGT, TokenLT, TokenGTE, TokenLTE:
		return evaluateDSLOrderedOp(expr, left, right)
	case TokenIn:
		return evaluateDSLInOp(expr, left, right)
	default:
		return false, &ExpressionEvalError{
			Position: expr.Pos(),
			Reason:   "unsupported comparison operator",
		}
	}
}

func evaluateDSLOrderedOp(expr *ComparisonExpr, left, right any) (bool, error) {
	lv, lok := toFloat64(left)
	rv, rok := toFloat64(right)
	if !lok || !rok {
		return false, &ExpressionEvalError{
			Position: expr.Pos(),
			Reason:   "ordered comparison requires numeric operands",
		}
	}

	switch expr.Operator {
	case TokenGT:
		return lv > rv, nil
	case TokenLT:
		return lv < rv, nil
	case TokenGTE:
		return lv >= rv, nil
	default:
		return lv <= rv, nil
	}
}

func evaluateDSLInOp(expr *ComparisonExpr, left, right any) (bool, error) {
	values, ok := right.([]any)
	if !ok {
		return false, &ExpressionEvalError{
			Position: expr.Pos(),
			Reason:   "IN requires array right operand",
		}
	}
	for _, item := range values {
		if compareEquality(left, item) {
			return true, nil
		}
	}
	return false, nil
}
