package tools

import (
	"context"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetObjectResult represents the result of getting an object.
type GetObjectResult struct {
	Bucket       string            `json:"bucket"`
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type,omitempty"`
	LastModified string            `json:"last_modified,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Content      string            `json:"content"`
	IsBase64     bool              `json:"is_base64"`
	Truncated    bool              `json:"truncated"`
}

// registerGetObject registers the s3_get_object tool.
func (t *Toolkit) registerGetObject(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolGetObject),
		Description: "Retrieve the content of an S3 object. For text content, returns the content directly. For binary content, returns base64-encoded data. Large objects may be truncated based on size limits.",
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
					"description": "Key (path) of the object to retrieve.",
				},
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handleGetObject)
}

// handleGetObject handles the s3_get_object tool request.
func (t *Toolkit) handleGetObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// Check size first via metadata if we have size limits
	if t.maxGetSize > 0 {
		meta, err := client.GetObjectMetadata(ctx, bucket, key)
		if err != nil {
			return ErrorResultf("failed to get object metadata: %v", err), nil
		}

		if meta.Size > t.maxGetSize {
			return ErrorResultf("%w: object size %d bytes exceeds limit of %d bytes", ErrSizeLimitExceeded, meta.Size, t.maxGetSize), nil
		}
	}

	// Get object
	content, err := client.GetObject(ctx, bucket, key)
	if err != nil {
		return ErrorResultf("failed to get object: %v", err), nil
	}

	// Build result
	result := GetObjectResult{
		Bucket:      bucket,
		Key:         key,
		Size:        content.Size,
		ContentType: content.ContentType,
		ETag:        content.ETag,
		Metadata:    content.Metadata,
		Truncated:   false,
	}

	if !content.LastModified.IsZero() {
		result.LastModified = content.LastModified.Format("2006-01-02T15:04:05Z")
	}

	// Determine if content is text or binary
	isText := isTextContent(content.ContentType, content.Body)

	if isText {
		result.Content = string(content.Body)
		result.IsBase64 = false
	} else {
		result.Content = base64.StdEncoding.EncodeToString(content.Body)
		result.IsBase64 = true
	}

	return JSONResult(result)
}

// isTextContent determines if the content is text based on content type and content inspection.
func isTextContent(contentType string, body []byte) bool {
	// Check content type first
	contentType = strings.ToLower(contentType)

	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-yaml",
		"application/yaml",
		"application/toml",
		"application/x-sh",
		"application/x-csh",
	}

	for _, tt := range textTypes {
		if strings.Contains(contentType, tt) {
			return true
		}
	}

	// Check if content is valid UTF-8 and doesn't contain null bytes
	if len(body) == 0 {
		return true
	}

	// Check for null bytes (common in binary files)
	for _, b := range body {
		if b == 0 {
			return false
		}
	}

	// Check if it's valid UTF-8
	return utf8.Valid(body)
}
