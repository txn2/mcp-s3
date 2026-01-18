package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListBucketsResult represents the result of listing buckets.
type ListBucketsResult struct {
	Buckets []BucketResult `json:"buckets"`
	Count   int            `json:"count"`
}

// BucketResult represents a bucket in the list results.
type BucketResult struct {
	Name         string `json:"name"`
	CreationDate string `json:"creation_date,omitempty"`
}

// registerListBuckets registers the s3_list_buckets tool.
func (t *Toolkit) registerListBuckets(s *server.MCPServer) {
	tool := mcp.Tool{
		Name:        t.toolName(ToolListBuckets),
		Description: "List all accessible S3 buckets. Returns bucket names and creation dates.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"connection": map[string]any{
					"type":        "string",
					"description": "Name of the S3 connection to use. If not specified, uses the default connection.",
				},
			},
		},
	}

	t.registerTool(s, tool, t.handleListBuckets)
}

// handleListBuckets handles the s3_list_buckets tool request.
func (t *Toolkit) handleListBuckets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	args, err := GetArgs(request)
	if err != nil {
		return ErrorResult(err), nil
	}

	// Get connection name
	connectionName := OptionalString(args, "connection", "")

	// Get client
	client, err := t.GetClient(connectionName)
	if err != nil {
		return ErrorResult(err), nil
	}

	// List buckets
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return ErrorResultf("failed to list buckets: %v", err), nil
	}

	// Build result
	result := ListBucketsResult{
		Buckets: make([]BucketResult, 0, len(buckets)),
		Count:   len(buckets),
	}

	for _, b := range buckets {
		br := BucketResult{
			Name: b.Name,
		}
		if !b.CreationDate.IsZero() {
			br.CreationDate = b.CreationDate.Format("2006-01-02T15:04:05Z")
		}
		result.Buckets = append(result.Buckets, br)
	}

	return JSONResult(result)
}
