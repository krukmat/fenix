package agent

import (
	"errors"
	"strings"
)

type ConformanceProfile string

const (
	ConformanceProfileSafe     ConformanceProfile = "safe"
	ConformanceProfileExtended ConformanceProfile = "extended"
	ConformanceProfileInvalid  ConformanceProfile = "invalid"
)

type ConformanceSeverity string

const (
	ConformanceSeverityInfo    ConformanceSeverity = "info"
	ConformanceSeverityWarning ConformanceSeverity = "warning"
	ConformanceSeverityError   ConformanceSeverity = "error"
)

type ConformanceResult struct {
	Profile ConformanceProfile     `json:"profile"`
	Details []ConformanceDetail    `json:"details"`
	Graph   *WorkflowSemanticGraph `json:"graph,omitempty"`
}

type ConformanceDetail struct {
	Code     string              `json:"code"`
	Severity ConformanceSeverity `json:"severity"`
	Message  string              `json:"message"`
	Line     int                 `json:"line,omitempty"`
	Column   int                 `json:"column,omitempty"`
}

func EvaluateWorkflowConformance(dslSource string, specSource string) ConformanceResult {
	program, err := ParseAndValidateDSL(dslSource)
	if err != nil {
		return invalidConformance("invalid_dsl", err.Error(), errPosition(err))
	}

	result := ConformanceResult{Profile: ConformanceProfileSafe, Details: []ConformanceDetail{}}
	var carta *CartaSummary
	trimmedSpec := strings.TrimSpace(specSource)
	switch {
	case trimmedSpec == "":
		result.Details = append(result.Details, ConformanceDetail{
			Code:     "missing_spec_source",
			Severity: ConformanceSeverityWarning,
			Message:  "missing spec_source is compatible but has no Carta graph nodes",
		})
	case strings.HasPrefix(trimmedSpec, cartaKeyword+" ") || trimmedSpec == cartaKeyword:
		parsed, parseErr := ParseCarta(specSource)
		if parseErr != nil {
			return invalidConformance("invalid_carta", parseErr.Error(), errPosition(parseErr))
		}
		carta = parsed
	default:
		result.Details = append(result.Details, ConformanceDetail{
			Code:     "legacy_spec_source",
			Severity: ConformanceSeverityInfo,
			Message:  "legacy free-form spec_source is compatible but has no Carta graph nodes",
		})
	}

	result.Graph = BuildWorkflowSemanticGraphWithCarta(program, carta)
	result.applyGraphConformance()
	return result
}

func EvaluateGraphConformance(graph *WorkflowSemanticGraph) ConformanceResult {
	result := ConformanceResult{
		Profile: ConformanceProfileSafe,
		Details: []ConformanceDetail{},
		Graph:   graph,
	}
	result.applyGraphConformance()
	return result
}

func (r *ConformanceResult) applyGraphConformance() {
	if r == nil {
		return
	}
	for _, node := range safeGraphNodes(r.Graph) {
		if isSupportedConformanceNode(node) {
			continue
		}
		r.Profile = ConformanceProfileExtended
		r.Details = append(r.Details, ConformanceDetail{
			Code:     "unsupported_semantic_node",
			Severity: ConformanceSeverityWarning,
			Message:  "semantic node is outside the stable Wave 0-4 tooling contract: " + string(node.Kind),
		})
	}
}

func isSupportedConformanceNode(node WorkflowSemanticNode) bool {
	switch node.Kind {
	case SemanticNodeWorkflow,
		SemanticNodeTrigger,
		SemanticNodeAction,
		SemanticNodeDecision,
		SemanticNodeGrounds,
		SemanticNodePermit,
		SemanticNodeDelegate,
		SemanticNodeInvariant,
		SemanticNodeBudget:
		return true
	default:
		return false
	}
}

func invalidConformance(code string, message string, position Position) ConformanceResult {
	detail := ConformanceDetail{
		Code:     code,
		Severity: ConformanceSeverityError,
		Message:  strings.TrimSpace(message),
		Line:     position.Line,
		Column:   position.Column,
	}
	return ConformanceResult{
		Profile: ConformanceProfileInvalid,
		Details: []ConformanceDetail{detail},
	}
}

func errPosition(err error) Position {
	var parserErr *ParserError
	if errors.As(err, &parserErr) {
		return Position{Line: parserErr.Line, Column: parserErr.Column}
	}
	var typed *DSLValidationError
	if errors.As(err, &typed) {
		return typed.Position
	}
	return Position{}
}

func safeGraphNodes(graph *WorkflowSemanticGraph) []WorkflowSemanticNode {
	if graph == nil {
		return nil
	}
	return graph.Nodes
}
