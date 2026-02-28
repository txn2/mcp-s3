package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

// registerGetObjectTool registers the s3_get_object tool.
func (t *Toolkit) registerGetObjectTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		getInput, ok := input.(GetObjectInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleGetObject(ctx, req, getInput)
	}

	wrappedHandler := t.wrapHandler(ToolGetObject, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:         t.toolName(ToolGetObject),
		Title:        t.getTitle(ToolGetObject, cfg),
		Description:  t.getDescription(ToolGetObject, cfg),
		Annotations:  t.getAnnotations(ToolGetObject, cfg),
		Icons:        t.getIcons(ToolGetObject, cfg),
		OutputSchema: t.getOutputSchema(ToolGetObject, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetObjectInput) (*mcp.CallToolResult, *GetObjectResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*GetObjectResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handleGetObject handles the s3_get_object tool request.
func (t *Toolkit) handleGetObject(ctx context.Context, _ *mcp.CallToolRequest, input GetObjectInput) (*mcp.CallToolResult, any, error) {
	// Validate required parameters
	if input.Bucket == "" {
		return ErrorResult("bucket parameter is required"), nil, nil
	}
	if input.Key == "" {
		return ErrorResult("key parameter is required"), nil, nil
	}

	// Get client
	s3Client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// Check size limit
	if err = t.checkGetSizeLimit(ctx, s3Client, input.Bucket, input.Key); err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// Get object
	content, err := s3Client.GetObject(ctx, input.Bucket, input.Key)
	if err != nil {
		return ErrorResultf("failed to get object: %v", err), nil, nil
	}

	// Build result
	result := buildGetResult(input.Bucket, input.Key, content)
	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}

func (t *Toolkit) checkGetSizeLimit(ctx context.Context, s3Client S3Client, bucket, key string) error {
	if t.maxGetSize <= 0 {
		return nil
	}
	meta, err := s3Client.GetObjectMetadata(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}
	if meta.Size > t.maxGetSize {
		return fmt.Errorf("%w: object size %d bytes exceeds limit of %d bytes", ErrSizeLimitExceeded, meta.Size, t.maxGetSize)
	}
	return nil
}

func buildGetResult(bucket, key string, content *client.ObjectContent) GetObjectResult {
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
