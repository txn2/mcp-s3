package tools

import (
	"context"
	"encoding/base64"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/txn2/mcp-s3/pkg/client"
)

// PutObjectResult represents the result of putting an object.
type PutObjectResult struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	Size      int64  `json:"size"`
	ETag      string `json:"etag,omitempty"`
	VersionID string `json:"version_id,omitempty"`
}

// registerPutObject registers the s3_put_object tool.
func (t *Toolkit) registerPutObject(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolPutObject),
		Description: "Upload an object to S3. For text content, provide the content directly. For binary content, provide base64-encoded content and set is_base64 to true. This operation may be blocked in read-only mode.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"bucket", "key", "content"},
			Properties: map[string]any{
				"bucket": map[string]any{
					"type":        "string",
					"description": "Name of the S3 bucket to upload to.",
				},
				"key": map[string]any{
					"type":        "string",
					"description": "Key (path) for the object in the bucket.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to upload. For text, provide directly. For binary, provide base64-encoded content.",
				},
				"content_type": map[string]any{
					"type":        "string",
					"description": "MIME type of the content (e.g., 'text/plain', 'application/json'). Defaults to 'application/octet-stream'.",
				},
				"is_base64": map[string]any{
					"type":        "boolean",
					"description": "Set to true if the content is base64-encoded binary data.",
				},
				"metadata": map[string]any{
					"type":        "object",
					"description": "Custom metadata key-value pairs to attach to the object.",
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

	t.registerTool(s, tool, t.handlePutObject)
}

// handlePutObject handles the s3_put_object tool request.
func (t *Toolkit) handlePutObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	contentStr, err := RequireString(args, "content")
	if err != nil {
		return ErrorResult(err), nil
	}

	contentType := OptionalString(args, "content_type", "application/octet-stream")
	isBase64 := OptionalBool(args, "is_base64", false)
	connectionName := OptionalString(args, "connection", "")

	// Decode content
	var body []byte
	if isBase64 {
		body, err = base64.StdEncoding.DecodeString(contentStr)
		if err != nil {
			return ErrorResultf("failed to decode base64 content: %v", err), nil
		}
	} else {
		body = []byte(contentStr)
	}

	// Check size limit
	if t.maxPutSize > 0 && int64(len(body)) > t.maxPutSize {
		return ErrorResultf("%w: content size %d bytes exceeds limit of %d bytes", ErrSizeLimitExceeded, len(body), t.maxPutSize), nil
	}

	// Extract metadata if provided
	var metadata map[string]string
	if metaVal, ok := args["metadata"]; ok {
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

	// Put object
	input := &client.PutObjectInput{
		Bucket:      bucket,
		Key:         key,
		Body:        body,
		ContentType: contentType,
		Metadata:    metadata,
	}

	output, err := s3Client.PutObject(ctx, input)
	if err != nil {
		return ErrorResultf("failed to put object: %v", err), nil
	}

	// Build result
	result := PutObjectResult{
		Bucket:    bucket,
		Key:       key,
		Size:      int64(len(body)),
		ETag:      output.ETag,
		VersionID: output.VersionID,
	}

	return JSONResult(result)
}
