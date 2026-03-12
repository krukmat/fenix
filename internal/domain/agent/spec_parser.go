package agent

import (
	"sort"
	"strings"
)

type SpecSummary struct {
	Blocks    map[string]bool
	Behaviors []SpecBehavior
	Warnings  []Warning
}

type SpecBehavior struct {
	Name string
	Line int
}

var specKnownBlocks = []string{
	"CONTEXT",
	"ACTORS",
	"BEHAVIOR",
	"CONSTRAINTS",
}

func ParsePartialSpec(source string) *SpecSummary {
	summary := newSpecSummary()
	if strings.TrimSpace(source) == "" {
		return summary
	}

	parseSpecLines(summary, strings.Split(source, "\n"))
	appendMissingBlockWarning(summary)
	return summary
}

func newSpecSummary() *SpecSummary {
	return &SpecSummary{Blocks: make(map[string]bool)}
}

func parseSpecLines(summary *SpecSummary, lines []string) {
	for i, raw := range lines {
		parseSpecLine(summary, strings.TrimSpace(raw), i+1)
	}
}

func parseSpecLine(summary *SpecSummary, line string, lineNumber int) {
	if line == "" {
		return
	}
	if markNonBehaviorBlock(summary, line) {
		return
	}
	appendBehaviorFromLine(summary, line, lineNumber)
}

func markNonBehaviorBlock(summary *SpecSummary, line string) bool {
	for _, block := range []string{"CONTEXT", "ACTORS", "CONSTRAINTS"} {
		if strings.HasPrefix(line, block) {
			summary.Blocks[block] = true
			return true
		}
	}
	return false
}

func appendBehaviorFromLine(summary *SpecSummary, line string, lineNumber int) {
	if !strings.HasPrefix(line, "BEHAVIOR ") {
		return
	}
	summary.Blocks["BEHAVIOR"] = true
	name := strings.TrimSpace(strings.TrimPrefix(line, "BEHAVIOR"))
	if name == "" {
		return
	}
	summary.Behaviors = append(summary.Behaviors, SpecBehavior{Name: name, Line: lineNumber})
}

func appendMissingBlockWarning(summary *SpecSummary) {
	missingBlocks := missingSpecBlocks(summary)
	if len(missingBlocks) == 0 {
		return
	}
	summary.Warnings = append(summary.Warnings, Warning{
		Code:        "spec_missing_blocks",
		Description: "missing spec blocks: " + strings.Join(missingBlocks, ", "),
		Location:    "spec_source",
	})
}

func missingSpecBlocks(summary *SpecSummary) []string {
	missingBlocks := make([]string, 0, len(specKnownBlocks))
	for _, block := range specKnownBlocks {
		if !summary.Blocks[block] {
			missingBlocks = append(missingBlocks, block)
		}
	}
	sort.Strings(missingBlocks)
	return missingBlocks
}
