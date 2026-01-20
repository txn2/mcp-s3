package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteObjectResult represents the result of deleting an object.
type DeleteObjectResult struct {
	Bucket  string `json:"bucket"`
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

// registerDeleteObjectTool registers the s3_delete_object tool.
func (t *Toolkit) registerDeleteObjectTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		deleteInput, ok := input.(DeleteObjectInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleDeleteObject(ctx, req, deleteInput)
	}

	wrappedHandler := t.wrapHandler(ToolDeleteObject, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolDeleteObject),
		Description: "Delete an object from S3. This operation is irreversible unless versioning is enabled on the bucket. This operation may be blocked in read-only mode.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteObjectInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

// handleDeleteObject handles the s3_delete_object tool request.
func (t *Toolkit) handleDeleteObject(ctx context.Context, _ *mcp.CallToolRequest, input DeleteObjectInput) (*mcp.CallToolResult, any, error) {
	// Check read-only mode
	if t.readOnly {
		return ErrorResult(ErrReadOnly.Error()), nil, nil
	}

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

	// Delete object
	err = client.DeleteObject(ctx, input.Bucket, input.Key)
	if err != nil {
		return ErrorResultf("failed to delete object: %v", err), nil, nil
	}

	// Build result
	result := DeleteObjectResult{
		Bucket:  input.Bucket,
		Key:     input.Key,
		Deleted: true,
	}

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, nil, nil
}
