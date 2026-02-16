package tools

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

// registerPutObjectTool registers the s3_put_object tool.
func (t *Toolkit) registerPutObjectTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		putInput, ok := input.(PutObjectInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handlePutObject(ctx, req, putInput)
	}

	wrappedHandler := t.wrapHandler(ToolPutObject, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        t.toolName(ToolPutObject),
		Description: t.getDescription(ToolPutObject, cfg),
		Annotations: t.getAnnotations(ToolPutObject, cfg),
		Icons:       t.getIcons(ToolPutObject, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PutObjectInput) (*mcp.CallToolResult, *PutObjectResult, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*PutObjectResult); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

// handlePutObject handles the s3_put_object tool request.
func (t *Toolkit) handlePutObject(ctx context.Context, _ *mcp.CallToolRequest, input PutObjectInput) (*mcp.CallToolResult, any, error) {
	if errResult := t.validatePutInput(input); errResult != nil {
		return errResult, nil, nil
	}

	body, errResult := t.preparePutBody(input)
	if errResult != nil {
		return errResult, nil, nil
	}

	s3Client, err := t.GetClient(input.Connection)
	if err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	output, err := s3Client.PutObject(ctx, &client.PutObjectInput{
		Bucket:      input.Bucket,
		Key:         input.Key,
		Body:        body,
		ContentType: defaultContentType(input.ContentType),
		Metadata:    input.Metadata,
	})
	if err != nil {
		return ErrorResultf("failed to put object: %v", err), nil, nil
	}

	return t.buildPutResult(input, body, output)
}

func (t *Toolkit) validatePutInput(input PutObjectInput) *mcp.CallToolResult {
	if t.readOnly {
		return ErrorResult(ErrReadOnly.Error())
	}
	if input.Bucket == "" {
		return ErrorResult("bucket parameter is required")
	}
	if input.Key == "" {
		return ErrorResult("key parameter is required")
	}
	if input.Content == "" {
		return ErrorResult("content parameter is required")
	}
	return nil
}

func (t *Toolkit) preparePutBody(input PutObjectInput) ([]byte, *mcp.CallToolResult) {
	body, err := decodeContent(input.Content, input.IsBase64)
	if err != nil {
		return nil, ErrorResultf("failed to decode base64 content: %v", err)
	}
	if err := t.checkPutSizeLimit(body); err != nil {
		return nil, ErrorResult(err.Error())
	}
	return body, nil
}

func defaultContentType(contentType string) string {
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func (t *Toolkit) buildPutResult(input PutObjectInput, body []byte, output *client.PutObjectOutput) (*mcp.CallToolResult, any, error) {
	result := PutObjectResult{
		Bucket:    input.Bucket,
		Key:       input.Key,
		Size:      int64(len(body)),
		ETag:      output.ETag,
		VersionID: output.VersionID,
	}
	jsonResult, err := JSONResult(result)
	if err != nil {
		return ErrorResultf("failed to format result: %v", err), nil, nil
	}
	return jsonResult, &result, nil
}

func decodeContent(content string, isBase64 bool) ([]byte, error) {
	if isBase64 {
		return base64.StdEncoding.DecodeString(content)
	}
	return []byte(content), nil
}

func (t *Toolkit) checkPutSizeLimit(body []byte) error {
	if t.maxPutSize > 0 && int64(len(body)) > t.maxPutSize {
		return fmt.Errorf("%w: content size %d bytes exceeds limit of %d bytes", ErrSizeLimitExceeded, len(body), t.maxPutSize)
	}
	return nil
}
