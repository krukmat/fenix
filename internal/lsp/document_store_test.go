package lsp_test

import (
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
)

func TestDocumentStore_GetUnknownURIReturnsNotFound(t *testing.T) {
	ds := lsp.NewDocumentStore()
	_, ok := ds.Get("file:///unknown.dsl")
	if ok {
		t.Error("expected not-found for unknown URI")
	}
}

func TestDocumentStore_OpenStoresDocument(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case_created\n  SET status = open\n")

	doc, ok := ds.Get("file:///wf.dsl")
	if !ok {
		t.Fatal("document not found after Open")
	}
	if doc.URI != "file:///wf.dsl" {
		t.Errorf("URI = %q, want file:///wf.dsl", doc.URI)
	}
	if doc.Version != 1 {
		t.Errorf("Version = %d, want 1", doc.Version)
	}
	if doc.Text == "" {
		t.Error("Text must not be empty after Open")
	}
}

func TestDocumentStore_ChangeUpdatesTextAndVersion(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case_created\n  SET status = open\n")

	updated := "WORKFLOW test\nON case_created\n  SET status = closed\n"
	ds.Change("file:///wf.dsl", 2, updated)

	doc, ok := ds.Get("file:///wf.dsl")
	if !ok {
		t.Fatal("document not found after Change")
	}
	if doc.Version != 2 {
		t.Errorf("Version = %d, want 2", doc.Version)
	}
	if doc.Text != updated {
		t.Errorf("Text = %q, want %q", doc.Text, updated)
	}
}

func TestDocumentStore_ChangeOnUnknownURIIsNoOp(t *testing.T) {
	ds := lsp.NewDocumentStore()
	// Should not panic or create a document for an unknown URI.
	ds.Change("file:///missing.dsl", 1, "WORKFLOW x\nON e\n  SET a = b\n")

	_, ok := ds.Get("file:///missing.dsl")
	if ok {
		t.Error("Change on unknown URI must not create a document")
	}
}

func TestDocumentStore_CloseRemovesDocument(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case_created\n  SET status = open\n")
	ds.Close("file:///wf.dsl")

	_, ok := ds.Get("file:///wf.dsl")
	if ok {
		t.Error("document must not exist after Close")
	}
}

func TestDocumentStore_MultipleDocumentsAreIsolated(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///a.dsl", 1, "WORKFLOW a\nON e\n  SET x = 1\n")
	ds.Open("file:///b.dsl", 1, "WORKFLOW b\nON e\n  SET y = 2\n")

	ds.Change("file:///a.dsl", 2, "WORKFLOW a\nON e\n  SET x = 99\n")

	docB, _ := ds.Get("file:///b.dsl")
	if docB.Version != 1 {
		t.Errorf("b.dsl version changed unexpectedly: %d", docB.Version)
	}
}
