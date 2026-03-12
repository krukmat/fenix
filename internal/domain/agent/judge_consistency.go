package agent

import (
	"fmt"
	"sort"
	"strings"
)

const judgeCheckBehaviorCoverage = 5

type DSLCoverageSummary struct {
	Labels []DSLCoverageLabel
}

type DSLCoverageLabel struct {
	Kind   string
	Name   string
	Tokens []string
}

func BuildDSLCoverageSummary(program *Program) *DSLCoverageSummary {
	summary := &DSLCoverageSummary{}
	if program == nil || program.Workflow == nil {
		return summary
	}

	workflow := program.Workflow
	summary.Labels = append(summary.Labels,
		newCoverageLabel("workflow", workflow.Name),
		newCoverageLabel("trigger", workflow.Trigger.Event),
	)
	collectStatementCoverage(&summary.Labels, workflow.Body)
	return summary
}

func RunInitialSpecDSLChecks(spec *SpecSummary, program *Program) ([]Violation, []Warning) {
	if spec == nil {
		return nil, nil
	}
	if len(spec.Behaviors) == 0 {
		return nil, nil
	}

	coverage := BuildDSLCoverageSummary(program)
	var violations []Violation

	for _, behavior := range spec.Behaviors {
		if isBehaviorCovered(behavior, coverage) {
			continue
		}
		violations = append(violations, normalizeViolation(Violation{
			CheckID:     judgeCheckBehaviorCoverage,
			Code:        "behavior_no_coverage",
			Type:        "behavior_no_coverage",
			Description: fmt.Sprintf("BEHAVIOR %s has no basic execution coverage in DSL", behavior.Name),
			Location:    fmt.Sprintf("BEHAVIOR %s", behavior.Name),
			Line:        behavior.Line,
		}))
	}

	return violations, nil
}

func collectStatementCoverage(out *[]DSLCoverageLabel, statements []Statement) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *SetStatement:
			*out = append(*out, newCoverageLabel("set", "set "+stmt.Target.Name))
		case *NotifyStatement:
			*out = append(*out, newCoverageLabel("notify", "notify "+stmt.Target.Name))
		case *AgentStatement:
			*out = append(*out, newCoverageLabel("agent", "agent "+stmt.Name.Name))
		case *IfStatement:
			*out = append(*out, newCoverageLabel("if", "if"))
			collectStatementCoverage(out, stmt.Body)
		}
	}
}

func isBehaviorCovered(behavior SpecBehavior, coverage *DSLCoverageSummary) bool {
	behaviorTokens := normalizeCoverageTokens(behavior.Name)
	if len(behaviorTokens) == 0 {
		return true
	}
	for _, label := range coverage.Labels {
		if tokensCoverBehavior(behaviorTokens, label.Tokens) {
			return true
		}
	}
	return false
}

func tokensCoverBehavior(behaviorTokens, labelTokens []string) bool {
	if len(labelTokens) == 0 {
		return false
	}
	overlap := 0
	labelSet := make(map[string]bool, len(labelTokens))
	for _, token := range labelTokens {
		labelSet[token] = true
	}
	for _, token := range behaviorTokens {
		if labelSet[token] {
			overlap++
		}
	}
	if len(behaviorTokens) == 1 {
		return overlap == 1
	}
	return overlap >= 2 || overlap == len(behaviorTokens)
}

func newCoverageLabel(kind, value string) DSLCoverageLabel {
	return DSLCoverageLabel{
		Kind:   kind,
		Name:   value,
		Tokens: normalizeCoverageTokens(value),
	}
}

func normalizeCoverageTokens(value string) []string {
	replacer := strings.NewReplacer(".", "_", "-", "_", ":", "_")
	value = replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})

	stop := map[string]bool{
		"workflow": true,
		"with":     true,
		"agent":    true,
	}

	set := make(map[string]bool)
	for _, part := range parts {
		if part == "" || stop[part] {
			continue
		}
		set[part] = true
	}

	out := make([]string, 0, len(set))
	for token := range set {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}
