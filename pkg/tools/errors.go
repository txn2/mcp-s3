package tools

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Common error types for S3 tools.
var (
	// ErrReadOnly is returned when a write operation is attempted in read-only mode.
	ErrReadOnly = errors.New("operation not permitted: server is in read-only mode")

	// ErrSizeLimitExceeded is returned when an object exceeds the size limit.
	ErrSizeLimitExceeded = errors.New("object size exceeds limit")

	// ErrMissingParameter is returned when a required parameter is missing.
	ErrMissingParameter = errors.New("missing required parameter")

	// ErrInvalidParameter is returned when a parameter has an invalid value.
	ErrInvalidParameter = errors.New("invalid parameter value")

	// ErrConnectionNotFound is returned when a requested connection doesn't exist.
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrNotFound is returned when a requested resource doesn't exist.
	ErrNotFound = errors.New("resource not found")
)

// ErrorResult creates an MCP CallToolResult with an error message.
func ErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Error: %s", message),
			},
		},
		IsError: true,
	}
}

// ErrorResultf creates an MCP CallToolResult with a formatted error message.
func ErrorResultf(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Error: "+format, args...),
			},
		},
		IsError: true,
	}
}

// TextResult creates an MCP CallToolResult with a text message.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}
}

// JSONResult creates an MCP CallToolResult with JSON-encoded data.
func JSONResult(data any) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}
