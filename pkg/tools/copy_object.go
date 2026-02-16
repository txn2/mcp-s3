package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

// registerCopyObjectTool registers the s3_copy_object tool.
func (t *Toolkit) registerCopyObjectTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		copyInput, ok := input.(CopyObjectInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleCopyObject(ctx, req, copyInput)
	}

	wrappedHandler := t.wrapHandler(ToolCopyObject, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolCopyObject),
		Description: t.getDescription(ToolCopyObject, cfg),
		Annotations: t.getAnnotations(ToolCopyObject, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CopyObjectInput) (*mcp.CallToolResult, *CopyObjectResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*CopyObjectResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handleCopyObject handles the s3_copy_object tool request.
func (t *Toolkit) handleCopyObject(ctx context.Context, _ *mcp.CallToolRequest, input CopyObjectInput) (*mcp.CallToolResult, any, error) {
	// Check read-only mode
	if t.readOnly {
		return ErrorResult(ErrReadOnly.Error()), nil, nil
	}

	// Validate required parameters
	if input.SourceBucket == "" {
		return ErrorResult("source_bucket parameter is required"), nil, nil
	}
	if input.SourceKey == "" {
		return ErrorResult("source_key parameter is required"), nil, nil
	}
	if input.DestBucket == "" {
		return ErrorResult("dest_bucket parameter is required"), nil, nil
	}
	if input.DestKey == "" {
		return ErrorResult("dest_key parameter is required"), nil, nil
	}

	// Get client
	s3Client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// Copy object
	output, err := s3Client.CopyObject(ctx, &client.CopyObjectInput{
		SourceBucket: input.SourceBucket,
		SourceKey:    input.SourceKey,
		DestBucket:   input.DestBucket,
		DestKey:      input.DestKey,
		Metadata:     input.Metadata,
	})
	if err != nil {
		return ErrorResultf("failed to copy object: %v", err), nil, nil
	}

	// Build result
	result := CopyObjectResult{
		SourceBucket: input.SourceBucket,
		SourceKey:    input.SourceKey,
		DestBucket:   input.DestBucket,
		DestKey:      input.DestKey,
		ETag:         output.ETag,
		VersionID:    output.VersionID,
	}
	if !output.LastModified.IsZero() {
		result.LastModified = output.LastModified.Format("2006-01-02T15:04:05Z")
	}

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}
