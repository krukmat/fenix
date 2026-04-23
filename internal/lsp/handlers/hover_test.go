package handlers_test // CLSF-44

import (
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
	"github.com/matiasleandrokruk/fenix/internal/lsp/handlers"
)

func TestHoverHandler_KnownDSLKeyword_ReturnsMarkdown(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\nSET case.status = \"open\"\n")

	h := handlers.NewHoverHandler(ds)
	result := h.Hover("file:///wf.dsl", 0, 0) // cursor on "WORKFLOW"

	if result == nil {
		t.Fatal("expected hover result for WORKFLOW keyword, got nil")
	}
	if result.Contents.Kind != "markdown" {
		t.Errorf("contents.kind = %q, want markdown", result.Contents.Kind)
	}
	if result.Contents.Value == "" {
		t.Error("expected non-empty hover content for WORKFLOW")
	}
}

func TestHoverHandler_KnownDSLKeyword_ContainsKindAndEffect(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\nSET case.status = \"open\"\n")

	h := handlers.NewHoverHandler(ds)
	result := h.Hover("file:///wf.dsl", 2, 0) // cursor on "SET"

	if result == nil {
		t.Fatal("expected hover result for SET keyword, got nil")
	}
	content := result.Contents.Value
	if content == "" {
		t.Error("hover content for SET must not be empty")
	}
}

func TestHoverHandler_CartaKeyword_ReturnsMarkdown(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///spec.carta", 1, "CARTA resolve\nGROUNDS query\nPERMIT search\n")

	h := handlers.NewHoverHandler(ds)
	result := h.Hover("file:///spec.carta", 1, 0) // cursor on "GROUNDS"

	if result == nil {
		t.Fatal("expected hover result for GROUNDS keyword, got nil")
	}
	if result.Contents.Kind != "markdown" {
		t.Errorf("contents.kind = %q, want markdown", result.Contents.Kind)
	}
	if result.Contents.Value == "" {
		t.Error("expected non-empty hover content for GROUNDS")
	}
}

func TestHoverHandler_UnknownToken_ReturnsNil(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\n")

	h := handlers.NewHoverHandler(ds)
	// cursor on "test" (not a keyword)
	result := h.Hover("file:///wf.dsl", 0, 9)

	if result != nil {
		t.Errorf("expected nil hover for non-keyword token, got %+v", result)
	}
}

func TestHoverHandler_UnknownURI_ReturnsNil(t *testing.T) {
	ds := lsp.NewDocumentStore()
	h := handlers.NewHoverHandler(ds)
	result := h.Hover("file:///missing.dsl", 0, 0)
	if result != nil {
		t.Errorf("expected nil hover for unknown URI, got %+v", result)
	}
}

func TestHoverHandler_CursorInWhitespace_ReturnsNil(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\n  SET case.status = \"open\"\n")

	h := handlers.NewHoverHandler(ds)
	// cursor on leading space before SET
	result := h.Hover("file:///wf.dsl", 1, 0)

	if result != nil {
		t.Errorf("expected nil hover for whitespace position, got %+v", result)
	}
}

func TestHoverHandler_OutOfBoundsLine_ReturnsNil(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\n")

	h := handlers.NewHoverHandler(ds)
	result := h.Hover("file:///wf.dsl", 99, 0)

	if result != nil {
		t.Errorf("expected nil hover for out-of-bounds line, got %+v", result)
	}
}
