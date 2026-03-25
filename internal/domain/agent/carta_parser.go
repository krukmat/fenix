package agent

import (
	"strconv"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

const (
	cartaKeyword                   = "CARTA"
	cartaInvariantModeNever        = "never"
	cartaInvariantModeAlways       = "always"
	cartaOnExceedPause             = "pause"
	cartaOnExceedDegrade           = "degrade"
	cartaOnExceedAbort             = "abort"
	errInvalidStringLiteral        = "invalid string literal"
	errInvalidRateUnit             = "invalid rate unit"
	errRateValueNonNegative        = "rate value must be non-negative"
	errNewlineAfterPermitClause    = "expected newline after PERMIT clause"
	errNewlineAfterDelegateClause  = "expected newline after DELEGATE clause"
	errNewlineAfterBudgetField     = "expected newline after BUDGET field"
)

type CartaParser struct {
	tokens   []Token
	pos      int
	warnings []Warning
}

func NewCartaParser(tokens []Token) *CartaParser {
	return &CartaParser{tokens: tokens}
}

func ParseCarta(source string) (*CartaSummary, error) {
	if !strings.HasPrefix(strings.TrimSpace(source), cartaKeyword+" ") && strings.TrimSpace(source) != cartaKeyword {
		return nil, &ParserError{
			Line:   1,
			Column: 1,
			Reason: "expected CARTA declaration",
			Found:  Token{Type: TokenIllegal, Literal: firstCartaTokenLiteral(source), Line: 1, Column: 1},
		}
	}

	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		return nil, err
	}

	parser := NewCartaParser(tokens)
	summary, err := parser.parseProgram()
	if err != nil {
		return nil, err
	}
	summary.Warnings = parser.Warnings()
	if summary.Grounds == nil {
		parser.addWarning(Token{Line: 1, Column: 1}, "carta_missing_grounds", "Carta has no GROUNDS block")
		summary.Warnings = parser.Warnings()
	}
	return summary, nil
}

func (p *CartaParser) Warnings() []Warning {
	if len(p.warnings) == 0 {
		return nil
	}
	out := make([]Warning, len(p.warnings))
	copy(out, p.warnings)
	return out
}

func (p *CartaParser) parseProgram() (*CartaSummary, error) {
	p.skipNewlines()

	if _, err := p.expect(TokenCarta, "expected CARTA declaration"); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdentifier, "expected Carta program name")
	if err != nil {
		return nil, err
	}
	if err := p.expectNewline("expected newline after CARTA header"); err != nil {
		return nil, err
	}

	summary := &CartaSummary{Name: nameTok.Literal}
	topIndented := p.consumeOptionalIndent()
	if err := p.parseProgramBlocks(summary, topIndented); err != nil {
		return nil, err
	}

	if len(summary.Agents) == 0 {
		return nil, p.errorAt(nameTok, "CARTA must declare at least one AGENT")
	}
	return summary, nil
}

func (p *CartaParser) consumeOptionalIndent() bool {
	if p.current().Type == TokenIndent {
		p.advance()
		return true
	}
	return false
}

func (p *CartaParser) parseProgramBlocks(summary *CartaSummary, topIndented bool) error {
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == TokenEOF {
			break
		}
		if topIndented && current.Type == TokenDedent {
			p.advance()
			break
		}
		if err := p.parseProgramBlock(summary, current); err != nil {
			return err
		}
	}
	return nil
}

func (p *CartaParser) parseProgramBlock(summary *CartaSummary, current Token) error {
	switch current.Type {
	case TokenAgent:
		agent, err := p.parseAgentBlock(summary)
		if err != nil {
			return err
		}
		summary.Agents = append(summary.Agents, *agent)
	case TokenBudget:
		if summary.Budget != nil {
			return p.errorAt(current, "duplicate BUDGET block")
		}
		budget, err := p.parseBudgetBlock()
		if err != nil {
			return err
		}
		summary.Budget = budget
	case TokenSkill:
		return p.errorAt(current, "SKILL blocks are not supported in this tranche")
	default:
		return p.errorAt(current, "unexpected Carta block")
	}
	return nil
}

