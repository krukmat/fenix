package agent

import "testing"

func TestCartaBudgetToLimits(t *testing.T) {
	t.Run("returns nil for nil budget", func(t *testing.T) {
		if got := CartaBudgetToLimits(nil); got != nil {
			t.Fatalf("expected nil map, got %#v", got)
		}
	})

	t.Run("returns nil for empty budget", func(t *testing.T) {
		if got := CartaBudgetToLimits(&CartaBudget{}); got != nil {
			t.Fatalf("expected nil map, got %#v", got)
		}
	})

	t.Run("maps populated budget fields", func(t *testing.T) {
		got := CartaBudgetToLimits(&CartaBudget{
			DailyTokens:      50000,
			DailyCostUSD:     5.0,
			ExecutionsPerDay: 12,
			OnExceed:         "pause",
		})

		if got["daily_tokens"] != 50000 {
			t.Fatalf("expected daily_tokens=50000, got %#v", got["daily_tokens"])
		}
		if got["daily_cost_usd"] != 5.0 {
			t.Fatalf("expected daily_cost_usd=5.0, got %#v", got["daily_cost_usd"])
		}
		if got["executions_per_day"] != 12 {
			t.Fatalf("expected executions_per_day=12, got %#v", got["executions_per_day"])
		}
		if got["on_exceed"] != "pause" {
			t.Fatalf("expected on_exceed=pause, got %#v", got["on_exceed"])
		}
	})

	t.Run("omits zero value fields", func(t *testing.T) {
		got := CartaBudgetToLimits(&CartaBudget{
			DailyCostUSD: 2.5,
		})

		if len(got) != 1 {
			t.Fatalf("expected one populated key, got %#v", got)
		}
		if got["daily_cost_usd"] != 2.5 {
			t.Fatalf("expected daily_cost_usd=2.5, got %#v", got["daily_cost_usd"])
		}
		if _, ok := got["daily_tokens"]; ok {
			t.Fatalf("expected daily_tokens to be omitted, got %#v", got["daily_tokens"])
		}
		if _, ok := got["executions_per_day"]; ok {
			t.Fatalf("expected executions_per_day to be omitted, got %#v", got["executions_per_day"])
		}
	})
}

func TestCartaInvariantsAsPolicyRules(t *testing.T) {
	t.Run("returns empty slice for empty input", func(t *testing.T) {
		got := CartaInvariantsAsPolicyRules(nil)
		if len(got) != 0 {
			t.Fatalf("expected empty slice, got %#v", got)
		}
	})

	t.Run("maps never invariants to deny rules", func(t *testing.T) {
		got := CartaInvariantsAsPolicyRules([]CartaInvariant{
			{Mode: "never", Statement: "send_pii"},
			{Mode: "never", Statement: "delete_case"},
		})

		if len(got) != 2 {
			t.Fatalf("expected 2 rules, got %d", len(got))
		}

		for i, action := range []string{"send_pii", "delete_case"} {
			if got[i]["action"] != action {
				t.Fatalf("expected action=%q, got %#v", action, got[i]["action"])
			}
			if got[i]["effect"] != "deny" {
				t.Fatalf("expected deny effect, got %#v", got[i]["effect"])
			}
			if got[i]["priority"] != 1000 {
				t.Fatalf("expected priority=1000, got %#v", got[i]["priority"])
			}
			if got[i]["resource"] != "tools" {
				t.Fatalf("expected resource=tools, got %#v", got[i]["resource"])
			}
		}
	})

	t.Run("maps always invariants to allow rules", func(t *testing.T) {
		got := CartaInvariantsAsPolicyRules([]CartaInvariant{
			{Mode: "always", Statement: "record_audit"},
		})

		if len(got) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(got))
		}
		if got[0]["action"] != "record_audit" {
			t.Fatalf("expected action=record_audit, got %#v", got[0]["action"])
		}
		if got[0]["effect"] != "allow" {
			t.Fatalf("expected allow effect, got %#v", got[0]["effect"])
		}
		if got[0]["priority"] != 1000 {
			t.Fatalf("expected priority=1000, got %#v", got[0]["priority"])
		}
	})
}
