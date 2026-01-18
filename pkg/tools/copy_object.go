package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/txn2/mcp-s3/pkg/client"
)

// CopyObjectResult represents the result of copying an object.
type CopyObjectResult struct {
	SourceBucket string `json:"source_bucket"`
	SourceKey    string `json:"source_key"`
	DestBucket   string `json:"dest_bucket"`
	DestKey      string `json:"dest_key"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	VersionID    string `json:"version_id,omitempty"`
}

// registerCopyObject registers the s3_copy_object tool.
func (t *Toolkit) registerCopyObject(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolCopyObject),
		Description: "Copy an object within S3, either within the same bucket or between different buckets. Can optionally update metadata during the copy. This operation may be blocked in read-only mode.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"source_bucket", "source_key", "dest_bucket", "dest_key"},
			Properties: map[string]any{
				"source_bucket": map[string]any{
					"type":        "string",
					"description": "Name of the source S3 bucket.",
				},
				"source_key": map[string]any{
					"type":        "string",
					"description": "Key (path) of the source object.",
				},
				"dest_bucket": map[string]any{
					"type":        "string",
					"description": "Name of the destination S3 bucket.",
				},
				"dest_key": map[string]any{
					"type":        "string",
					"description": "Key (path) for the destination object.",
				},
				"metadata": map[string]any{
					"type":        "object",
					"description": "New metadata to assign to the copied object. If provided, replaces source metadata.",
					"additionalProperties": map[string]any{
						"type": "string",
					},
				},
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handleCopyObject)
}

// handleCopyObject handles the s3_copy_object tool request.
func (t *Toolkit) handleCopyObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if t.readOnly {
		return ErrorResult(ErrReadOnly), nil
	}

	// Extract parameters
	sourceBucket, err := RequireString(request.Params.Arguments, "source_bucket")
	if err != nil {
		return ErrorResult(err), nil
	}

	sourceKey, err := RequireString(request.Params.Arguments, "source_key")
	if err != nil {
		return ErrorResult(err), nil
	}

	destBucket, err := RequireString(request.Params.Arguments, "dest_bucket")
	if err != nil {
		return ErrorResult(err), nil
	}

	destKey, err := RequireString(request.Params.Arguments, "dest_key")
	if err != nil {
		return ErrorResult(err), nil
	}

	connectionName := OptionalString(request.Params.Arguments, "connection", "")

	// Extract metadata if provided
	var metadata map[string]string
	if metaVal, ok := request.Params.Arguments["metadata"]; ok {
		if metaMap, ok := metaVal.(map[string]any); ok {
			metadata = make(map[string]string)
			for k, v := range metaMap {
				if strVal, ok := v.(string); ok {
					metadata[k] = strVal
				}
			}
		}
	}

	// Get client
	s3Client, err := t.GetClient(connectionName)
	if err != nil {
		return ErrorResult(err), nil
	}

	// Copy object
	input := &client.CopyObjectInput{
		SourceBucket: sourceBucket,
		SourceKey:    sourceKey,
		DestBucket:   destBucket,
		DestKey:      destKey,
		Metadata:     metadata,
	}

	output, err := s3Client.CopyObject(ctx, input)
	if err != nil {
		return ErrorResultf("failed to copy object: %v", err), nil
	}

	// Build result
	result := CopyObjectResult{
		SourceBucket: sourceBucket,
		SourceKey:    sourceKey,
		DestBucket:   destBucket,
		DestKey:      destKey,
		ETag:         output.ETag,
		VersionID:    output.VersionID,
	}

	if !output.LastModified.IsZero() {
		result.LastModified = output.LastModified.Format("2006-01-02T15:04:05Z")
	}

	return JSONResult(result)
}
