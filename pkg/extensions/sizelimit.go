package extensions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-s3/pkg/tools"
)

// SizeLimitInterceptor enforces size limits on object operations.
type SizeLimitInterceptor struct {
	maxGetSize int64
	maxPutSize int64
}

// NewSizeLimitInterceptor creates a new size limit interceptor.
func NewSizeLimitInterceptor(maxGetSize, maxPutSize int64) *SizeLimitInterceptor {
	return &SizeLimitInterceptor{
		maxGetSize: maxGetSize,
		maxPutSize: maxPutSize,
	}
}

// Name returns the interceptor name.
func (i *SizeLimitInterceptor) Name() string {
	return "sizelimit"
}

// Intercept checks size limits for PUT operations.
// GET size limits are handled in the tool itself since we need to check the object size.
func (i *SizeLimitInterceptor) Intercept(ctx context.Context, tc *tools.ToolContext, request *mcp.CallToolRequest) tools.InterceptResult {
	// Only check PUT operations for content size
	if tc.ToolName != tools.ToolPutObject {
		// Store the limits in context for tools to use
		tc.Set("max_get_size", i.maxGetSize)
		tc.Set("max_put_size", i.maxPutSize)
		return tools.Allowed()
	}

	// Check content size for PUT
	if i.maxPutSize <= 0 {
		return tools.Allowed()
	}

	// Extract arguments from request - the request has Params.Arguments as raw JSON
	args, err := extractArgsFromRequest(request)
	if err != nil || args == nil {
		return tools.Allowed()
	}

	content, ok := args["content"].(string)
	if !ok {
		return tools.Allowed()
	}

	isBase64 := false
	if b, ok := args["is_base64"].(bool); ok {
		isBase64 = b
	}

	var size int64
	if isBase64 {
		// Calculate decoded size
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return tools.Blocked("invalid base64 content")
		}
		size = int64(len(decoded))
	} else {
		size = int64(len(content))
	}

	if size > i.maxPutSize {
		return tools.Blocked(fmt.Sprintf("content size %d bytes exceeds limit of %d bytes", size, i.maxPutSize))
	}

	return tools.Allowed()
}

// extractArgsFromRequest extracts arguments from an MCP request.
func extractArgsFromRequest(request *mcp.CallToolRequest) (map[string]any, error) {
	if request == nil || len(request.Params.Arguments) == 0 {
		return nil, nil
	}

	// Arguments in the official SDK is json.RawMessage
	var args map[string]any
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// Ensure SizeLimitInterceptor implements RequestInterceptor.
var _ tools.RequestInterceptor = (*SizeLimitInterceptor)(nil)