func (p *CartaParser) parseAgentBlock(summary *CartaSummary) (*CartaAgent, error) {
	if _, err := p.expect(TokenAgent, "expected AGENT"); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdentifier, "expected agent name after AGENT")
	if err != nil {
		return nil, err
	}
	if err := p.expectNewline("expected newline after AGENT header"); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenIndent, "expected indented block after AGENT"); err != nil {
		return nil, err
	}

	agent := &CartaAgent{Name: nameTok.Literal}
	if err := p.parseAgentDirectives(summary); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenDedent, "expected end of AGENT block"); err != nil {
		return nil, err
	}
	return agent, nil
}

func (p *CartaParser) parseAgentDirectives(summary *CartaSummary) error {
	seen := agentBlockSeen{}
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == TokenDedent {
			break
		}
		if current.Type == TokenEOF {
			return p.errorAt(current, "expected end of AGENT block")
		}
		if err := p.parseAgentDirective(summary, &seen, current); err != nil {
			return err
		}
	}
	return nil
}

type agentBlockSeen struct {
	grounds   bool
	delegate  bool
	invariant bool
}

func (p *CartaParser) parseAgentDirective(summary *CartaSummary, seen *agentBlockSeen, current Token) error {
	switch current.Type {
	case TokenGrounds:
		return p.parseAgentGroundsDirective(summary, seen, current)
	case TokenPermit:
		return p.parseAgentPermitDirective(summary)
	case TokenDelegate:
		return p.parseAgentDelegateDirective(summary, seen, current)
	case TokenInvariant:
		return p.parseAgentInvariantDirective(summary, seen, current)
	default:
		return p.errorAt(current, "unexpected AGENT directive")
	}
}

func (p *CartaParser) parseAgentGroundsDirective(summary *CartaSummary, seen *agentBlockSeen, current Token) error {
	if seen.grounds {
		return p.errorAt(current, "duplicate GROUNDS block")
	}
	grounds, err := p.parseGroundsBlock()
	if err != nil {
		return err
	}
	summary.Grounds = grounds
	seen.grounds = true
	return nil
}

func (p *CartaParser) parseAgentPermitDirective(summary *CartaSummary) error {
	permit, err := p.parsePermitBlock()
	if err != nil {
		return err
	}
	summary.Permits = append(summary.Permits, *permit)
	return nil
}

func (p *CartaParser) parseAgentDelegateDirective(summary *CartaSummary, seen *agentBlockSeen, current Token) error {
	if seen.delegate {
		return p.errorAt(current, "duplicate DELEGATE block")
	}
	delegate, err := p.parseDelegateBlock()
	if err != nil {
		return err
	}
	summary.Delegates = append(summary.Delegates, *delegate)
	seen.delegate = true
	return nil
}

func (p *CartaParser) parseAgentInvariantDirective(summary *CartaSummary, seen *agentBlockSeen, current Token) error {
	if seen.invariant {
		return p.errorAt(current, "duplicate INVARIANT block")
	}
	invariants, err := p.parseInvariantBlock()
	if err != nil {
		return err
	}
	summary.Invariants = append(summary.Invariants, invariants...)
	seen.invariant = true
	return nil
}

func (p *CartaParser) parseGroundsBlock() (*CartaGrounds, error) {
	grounds := &CartaGrounds{}
	err := p.parseFieldBlock(TokenGrounds, "GROUNDS", func() error {
		return p.parseGroundsField(grounds)
	})
	if err != nil {
		return nil, err
	}
	return grounds, nil
}

func (p *CartaParser) parseFieldBlock(keyword TokenType, name string, parseFn func() error) error {
	if err := p.expectBlockHeader(keyword, name); err != nil {
		return err
	}
	fieldCount, err := p.iterateBlockFields(name, parseFn)
	if err != nil {
		return err
	}
	if _, err := p.expect(TokenDedent, "expected end of "+name+" block"); err != nil {
		return err
	}
	if fieldCount == 0 {
		return p.errorAt(p.current(), name+" block must contain at least one field")
	}
	return nil
}

func (p *CartaParser) expectBlockHeader(keyword TokenType, name string) error {
	if _, err := p.expect(keyword, "expected "+name); err != nil {
		return err
	}
	if err := p.expectNewline("expected newline after " + name); err != nil {
		return err
	}
	_, err := p.expect(TokenIndent, "expected indented block after "+name)
	return err
}

