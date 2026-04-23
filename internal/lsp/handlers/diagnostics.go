// Package handlers contains LSP request and notification handlers. // CLSF-42
package handlers

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
	"github.com/matiasleandrokruk/fenix/internal/lsp"
)

const (
	lspSeverityError   = 1
	lspSeverityWarning = 2
)

// lspPosition is a zero-based line/character position per the LSP spec.
type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// lspRange is a start/end span in a text document.
type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

// lspDiagnostic is an LSP Diagnostic object.
type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Message  string   `json:"message"`
}

// publishDiagnosticsParams is the params for textDocument/publishDiagnostics.
type publishDiagnosticsParams struct {
	URI         string          `json:"uri"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
}

// lspNotification is a server→client JSON-RPC notification (no id field).
type lspNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// DiagnosticsHandler validates documents from a DocumentStore and publishes
// textDocument/publishDiagnostics notifications. // CLSF-42
type DiagnosticsHandler struct {
	store *lsp.DocumentStore
}

// NewDiagnosticsHandler creates a handler backed by the given store.
func NewDiagnosticsHandler(store *lsp.DocumentStore) *DiagnosticsHandler {
	return &DiagnosticsHandler{store: store}
}

// Publish validates the document at uri and writes a publishDiagnostics
// notification to out. Returns an error if the URI is not in the store.
func (h *DiagnosticsHandler) Publish(uri string, out io.Writer) error {
	doc, ok := h.store.Get(uri)
	if !ok {
		return fmt.Errorf("document not found: %s", uri)
	}

	diags := h.diagnose(doc)
	return writeNotification(out, "textDocument/publishDiagnostics", publishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

func (h *DiagnosticsHandler) diagnose(doc lsp.Document) []lspDiagnostic {
	synthetic := &workflowdomain.Workflow{DSLSource: doc.Text}
	result := agent.ValidateWorkflowDSLSyntax(synthetic)

	diags := make([]lspDiagnostic, 0, len(result.Violations)+len(result.Warnings))
	for _, v := range result.Violations {
		diags = append(diags, violationToDiagnostic(v))
	}
	for _, w := range result.Warnings {
		diags = append(diags, warningToDiagnostic(w))
	}
	return diags
}

func violationToDiagnostic(v agent.Violation) lspDiagnostic {
	return lspDiagnostic{
		Range:    domainPosToRange(v.Line, v.Column),
		Severity: lspSeverityError,
		Message:  v.Description,
	}
}

func warningToDiagnostic(w agent.Warning) lspDiagnostic {
	return lspDiagnostic{
		Range:    domainPosToRange(w.Line, w.Column),
		Severity: lspSeverityWarning,
		Message:  w.Description,
	}
}

// domainPosToRange converts a 1-based domain line/col to a zero-length LSP range.
// Domain line/col of 0 means "no position" and maps to the start of the file.
func domainPosToRange(line, col int) lspRange {
	lspLine := 0
	lspChar := 0
	if line > 0 {
		lspLine = line - 1
	}
	if col > 0 {
		lspChar = col - 1
	}
	pos := lspPosition{Line: lspLine, Character: lspChar}
	return lspRange{Start: pos, End: pos}
}

func writeNotification(out io.Writer, method string, params any) error {
	notif := lspNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, writeErr := io.WriteString(out, header); writeErr != nil {
		return writeErr
	}
	_, err = out.Write(data)
	return err
}
