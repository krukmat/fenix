package agent

import workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"

func init() {
	workflowdomain.RegisterCartaBudgetLimitsResolver(resolveCartaBudgetLimits)
	workflowdomain.RegisterCartaInvariantRulesResolver(resolveCartaInvariantRules)
}

func resolveCartaBudgetLimits(source string) (map[string]any, error) {
	if !isCartaSource(source) {
		return nil, nil
	}
	carta, err := ParseCarta(source)
	if err != nil {
		return nil, err
	}
	return CartaBudgetToLimits(carta.Budget), nil
}

func resolveCartaInvariantRules(source string) ([]map[string]any, error) {
	if !isCartaSource(source) {
		return nil, nil
	}
	carta, err := ParseCarta(source)
	if err != nil {
		return nil, err
	}
	return CartaInvariantsAsPolicyRules(carta.Invariants), nil
}