func (p *CartaParser) iterateBlockFields(name string, parseFn func() error) (int, error) {
	fieldCount := 0
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == TokenDedent {
			break
		}
		if current.Type == TokenEOF {
			return 0, p.errorAt(current, "expected end of "+name+" block")
		}
		if err := parseFn(); err != nil {
			return 0, err
		}
		fieldCount++
	}
	return fieldCount, nil
}

func (p *CartaParser) parsePermitBlock() (*CartaPermit, error) {
	toolName, err := p.expectPermitHeader()
	if err != nil {
		return nil, err
	}

	permit := &CartaPermit{Tool: toolName}
	if p.current().Type != TokenIndent {
		return permit, nil
	}
	if _, err := p.expect(TokenIndent, "expected indented block after PERMIT"); err != nil {
		return nil, err
	}

	clauseCount, err := p.parsePermitClauses(permit)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenDedent, "expected end of PERMIT block"); err != nil {
		return nil, err
	}
	if clauseCount == 0 {
		return nil, p.errorAt(p.current(), "PERMIT block must contain at least one clause")
	}

	return permit, nil
}

func (p *CartaParser) expectPermitHeader() (string, error) {
	if _, err := p.expect(TokenPermit, "expected PERMIT"); err != nil {
		return "", err
	}
	toolTok, err := p.expect(TokenIdentifier, "expected tool name after PERMIT")
	if err != nil {
		return "", err
	}
	if err := p.expectNewline("expected newline after PERMIT header"); err != nil {
		return "", err
	}
	return toolTok.Literal, nil
}

func (p *CartaParser) parsePermitClauses(permit *CartaPermit) (int, error) {
	clauseCount := 0
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == TokenDedent {
			break
		}
		if current.Type == TokenEOF {
			return 0, p.errorAt(current, "expected end of PERMIT block")
		}
		if err := p.parsePermitClause(permit); err != nil {
			return 0, err
		}
		clauseCount++
	}
	return clauseCount, nil
}

func (p *CartaParser) parseDelegateBlock() (*CartaDelegate, error) {
	if err := p.expectDelegateHeader(); err != nil {
		return nil, err
	}

	delegate := &CartaDelegate{}
	clauseCount, err := p.parseDelegateClauses(delegate)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenDedent, "expected end of DELEGATE block"); err != nil {
		return nil, err
	}
	if clauseCount == 0 {
		return nil, p.errorAt(p.current(), "DELEGATE TO HUMAN block must contain at least one clause")
	}
	return delegate, nil
}

func (p *CartaParser) expectDelegateHeader() error {
	if _, err := p.expect(TokenDelegate, "expected DELEGATE"); err != nil {
		return err
	}
	if _, err := p.expect(TokenTo, "expected TO after DELEGATE"); err != nil {
		return err
	}
	if _, err := p.expect(TokenHuman, "expected HUMAN after DELEGATE TO"); err != nil {
		return err
	}
	if err := p.expectNewline("expected newline after DELEGATE TO HUMAN"); err != nil {
		return err
	}
	if _, err := p.expect(TokenIndent, "expected indented block after DELEGATE TO HUMAN"); err != nil {
		return err
	}
	return nil
}

func (p *CartaParser) parseDelegateClauses(delegate *CartaDelegate) (int, error) {
	clauseCount := 0
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == TokenDedent {
			break
		}
		if current.Type == TokenEOF {
			return 0, p.errorAt(current, "expected end of DELEGATE block")
		}
		if err := p.parseDelegateClause(delegate); err != nil {
			return 0, err
		}
		clauseCount++
	}
	return clauseCount, nil
}

func (p *CartaParser) parseInvariantBlock() ([]CartaInvariant, error) {
	if err := p.expectInvariantHeader(); err != nil {
		return nil, err
	}

	invariants, err := p.parseInvariantEntries()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenDedent, "expected end of INVARIANT block"); err != nil {
		return nil, err
	}
	return invariants, nil
}

func (p *CartaParser) expectInvariantHeader() error {
	if _, err := p.expect(TokenInvariant, "expected INVARIANT"); err != nil {
		return err
	}
	if err := p.expectNewline("expected newline after INVARIANT"); err != nil {
		return err
	}
	if _, err := p.expect(TokenIndent, "expected indented block after INVARIANT"); err != nil {
		return err
	}
	return nil
}

