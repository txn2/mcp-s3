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
	if t.readOnly {
		return ErrorResult(ErrReadOnly), nil
	}

	args, err := GetArgs(request)
	if err != nil {
		return ErrorResult(err), nil
	}

	params, err := extractCopyParams(args)
	if err != nil {
		return ErrorResult(err), nil
	}

	s3Client, err := t.GetClient(params.connection)
	if err != nil {
		return ErrorResult(err), nil
	}

	output, err := s3Client.CopyObject(ctx, &client.CopyObjectInput{
		SourceBucket: params.sourceBucket,
		SourceKey:    params.sourceKey,
		DestBucket:   params.destBucket,
		DestKey:      params.destKey,
		Metadata:     params.metadata,
	})
	if err != nil {
		return ErrorResultf("failed to copy object: %v", err), nil
	}

	return JSONResult(buildCopyResult(params, output))
}

type copyParams struct {
	sourceBucket, sourceKey, destBucket, destKey, connection string
	metadata                                                 map[string]string
}

func extractCopyParams(args map[string]any) (*copyParams, error) {
	sourceBucket, err := RequireString(args, "source_bucket")
	if err != nil {
		return nil, err
	}
	sourceKey, err := RequireString(args, "source_key")
	if err != nil {
		return nil, err
	}
	destBucket, err := RequireString(args, "dest_bucket")
	if err != nil {
		return nil, err
	}
	destKey, err := RequireString(args, "dest_key")
	if err != nil {
		return nil, err
	}
	return &copyParams{
		sourceBucket: sourceBucket,
		sourceKey:    sourceKey,
		destBucket:   destBucket,
		destKey:      destKey,
		connection:   OptionalString(args, "connection", ""),
		metadata:     OptionalMetadata(args, "metadata"),
	}, nil
}

func buildCopyResult(params *copyParams, output *client.CopyObjectOutput) CopyObjectResult {
	result := CopyObjectResult{
		SourceBucket: params.sourceBucket,
		SourceKey:    params.sourceKey,
		DestBucket:   params.destBucket,
		DestKey:      params.destKey,
		ETag:         output.ETag,
		VersionID:    output.VersionID,
	}
	if !output.LastModified.IsZero() {
		result.LastModified = output.LastModified.Format("2006-01-02T15:04:05Z")
	}
	return result
}
