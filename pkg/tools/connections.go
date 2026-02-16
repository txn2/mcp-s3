package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListConnectionsResult represents the result of listing connections.
type ListConnectionsResult struct {
	Connections       []ConnectionInfo `json:"connections"`
	DefaultConnection string           `json:"default_connection"`
	Count             int              `json:"count"`
}

// ConnectionInfo represents information about an S3 connection.
type ConnectionInfo struct {
	Name     string `json:"name"`
	Region   string `json:"region,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// registerListConnectionsTool registers the s3_list_connections tool.
func (t *Toolkit) registerListConnectionsTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		_, ok := input.(ListConnectionsInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleListConnections(ctx, req)
	}

	wrappedHandler := t.wrapHandler(ToolListConnections, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolListConnections),
		Description: t.getDescription(ToolListConnections, cfg),
		Annotations: t.getAnnotations(ToolListConnections, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListConnectionsInput) (*mcp.CallToolResult, *ListConnectionsResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*ListConnectionsResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handleListConnections handles the s3_list_connections tool request.
func (t *Toolkit) handleListConnections(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	connections := t.ListConnections()

	result := ListConnectionsResult{
		Connections:       make([]ConnectionInfo, 0, len(connections)),
		DefaultConnection: t.defaultConnection,
		Count:             len(connections),
	}

	for _, name := range connections {
		client, err := t.GetClient(name)
		if err != nil {
			continue
		}

		cfg := client.Config()
		info := ConnectionInfo{
			Name:   name,
			Region: cfg.Region,
		}

		if cfg.Endpoint != "" {
			info.Endpoint = cfg.Endpoint
		}

		result.Connections = append(result.Connections, info)
	}

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}
