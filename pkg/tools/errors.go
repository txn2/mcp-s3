package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/mark3labs/mcp-go/mcp"
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

	// ErrAccessDenied is returned when access to a resource is denied.
	ErrAccessDenied = errors.New("access denied")

	// ErrNotFound is returned when a requested resource doesn't exist.
	ErrNotFound = errors.New("resource not found")
)

// ErrorResult creates an MCP CallToolResult with an error message.
func ErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error: %s", err.Error()),
			},
		},
		IsError: true,
	}
}

// ErrorResultf creates an MCP CallToolResult with a formatted error message.
func ErrorResultf(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
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
			mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// TextResultf creates an MCP CallToolResult with a formatted text message.
func TextResultf(format string, args ...any) *mcp.CallToolResult {
	return TextResult(fmt.Sprintf(format, args...))
}

// JSONResult creates an MCP CallToolResult with JSON-encoded data.
func JSONResult(data any) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonBytes),
			},
		},
	}, nil
}

// MustJSONResult creates an MCP CallToolResult with JSON-encoded data.
// Panics if marshaling fails.
func MustJSONResult(data any) *mcp.CallToolResult {
	result, err := JSONResult(data)
	if err != nil {
		panic(err)
	}
	return result
}

// BinaryResult creates an MCP CallToolResult with binary/blob content as base64.
func BinaryResult(data []byte, mimeType string) *mcp.CallToolResult {
	// Use TextContent with base64-encoded data since BlobResourceContents
	// is not directly usable as Content in tool results
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("data:%s;base64,%s", mimeType, string(data)),
			},
		},
	}
}

// GetArgs extracts the arguments map from the request.
// Returns an error if the arguments are not a map.
func GetArgs(request mcp.CallToolRequest) (map[string]any, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: arguments must be an object", ErrInvalidParameter)
	}
	return args, nil
}

// RequireString extracts a required string parameter from the request arguments.
// Returns an error if the parameter is missing or not a string.
func RequireString(args map[string]any, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrMissingParameter, key)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s must be a string", ErrInvalidParameter, key)
	}

	if str == "" {
		return "", fmt.Errorf("%w: %s cannot be empty", ErrInvalidParameter, key)
	}

	return str, nil
}

// OptionalString extracts an optional string parameter from the request arguments.
// Returns the default value if the parameter is missing.
func OptionalString(args map[string]any, key string, defaultValue string) string {
	val, ok := args[key]
	if !ok {
		return defaultValue
	}

	str, ok := val.(string)
	if !ok {
		return defaultValue
	}

	return str
}

// RequireInt extracts a required integer parameter from the request arguments.
// Returns an error if the parameter is missing or not a number.
func RequireInt(args map[string]any, key string) (int, error) {
	val, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMissingParameter, key)
	}

	switch v := val.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("%w: %s must be a number", ErrInvalidParameter, key)
	}
}

// OptionalInt extracts an optional integer parameter from the request arguments.
// Returns the default value if the parameter is missing.
func OptionalInt(args map[string]any, key string, defaultValue int) int {
	val, ok := args[key]
	if !ok {
		return defaultValue
	}

	switch v := val.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// OptionalInt32 extracts an optional int32 parameter from the request arguments.
func OptionalInt32(args map[string]any, key string, defaultValue int32) int32 {
	v := OptionalInt(args, key, int(defaultValue))
	// Clamp to int32 range to prevent overflow
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// OptionalBool extracts an optional boolean parameter from the request arguments.
// Returns the default value if the parameter is missing.
func OptionalBool(args map[string]any, key string, defaultValue bool) bool {
	val, ok := args[key]
	if !ok {
		return defaultValue
	}

	b, ok := val.(bool)
	if !ok {
		return defaultValue
	}

	return b
}