func (p *CartaParser) parseInvariantEntries() ([]CartaInvariant, error) {
	invariants := make([]CartaInvariant, 0)
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == TokenDedent {
			break
		}
		if current.Type == TokenEOF {
			return nil, p.errorAt(current, "expected end of INVARIANT block")
		}
		inv, err := p.parseInvariantEntry(current)
		if err != nil {
			return nil, err
		}
		invariants = append(invariants, *inv)
	}
	return invariants, nil
}

func (p *CartaParser) parseInvariantEntry(current Token) (*CartaInvariant, error) {
	mode, err := p.parseInvariantMode(current)
	if err != nil {
		return nil, err
	}
	p.advance()

	if _, err := p.expect(TokenColon, "expected ':' after invariant mode"); err != nil {
		return nil, err
	}
	statementTok, err := p.expect(TokenString, "expected string literal after invariant mode")
	if err != nil {
		return nil, err
	}
	statement, err := strconv.Unquote(statementTok.Literal)
	if err != nil {
		return nil, p.errorAt(statementTok, errInvalidStringLiteral)
	}
	if err := p.expectNewline("expected newline after invariant statement"); err != nil {
		return nil, err
	}
	return &CartaInvariant{Mode: mode, Statement: statement}, nil
}

func (p *CartaParser) parseInvariantMode(current Token) (string, error) {
	switch current.Type {
	case TokenNever:
		return cartaInvariantModeNever, nil
	case TokenAlways:
		return cartaInvariantModeAlways, nil
	default:
		return "", p.errorAt(current, "expected never or always in INVARIANT block")
	}
}

func (p *CartaParser) parseBudgetBlock() (*CartaBudget, error) {
	budget := &CartaBudget{}
	err := p.parseFieldBlock(TokenBudget, "BUDGET", func() error {
		return p.parseBudgetField(budget)
	})
	if err != nil {
		return nil, err
	}
	return budget, nil
}

func (p *CartaParser) parseGroundsField(grounds *CartaGrounds) error {
	fieldTok, err := p.expect(TokenIdentifier, "expected GROUNDS field name")
	if err != nil {
		return err
	}
	if _, err := p.expect(TokenColon, "expected ':' after GROUNDS field"); err != nil {
		return err
	}

	switch fieldTok.Literal {
	case "min_sources":
		return p.parseGroundsFieldMinSources(grounds, fieldTok)
	case "min_confidence":
		return p.parseGroundsFieldMinConfidence(grounds, fieldTok)
	case "max_staleness":
		return p.parseGroundsFieldMaxStaleness(grounds, fieldTok)
	case "types":
		return p.parseGroundsFieldTypes(grounds)
	default:
		p.addWarning(fieldTok, "carta_unknown_grounds_field", "unknown GROUNDS field: "+fieldTok.Literal)
		p.skipUntilNewline()
		return p.expectNewline("expected newline after GROUNDS field")
	}
}

func (p *CartaParser) parseGroundsFieldMinSources(grounds *CartaGrounds, fieldTok Token) error {
	valueTok, err := p.expect(TokenNumber, "expected integer after min_sources")
	if err != nil {
		return err
	}
	value, err := parseCartaInt(valueTok, "invalid min_sources value")
	if err != nil {
		return err
	}
	grounds.MinSources = value
	return p.expectNewline("expected newline after GROUNDS field")
}

func (p *CartaParser) parseGroundsFieldMinConfidence(grounds *CartaGrounds, fieldTok Token) error {
	valueTok, err := p.expect(TokenIdentifier, "expected confidence level after min_confidence")
	if err != nil {
		return err
	}
	confidence, err := parseConfidenceLevel(valueTok)
	if err != nil {
		return err
	}
	grounds.MinConfidence = confidence
	return p.expectNewline("expected newline after GROUNDS field")
}

func (p *CartaParser) parseGroundsFieldMaxStaleness(grounds *CartaGrounds, fieldTok Token) error {
	valueTok, err := p.expect(TokenNumber, "expected integer after max_staleness")
	if err != nil {
		return err
	}
	value, err := parseCartaInt(valueTok, "invalid max_staleness value")
	if err != nil {
		return err
	}
	unitTok, err := p.expect(TokenIdentifier, "expected duration unit after max_staleness")
	if err != nil {
		return err
	}
	if !isCartaDurationUnit(unitTok.Literal) {
		return p.errorAt(unitTok, "invalid duration unit")
	}
	grounds.MaxStaleness = value
	grounds.MaxAgeUnit = unitTok.Literal
	return p.expectNewline("expected newline after GROUNDS field")
}

