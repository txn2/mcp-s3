package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListObjectsResult represents the result of listing objects.
type ListObjectsResult struct {
	Bucket            string         `json:"bucket"`
	Prefix            string         `json:"prefix,omitempty"`
	Delimiter         string         `json:"delimiter,omitempty"`
	Objects           []ObjectResult `json:"objects"`
	CommonPrefixes    []string       `json:"common_prefixes,omitempty"`
	Count             int            `json:"count"`
	IsTruncated       bool           `json:"is_truncated"`
	NextContinueToken string         `json:"next_continuation_token,omitempty"`
}

// ObjectResult represents an object in the list results.
type ObjectResult struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified,omitempty"`
	ETag         string `json:"etag,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
}

// registerListObjects registers the s3_list_objects tool.
func (t *Toolkit) registerListObjects(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolListObjects),
		Description: "List objects in an S3 bucket. Supports prefix filtering, delimiter for folder simulation, and pagination.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"bucket"},
			Properties: map[string]any{
				"bucket": map[string]any{
					"type":        "string",
					"description": "Name of the S3 bucket to list objects from.",
				},
				"prefix": map[string]any{
					"type":        "string",
					"description": "Filter objects by key prefix. Only objects with keys starting with this prefix are returned.",
				},
				"delimiter": map[string]any{
					"type":        "string",
					"description": "Character used to group keys. Commonly '/' to simulate folders. Common prefixes are returned separately.",
				},
				"max_keys": map[string]any{
					"type":        "integer",
					"description": "Maximum number of objects to return (1-1000). Default: 1000.",
					"minimum":     1,
					"maximum":     1000,
				},
				"continuation_token": map[string]any{
					"type":        "string",
					"description": "Token from a previous response to continue listing from where you left off.",
				},
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handleListObjects)
}

// handleListObjects handles the s3_list_objects tool request.
func (t *Toolkit) handleListObjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	bucket, err := RequireString(request.Params.Arguments, "bucket")
	if err != nil {
		return ErrorResult(err), nil
	}

	prefix := OptionalString(request.Params.Arguments, "prefix", "")
	delimiter := OptionalString(request.Params.Arguments, "delimiter", "")
	maxKeys := OptionalInt32(request.Params.Arguments, "max_keys", 1000)
	continueToken := OptionalString(request.Params.Arguments, "continuation_token", "")
	connectionName := OptionalString(request.Params.Arguments, "connection", "")

	// Get client
	client, err := t.GetClient(connectionName)
	if err != nil {
		return ErrorResult(err), nil
	}

	// List objects
	output, err := client.ListObjects(ctx, bucket, prefix, delimiter, maxKeys, continueToken)
	if err != nil {
		return ErrorResultf("failed to list objects: %v", err), nil
	}

	// Build result
	result := ListObjectsResult{
		Bucket:            bucket,
		Prefix:            prefix,
		Delimiter:         delimiter,
		Objects:           make([]ObjectResult, 0, len(output.Objects)),
		CommonPrefixes:    output.CommonPrefixes,
		Count:             len(output.Objects),
		IsTruncated:       output.IsTruncated,
		NextContinueToken: output.NextContinueToken,
	}

	for _, obj := range output.Objects {
		or := ObjectResult{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			StorageClass: obj.StorageClass,
		}
		if !obj.LastModified.IsZero() {
			or.LastModified = obj.LastModified.Format("2006-01-02T15:04:05Z")
		}
		result.Objects = append(result.Objects, or)
	}

	return JSONResult(result)
}
