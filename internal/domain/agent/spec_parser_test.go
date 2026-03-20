package agent

import "testing"

func TestParsePartialSpec_ExtractsBehaviorsAndBlocks(t *testing.T) {
	t.Parallel()

	spec := `CONTEXT
  system = crm

ACTORS
  admin

BEHAVIOR verify_workflow
  GIVEN a workflow

BEHAVIOR verify_workflow_no_spec
  GIVEN no spec

CONSTRAINTS
  one active per name`

	summary := ParsePartialSpec(spec)
	if !summary.Blocks["CONTEXT"] || !summary.Blocks["ACTORS"] || !summary.Blocks["BEHAVIOR"] || !summary.Blocks["CONSTRAINTS"] {
		t.Fatalf("Blocks = %#v", summary.Blocks)
	}
	if len(summary.Behaviors) != 2 {
		t.Fatalf("len(Behaviors) = %d, want 2", len(summary.Behaviors))
	}
	if summary.Behaviors[0].Name != "verify_workflow" {
		t.Fatalf("behavior[0] = %#v", summary.Behaviors[0])
	}
	if len(summary.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", summary.Warnings)
	}
}

func TestParsePartialSpec_AddsWarningForMissingBlocks(t *testing.T) {
	t.Parallel()

	spec := `CONTEXT
  system = crm

ACTORS
  admin`

	summary := ParsePartialSpec(spec)
	if len(summary.Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(summary.Warnings))
	}
	got := summary.Warnings[0]
	if got.Code != "spec_missing_blocks" {
		t.Fatalf("Code = %q, want spec_missing_blocks", got.Code)
	}
	if got.Description == "" {
		t.Fatal("expected warning description")
	}
}

func TestParsePartialSpec_EmptySpecReturnsEmptySummary(t *testing.T) {
	t.Parallel()

	summary := ParsePartialSpec("   ")
	if len(summary.Blocks) != 0 {
		t.Fatalf("Blocks = %#v, want empty", summary.Blocks)
	}
	if len(summary.Behaviors) != 0 {
		t.Fatalf("Behaviors = %#v, want empty", summary.Behaviors)
	}
	if len(summary.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want empty", summary.Warnings)
	}
}