func (p *CartaParser) parseGroundsFieldTypes(grounds *CartaGrounds) error {
	types, err := p.parseStringList()
	if err != nil {
		return err
	}
	grounds.Types = types
	return p.expectNewline("expected newline after GROUNDS field")
}

func (p *CartaParser) parsePermitClause(permit *CartaPermit) error {
	fieldTok, err := p.expect(TokenIdentifier, "expected PERMIT clause name")
	if err != nil {
		return err
	}
	if _, err := p.expect(TokenColon, "expected ':' after PERMIT clause"); err != nil {
		return err
	}

	switch fieldTok.Literal {
	case "when":
		return p.parsePermitWhen(permit)
	case "rate":
		return p.parsePermitRate(permit, fieldTok)
	case "approval":
		return p.parsePermitApproval(permit, fieldTok)
	default:
		p.addWarning(fieldTok, "carta_unknown_permit_clause", "unknown PERMIT clause: "+fieldTok.Literal)
		p.skipUntilNewline()
		return p.expectNewline(errNewlineAfterPermitClause)
	}
}

func (p *CartaParser) parsePermitWhen(permit *CartaPermit) error {
	permit.When = p.readLineLiteral()
	return p.expectNewline(errNewlineAfterPermitClause)
}

func (p *CartaParser) parsePermitRate(permit *CartaPermit, fieldTok Token) error {
	valueTok, err := p.expect(TokenNumber, "expected integer after rate")
	if err != nil {
		return err
	}
	value, err := parseCartaInt(valueTok, "invalid rate value")
	if err != nil {
		return err
	}
	if value < 0 {
		return p.errorAt(valueTok, errRateValueNonNegative)
	}
	if _, err := p.expect(TokenSlash, "expected '/' in rate clause"); err != nil {
		return err
	}
	unitTok, err := p.expect(TokenIdentifier, "expected rate unit after '/'")
	if err != nil {
		return err
	}
	if !isCartaRateUnit(unitTok.Literal) {
		return p.errorAt(unitTok, errInvalidRateUnit)
	}
	permit.Rate = &CartaRate{Value: value, Unit: unitTok.Literal}
	return p.expectNewline(errNewlineAfterPermitClause)
}

func (p *CartaParser) parsePermitApproval(permit *CartaPermit, fieldTok Token) error {
	modeTok, err := p.expect(TokenIdentifier, "expected approval mode after approval")
	if err != nil {
		return err
	}
	if !isCartaApprovalMode(modeTok.Literal) {
		return p.errorAt(modeTok, "invalid approval mode")
	}
	permit.Approval = &CartaApprovalConfig{Mode: modeTok.Literal}
	return p.expectNewline(errNewlineAfterPermitClause)
}

func (p *CartaParser) parseDelegateClause(delegate *CartaDelegate) error {
	fieldTok, err := p.expect(TokenIdentifier, "expected DELEGATE clause name")
	if err != nil {
		return err
	}
	if _, err := p.expect(TokenColon, "expected ':' after DELEGATE clause"); err != nil {
		return err
	}

	switch fieldTok.Literal {
	case "when":
		return p.parseDelegateWhen(delegate)
	case StepTypeReason:
		return p.parseDelegateReason(delegate, fieldTok)
	case "package":
		return p.parseDelegatePackage(delegate)
	default:
		p.addWarning(fieldTok, "carta_unknown_delegate_clause", "unknown DELEGATE clause: "+fieldTok.Literal)
		p.skipUntilNewline()
		return p.expectNewline(errNewlineAfterDelegateClause)
	}
}

func (p *CartaParser) parseDelegateWhen(delegate *CartaDelegate) error {
	delegate.When = p.readLineLiteral()
	return p.expectNewline(errNewlineAfterDelegateClause)
}

func (p *CartaParser) parseDelegateReason(delegate *CartaDelegate, fieldTok Token) error {
	reasonTok, err := p.expect(TokenString, "expected string after reason")
	if err != nil {
		return err
	}
	reason, err := strconv.Unquote(reasonTok.Literal)
	if err != nil {
		return p.errorAt(reasonTok, errInvalidStringLiteral)
	}
	delegate.Reason = reason
	return p.expectNewline(errNewlineAfterDelegateClause)
}

