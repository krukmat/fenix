package agent

import (
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

func TestCartaParserGroundsParsesValidBlock(t *testing.T) {
	t.Parallel()

	source := "GROUNDS\n  min_sources: 2\n  min_confidence: medium\n  max_staleness: 30 days\n  types: [\"case\", \"kb_article\"]\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	grounds, err := NewCartaParser(tokens).parseGroundsBlock()
	if err != nil {
		t.Fatalf("parseGroundsBlock() error = %v", err)
	}

	if grounds.MinSources != 2 {
		t.Fatalf("MinSources = %d, want 2", grounds.MinSources)
	}
	if grounds.MinConfidence != knowledge.ConfidenceMedium {
		t.Fatalf("MinConfidence = %s, want %s", grounds.MinConfidence, knowledge.ConfidenceMedium)
	}
	if grounds.MaxStaleness != 30 || grounds.MaxAgeUnit != "days" {
		t.Fatalf("staleness = %d %s, want 30 days", grounds.MaxStaleness, grounds.MaxAgeUnit)
	}
	if len(grounds.Types) != 2 || grounds.Types[0] != "case" || grounds.Types[1] != "kb_article" {
		t.Fatalf("Types = %#v", grounds.Types)
	}
}

func TestCartaParserGroundsAddsWarningForUnknownField(t *testing.T) {
	t.Parallel()

	source := "GROUNDS\n  mystery: 42\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	parser := NewCartaParser(tokens)
	_, err = parser.parseGroundsBlock()
	if err != nil {
		t.Fatalf("parseGroundsBlock() error = %v", err)
	}

	warnings := parser.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(warnings))
	}
	if warnings[0].Code != "carta_unknown_grounds_field" {
		t.Fatalf("Warning.Code = %q, want %q", warnings[0].Code, "carta_unknown_grounds_field")
	}
	if warnings[0].Description != "unknown GROUNDS field: mystery" {
		t.Fatalf("Warning.Description = %q, want %q", warnings[0].Description, "unknown GROUNDS field: mystery")
	}
	if warnings[0].Line != 2 || warnings[0].Column != 3 {
		t.Fatalf("Warning location = %d:%d, want 2:3", warnings[0].Line, warnings[0].Column)
	}
}

func TestCartaParserGroundsRejectsInvalidConfidence(t *testing.T) {
	t.Parallel()

	source := "GROUNDS\n  min_confidence: foobar\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parseGroundsBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}

	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.Reason != "invalid confidence level" {
		t.Fatalf("Reason = %q, want %q", parseErr.Reason, "invalid confidence level")
	}
	if parseErr.Line != 2 || parseErr.Column != 19 {
		t.Fatalf("location = %d:%d, want 2:19", parseErr.Line, parseErr.Column)
	}
}

func TestCartaParserPermitParsesWithoutClauses(t *testing.T) {
	t.Parallel()

	tokens, err := NewCartaLexer().Lex("PERMIT send_reply\n")
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	permit, err := NewCartaParser(tokens).parsePermitBlock()
	if err != nil {
		t.Fatalf("parsePermitBlock() error = %v", err)
	}

	if permit.Tool != "send_reply" {
		t.Fatalf("Tool = %q, want %q", permit.Tool, "send_reply")
	}
	if permit.Rate != nil || permit.Approval != nil || permit.When != "" {
		t.Fatalf("permit = %#v, want empty clauses", permit)
	}
}

func TestCartaParserPermitParsesRateAndApproval(t *testing.T) {
	t.Parallel()

	source := "PERMIT send_reply\n  rate: 10 / hour\n  approval: none\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	permit, err := NewCartaParser(tokens).parsePermitBlock()
	if err != nil {
		t.Fatalf("parsePermitBlock() error = %v", err)
	}

	if permit.Rate == nil || permit.Rate.Value != 10 || permit.Rate.Unit != "hour" {
		t.Fatalf("Rate = %#v, want 10/hour", permit.Rate)
	}
	if permit.Approval == nil || permit.Approval.Mode != "none" {
		t.Fatalf("Approval = %#v, want none", permit.Approval)
	}
}

func TestCartaParserPermitRejectsNegativeRate(t *testing.T) {
	t.Parallel()

	source := "PERMIT send_reply\n  rate: -5 / hour\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parsePermitBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}

	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.Reason != "rate value must be non-negative" {
		t.Fatalf("Reason = %q, want %q", parseErr.Reason, "rate value must be non-negative")
	}
}

