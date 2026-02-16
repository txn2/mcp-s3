package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PresignURLResult represents the result of generating a presigned URL.
type PresignURLResult struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	URL       string `json:"url"`
	Method    string `json:"method"`
	ExpiresIn int    `json:"expires_in_seconds"`
	ExpiresAt string `json:"expires_at"`
}

// registerPresignURLTool registers the s3_presign_url tool.
func (t *Toolkit) registerPresignURLTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		presignInput, ok := input.(PresignURLInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handlePresignURL(ctx, req, presignInput)
	}

	wrappedHandler := t.wrapHandler(ToolPresignURL, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolPresignURL),
		Description: t.getDescription(ToolPresignURL, cfg),
		Annotations: t.getAnnotations(ToolPresignURL, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PresignURLInput) (*mcp.CallToolResult, *PresignURLResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*PresignURLResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handlePresignURL handles the s3_presign_url tool request.
func (t *Toolkit) handlePresignURL(ctx context.Context, _ *mcp.CallToolRequest, input PresignURLInput) (*mcp.CallToolResult, any, error) {
	// Validate required parameters
	if input.Bucket == "" {
		return ErrorResult("bucket parameter is required"), nil, nil
	}
	if input.Key == "" {
		return ErrorResult("key parameter is required"), nil, nil
	}

	// Apply defaults and validate method
	method := input.Method
	if method == "" {
		method = "GET"
	}
	if method != "GET" && method != "PUT" {
		return ErrorResultf("invalid method: %s (must be GET or PUT)", method), nil, nil
	}

	// Clamp expiration
	expiresIn := clampExpiration(input.ExpiresIn)

	// Get client
	s3Client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// Generate presigned URL
	presigned, err := generatePresignedURL(ctx, s3Client, input.Bucket, input.Key, method, expiresIn)
	if err != nil {
		return ErrorResultf("failed to generate presigned URL: %v", err), nil, nil
	}

	// Build result
	result := PresignURLResult{
		Bucket:    input.Bucket,
		Key:       input.Key,
		URL:       presigned.URL,
		Method:    presigned.Method,
		ExpiresIn: expiresIn,
		ExpiresAt: presigned.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}

func clampExpiration(expiresIn int) int {
	if expiresIn < 1 {
		return 3600
	}
	if expiresIn > 604800 {
		return 604800
	}
	return expiresIn
}

func generatePresignedURL(ctx context.Context, client S3Client, bucket, key, method string, expiresIn int) (*presignedURLResult, error) {
	expires := time.Duration(expiresIn) * time.Second
	if method == "GET" {
		url, err := client.PresignGetURL(ctx, bucket, key, expires)
		if err != nil {
			return nil, fmt.Errorf("failed to generate presigned GET URL: %w", err)
		}
		return &presignedURLResult{URL: url.URL, Method: url.Method, ExpiresAt: url.ExpiresAt}, nil
	}
	url, err := client.PresignPutURL(ctx, bucket, key, expires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}
	return &presignedURLResult{URL: url.URL, Method: url.Method, ExpiresAt: url.ExpiresAt}, nil
}

// presignedURLResult is an internal type for holding presigned URL info.
type presignedURLResult struct {
	URL       string
	Method    string
	ExpiresAt time.Time
}
