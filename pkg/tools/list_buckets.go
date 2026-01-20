package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// registerListBucketsTool registers the s3_list_buckets tool.
func (t *Toolkit) registerListBucketsTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		listInput, ok := input.(ListBucketsInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleListBuckets(ctx, req, listInput)
	}

	wrappedHandler := t.wrapHandler(ToolListBuckets, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolListBuckets),
		Description: "List all accessible S3 buckets. Returns bucket names and creation dates.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListBucketsInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

// handleListBuckets handles the s3_list_buckets tool request.
func (t *Toolkit) handleListBuckets(ctx context.Context, _ *mcp.CallToolRequest, input ListBucketsInput) (*mcp.CallToolResult, any, error) {
	// Get client
	client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// List buckets
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return ErrorResultf("failed to list buckets: %v", err), nil, nil
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

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, nil, nil
}
