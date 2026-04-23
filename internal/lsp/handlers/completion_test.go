package handlers_test

import (
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
	"github.com/matiasleandrokruk/fenix/internal/lsp/handlers"
)

func TestCompletionHandler_EmptyLine_ReturnsDSLKeywords(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\n")

	h := handlers.NewCompletionHandler(ds)
	items := h.Complete("file:///wf.dsl", 2, 0) // line 3, col 0 (0-based)

	assertContainsLabel(t, items, "SET")
	assertContainsLabel(t, items, "IF")
	assertContainsLabel(t, items, "NOTIFY")
	assertContainsLabel(t, items, "AGENT")
}

func TestCompletionHandler_PrefixFilter_ReturnsMatchingKeywords(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\nSE")

	h := handlers.NewCompletionHandler(ds)
	items := h.Complete("file:///wf.dsl", 2, 2) // cursor after "SE"

	assertContainsLabel(t, items, "SET")
	for _, item := range items {
		if item.Label != "SET" {
			t.Errorf("unexpected item %q with prefix SE", item.Label)
		}
	}
}

func TestCompletionHandler_CartaDoc_ReturnsCartaKeywords(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///spec.carta", 1, "CARTA resolve\nAGENT search\n")

	h := handlers.NewCompletionHandler(ds)
	items := h.Complete("file:///spec.carta", 2, 0)

	assertContainsLabel(t, items, "GROUNDS")
	assertContainsLabel(t, items, "PERMIT")
	assertContainsLabel(t, items, "DELEGATE")
	assertContainsLabel(t, items, "INVARIANT")
	assertContainsLabel(t, items, "BUDGET")
}

func TestCompletionHandler_AllItemsAreKeywordKind(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\n")

	h := handlers.NewCompletionHandler(ds)
	items := h.Complete("file:///wf.dsl", 2, 0)

	for _, item := range items {
		if item.Kind != handlers.CompletionKindKeyword {
			t.Errorf("item %q has kind %d, want %d (Keyword)", item.Label, item.Kind, handlers.CompletionKindKeyword)
		}
	}
}

func TestCompletionHandler_UnknownURI_ReturnsEmpty(t *testing.T) {
	ds := lsp.NewDocumentStore()
	h := handlers.NewCompletionHandler(ds)
	items := h.Complete("file:///missing.dsl", 0, 0)
	if len(items) != 0 {
		t.Errorf("expected empty completions for unknown URI, got %d", len(items))
	}
}

func TestCompletionHandler_NoPrefixMatch_ReturnsEmpty(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\nZZZZ")

	h := handlers.NewCompletionHandler(ds)
	items := h.Complete("file:///wf.dsl", 2, 4) // cursor after "ZZZZ"

	if len(items) != 0 {
		t.Errorf("expected no completions for unknown prefix ZZZZ, got %v", items)
	}
}

func assertContainsLabel(t *testing.T, items []lsp.CompletionItem, label string) {
	t.Helper()
	for _, item := range items {
		if item.Label == label {
			return
		}
	}
	t.Errorf("completion items %v do not contain %q", labelSlice(items), label)
}

func labelSlice(items []lsp.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}
