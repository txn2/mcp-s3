package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/txn2/mcp-s3/pkg/client"
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
	args, err := GetArgs(request)
	if err != nil {
		return ErrorResult(err), nil
	}

	params, err := extractGetParams(args)
	if err != nil {
		return ErrorResult(err), nil
	}

	s3Client, err := t.GetClient(params.connection)
	if err != nil {
		return ErrorResult(err), nil
	}

	if err := t.checkGetSizeLimit(ctx, s3Client, params.bucket, params.key); err != nil {
		return ErrorResult(err), nil
	}

	content, err := s3Client.GetObject(ctx, params.bucket, params.key)
	if err != nil {
		return ErrorResultf("failed to get object: %v", err), nil
	}

	return JSONResult(buildGetResult(params, content))
}

type getParams struct {
	bucket, key, connection string
}

func extractGetParams(args map[string]any) (*getParams, error) {
	bucket, err := RequireString(args, "bucket")
	if err != nil {
		return nil, err
	}
	key, err := RequireString(args, "key")
	if err != nil {
		return nil, err
	}
	return &getParams{
		bucket:     bucket,
		key:        key,
		connection: OptionalString(args, "connection", ""),
	}, nil
}

func (t *Toolkit) checkGetSizeLimit(ctx context.Context, client S3Client, bucket, key string) error {
	if t.maxGetSize <= 0 {
		return nil
	}
	meta, err := client.GetObjectMetadata(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}
	if meta.Size > t.maxGetSize {
		return fmt.Errorf("%w: object size %d bytes exceeds limit of %d bytes", ErrSizeLimitExceeded, meta.Size, t.maxGetSize)
	}
	return nil
}

func buildGetResult(params *getParams, content *client.ObjectContent) GetObjectResult {
	result := GetObjectResult{
		Bucket:      params.bucket,
		Key:         params.key,
		Size:        content.Size,
		ContentType: content.ContentType,
		ETag:        content.ETag,
		Metadata:    content.Metadata,
		Truncated:   false,
	}
	if !content.LastModified.IsZero() {
		result.LastModified = content.LastModified.Format("2006-01-02T15:04:05Z")
	}
	result.Content, result.IsBase64 = encodeContent(content.ContentType, content.Body)
	return result
}

func encodeContent(contentType string, body []byte) (string, bool) {
	if isTextContent(contentType, body) {
		return string(body), false
	}
	return base64.StdEncoding.EncodeToString(body), true
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
