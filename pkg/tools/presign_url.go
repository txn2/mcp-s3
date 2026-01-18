package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// registerPresignURL registers the s3_presign_url tool.
func (t *Toolkit) registerPresignURL(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolPresignURL),
		Description: "Generate a presigned URL for temporary access to an S3 object. The URL allows temporary access without requiring AWS credentials. Supports both GET (download) and PUT (upload) operations.",
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
					"description": "Key (path) of the object to generate a URL for.",
				},
				"method": map[string]any{
					"type":        "string",
					"description": "HTTP method for the presigned URL. 'GET' for downloads, 'PUT' for uploads.",
					"enum":        []string{"GET", "PUT"},
					"default":     "GET",
				},
				"expires_in": map[string]any{
					"type":        "integer",
					"description": "URL expiration time in seconds. Default: 3600 (1 hour). Maximum: 604800 (7 days).",
					"minimum":     1,
					"maximum":     604800,
				},
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handlePresignURL)
}

// handlePresignURL handles the s3_presign_url tool request.
func (t *Toolkit) handlePresignURL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := GetArgs(request)
	if err != nil {
		return ErrorResult(err), nil
	}

	params, err := extractPresignParams(args)
	if err != nil {
		return ErrorResult(err), nil
	}

	s3Client, err := t.GetClient(params.connection)
	if err != nil {
		return ErrorResult(err), nil
	}

	presigned, err := generatePresignedURL(ctx, s3Client, params)
	if err != nil {
		return ErrorResultf("failed to generate presigned URL: %v", err), nil
	}

	return JSONResult(PresignURLResult{
		Bucket:    params.bucket,
		Key:       params.key,
		URL:       presigned.URL,
		Method:    presigned.Method,
		ExpiresIn: params.expiresIn,
		ExpiresAt: presigned.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

type presignParams struct {
	bucket, key, method, connection string
	expiresIn                       int
}

func extractPresignParams(args map[string]any) (*presignParams, error) {
	bucket, err := RequireString(args, "bucket")
	if err != nil {
		return nil, err
	}
	key, err := RequireString(args, "key")
	if err != nil {
		return nil, err
	}
	method := OptionalString(args, "method", "GET")
	if method != "GET" && method != "PUT" {
		return nil, fmt.Errorf("invalid method: %s (must be GET or PUT)", method)
	}
	expiresIn := clampExpiration(OptionalInt(args, "expires_in", 3600))
	return &presignParams{
		bucket:     bucket,
		key:        key,
		method:     method,
		connection: OptionalString(args, "connection", ""),
		expiresIn:  expiresIn,
	}, nil
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

func generatePresignedURL(ctx context.Context, client S3Client, params *presignParams) (*presignedURLResult, error) {
	expires := time.Duration(params.expiresIn) * time.Second
	if params.method == "GET" {
		url, err := client.PresignGetURL(ctx, params.bucket, params.key, expires)
		if err != nil {
			return nil, err
		}
		return &presignedURLResult{URL: url.URL, Method: url.Method, ExpiresAt: url.ExpiresAt}, nil
	}
	url, err := client.PresignPutURL(ctx, params.bucket, params.key, expires)
	if err != nil {
		return nil, err
	}
	return &presignedURLResult{URL: url.URL, Method: url.Method, ExpiresAt: url.ExpiresAt}, nil
}

// presignedURLResult is an internal type for holding presigned URL info.
type presignedURLResult struct {
	URL       string
	Method    string
	ExpiresAt time.Time
}
