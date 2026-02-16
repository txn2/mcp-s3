package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListObjectsResult represents the result of listing objects.
type ListObjectsResult struct {
	Bucket            string         `json:"bucket"`
	Prefix            string         `json:"prefix,omitempty"`
	Delimiter         string         `json:"delimiter,omitempty"`
	Objects           []ObjectResult `json:"objects"`
	CommonPrefixes    []string       `json:"common_prefixes,omitempty"`
	Count             int            `json:"count"`
	IsTruncated       bool           `json:"is_truncated"`
	NextContinueToken string         `json:"next_continuation_token,omitempty"`
}

// ObjectResult represents an object in the list results.
type ObjectResult struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified,omitempty"`
	ETag         string `json:"etag,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
}

// registerListObjectsTool registers the s3_list_objects tool.
func (t *Toolkit) registerListObjectsTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		listInput, ok := input.(ListObjectsInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleListObjects(ctx, req, listInput)
	}

	wrappedHandler := t.wrapHandler(ToolListObjects, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolListObjects),
		Description: t.getDescription(ToolListObjects, cfg),
		Annotations: t.getAnnotations(ToolListObjects, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListObjectsInput) (*mcp.CallToolResult, *ListObjectsResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*ListObjectsResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handleListObjects handles the s3_list_objects tool request.
func (t *Toolkit) handleListObjects(ctx context.Context, _ *mcp.CallToolRequest, input ListObjectsInput) (*mcp.CallToolResult, any, error) {
	// Validate required parameters
	if input.Bucket == "" {
		return ErrorResult("bucket parameter is required"), nil, nil
	}

	// Apply defaults
	maxKeys := input.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}
	if maxKeys > 1000 {
		maxKeys = 1000
	}

	// Get client
	client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	// List objects
	output, err := client.ListObjects(ctx, input.Bucket, input.Prefix, input.Delimiter, maxKeys, input.ContinuationToken)
	if err != nil {
		return ErrorResultf("failed to list objects: %v", err), nil, nil
	}

	// Build result
	result := ListObjectsResult{
		Bucket:            input.Bucket,
		Prefix:            input.Prefix,
		Delimiter:         input.Delimiter,
		Objects:           make([]ObjectResult, 0, len(output.Objects)),
		CommonPrefixes:    output.CommonPrefixes,
		Count:             len(output.Objects),
		IsTruncated:       output.IsTruncated,
		NextContinueToken: output.NextContinueToken,
	}

	for _, obj := range output.Objects {
		or := ObjectResult{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			StorageClass: obj.StorageClass,
		}
		if !obj.LastModified.IsZero() {
			or.LastModified = obj.LastModified.Format("2006-01-02T15:04:05Z")
		}
		result.Objects = append(result.Objects, or)
	}

	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}
