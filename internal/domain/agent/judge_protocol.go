package agent

import (
	"fmt"
	"strings"
)

const (
	judgeCheckProtocolAmbiguity           = 8
	judgeCheckProtocolContract            = 9
	judgeViolationDispatchContractMissing = "dispatch_contract_missing"
	judgeWarningSurfaceViewAmbiguous      = "surface_view_ambiguous"
)

func RunProtocolJudgeChecks(program *Program) ([]Violation, []Warning) {
	if program == nil || program.Workflow == nil {
		return nil, nil
	}

	collector := newProtocolCheckCollector(program.Workflow.Name)
	collector.collect(program.Workflow.Body)
	return collector.violations, collector.warnings
}

type protocolCheckCollector struct {
	workflowName string
	violations   []Violation
	warnings     []Warning
}

func newProtocolCheckCollector(workflowName string) *protocolCheckCollector {
	return &protocolCheckCollector{workflowName: strings.TrimSpace(workflowName)}
}

func (c *protocolCheckCollector) collect(statements []Statement) {
	for _, stmt := range statements {
		c.checkStatement(stmt)
		if ifStmt, ok := stmt.(*IfStatement); ok {
			c.collect(ifStmt.Body)
		}
	}
}

func (c *protocolCheckCollector) checkStatement(stmt Statement) {
	switch node := stmt.(type) {
	case *DispatchStatement:
		c.checkDispatch(node)
	case *SurfaceStatement:
		c.checkSurface(node)
	}
}

func (c *protocolCheckCollector) checkDispatch(stmt *DispatchStatement) {
	dispatchKind := string(TokenDispatch)
	payload, ok := objectPayload(stmt.Payload)
	if !ok {
		c.addViolation(judgeCheckProtocolContract, "dispatch_payload_invalid", "DISPATCH payload must be an object for protocol validation", stmt.Pos(), stmtLocation(dispatchKind, identifierName(stmt.Target)))
		return
	}

	hasWorkflowName := hasNonEmptyObjectField(payload, "workflow_name")
	hasDSLSource := hasNonEmptyObjectField(payload, "dsl_source")

	switch {
	case hasWorkflowName && hasDSLSource:
		c.addViolation(judgeCheckProtocolContract, "dispatch_contract_ambiguous", "DISPATCH must provide workflow_name or dsl_source, not both", stmt.Pos(), stmtLocation(dispatchKind, identifierName(stmt.Target)))
	case !hasWorkflowName && !hasDSLSource:
		c.addViolation(judgeCheckProtocolContract, judgeViolationDispatchContractMissing, "DISPATCH requires workflow_name or dsl_source in payload", stmt.Pos(), stmtLocation(dispatchKind, identifierName(stmt.Target)))
	case hasDSLSource:
		c.addWarning(judgeCheckProtocolAmbiguity, "dispatch_inline_dsl_portability", "DISPATCH with inline dsl_source may reduce A2A portability; prefer workflow_name when possible", stmt.Pos(), stmtLocation(dispatchKind, identifierName(stmt.Target)))
	}
}

func (c *protocolCheckCollector) checkSurface(stmt *SurfaceStatement) {
	surfaceKind := string(TokenSurface)
	payload, ok := objectPayload(stmt.Payload)
	if ok {
		if value, exists := payloadField(payload, "confidence"); exists && !isConfidenceValue(value) {
			c.addViolation(judgeCheckProtocolContract, "surface_confidence_invalid", "SURFACE confidence must be numeric in range [0.0, 1.0]", stmt.Pos(), stmtLocation(surfaceKind, identifierName(stmt.View)))
		}
	}
	if !strings.Contains(identifierName(stmt.View), ".") {
		c.addWarning(judgeCheckProtocolAmbiguity, judgeWarningSurfaceViewAmbiguous, "SURFACE target view should be dotted (for example salesperson.view) to avoid ambiguous consumers", stmt.Pos(), stmtLocation(surfaceKind, identifierName(stmt.View)))
	}
}

func (c *protocolCheckCollector) addViolation(checkID int, code, description string, pos Position, location string) {
	c.violations = append(c.violations, normalizeViolation(Violation{
		CheckID:     checkID,
		Code:        code,
		Type:        "protocol_contract",
		Description: description,
		Location:    location,
		Line:        pos.Line,
		Column:      pos.Column,
	}))
}

func (c *protocolCheckCollector) addWarning(checkID int, code, description string, pos Position, location string) {
	c.warnings = append(c.warnings, normalizeWarning(Warning{
		CheckID:     checkID,
		Code:        code,
		Description: description,
		Location:    location,
		Line:        pos.Line,
		Column:      pos.Column,
	}))
}

func objectPayload(expr Expression) (map[string]any, bool) {
	object, ok := expr.(*ObjectLiteralExpr)
	if !ok {
		return nil, false
	}
	out := make(map[string]any, len(object.Fields))
	for _, field := range object.Fields {
		out[field.Key] = literalValue(field.Value)
	}
	return out, true
}

func payloadField(payload map[string]any, key string) (any, bool) {
	value, ok := payload[strings.TrimSpace(key)]
	return value, ok
}

func hasNonEmptyObjectField(payload map[string]any, key string) bool {
	value, ok := payloadField(payload, key)
	if !ok || value == nil {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(value)) != ""
}

func literalValue(expr Expression) any {
	switch value := expr.(type) {
	case *LiteralExpr:
		return value.Value
	case *IdentifierExpr:
		return value.Name
	case *ObjectLiteralExpr:
		out := make(map[string]any, len(value.Fields))
		for _, field := range value.Fields {
			out[field.Key] = literalValue(field.Value)
		}
		return out
	case *ArrayLiteralExpr:
		out := make([]any, 0, len(value.Elements))
		for _, item := range value.Elements {
			out = append(out, literalValue(item))
		}
		return out
	default:
		return nil
	}
}

func isConfidenceValue(value any) bool {
	number, ok := floatValue(value)
	return ok && number >= 0 && number <= 1
}

func stmtLocation(kind, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return kind
	}
	return kind + " " + target
}
