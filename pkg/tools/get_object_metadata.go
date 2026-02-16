package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// registerGetObjectMetadataTool registers the s3_get_object_metadata tool.
func (t *Toolkit) registerGetObjectMetadataTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		metaInput, ok := input.(GetObjectMetadataInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleGetObjectMetadata(ctx, req, metaInput)
	}

	wrappedHandler := t.wrapHandler(ToolGetObjectMetadata, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolGetObjectMetadata),
		Description: t.getDescription(ToolGetObjectMetadata, cfg),
		Annotations: t.getAnnotations(ToolGetObjectMetadata, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetObjectMetadataInput) (*mcp.CallToolResult, *GetObjectMetadataResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*GetObjectMetadataResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handleGetObjectMetadata handles the s3_get_object_metadata tool request.
func (t *Toolkit) handleGetObjectMetadata(ctx context.Context, _ *mcp.CallToolRequest, input GetObjectMetadataInput) (*mcp.CallToolResult, any, error) {
	// Validate required parameters
	if input.Bucket == "" {
		return ErrorResult("bucket parameter is required"), nil, nil
	}
	if input.Key == "" {
		return ErrorResult("key parameter is required"), nil, nil
	}

	// Get client
	client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// Get metadata
	meta, err := client.GetObjectMetadata(ctx, input.Bucket, input.Key)
	if err != nil {
		return ErrorResultf("failed to get object metadata: %v", err), nil, nil
	}

	// Build result
	result := GetObjectMetadataResult{
		Bucket:        input.Bucket,
		Key:           input.Key,
		Size:          meta.Size,
		ContentType:   meta.ContentType,
		ContentLength: meta.ContentLength,
		ETag:          meta.ETag,
		Metadata:      meta.Metadata,
	}

	if !meta.LastModified.IsZero() {
		result.LastModified = meta.LastModified.Format("2006-01-02T15:04:05Z")
	}

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}