func (p *CartaParser) parseDelegatePackage(delegate *CartaDelegate) error {
	values, err := p.parseIdentifierList(
		"expected identifier in package list",
		"expected '[' after package",
		"expected ',' or ']' in package list",
		"expected ']' after package list",
	)
	if err != nil {
		return err
	}
	delegate.Package = values
	return p.expectNewline(errNewlineAfterDelegateClause)
}

func (p *CartaParser) parseBudgetField(budget *CartaBudget) error {
	fieldTok, err := p.expect(TokenIdentifier, "expected BUDGET field name")
	if err != nil {
		return err
	}
	if _, err := p.expect(TokenColon, "expected ':' after BUDGET field"); err != nil {
		return err
	}

	switch fieldTok.Literal {
	case "daily_tokens":
		return p.parseBudgetFieldDailyTokens(budget, fieldTok)
	case "daily_cost_usd":
		return p.parseBudgetFieldDailyCostUSD(budget, fieldTok)
	case "executions_per_day":
		return p.parseBudgetFieldExecutionsPerDay(budget, fieldTok)
	case "on_exceed":
		return p.parseBudgetFieldOnExceed(budget, fieldTok)
	default:
		p.addWarning(fieldTok, "carta_unknown_budget_field", "unknown BUDGET field: "+fieldTok.Literal)
		p.skipUntilNewline()
		return p.expectNewline(errNewlineAfterBudgetField)
	}
}

func (p *CartaParser) parseBudgetFieldDailyTokens(budget *CartaBudget, fieldTok Token) error {
	valueTok, err := p.expect(TokenNumber, "expected integer after daily_tokens")
	if err != nil {
		return err
	}
	value, err := parseCartaInt(valueTok, "invalid daily_tokens value")
	if err != nil {
		return err
	}
	budget.DailyTokens = value
	return p.expectNewline(errNewlineAfterBudgetField)
}

func (p *CartaParser) parseBudgetFieldDailyCostUSD(budget *CartaBudget, fieldTok Token) error {
	valueTok, err := p.expect(TokenNumber, "expected number after daily_cost_usd")
	if err != nil {
		return err
	}
	value, err := strconv.ParseFloat(valueTok.Literal, 64)
	if err != nil {
		return p.errorAt(valueTok, "invalid daily_cost_usd value")
	}
	budget.DailyCostUSD = value
	return p.expectNewline(errNewlineAfterBudgetField)
}

func (p *CartaParser) parseBudgetFieldExecutionsPerDay(budget *CartaBudget, fieldTok Token) error {
	valueTok, err := p.expect(TokenNumber, "expected integer after executions_per_day")
	if err != nil {
		return err
	}
	value, err := parseCartaInt(valueTok, "invalid executions_per_day value")
	if err != nil {
		return err
	}
	budget.ExecutionsPerDay = value
	return p.expectNewline(errNewlineAfterBudgetField)
}

func (p *CartaParser) parseBudgetFieldOnExceed(budget *CartaBudget, fieldTok Token) error {
	modeTok, err := p.expect(TokenIdentifier, "expected on_exceed mode")
	if err != nil {
		return err
	}
	if !isCartaOnExceed(modeTok.Literal) {
		return p.errorAt(modeTok, "invalid on_exceed mode")
	}
	budget.OnExceed = modeTok.Literal
	return p.expectNewline(errNewlineAfterBudgetField)
}

func (p *CartaParser) parseStringList() ([]string, error) {
	return p.parseList("expected '[' after types", "expected ']' after types list", "expected ',' or ']' in types list", func() (string, error) {
		return p.parseStringListItem()
	})
}

func (p *CartaParser) parseStringListItem() (string, error) {
	valueTok, err := p.expect(TokenString, "expected string in types list")
	if err != nil {
		return "", err
	}
	value, err := strconv.Unquote(valueTok.Literal)
	if err != nil {
		return "", p.errorAt(valueTok, errInvalidStringLiteral)
	}
	return value, nil
}

func (p *CartaParser) parseIdentifierList(itemErr, openErr, delimErr, closeErr string) ([]string, error) {
	return p.parseList(openErr, closeErr, delimErr, func() (string, error) {
		tok, err := p.expect(TokenIdentifier, itemErr)
		if err != nil {
			return "", err
		}
		return tok.Literal, nil
	})
}

