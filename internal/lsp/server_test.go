package lsp_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
	"github.com/matiasleandrokruk/fenix/internal/lsp/handlers"
)

// lspMessage frames a JSON body as a Content-Length LSP message.
func lspMessage(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

// readLSPResponse reads one framed LSP response from r and returns the parsed JSON.
func readLSPResponse(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	br := bufio.NewReader(r)
	var contentLength int
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
			n := 0
			fmt.Sscanf(strings.TrimPrefix(line, "Content-Length: "), "%d", &n)
			contentLength = n
		}
	}
	if contentLength == 0 {
		t.Fatal("response missing Content-Length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(br, body); err != nil {
		t.Fatalf("reading body: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing response JSON: %v", err)
	}
	return result
}

// runServer starts a Server (with diagnostics wired) using in-memory pipes.
// Returns writer (client input) and reader (server output).
func runServer(t *testing.T) (clientIn io.WriteCloser, clientOut io.ReadCloser) {
	t.Helper()
	serverIn, clientIn := io.Pipe()
	clientOut, serverOut := io.Pipe()
	srv := lsp.NewServer(serverIn, serverOut)
	srv.WithDiagnostics(handlers.NewDiagnosticsHandler(srv.Docs()))
	srv.WithCompletion(handlers.NewCompletionHandler(srv.Docs())) // CLSF-43
	srv.WithHover(handlers.NewHoverHandler(srv.Docs()))           // CLSF-44
	go func() {
		_ = srv.Run()
		serverOut.Close()
	}()
	t.Cleanup(func() {
		clientIn.Close()
		clientOut.Close()
	})
	return clientIn, clientOut
}

