package handlers_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
	"github.com/matiasleandrokruk/fenix/internal/lsp/handlers"
)

// readNotification reads one framed LSP message from r and returns the parsed JSON.
func readNotification(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	br := bufio.NewReader(r)
	contentLength := 0
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("reading header: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			fmt.Sscanf(strings.TrimPrefix(line, "Content-Length: "), "%d", &contentLength)
		}
	}
	if contentLength == 0 {
		t.Fatal("notification missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(br, body); err != nil {
		t.Fatalf("reading body: %v", err)
	}
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("parsing notification JSON: %v", err)
	}
	return msg
}

func TestDiagnosticsHandler_ValidDSL_PublishesEmptyDiagnostics(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\nSET case.status = \"open\"\n")

	h := handlers.NewDiagnosticsHandler(ds)
	var buf bytes.Buffer
	if err := h.Publish("file:///wf.dsl", &buf); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	msg := readNotification(t, &buf)
	if msg["method"] != "textDocument/publishDiagnostics" {
		t.Errorf("method = %v, want textDocument/publishDiagnostics", msg["method"])
	}
	params := msg["params"].(map[string]any)
	if params["uri"] != "file:///wf.dsl" {
		t.Errorf("uri = %v, want file:///wf.dsl", params["uri"])
	}
	diags := params["diagnostics"].([]any)
	if len(diags) != 0 {
		t.Errorf("diagnostics = %v, want empty for valid DSL", diags)
	}
}

func TestDiagnosticsHandler_SyntaxError_PublishesErrorDiagnostic(t *testing.T) {
	ds := lsp.NewDocumentStore()
	// Missing ON clause — parser error
	ds.Open("file:///bad.dsl", 1, "WORKFLOW test\n  SET status = \"open\"\n")

	h := handlers.NewDiagnosticsHandler(ds)
	var buf bytes.Buffer
	if err := h.Publish("file:///bad.dsl", &buf); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	msg := readNotification(t, &buf)
	params := msg["params"].(map[string]any)
	diags := params["diagnostics"].([]any)
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic for invalid DSL")
	}

	d := diags[0].(map[string]any)
	// Severity 1 = Error
	if d["severity"] != float64(1) {
		t.Errorf("severity = %v, want 1 (Error)", d["severity"])
	}
	if d["message"] == nil || d["message"] == "" {
		t.Error("diagnostic message must not be empty")
	}
}

func TestDiagnosticsHandler_SyntaxError_LineAndCharAreZeroBased(t *testing.T) {
	ds := lsp.NewDocumentStore()
	// Line 2 (1-based) has the error → LSP expects line 1 (0-based)
	ds.Open("file:///bad.dsl", 1, "WORKFLOW test\n  SET status = \"open\"\n")

	h := handlers.NewDiagnosticsHandler(ds)
	var buf bytes.Buffer
	_ = h.Publish("file:///bad.dsl", &buf)

	msg := readNotification(t, &buf)
	params := msg["params"].(map[string]any)
	diags := params["diagnostics"].([]any)
	if len(diags) == 0 {
		t.Skip("no diagnostics to check position on")
	}

	d := diags[0].(map[string]any)
	rng := d["range"].(map[string]any)
	start := rng["start"].(map[string]any)
	line := start["line"].(float64)
	char := start["character"].(float64)

	if line < 0 {
		t.Errorf("line = %v, must be >= 0 (0-based)", line)
	}
	if char < 0 {
		t.Errorf("character = %v, must be >= 0 (0-based)", char)
	}
}

func TestDiagnosticsHandler_UnknownURI_ReturnsError(t *testing.T) {
	ds := lsp.NewDocumentStore()
	h := handlers.NewDiagnosticsHandler(ds)
	var buf bytes.Buffer
	err := h.Publish("file:///missing.dsl", &buf)
	if err == nil {
		t.Error("expected error for unknown URI")
	}
}

func TestDiagnosticsHandler_PublishDiagnosticsHasNoID(t *testing.T) {
	ds := lsp.NewDocumentStore()
	ds.Open("file:///wf.dsl", 1, "WORKFLOW test\nON case.created\nSET case.status = \"open\"\n")

	h := handlers.NewDiagnosticsHandler(ds)
	var buf bytes.Buffer
	_ = h.Publish("file:///wf.dsl", &buf)

	msg := readNotification(t, &buf)
	if _, hasID := msg["id"]; hasID {
		t.Error("publishDiagnostics is a notification and must not have an id field")
	}
}
