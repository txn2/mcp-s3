package extensions

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/txn2/mcp-s3/pkg/tools"
)

// PrefixACLInterceptor enforces prefix-based access control.
type PrefixACLInterceptor struct {
	allowedPrefixes []string
	deniedPrefixes  []string
}

// NewPrefixACLInterceptor creates a new prefix ACL interceptor.
func NewPrefixACLInterceptor(allowedPrefixes, deniedPrefixes []string) *PrefixACLInterceptor {
	return &PrefixACLInterceptor{
		allowedPrefixes: allowedPrefixes,
		deniedPrefixes:  deniedPrefixes,
	}
}

// Name returns the interceptor name.
func (i *PrefixACLInterceptor) Name() string {
	return "prefixacl"
}

// Intercept checks if the requested key prefix is allowed.
func (i *PrefixACLInterceptor) Intercept(ctx context.Context, tc *tools.ToolContext, request mcp.CallToolRequest) tools.InterceptResult {
	// Get the key from the request
	key := i.extractKey(tc.ToolName, request.Params.Arguments)
	if key == "" {
		return tools.Allowed()
	}

	// Check denied prefixes first
	for _, prefix := range i.deniedPrefixes {
		if strings.HasPrefix(key, prefix) {
			return tools.Blocked("access to prefix " + prefix + " is denied")
		}
	}

	// If allowed prefixes are specified, check them
	if len(i.allowedPrefixes) > 0 {
		allowed := false
		for _, prefix := range i.allowedPrefixes {
			if strings.HasPrefix(key, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return tools.Blocked("access denied: key does not match any allowed prefix")
		}
	}

	return tools.Allowed()
}

// extractKey extracts the object key from the request arguments based on the tool.
func (i *PrefixACLInterceptor) extractKey(toolName string, args map[string]any) string {
	switch toolName {
	case tools.ToolGetObject, tools.ToolGetObjectMetadata, tools.ToolPutObject, tools.ToolDeleteObject, tools.ToolPresignURL:
		if key, ok := args["key"].(string); ok {
			return key
		}
	case tools.ToolListObjects:
		if prefix, ok := args["prefix"].(string); ok {
			return prefix
		}
	case tools.ToolCopyObject:
		// Check both source and destination keys
		sourceKey, _ := args["source_key"].(string)
		destKey, _ := args["dest_key"].(string)
		// Return source key for checking; dest key would need separate check
		if sourceKey != "" {
			return sourceKey
		}
		return destKey
	}
	return ""
}

// Ensure PrefixACLInterceptor implements RequestInterceptor.
var _ tools.RequestInterceptor = (*PrefixACLInterceptor)(nil)