func TestServer_Initialize_ReturnsServerInfoAndCapabilities(t *testing.T) {
	clientIn, clientOut := runServer(t)

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	if _, err := io.WriteString(clientIn, lspMessage(req)); err != nil {
		t.Fatalf("writing initialize: %v", err)
	}

	resp := readLSPResponse(t, clientOut)

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	// id round-trips as float64 when decoded into any
	if resp["id"] != float64(1) {
		t.Errorf("id = %v, want 1", resp["id"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %v", resp["result"])
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("serverInfo missing in result: %v", result)
	}
	if serverInfo["name"] != "fenixlsp" {
		t.Errorf("serverInfo.name = %v, want fenixlsp", serverInfo["name"])
	}
	if _, ok := result["capabilities"]; !ok {
		t.Error("capabilities missing in result")
	}
}

func TestServer_Initialize_CapabilitiesIncludeTextDocumentSync(t *testing.T) {
	clientIn, clientOut := runServer(t)

	req := `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{}}`
	if _, err := io.WriteString(clientIn, lspMessage(req)); err != nil {
		t.Fatalf("writing initialize: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	result := resp["result"].(map[string]any)
	caps := result["capabilities"].(map[string]any)

	if caps["textDocumentSync"] == nil {
		t.Error("textDocumentSync missing in capabilities")
	}
}

func TestServer_Shutdown_ReturnsNullResult(t *testing.T) {
	clientIn, clientOut := runServer(t)

	// Must initialize first per LSP spec, but our shell does not enforce ordering.
	shutdown := `{"jsonrpc":"2.0","id":10,"method":"shutdown","params":null}`
	if _, err := io.WriteString(clientIn, lspMessage(shutdown)); err != nil {
		t.Fatalf("writing shutdown: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	if resp["id"] != float64(10) {
		t.Errorf("shutdown id = %v, want 10", resp["id"])
	}
	if _, hasError := resp["error"]; hasError {
		t.Errorf("shutdown response must not contain error: %v", resp["error"])
	}
}

func TestServer_UnknownRequest_ReturnsMethodNotFound(t *testing.T) {
	clientIn, clientOut := runServer(t)

	req := `{"jsonrpc":"2.0","id":99,"method":"unknownMethod","params":{}}`
	if _, err := io.WriteString(clientIn, lspMessage(req)); err != nil {
		t.Fatalf("writing unknown request: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error field for unknown method, got: %v", resp)
	}
	if errObj["code"] != float64(-32601) {
		t.Errorf("error code = %v, want -32601", errObj["code"])
	}
}

func TestServer_UnknownNotification_ProducesNoResponse(t *testing.T) {
	clientIn, clientOut := runServer(t)

	// Send notification (no id), then send shutdown to get a response to unblock.
	notification := `{"jsonrpc":"2.0","method":"unknownNotification","params":{}}`
	shutdown := `{"jsonrpc":"2.0","id":5,"method":"shutdown","params":null}`

	if _, err := io.WriteString(clientIn, lspMessage(notification)+lspMessage(shutdown)); err != nil {
		t.Fatalf("writing messages: %v", err)
	}

	// The only response should be the shutdown one.
	resp := readLSPResponse(t, clientOut)
	if resp["id"] != float64(5) {
		t.Errorf("expected shutdown response id=5, got id=%v — notification may have produced a spurious response", resp["id"])
	}
}

func TestServer_DidOpen_PublishesDiagnostics(t *testing.T) {
	clientIn, clientOut := runServer(t)

	// Valid DSL → publishDiagnostics with empty list
	params := `{"textDocument":{"uri":"file:///wf.dsl","languageId":"dsl","version":1,"text":"WORKFLOW test\nON case.created\nSET case.status = \"open\"\n"}}`
	notif := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":` + params + `}`
	if _, err := io.WriteString(clientIn, lspMessage(notif)); err != nil {
		t.Fatalf("writing didOpen: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	if resp["method"] != "textDocument/publishDiagnostics" {
		t.Errorf("method = %v, want textDocument/publishDiagnostics", resp["method"])
	}
	p := resp["params"].(map[string]any)
	if p["uri"] != "file:///wf.dsl" {
		t.Errorf("uri = %v, want file:///wf.dsl", p["uri"])
	}
}

func TestServer_DidChange_PublishesDiagnostics(t *testing.T) {
	clientIn, clientOut := runServer(t)

	// Open first
	openParams := `{"textDocument":{"uri":"file:///wf.dsl","languageId":"dsl","version":1,"text":"WORKFLOW test\nON case.created\nSET case.status = \"open\"\n"}}`
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":` + openParams + `}`
	if _, err := io.WriteString(clientIn, lspMessage(open)); err != nil {
		t.Fatalf("writing didOpen: %v", err)
	}
	readLSPResponse(t, clientOut) // consume publishDiagnostics from open

	// Change to invalid DSL
	changeParams := `{"textDocument":{"uri":"file:///wf.dsl","version":2},"contentChanges":[{"text":"WORKFLOW test\nSET case.status = \"broken\"\n"}]}`
	change := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":` + changeParams + `}`
	if _, err := io.WriteString(clientIn, lspMessage(change)); err != nil {
		t.Fatalf("writing didChange: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	if resp["method"] != "textDocument/publishDiagnostics" {
		t.Errorf("method = %v, want textDocument/publishDiagnostics", resp["method"])
	}
	p := resp["params"].(map[string]any)
	diags := p["diagnostics"].([]any)
	if len(diags) == 0 {
		t.Error("expected diagnostics for invalid DSL after change")
	}
}

func TestServer_Completion_ReturnsKeywordItems(t *testing.T) { // CLSF-43
	clientIn, clientOut := runServer(t)

	// Open a DSL document first so the store knows the URI.
	openParams := `{"textDocument":{"uri":"file:///wf.dsl","languageId":"dsl","version":1,"text":"WORKFLOW test\nON case.created\n"}}`
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":` + openParams + `}`
	if _, err := io.WriteString(clientIn, lspMessage(open)); err != nil {
		t.Fatalf("writing didOpen: %v", err)
	}
	readLSPResponse(t, clientOut) // consume publishDiagnostics

	req := `{"jsonrpc":"2.0","id":20,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///wf.dsl"},"position":{"line":2,"character":0}}}`
	if _, err := io.WriteString(clientIn, lspMessage(req)); err != nil {
		t.Fatalf("writing completion: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	if resp["id"] != float64(20) {
		t.Errorf("id = %v, want 20", resp["id"])
	}
	result, ok := resp["result"].([]any)
	if !ok {
		t.Fatalf("result is not an array: %v", resp["result"])
	}
	if len(result) == 0 {
		t.Error("expected at least one completion item")
	}
}

func TestServer_Hover_KnownKeyword_ReturnsMarkdown(t *testing.T) { // CLSF-44
	clientIn, clientOut := runServer(t)

	openParams := `{"textDocument":{"uri":"file:///wf.dsl","languageId":"dsl","version":1,"text":"WORKFLOW test\nON case.created\n"}}`
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":` + openParams + `}`
	if _, err := io.WriteString(clientIn, lspMessage(open)); err != nil {
		t.Fatalf("writing didOpen: %v", err)
	}
	readLSPResponse(t, clientOut) // consume publishDiagnostics

	req := `{"jsonrpc":"2.0","id":30,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///wf.dsl"},"position":{"line":0,"character":0}}}`
	if _, err := io.WriteString(clientIn, lspMessage(req)); err != nil {
		t.Fatalf("writing hover: %v", err)
	}

	resp := readLSPResponse(t, clientOut)
	if resp["id"] != float64(30) {
		t.Errorf("id = %v, want 30", resp["id"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %v", resp["result"])
	}
	contents, ok := result["contents"].(map[string]any)
	if !ok {
		t.Fatalf("contents missing or not object: %v", result)
	}
	if contents["kind"] != "markdown" {
		t.Errorf("contents.kind = %v, want markdown", contents["kind"])
	}
}

func TestServer_Exit_StopsServer(t *testing.T) {
	clientIn, clientOut := runServer(t)

	exit := `{"jsonrpc":"2.0","method":"exit"}`
	if _, err := io.WriteString(clientIn, lspMessage(exit)); err != nil {
		t.Fatalf("writing exit: %v", err)
	}
	clientIn.Close()

	// After exit the server should close its output; reading should return EOF.
	buf := make([]byte, 1)
	n, err := clientOut.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("expected EOF after exit, got n=%d err=%v", n, err)
	}
}
