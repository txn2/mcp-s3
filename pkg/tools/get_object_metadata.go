package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetObjectMetadataResult represents the result of getting object metadata.
type GetObjectMetadataResult struct {
	Bucket        string            `json:"bucket"`
	Key           string            `json:"key"`
	Size          int64             `json:"size"`
	ContentType   string            `json:"content_type,omitempty"`
	ContentLength int64             `json:"content_length"`
	LastModified  string            `json:"last_modified,omitempty"`
	ETag          string            `json:"etag,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// registerGetObjectMetadata registers the s3_get_object_metadata tool.
func (t *Toolkit) registerGetObjectMetadata(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolGetObjectMetadata),
		Description: "Get metadata for an S3 object without downloading its content. Returns size, content type, last modified date, ETag, and custom metadata.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"bucket", "key"},
			Properties: map[string]any{
				"bucket": map[string]any{
					"type":        "string",
					"description": "Name of the S3 bucket containing the object.",
				},
				"key": map[string]any{
					"type":        "string",
					"description": "Key (path) of the object to get metadata for.",
				},
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handleGetObjectMetadata)
}

// handleGetObjectMetadata handles the s3_get_object_metadata tool request.
func (t *Toolkit) handleGetObjectMetadata(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	bucket, err := RequireString(request.Params.Arguments, "bucket")
	if err != nil {
		return ErrorResult(err), nil
	}

	key, err := RequireString(request.Params.Arguments, "key")
	if err != nil {
		return ErrorResult(err), nil
	}

	connectionName := OptionalString(request.Params.Arguments, "connection", "")

	// Get client
	client, err := t.GetClient(connectionName)
	if err != nil {
		return ErrorResult(err), nil
	}

	// Get metadata
	meta, err := client.GetObjectMetadata(ctx, bucket, key)
	if err != nil {
		return ErrorResultf("failed to get object metadata: %v", err), nil
	}

	// Build result
	result := GetObjectMetadataResult{
		Bucket:        bucket,
		Key:           key,
		Size:          meta.Size,
		ContentType:   meta.ContentType,
		ContentLength: meta.ContentLength,
		ETag:          meta.ETag,
		Metadata:      meta.Metadata,
	}

	if !meta.LastModified.IsZero() {
		result.LastModified = meta.LastModified.Format("2006-01-02T15:04:05Z")
	}

	return JSONResult(result)
}
