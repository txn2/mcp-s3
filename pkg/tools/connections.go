package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// registerListConnections registers the s3_list_connections tool.
func (t *Toolkit) registerListConnections(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolListConnections),
		Description: "List all configured S3 connections. Returns connection names, regions, and endpoints (if custom endpoints are configured).",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}

	t.registerTool(s, tool, t.handleListConnections)
}

// handleListConnections handles the s3_list_connections tool request.
func (t *Toolkit) handleListConnections(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	return JSONResult(result)
}
