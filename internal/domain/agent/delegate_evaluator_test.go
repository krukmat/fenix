package agent

import "testing"

func TestDelegateEvaluator(t *testing.T) {
	t.Run("returns matched when condition is true", func(t *testing.T) {
		evaluator := NewDelegateEvaluator()

		got, err := evaluator.EvaluateDelegate([]CartaDelegate{
			{When: `case.priority == "high"`, Reason: "handoff_high_priority"},
		}, map[string]any{
			"case": map[string]any{"priority": "high"},
		})
		if err != nil {
			t.Fatalf("EvaluateDelegate() error = %v", err)
		}
		if !got.Matched || got.Delegate == nil {
			t.Fatalf("expected matched decision, got %#v", got)
		}
		if got.Delegate.Reason != "handoff_high_priority" {
			t.Fatalf("delegate reason = %q, want handoff_high_priority", got.Delegate.Reason)
		}
	})

	t.Run("returns unmatched when condition is false", func(t *testing.T) {
		evaluator := NewDelegateEvaluator()

		got, err := evaluator.EvaluateDelegate([]CartaDelegate{
			{When: `case.priority == "high"`, Reason: "handoff_high_priority"},
		}, map[string]any{
			"case": map[string]any{"priority": "low"},
		})
		if err != nil {
			t.Fatalf("EvaluateDelegate() error = %v", err)
		}
		if got.Matched {
			t.Fatalf("expected unmatched decision, got %#v", got)
		}
	})

	t.Run("empty when delegates immediately per current runtime contract", func(t *testing.T) {
		evaluator := NewDelegateEvaluator()

		got, err := evaluator.EvaluateDelegate([]CartaDelegate{
			{When: "", Reason: "manual_review"},
		}, map[string]any{
			"case": map[string]any{"priority": "low"},
		})
		if err != nil {
			t.Fatalf("EvaluateDelegate() error = %v", err)
		}
		if !got.Matched || got.Delegate == nil {
			t.Fatalf("expected matched decision, got %#v", got)
		}
		if got.Delegate.Reason != "manual_review" {
			t.Fatalf("delegate reason = %q, want manual_review", got.Delegate.Reason)
		}
	})

	t.Run("returns second matching delegate", func(t *testing.T) {
		evaluator := NewDelegateEvaluator()

		got, err := evaluator.EvaluateDelegate([]CartaDelegate{
			{When: `case.priority == "high"`, Reason: "first"},
			{When: `case.priority == "low"`, Reason: "second"},
		}, map[string]any{
			"case": map[string]any{"priority": "low"},
		})
		if err != nil {
			t.Fatalf("EvaluateDelegate() error = %v", err)
		}
		if !got.Matched || got.Delegate == nil {
			t.Fatalf("expected matched decision, got %#v", got)
		}
		if got.Delegate.Reason != "second" {
			t.Fatalf("delegate reason = %q, want second", got.Delegate.Reason)
		}
	})
}
