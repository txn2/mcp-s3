package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DeleteObjectResult represents the result of deleting an object.
type DeleteObjectResult struct {
	Bucket  string `json:"bucket"`
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

// registerDeleteObject registers the s3_delete_object tool.
func (t *Toolkit) registerDeleteObject(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolDeleteObject),
		Description: "Delete an object from S3. This operation is irreversible unless versioning is enabled on the bucket. This operation may be blocked in read-only mode.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"bucket", "key"},
			Properties: map[string]any{
				"bucket": map[string]any{
					"type":        "string",
					"description": "Name of the S3 bucket containing the object to delete.",
				},
				"key": map[string]any{
					"type":        "string",
					"description": "Key (path) of the object to delete.",
				},
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handleDeleteObject)
}

// handleDeleteObject handles the s3_delete_object tool request.
func (t *Toolkit) handleDeleteObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if t.readOnly {
		return ErrorResult(ErrReadOnly), nil
	}

	// Extract arguments
	args, err := GetArgs(request)
	if err != nil {
		return ErrorResult(err), nil
	}

	// Extract parameters
	bucket, err := RequireString(args, "bucket")
	if err != nil {
		return ErrorResult(err), nil
	}

	key, err := RequireString(args, "key")
	if err != nil {
		return ErrorResult(err), nil
	}

	connectionName := OptionalString(args, "connection", "")

	// Get client
	client, err := t.GetClient(connectionName)
	if err != nil {
		return ErrorResult(err), nil
	}

	// Delete object
	err = client.DeleteObject(ctx, bucket, key)
	if err != nil {
		return ErrorResultf("failed to delete object: %v", err), nil
	}

	// Build result
	result := DeleteObjectResult{
		Bucket:  bucket,
		Key:     key,
		Deleted: true,
	}

	return JSONResult(result)
}
