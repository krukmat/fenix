// Package lsp implements a Language Server Protocol (LSP) shell for Carta DSL workflows. // CLSF-40
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const contentLengthHeader = "Content-Length: "

// diagnosticsPublisher abstracts the diagnostics handler for wiring and testing. // CLSF-42
type diagnosticsPublisher interface {
	Publish(uri string, out io.Writer) error
}

// completionProvider abstracts the completion handler for wiring and testing. // CLSF-43
type completionProvider interface {
	Complete(uri string, line, character int) []CompletionItem
}

// hoverProvider abstracts the hover handler for wiring and testing. // CLSF-44
type hoverProvider interface {
	Hover(uri string, line, character int) *HoverResult
}

// HoverResult is the LSP hover response payload shared between server and handlers. // CLSF-44
type HoverResult struct {
	Contents HoverMarkupContent `json:"contents"`
}

// HoverMarkupContent holds a Markdown string for LSP hover.
type HoverMarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// CompletionItem is the minimal shape shared between the server and completion handlers. // CLSF-43
type CompletionItem struct {
	Label string `json:"label"`
	Kind  int    `json:"kind"`
}

// RequestMessage represents an incoming JSON-RPC 2.0 message (request or notification).
type RequestMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ResponseMessage is an outgoing JSON-RPC 2.0 response.
type ResponseMessage struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Result  any            `json:"result,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

// ResponseError is the JSON-RPC error object.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeResult is the response payload for the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerCapabilities describes what this LSP server can do.
type ServerCapabilities struct {
	// TextDocumentSync: 1 = full document sync.
	TextDocumentSync int `json:"textDocumentSync"`
}

// ServerInfo identifies the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Server is a minimal LSP server that speaks JSON-RPC 2.0 over stdio. // CLSF-40
type Server struct {
	in         io.Reader
	out        io.Writer
	docs       *DocumentStore
	diag       diagnosticsPublisher
	completion completionProvider
	hover      hoverProvider
}

// NewServer creates a Server that reads from in and writes to out.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{in: in, out: out, docs: NewDocumentStore()}
}

// WithDiagnostics attaches a diagnostics publisher. // CLSF-42
func (s *Server) WithDiagnostics(d diagnosticsPublisher) *Server {
	s.diag = d
	return s
}

// WithCompletion attaches a completion provider. // CLSF-43
func (s *Server) WithCompletion(c completionProvider) *Server {
	s.completion = c
	return s
}

// WithHover attaches a hover provider. // CLSF-44
func (s *Server) WithHover(h hoverProvider) *Server {
	s.hover = h
	return s
}

// Docs returns the server's document store for external wiring. // CLSF-42
func (s *Server) Docs() *DocumentStore {
	return s.docs
}

// Run reads and dispatches LSP messages until the connection closes or an exit
// notification is received.
func (s *Server) Run() error {
	reader := bufio.NewReader(s.in)
	for {
		stop, err := s.processNextMessage(reader)
		if err != nil || stop {
			return err
		}
	}
}

func (s *Server) processNextMessage(reader *bufio.Reader) (bool, error) {
	msg, err := readMessage(reader)
	if err != nil {
		return errors.Is(err, io.EOF), nonEOFError(err)
	}
	return s.handle(msg)
}

func nonEOFError(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func readMessage(r *bufio.Reader) (*RequestMessage, error) {
	contentLength, err := readHeaders(r)
	if err != nil {
		return nil, err
	}
	body := make([]byte, contentLength)
	if _, readErr := io.ReadFull(r, body); readErr != nil {
		return nil, readErr
	}
	var msg RequestMessage
	if decodeErr := json.Unmarshal(body, &msg); decodeErr != nil {
		return nil, decodeErr
	}
	return &msg, nil
}

func readHeaders(r *bufio.Reader) (int, error) {
	contentLength := 0
	for {
		done, n, matched, err := readHeaderContentLength(r)
		if err != nil {
			return 0, err
		}
		if done {
			break
		}
		if matched {
			contentLength = n
		}
	}
	return contentLength, nil
}

func readHeaderContentLength(r *bufio.Reader) (bool, int, bool, error) {
	line, err := readHeaderLine(r)
	if err != nil {
		return false, 0, false, err
	}
	if line == "" {
		return true, 0, false, nil
	}
	n, matched, parseErr := parseContentLengthHeader(line)
	if parseErr != nil {
		return false, 0, false, parseErr
	}
	return false, n, matched, nil
}

func readHeaderLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseContentLengthHeader(line string) (int, bool, error) {
	if !strings.HasPrefix(line, contentLengthHeader) {
		return 0, false, nil
	}
	n, err := parseContentLength(line)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

func parseContentLength(line string) (int, error) {
	val := strings.TrimPrefix(line, contentLengthHeader)
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid Content-Length %q: %w", val, err)
	}
	return n, nil
}

// handle dispatches one message and reports whether the server should stop.
func (s *Server) handle(msg *RequestMessage) (stop bool, err error) {
	switch msg.Method {
	case "initialize":
		return false, s.handleInitialize(msg)
	case "initialized":
		return false, nil
	case "shutdown":
		return false, s.writeResponse(ResponseMessage{JSONRPC: "2.0", ID: msg.ID})
	case "exit":
		return true, nil
	default:
		return s.handleTextDocument(msg)
	}
}

// handleTextDocument dispatches textDocument/* notifications and falls back to
// method-not-found for unknown methods.
func (s *Server) handleTextDocument(msg *RequestMessage) (bool, error) {
	switch msg.Method {
	case "textDocument/didOpen":
		return false, s.handleDidOpen(msg)
	case "textDocument/didChange":
		return false, s.handleDidChange(msg)
	case "textDocument/didClose":
		return false, s.handleDidClose(msg)
	case "textDocument/completion":
		return false, s.handleCompletion(msg)
	case "textDocument/hover":
		return false, s.handleHover(msg)
	default:
		return false, s.handleUnknown(msg)
	}
}

// didOpenParams mirrors the textDocument/didOpen notification params.
type didOpenParams struct {
	TextDocument struct {
		URI     string `json:"uri"`
		Version int    `json:"version"`
		Text    string `json:"text"`
	} `json:"textDocument"`
}

// didChangeParams mirrors the textDocument/didChange notification params.
type didChangeParams struct {
	TextDocument struct {
		URI     string `json:"uri"`
		Version int    `json:"version"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

