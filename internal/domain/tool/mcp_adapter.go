package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	ErrMCPGatewayInvalid = errors.New("invalid mcp gateway")
	ErrMCPClientSession  = errors.New("mcp client session is required")
)

type MCPResourceDescriptor struct {
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
}

type MCPResourcePayload struct {
	URI      string
	MIMEType string
	Text     string
	Blob     []byte
	Meta     map[string]any
}

type MCPResourceProvider interface {
	ListResources(ctx context.Context, workspaceID string) ([]*MCPResourceDescriptor, error)
	ReadResource(ctx context.Context, workspaceID, uri string) (*MCPResourcePayload, error)
}

type MCPGateway struct {
	workspaceID string
	registry    *ToolRegistry
	resources   MCPResourceProvider
}

func NewMCPGateway(workspaceID string, registry *ToolRegistry, resources MCPResourceProvider) (*MCPGateway, error) {
	if strings.TrimSpace(workspaceID) == "" || registry == nil {
		return nil, ErrMCPGatewayInvalid
	}
	return &MCPGateway{
		workspaceID: strings.TrimSpace(workspaceID),
		registry:    registry,
		resources:   resources,
	}, nil
}

func (g *MCPGateway) BuildServer(ctx context.Context) (*mcp.Server, error) {
	if err := g.registry.EnsureBuiltInToolDefinitions(ctx, g.workspaceID); err != nil {
		return nil, err
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "fenix-mcp", Version: "v0.0.1"}, nil)

	defs, err := g.registry.ListToolDefinitions(ctx, g.workspaceID)
	if err != nil {
		return nil, err
	}
	for _, def := range defs {
		g.addTool(server, def)
	}

	if addErr := g.addResources(ctx, server); addErr != nil {
		return nil, addErr
	}
	return server, nil
}

func (g *MCPGateway) ConnectInMemory(ctx context.Context) (*mcp.ClientSession, func(), error) {
	server, err := g.BuildServer(ctx)
	if err != nil {
		return nil, nil, err
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, nil, err
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "fenix-mcp-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		return nil, nil, err
	}

	cleanup := func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
	return clientSession, cleanup, nil
}

func (g *MCPGateway) CallTool(ctx context.Context, session *mcp.ClientSession, name string, arguments any) (*mcp.CallToolResult, error) {
	if session == nil {
		return nil, ErrMCPClientSession
	}
	return session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
}

func (g *MCPGateway) ListResources(ctx context.Context, session *mcp.ClientSession) (*mcp.ListResourcesResult, error) {
	if session == nil {
		return nil, ErrMCPClientSession
	}
	return session.ListResources(ctx, nil)
}

func (g *MCPGateway) ReadResource(ctx context.Context, session *mcp.ClientSession, uri string) (*mcp.ReadResourceResult, error) {
	if session == nil {
		return nil, ErrMCPClientSession
	}
	return session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
}

func (g *MCPGateway) addTool(server *mcp.Server, def *ToolDefinition) {
	server.AddTool(&mcp.Tool{
		Name:        def.Name,
		Description: derefString(def.Description),
		InputSchema: json.RawMessage(def.InputSchema),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, err := g.registry.Execute(ctx, g.workspaceID, def.Name, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return buildMCPToolResult(out), nil
	})
}

func (g *MCPGateway) addResources(ctx context.Context, server *mcp.Server) error {
	if g.resources == nil {
		return nil
	}

	resources, err := g.resources.ListResources(ctx, g.workspaceID)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		server.AddResource(&mcp.Resource{
			URI:         resource.URI,
			Name:        resource.Name,
			Title:       resource.Title,
			Description: resource.Description,
			MIMEType:    resource.MIMEType,
		}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			payload, readErr := g.resources.ReadResource(ctx, g.workspaceID, req.Params.URI)
			if readErr != nil {
				return nil, readErr
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      payload.URI,
					MIMEType: payload.MIMEType,
					Text:     payload.Text,
					Blob:     payload.Blob,
					Meta:     mcp.Meta(payload.Meta),
				}},
			}, nil
		})
	}
	return nil
}

func buildMCPToolResult(out json.RawMessage) *mcp.CallToolResult {
	text := string(out)
	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}

	var structured map[string]any
	if err := json.Unmarshal(out, &structured); err == nil && structured != nil {
		result.StructuredContent = structured
	}
	return result
}

func (p *MCPResourcePayload) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: resource payload is required", ErrMCPGatewayInvalid)
	}
	if strings.TrimSpace(p.URI) == "" {
		return fmt.Errorf("%w: resource uri is required", ErrMCPGatewayInvalid)
	}
	return nil
}
