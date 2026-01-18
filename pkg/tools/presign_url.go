package tools

import (
	"context"
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

	method := OptionalString(args, "method", "GET")
	expiresIn := OptionalInt(args, "expires_in", 3600)
	connectionName := OptionalString(args, "connection", "")

	// Validate method
	if method != "GET" && method != "PUT" {
		return ErrorResultf("invalid method: %s (must be GET or PUT)", method), nil
	}

	// Validate and cap expiration
	if expiresIn < 1 {
		expiresIn = 3600
	}
	if expiresIn > 604800 {
		expiresIn = 604800
	}

	expires := time.Duration(expiresIn) * time.Second

	// Get client
	client, err := t.GetClient(connectionName)
	if err != nil {
		return ErrorResult(err), nil
	}

	// Generate presigned URL
	var presigned *presignedURLResult
	if method == "GET" {
		url, err := client.PresignGetURL(ctx, bucket, key, expires)
		if err != nil {
			return ErrorResultf("failed to generate presigned URL: %v", err), nil
		}
		presigned = &presignedURLResult{
			URL:       url.URL,
			Method:    url.Method,
			ExpiresAt: url.ExpiresAt,
		}
	} else {
		url, err := client.PresignPutURL(ctx, bucket, key, expires)
		if err != nil {
			return ErrorResultf("failed to generate presigned URL: %v", err), nil
		}
		presigned = &presignedURLResult{
			URL:       url.URL,
			Method:    url.Method,
			ExpiresAt: url.ExpiresAt,
		}
	}

	// Build result
	result := PresignURLResult{
		Bucket:    bucket,
		Key:       key,
		URL:       presigned.URL,
		Method:    presigned.Method,
		ExpiresIn: expiresIn,
		ExpiresAt: presigned.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}

	return JSONResult(result)
}

// presignedURLResult is an internal type for holding presigned URL info.
type presignedURLResult struct {
	URL       string
	Method    string
	ExpiresAt time.Time
}
