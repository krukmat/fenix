package handlers // CLSF-44

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
)

// keywordMeta describes a DSL or Carta keyword for hover display.
type keywordMeta struct {
	nodeKind    string
	conformance string
	effect      string
}

// dslKeywordMeta maps DSL v0 keywords to their semantic metadata.
var dslKeywordMeta = map[string]keywordMeta{
	"WORKFLOW": {nodeKind: "workflow", conformance: "safe", effect: "Declares a named workflow scope."},
	"ON":       {nodeKind: "trigger", conformance: "safe", effect: "Binds workflow execution to a domain event."},
	"IF":       {nodeKind: "decision", conformance: "safe", effect: "Guards execution on a boolean condition."},
	"SET":      {nodeKind: "action", conformance: "safe", effect: "Assigns a value to a CRM field."},
	"NOTIFY":   {nodeKind: "action", conformance: "safe", effect: "Sends a notification to a user or channel."},
	"AGENT":    {nodeKind: "action", conformance: "extended", effect: "Delegates execution to a named agent."},
	"WAIT":     {nodeKind: "action", conformance: "safe", effect: "Pauses execution until a condition or timeout."},
	"DISPATCH": {nodeKind: "action", conformance: "extended", effect: "Dispatches a domain event or sub-workflow."},
	"SURFACE":  {nodeKind: "action", conformance: "safe", effect: "Surfaces a UI widget or copilot suggestion."},
}

// cartaKeywordMeta maps Carta spec keywords to their semantic metadata.
var cartaKeywordMeta = map[string]keywordMeta{
	"CARTA":     {nodeKind: "carta", conformance: "safe", effect: "Declares a Carta governance spec for a workflow."},
	"GROUNDS":   {nodeKind: "grounds", conformance: "safe", effect: "Requires retrieval evidence before execution."},
	"PERMIT":    {nodeKind: "permit", conformance: "safe", effect: "Allowlists a tool the agent may call."},
	"DELEGATE":  {nodeKind: "delegate", conformance: "safe", effect: "Routes execution to another named agent."},
	"INVARIANT": {nodeKind: "invariant", conformance: "safe", effect: "Declares a policy invariant that must hold."},
	"BUDGET":    {nodeKind: "budget", conformance: "safe", effect: "Sets cost or token quota for the workflow run."},
}

// HoverHandler provides hover information for DSL v0 and Carta keywords. // CLSF-44
type HoverHandler struct {
	store *lsp.DocumentStore
}

// NewHoverHandler creates a HoverHandler backed by the given store.
func NewHoverHandler(store *lsp.DocumentStore) *HoverHandler {
	return &HoverHandler{store: store}
}

// Hover returns hover content for the given URI at the 0-based line/character position.
// Returns nil when the URI is unknown, position is out of bounds, or the token is not a known keyword.
func (h *HoverHandler) Hover(uri string, line, character int) *lsp.HoverResult {
	doc, ok := h.store.Get(uri)
	if !ok {
		return nil
	}
	token := tokenAt(doc.Text, line, character)
	if token == "" {
		return nil
	}
	meta, found := lookupKeyword(doc.Text, token)
	if !found {
		return nil
	}
	return &lsp.HoverResult{
		Contents: lsp.HoverMarkupContent{
			Kind:  "markdown",
			Value: formatHover(token, meta),
		},
	}
}

// tokenAt extracts the identifier token at the given 0-based line/character in text.
// Returns "" when the character is not inside a word.
func tokenAt(text string, line, character int) string {
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	return tokenInLine(lines[line], character)
}

// tokenInLine extracts the word token at character position within a single line.
func tokenInLine(lineText string, character int) string {
	if character >= len(lineText) || !isWordChar(rune(lineText[character])) {
		return ""
	}
	start := character
	for start > 0 && isWordChar(rune(lineText[start-1])) {
		start--
	}
	end := character
	for end < len(lineText) && isWordChar(rune(lineText[end])) {
		end++
	}
	return lineText[start:end]
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// lookupKeyword returns the metadata for the token in context of the document.
// It selects the DSL or Carta keyword table based on the document's first non-empty line.
func lookupKeyword(text, token string) (keywordMeta, bool) {
	table := dslKeywordMeta
	for _, l := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "CARTA") {
			table = cartaKeywordMeta
		}
		break
	}
	meta, ok := table[token]
	return meta, ok
}

func formatHover(keyword string, meta keywordMeta) string {
	return fmt.Sprintf("**`%s`** — %s\n\n- **Node kind**: `%s`\n- **Conformance**: `%s`\n- **Effect**: %s",
		keyword, meta.effect, meta.nodeKind, meta.conformance, meta.effect)
}
