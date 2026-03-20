package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpEchoExecutor struct{}

func (mcpEchoExecutor) Execute(_ context.Context, params json.RawMessage) (json.RawMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(params, &raw); err != nil {
		return nil, err
	}
	return mustMarshalRaw(map[string]any{
		"ok":   true,
		"echo": raw["message"],
	}), nil
}

type stubMCPResourceProvider struct {
	items []*MCPResourceDescriptor
	read  map[string]*MCPResourcePayload
	err   error
}

func (s *stubMCPResourceProvider) ListResources(_ context.Context, _ string) ([]*MCPResourceDescriptor, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func (s *stubMCPResourceProvider) ReadResource(_ context.Context, _ string, uri string) (*MCPResourcePayload, error) {
	if s.err != nil {
		return nil, s.err
	}
	item, ok := s.read[uri]
	if !ok {
		return nil, errors.New("resource not found")
	}
	return item, nil
}

func TestMCPGateway_ConnectInMemory_ExposesToolsAndResources(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	registry := NewToolRegistry(db)

	if err := registry.Register("echo_tool", mcpEchoExecutor{}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := registry.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID: wsID,
		Name:        "echo_tool",
		InputSchema: json.RawMessage(`{"type":"object","required":["message"],"properties":{"message":{"type":"string"}},"additionalProperties":false}`),
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition() error = %v", err)
	}

	resourceProvider := &stubMCPResourceProvider{
		items: []*MCPResourceDescriptor{{
			URI:         "fenix://context/summary",
			Name:        "workspace_summary",
			Title:       "Workspace Summary",
			Description: "Synthetic workspace context",
			MIMEType:    "application/json",
		}},
		read: map[string]*MCPResourcePayload{
			"fenix://context/summary": {
				URI:      "fenix://context/summary",
				MIMEType: "application/json",
				Text:     `{"workspace":"demo"}`,
				Meta:     map[string]any{"kind": "summary"},
			},
		},
	}

	gateway, err := NewMCPGateway(wsID, registry, resourceProvider)
	if err != nil {
		t.Fatalf("NewMCPGateway() error = %v", err)
	}

	session, cleanup, err := gateway.ConnectInMemory(context.Background())
	if err != nil {
		t.Fatalf("ConnectInMemory() error = %v", err)
	}
	defer cleanup()

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("expected at least one tool")
	}
	if !containsTool(tools.Tools, "echo_tool") {
		t.Fatalf("expected echo_tool in MCP tools list")
	}

	toolResult, err := gateway.CallTool(context.Background(), session, "echo_tool", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if len(toolResult.Content) != 1 {
		t.Fatalf("unexpected content length: %d", len(toolResult.Content))
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", toolResult.Content[0])
	}
	if text.Text == "" {
		t.Fatal("expected textual tool result")
	}
	if toolResult.StructuredContent == nil {
		t.Fatal("expected structured content for object output")
	}

	resources, err := gateway.ListResources(context.Background(), session)
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if len(resources.Resources) != 1 || resources.Resources[0].URI != "fenix://context/summary" {
		t.Fatalf("unexpected resources list: %#v", resources.Resources)
	}

	readResult, err := gateway.ReadResource(context.Background(), session, "fenix://context/summary")
	if err != nil {
		t.Fatalf("ReadResource() error = %v", err)
	}
	if len(readResult.Contents) != 1 {
		t.Fatalf("unexpected read contents length: %d", len(readResult.Contents))
	}
	if got := readResult.Contents[0].Text; got != `{"workspace":"demo"}` {
		t.Fatalf("ReadResource() text = %q", got)
	}
}

func TestMCPGateway_NewMCPGateway_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	registry := NewToolRegistry(db)

	if _, err := NewMCPGateway("", registry, nil); !errors.Is(err, ErrMCPGatewayInvalid) {
		t.Fatalf("expected ErrMCPGatewayInvalid for empty workspace, got %v", err)
	}
	if _, err := NewMCPGateway("ws", nil, nil); !errors.Is(err, ErrMCPGatewayInvalid) {
		t.Fatalf("expected ErrMCPGatewayInvalid for nil registry, got %v", err)
	}
}

func TestMCPGateway_BuildServer_PropagatesResourceProviderErrors(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	registry := NewToolRegistry(db)

	gateway, err := NewMCPGateway(wsID, registry, &stubMCPResourceProvider{err: errors.New("boom")})
	if err != nil {
		t.Fatalf("NewMCPGateway() error = %v", err)
	}

	if _, err := gateway.BuildServer(context.Background()); err == nil || err.Error() != "boom" {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestMCPGateway_ClientSessionGuardsAndPayloadValidation(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	registry := NewToolRegistry(db)
	gateway, err := NewMCPGateway(wsID, registry, nil)
	if err != nil {
		t.Fatalf("NewMCPGateway() error = %v", err)
	}

	if _, err := gateway.CallTool(context.Background(), nil, "echo_tool", nil); !errors.Is(err, ErrMCPClientSession) {
		t.Fatalf("CallTool(nil) err = %v, want ErrMCPClientSession", err)
	}
	if _, err := gateway.ListResources(context.Background(), nil); !errors.Is(err, ErrMCPClientSession) {
		t.Fatalf("ListResources(nil) err = %v, want ErrMCPClientSession", err)
	}
	if _, err := gateway.ReadResource(context.Background(), nil, "fenix://x"); !errors.Is(err, ErrMCPClientSession) {
		t.Fatalf("ReadResource(nil) err = %v, want ErrMCPClientSession", err)
	}

	if err := (&MCPResourcePayload{}).Validate(); !errors.Is(err, ErrMCPGatewayInvalid) {
		t.Fatalf("Validate(empty) err = %v, want ErrMCPGatewayInvalid", err)
	}
	if err := (&MCPResourcePayload{URI: "fenix://ok"}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}

func TestBuildMCPToolResult_StructuredAndTextFallback(t *testing.T) {
	t.Parallel()

	structured := buildMCPToolResult(json.RawMessage(`{"ok":true,"message":"hi"}`))
	if len(structured.Content) != 1 {
		t.Fatalf("structured content len = %d, want 1", len(structured.Content))
	}
	text, ok := structured.Content[0].(*mcp.TextContent)
	if !ok || text.Text == "" {
		t.Fatalf("unexpected structured text content = %#v", structured.Content)
	}
	if structured.StructuredContent == nil {
		t.Fatal("expected structured content")
	}

	unstructured := buildMCPToolResult(json.RawMessage(`not-json`))
	if unstructured.StructuredContent != nil {
		t.Fatalf("expected nil structured content, got %#v", unstructured.StructuredContent)
	}
}

func TestMCPGateway_BuildServerAndConnectInMemory(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	registry := NewToolRegistry(db)

	gateway, err := NewMCPGateway(wsID, registry, nil)
	if err != nil {
		t.Fatalf("NewMCPGateway() error = %v", err)
	}

	server, err := gateway.BuildServer(context.Background())
	if err != nil {
		t.Fatalf("BuildServer() error = %v", err)
	}
	if server == nil {
		t.Fatal("BuildServer() returned nil server")
	}

	session, cleanup, err := gateway.ConnectInMemory(context.Background())
	if err != nil {
		t.Fatalf("ConnectInMemory() error = %v", err)
	}
	defer cleanup()

	if _, err := gateway.ListResources(context.Background(), session); err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
}

func containsTool(items []*mcp.Tool, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func mustMarshalRaw(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
