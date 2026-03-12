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
	summary := &SpecSummary{
		Blocks: make(map[string]bool),
	}

	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return summary
	}

	lines := strings.Split(source, "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "CONTEXT"):
			summary.Blocks["CONTEXT"] = true
		case strings.HasPrefix(line, "ACTORS"):
			summary.Blocks["ACTORS"] = true
		case strings.HasPrefix(line, "CONSTRAINTS"):
			summary.Blocks["CONSTRAINTS"] = true
		case strings.HasPrefix(line, "BEHAVIOR "):
			summary.Blocks["BEHAVIOR"] = true
			name := strings.TrimSpace(strings.TrimPrefix(line, "BEHAVIOR"))
			if name != "" {
				summary.Behaviors = append(summary.Behaviors, SpecBehavior{
					Name: name,
					Line: i + 1,
				})
			}
		}
	}

	missingBlocks := make([]string, 0, len(specKnownBlocks))
	for _, block := range specKnownBlocks {
		if !summary.Blocks[block] {
			missingBlocks = append(missingBlocks, block)
		}
	}
	if len(missingBlocks) > 0 {
		sort.Strings(missingBlocks)
		summary.Warnings = append(summary.Warnings, Warning{
			Code:        "spec_missing_blocks",
			Description: "missing spec blocks: " + strings.Join(missingBlocks, ", "),
			Location:    "spec_source",
		})
	}

	return summary
}