// didCloseParams mirrors the textDocument/didClose notification params.
type didCloseParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

func (s *Server) handleDidOpen(msg *RequestMessage) error {
	var p didOpenParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil // malformed notification — ignore per LSP spec
	}
	s.docs.Open(p.TextDocument.URI, p.TextDocument.Version, p.TextDocument.Text)
	return s.publishDiagnostics(p.TextDocument.URI)
}

func (s *Server) handleDidChange(msg *RequestMessage) error {
	var p didChangeParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil
	}
	if len(p.ContentChanges) == 0 {
		return nil
	}
	s.docs.Change(p.TextDocument.URI, p.TextDocument.Version, p.ContentChanges[0].Text)
	return s.publishDiagnostics(p.TextDocument.URI)
}

func (s *Server) handleDidClose(msg *RequestMessage) error {
	var p didCloseParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil
	}
	s.docs.Close(p.TextDocument.URI)
	return nil
}

// completionParams mirrors the textDocument/completion request params.
type completionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
}

func (s *Server) handleCompletion(msg *RequestMessage) error {
	var p completionParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return s.writeResponse(ResponseMessage{JSONRPC: "2.0", ID: msg.ID, Result: []CompletionItem{}})
	}
	items := s.completeAt(p.TextDocument.URI, p.Position.Line, p.Position.Character)
	return s.writeResponse(ResponseMessage{JSONRPC: "2.0", ID: msg.ID, Result: items})
}

func (s *Server) completeAt(uri string, line, character int) []CompletionItem {
	if s.completion == nil {
		return []CompletionItem{}
	}
	raw := s.completion.Complete(uri, line, character)
	items := make([]CompletionItem, len(raw))
	for i, r := range raw {
		items[i] = CompletionItem{Label: r.Label, Kind: r.Kind}
	}
	return items
}

// hoverParams mirrors the textDocument/hover request params.
type hoverParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"position"`
}

func (s *Server) handleHover(msg *RequestMessage) error { // CLSF-44
	var p hoverParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return s.writeResponse(ResponseMessage{JSONRPC: "2.0", ID: msg.ID, Result: nil})
	}
	result := s.hoverAt(p.TextDocument.URI, p.Position.Line, p.Position.Character)
	return s.writeResponse(ResponseMessage{JSONRPC: "2.0", ID: msg.ID, Result: result})
}

func (s *Server) hoverAt(uri string, line, character int) *HoverResult {
	if s.hover == nil {
		return nil
	}
	return s.hover.Hover(uri, line, character)
}

func (s *Server) publishDiagnostics(uri string) error {
	if s.diag == nil {
		return nil
	}
	return s.diag.Publish(uri, s.out)
}

func (s *Server) handleInitialize(msg *RequestMessage) error {
	return s.writeResponse(ResponseMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: InitializeResult{
			Capabilities: ServerCapabilities{TextDocumentSync: 1},
			ServerInfo:   &ServerInfo{Name: "fenixlsp", Version: "0.1.0"},
		},
	})
}

// handleUnknown responds to requests (messages with an id) with method-not-found.
// Notifications (no id) are silently ignored per LSP spec.
func (s *Server) handleUnknown(msg *RequestMessage) error {
	if msg.ID == nil {
		return nil
	}
	return s.writeResponse(ResponseMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Error:   &ResponseError{Code: -32601, Message: "method not found"},
	})
}

func (s *Server) writeResponse(resp ResponseMessage) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, writeErr := io.WriteString(s.out, header); writeErr != nil {
		return writeErr
	}
	_, err = s.out.Write(data)
	return err
}
