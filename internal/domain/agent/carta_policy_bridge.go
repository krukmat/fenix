package agent

import (
	"fmt"
	"strings"
)

func CartaBudgetToLimits(budget *CartaBudget) map[string]any {
	if budget == nil {
		return nil
	}

	limits := make(map[string]any)
	if budget.DailyTokens > 0 {
		limits["daily_tokens"] = budget.DailyTokens
	}
	if budget.DailyCostUSD > 0 {
		limits["daily_cost_usd"] = budget.DailyCostUSD
	}
	if budget.ExecutionsPerDay > 0 {
		limits["executions_per_day"] = budget.ExecutionsPerDay
	}
	if budget.OnExceed != "" {
		limits["on_exceed"] = budget.OnExceed
	}
	if len(limits) == 0 {
		return nil
	}
	return limits
}

func CartaInvariantsAsPolicyRules(invariants []CartaInvariant) []map[string]any {
	if len(invariants) == 0 {
		return []map[string]any{}
	}

	rules := make([]map[string]any, 0, len(invariants))
	for i, invariant := range invariants {
		statement := strings.TrimSpace(invariant.Statement)
		if statement == "" {
			continue
		}

		effect := "deny"
		if strings.EqualFold(strings.TrimSpace(invariant.Mode), "always") {
			effect = "allow"
		}

		rules = append(rules, map[string]any{
			"id":       fmt.Sprintf("carta_invariant_%d", i+1),
			"resource": "tools",
			"action":   statement,
			"effect":   effect,
			"priority": 1000,
		})
	}

	if len(rules) == 0 {
		return []map[string]any{}
	}
	return rules
}