func TestCartaParserPermitRejectsInvalidRateUnit(t *testing.T) {
	t.Parallel()

	source := "PERMIT send_reply\n  rate: 5 / week\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parsePermitBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}

	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.Reason != "invalid rate unit" {
		t.Fatalf("Reason = %q, want %q", parseErr.Reason, "invalid rate unit")
	}
}

func TestCartaParserDelegateParsesPackageIdentifiers(t *testing.T) {
	t.Parallel()

	source := "DELEGATE TO HUMAN\n  when: case.tier == \"enterprise\"\n  reason: \"Enterprise cases require review\"\n  package: [evidence_ids, case_summary]\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	delegate, err := NewCartaParser(tokens).parseDelegateBlock()
	if err != nil {
		t.Fatalf("parseDelegateBlock() error = %v", err)
	}

	if len(delegate.Package) != 2 || delegate.Package[0] != "evidence_ids" || delegate.Package[1] != "case_summary" {
		t.Fatalf("Package = %#v", delegate.Package)
	}
}

func TestCartaParserDelegateRejectsMissingBody(t *testing.T) {
	t.Parallel()

	source := "DELEGATE TO HUMAN\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parseDelegateBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}
}

func TestCartaParserInvariantParsesMultipleLines(t *testing.T) {
	t.Parallel()

	source := "INVARIANT\n  never: \"send_pii\"\n  always: \"record_audit\"\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	invariants, err := NewCartaParser(tokens).parseInvariantBlock()
	if err != nil {
		t.Fatalf("parseInvariantBlock() error = %v", err)
	}

	if len(invariants) != 2 {
		t.Fatalf("len(invariants) = %d, want 2", len(invariants))
	}
	if invariants[0].Mode != "never" || invariants[0].Statement != "send_pii" {
		t.Fatalf("invariants[0] = %#v", invariants[0])
	}
	if invariants[1].Mode != "always" || invariants[1].Statement != "record_audit" {
		t.Fatalf("invariants[1] = %#v", invariants[1])
	}
}

func TestCartaParserInvariantRejectsMissingString(t *testing.T) {
	t.Parallel()

	source := "INVARIANT\n  never: alert_user\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parseInvariantBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}
}

func TestCartaParserBudgetParsesPartialBudget(t *testing.T) {
	t.Parallel()

	source := "BUDGET\n  daily_tokens: 50000\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	budget, err := NewCartaParser(tokens).parseBudgetBlock()
	if err != nil {
		t.Fatalf("parseBudgetBlock() error = %v", err)
	}

	if budget.DailyTokens != 50000 {
		t.Fatalf("DailyTokens = %d, want 50000", budget.DailyTokens)
	}
	if budget.DailyCostUSD != 0 || budget.ExecutionsPerDay != 0 || budget.OnExceed != "" {
		t.Fatalf("budget = %#v, want partial zero-value budget", budget)
	}
}

func TestCartaParserBudgetRejectsInvalidOnExceed(t *testing.T) {
	t.Parallel()

	source := "BUDGET\n  on_exceed: invalid\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parseBudgetBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}
}

func TestParseCartaParsesFullProgram(t *testing.T) {
	t.Parallel()

	source := `CARTA resolve_support_case
BUDGET
  daily_tokens: 50000
  daily_cost_usd: 5.00
  executions_per_day: 100
  on_exceed: pause
AGENT search_knowledge
  GROUNDS
    min_sources: 2
    min_confidence: medium
    max_staleness: 30 days
    types: ["case", "kb_article"]
  PERMIT send_reply
    rate: 10 / hour
    approval: none
  DELEGATE TO HUMAN
    when: case.tier == "enterprise"
    reason: "Enterprise review"
    package: [evidence_ids, case_summary]
  INVARIANT
    never: "send_pii"`

	summary, err := ParseCarta(source)
	if err != nil {
		t.Fatalf("ParseCarta() error = %v", err)
	}

	if summary.Name != "resolve_support_case" {
		t.Fatalf("Name = %q, want %q", summary.Name, "resolve_support_case")
	}
	if len(summary.Agents) != 1 || summary.Agents[0].Name != "search_knowledge" {
		t.Fatalf("Agents = %#v", summary.Agents)
	}
	if summary.Grounds == nil || summary.Grounds.MinSources != 2 {
		t.Fatalf("Grounds = %#v", summary.Grounds)
	}
	if len(summary.Permits) != 1 || summary.Permits[0].Tool != "send_reply" {
		t.Fatalf("Permits = %#v", summary.Permits)
	}
	if len(summary.Delegates) != 1 || summary.Delegates[0].Reason != "Enterprise review" {
		t.Fatalf("Delegates = %#v", summary.Delegates)
	}
	if len(summary.Invariants) != 1 || summary.Invariants[0].Mode != "never" {
		t.Fatalf("Invariants = %#v", summary.Invariants)
	}
	if summary.Budget == nil || summary.Budget.OnExceed != "pause" {
		t.Fatalf("Budget = %#v", summary.Budget)
	}
}