func (p *CartaParser) parseList(openErr, closeErr, delimErr string, itemFn func() (string, error)) ([]string, error) {
	if _, err := p.expect(TokenLBracket, openErr); err != nil {
		return nil, err
	}
	values, err := p.collectListItems(delimErr, itemFn)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenRBracket, closeErr); err != nil {
		return nil, err
	}
	return values, nil
}

func (p *CartaParser) collectListItems(delimErr string, itemFn func() (string, error)) ([]string, error) {
	values := make([]string, 0)
	for p.current().Type != TokenRBracket {
		item, err := itemFn()
		if err != nil {
			return nil, err
		}
		values = append(values, item)
		if err := p.advanceListDelimiter(delimErr); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (p *CartaParser) advanceListDelimiter(delimErr string) error {
	if p.current().Type == TokenComma {
		p.advance()
		return nil
	}
	if p.current().Type != TokenRBracket {
		return p.errorAt(p.current(), delimErr)
	}
	return nil
}

func (p *CartaParser) readLineLiteral() string {
	parts := make([]string, 0)
	for {
		current := p.current()
		if current.Type == TokenNewline || current.Type == TokenEOF || current.Type == TokenDedent {
			break
		}
		parts = append(parts, current.Literal)
		p.advance()
	}
	return strings.Join(parts, " ")
}

func parseConfidenceLevel(tok Token) (knowledge.ConfidenceLevel, error) {
	switch strings.ToLower(tok.Literal) {
	case string(knowledge.ConfidenceLow):
		return knowledge.ConfidenceLow, nil
	case string(knowledge.ConfidenceMedium):
		return knowledge.ConfidenceMedium, nil
	case string(knowledge.ConfidenceHigh):
		return knowledge.ConfidenceHigh, nil
	default:
		return "", &ParserError{
			Line:   tok.Line,
			Column: tok.Column,
			Reason: "invalid confidence level",
			Found:  tok,
		}
	}
}

func isCartaDurationUnit(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "days", waitUnitHours, "minutes":
		return true
	default:
		return false
	}
}

func isCartaRateUnit(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "min", "hour", waitUnitDay:
		return true
	default:
		return false
	}
}

func isCartaApprovalMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "required":
		return true
	default:
		return false
	}
}

func isCartaOnExceed(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case cartaOnExceedPause, cartaOnExceedDegrade, cartaOnExceedAbort:
		return true
	default:
		return false
	}
}

func parseCartaInt(tok Token, reason string) (int, error) {
	if strings.Contains(tok.Literal, ".") {
		return 0, &ParserError{Line: tok.Line, Column: tok.Column, Reason: reason, Found: tok}
	}
	value, err := strconv.Atoi(tok.Literal)
	if err != nil {
		return 0, &ParserError{Line: tok.Line, Column: tok.Column, Reason: reason, Found: tok}
	}
	return value, nil
}

func (p *CartaParser) addWarning(tok Token, code, description string) {
	p.warnings = append(p.warnings, Warning{
		Code:        code,
		Description: description,
		Location:    "line " + strconv.Itoa(tok.Line),
		Line:        tok.Line,
		Column:      tok.Column,
	})
}

func (p *CartaParser) skipUntilNewline() {
	for {
		current := p.current()
		if current.Type == TokenNewline || current.Type == TokenEOF || current.Type == TokenDedent {
			return
		}
		p.advance()
	}
}

func (p *CartaParser) expect(tokenType TokenType, reason string) (Token, error) {
	current := p.current()
	if current.Type != tokenType {
		return Token{}, p.errorAt(current, reason)
	}
	p.advance()
	return current, nil
}

func (p *CartaParser) expectNewline(reason string) error {
	if p.current().Type != TokenNewline {
		return p.errorAt(p.current(), reason)
	}
	p.skipNewlines()
	return nil
}

func (p *CartaParser) skipNewlines() {
	for p.current().Type == TokenNewline {
		p.advance()
	}
}

func (p *CartaParser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *CartaParser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *CartaParser) errorAt(tok Token, reason string) error {
	return &ParserError{
		Line:   tok.Line,
		Column: tok.Column,
		Reason: reason,
		Found:  tok,
	}
}

func firstCartaTokenLiteral(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
