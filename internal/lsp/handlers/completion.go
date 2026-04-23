package handlers // CLSF-43

import (
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
)

// CompletionKindKeyword is the LSP CompletionItemKind value for keywords.
const CompletionKindKeyword = 14

// dslKeywords are the stable DSL v0 statement keywords. // CLSF-43
var dslKeywords = []string{
	"WORKFLOW", "ON", "IF", "SET", "NOTIFY", "AGENT", "WAIT", "DISPATCH", "SURFACE",
}

// cartaKeywords are the Carta spec block keywords. // CLSF-43
var cartaKeywords = []string{
	"CARTA", "AGENT", "GROUNDS", "PERMIT", "DELEGATE", "INVARIANT", "BUDGET",
}

// CompletionHandler provides context-aware keyword completions from the DocumentStore. // CLSF-43
type CompletionHandler struct {
	store *lsp.DocumentStore
}

// NewCompletionHandler creates a CompletionHandler backed by the given store.
func NewCompletionHandler(store *lsp.DocumentStore) *CompletionHandler {
	return &CompletionHandler{store: store}
}

// Complete returns completion items for the given URI at the 0-based line/character position.
// Returns an empty slice when the URI is unknown or no keywords match the prefix.
func (h *CompletionHandler) Complete(uri string, line, character int) []lsp.CompletionItem {
	doc, ok := h.store.Get(uri)
	if !ok {
		return []lsp.CompletionItem{}
	}
	prefix := linePrefix(doc.Text, line, character)
	keywords := keywordsForDoc(doc.Text)
	return filterByPrefix(keywords, prefix)
}

// linePrefix returns the text on the given 0-based line up to the given character.
func linePrefix(text string, line, character int) string {
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	lineText := lines[line]
	if character > len(lineText) {
		character = len(lineText)
	}
	return strings.TrimLeft(lineText[:character], " \t")
}

// keywordsForDoc returns the keyword set appropriate for the document content.
// A document whose first non-empty line starts with "CARTA" uses Carta keywords.
func keywordsForDoc(text string) []string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "CARTA") {
			return cartaKeywords
		}
		return dslKeywords
	}
	return dslKeywords
}

// filterByPrefix returns items whose label starts with prefix (case-sensitive).
// An empty prefix returns all keywords.
func filterByPrefix(keywords []string, prefix string) []lsp.CompletionItem {
	items := make([]lsp.CompletionItem, 0, len(keywords))
	for _, kw := range keywords {
		if strings.HasPrefix(kw, prefix) {
			items = append(items, lsp.CompletionItem{Label: kw, Kind: CompletionKindKeyword})
		}
	}
	return items
}
