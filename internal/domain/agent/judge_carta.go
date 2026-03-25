package agent

import (
	"fmt"
	"strings"
)

const (
	CartaCheckPermit   = 10
	CartaCheckCoverage = 11
	CartaCheckGrounds  = 12
)

func RunCartaPermitChecks(carta *CartaSummary, program *Program) []Violation {
	if carta == nil || program == nil || program.Workflow == nil {
		return nil
	}

	permitted := make(map[string]bool, len(carta.Permits))
	for _, permit := range carta.Permits {
		toolName := strings.ToLower(strings.TrimSpace(permit.Tool))
		if toolName == "" {
			continue
		}
		permitted[toolName] = true
	}

	var violations []Violation
	collectCartaPermitViolations(&violations, program.Workflow.Body, permitted)
	return violations
}

func collectCartaPermitViolations(out *[]Violation, statements []Statement, permitted map[string]bool) {
	for _, stmt := range statements {
		switch node := stmt.(type) {
		case *IfStatement:
			collectCartaPermitViolations(out, node.Body, permitted)
		default:
			toolName := strings.ToLower(strings.TrimSpace(ToolNameForStatement(stmt)))
			if toolName == "" || permitted[toolName] {
				continue
			}
			*out = append(*out, normalizeViolation(Violation{
				CheckID:     CartaCheckPermit,
				Code:        "tool_not_permitted",
				Type:        "tool_not_permitted",
				Description: fmt.Sprintf("DSL tool %s is not permitted by CARTA", toolName),
				Location:    cartaStatementLocation(stmt),
				Line:        stmt.Pos().Line,
				Column:      stmt.Pos().Column,
			}))
		}
	}
}

func cartaStatementLocation(stmt Statement) string {
	switch node := stmt.(type) {
	case *SetStatement:
		return stmtLocation("SET", identifierName(node.Target))
	case *NotifyStatement:
		return stmtLocation("NOTIFY", identifierName(node.Target))
	default:
		return ""
	}
}

func RunCartaCoverageChecks(carta *CartaSummary, spec *SpecSummary) []Violation {
	if carta == nil || spec == nil || len(spec.Behaviors) == 0 {
		return nil
	}
	if len(carta.Delegates) != 0 {
		return nil
	}

	permitLabels := make([]DSLCoverageLabel, 0, len(carta.Permits))
	for _, permit := range carta.Permits {
		toolName := strings.TrimSpace(permit.Tool)
		if toolName == "" {
			continue
		}
		permitLabels = append(permitLabels, newCoverageLabel("permit", toolName))
	}

	var violations []Violation
	for _, behavior := range spec.Behaviors {
		if cartaBehaviorCovered(behavior, permitLabels) {
			continue
		}
		violations = append(violations, normalizeViolation(Violation{
			CheckID:     CartaCheckCoverage,
			Code:        "behavior_no_permit_or_delegate",
			Type:        "behavior_no_permit_or_delegate",
			Description: fmt.Sprintf("BEHAVIOR %s has no PERMIT or DELEGATE coverage in CARTA", behavior.Name),
			Location:    fmt.Sprintf("BEHAVIOR %s", behavior.Name),
			Line:        behavior.Line,
		}))
	}
	return violations
}

func cartaBehaviorCovered(behavior SpecBehavior, permitLabels []DSLCoverageLabel) bool {
	behaviorTokens := normalizeCoverageTokens(behavior.Name)
	if len(behaviorTokens) == 0 {
		return true
	}
	for _, label := range permitLabels {
		if tokensCoverBehavior(behaviorTokens, label.Tokens) {
			return true
		}
	}
	return false
}

func RunCartaGroundsPresenceCheck(carta *CartaSummary) []Warning {
	if carta == nil || carta.Grounds != nil {
		return nil
	}
	return []Warning{normalizeWarning(Warning{
		CheckID:     CartaCheckGrounds,
		Code:        "carta_missing_grounds",
		Description: "Carta has no GROUNDS block",
		Location:    "spec_source",
	})}
}

// RunCartaSpecDSLChecks is introduced in FC-J.1 so the Carta dispatch path can
// compile as soon as ParseCarta is wired into the Judge. The detailed checks are
// added in FC-J.3, FC-J.4, FC-J.5 and composed in FC-J.6.
func RunCartaSpecDSLChecks(carta *CartaSummary, program *Program, spec *SpecSummary) ([]Violation, []Warning) {
	violations := RunCartaPermitChecks(carta, program)
	violations = append(violations, RunCartaCoverageChecks(carta, spec)...)
	warnings := RunCartaGroundsPresenceCheck(carta)
	return violations, warnings
}