func TestParseCartaRejectsMissingHeader(t *testing.T) {
	t.Parallel()

	_, err := ParseCarta("AGENT x\n  GROUNDS\n    min_sources: 2\n")
	if err == nil {
		t.Fatal("expected parser error")
	}
}

func TestParseCartaRejectsDuplicateGrounds(t *testing.T) {
	t.Parallel()

	source := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  GROUNDS
    min_sources: 3`

	_, err := ParseCarta(source)
	if err == nil {
		t.Fatal("expected parser error")
	}
}

func TestParseCartaWarnsWhenGroundsMissing(t *testing.T) {
	t.Parallel()

	source := `CARTA resolve_support_case
AGENT search_knowledge
  PERMIT send_reply`

	summary, err := ParseCarta(source)
	if err != nil {
		t.Fatalf("ParseCarta() error = %v", err)
	}
	if summary.Grounds != nil {
		t.Fatalf("Grounds = %#v, want nil", summary.Grounds)
	}
	if len(summary.Warnings) == 0 {
		t.Fatal("expected parser warning for missing grounds")
	}
}

func TestParseCartaRejectsUnsupportedSkillBlock(t *testing.T) {
	t.Parallel()

	source := `CARTA resolve_support_case
SKILL triage
AGENT search_knowledge
  GROUNDS
    min_sources: 2`

	_, err := ParseCarta(source)
	if err == nil {
		t.Fatal("expected parser error")
	}

	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.Reason != "SKILL blocks are not supported in this tranche" {
		t.Fatalf("Reason = %q, want SKILL unsupported error", parseErr.Reason)
	}
}

func TestParseCartaRejectsCartaWithoutAgent(t *testing.T) {
	t.Parallel()

	source := `CARTA resolve_support_case
BUDGET
  daily_tokens: 50000`

	_, err := ParseCarta(source)
	if err == nil {
		t.Fatal("expected parser error")
	}

	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.Reason != "CARTA must declare at least one AGENT" {
		t.Fatalf("Reason = %q, want missing agent error", parseErr.Reason)
	}
}

func TestParseCartaParsesMultipleAgents(t *testing.T) {
	t.Parallel()

	source := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
AGENT send_reply
  PERMIT send_reply`

	summary, err := ParseCarta(source)
	if err != nil {
		t.Fatalf("ParseCarta() error = %v", err)
	}
	if len(summary.Agents) != 2 {
		t.Fatalf("len(Agents) = %d, want 2", len(summary.Agents))
	}
	if summary.Agents[0].Name != "search_knowledge" || summary.Agents[1].Name != "send_reply" {
		t.Fatalf("Agents = %#v", summary.Agents)
	}
	if len(summary.Permits) != 1 || summary.Permits[0].Tool != "send_reply" {
		t.Fatalf("Permits = %#v", summary.Permits)
	}
}

func TestCartaParserBudgetParsesAllFields(t *testing.T) {
	t.Parallel()

	source := "BUDGET\n  daily_tokens: 50000\n  daily_cost_usd: 5.25\n  executions_per_day: 100\n  on_exceed: degrade\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	budget, err := NewCartaParser(tokens).parseBudgetBlock()
	if err != nil {
		t.Fatalf("parseBudgetBlock() error = %v", err)
	}

	if budget.DailyTokens != 50000 || budget.DailyCostUSD != 5.25 || budget.ExecutionsPerDay != 100 || budget.OnExceed != "degrade" {
		t.Fatalf("budget = %#v", budget)
	}
}

func TestCartaParserDelegateRejectsUnquotedReason(t *testing.T) {
	t.Parallel()

	source := "DELEGATE TO HUMAN\n  reason: enterprise_review\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	_, err = NewCartaParser(tokens).parseDelegateBlock()
	if err == nil {
		t.Fatal("expected parser error")
	}
}
