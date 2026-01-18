package extensions

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/txn2/mcp-s3/pkg/tools"
)

// ReadOnlyInterceptor blocks write operations when enabled.
type ReadOnlyInterceptor struct {
	enabled bool
}

// NewReadOnlyInterceptor creates a new read-only interceptor.
func NewReadOnlyInterceptor(enabled bool) *ReadOnlyInterceptor {
	return &ReadOnlyInterceptor{
		enabled: enabled,
	}
}

// Name returns the interceptor name.
func (i *ReadOnlyInterceptor) Name() string {
	return "readonly"
}

// Intercept checks if the operation is a write and blocks it if read-only mode is enabled.
func (i *ReadOnlyInterceptor) Intercept(ctx context.Context, tc *tools.ToolContext, request mcp.CallToolRequest) tools.InterceptResult {
	if !i.enabled {
		return tools.Allowed()
	}

	if tools.IsWriteTool(tc.ToolName) {
		return tools.Blocked("server is in read-only mode")
	}

	return tools.Allowed()
}

// Ensure ReadOnlyInterceptor implements RequestInterceptor.
var _ tools.RequestInterceptor = (*ReadOnlyInterceptor)(nil)
