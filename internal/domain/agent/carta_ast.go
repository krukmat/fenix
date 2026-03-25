package agent

import "github.com/matiasleandrokruk/fenix/internal/domain/knowledge"

// CartaSummary is flat for the current MVP. Carta directives are aggregated at
// the summary level because downstream task specs consume carta.Grounds,
// carta.Permits, carta.Delegates and carta.Invariants directly.
type CartaSummary struct {
	Name       string
	Agents     []CartaAgent
	Grounds    *CartaGrounds
	Permits    []CartaPermit
	Delegates  []CartaDelegate
	Invariants []CartaInvariant
	Budget     *CartaBudget
	Warnings   []Warning
}

type CartaAgent struct {
	Name string
}

type CartaGrounds struct {
	MinSources    int
	MinConfidence knowledge.ConfidenceLevel
	MaxStaleness  int
	MaxAgeUnit    string
	Types         []string
}

type CartaPermit struct {
	Tool     string
	When     string
	Rate     *CartaRate
	Approval *CartaApprovalConfig
}

type CartaRate struct {
	Value int
	Unit  string
}

type CartaApprovalConfig struct {
	Mode string
}

type CartaDelegate struct {
	When    string
	Reason  string
	Package []string
}

type CartaInvariant struct {
	Mode      string
	Statement string
}

type CartaBudget struct {
	DailyTokens      int
	DailyCostUSD     float64
	ExecutionsPerDay int
	OnExceed         string
}
