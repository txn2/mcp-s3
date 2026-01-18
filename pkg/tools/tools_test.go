package tools

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestListBuckets(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddBucket("bucket1", time.Now().Add(-24*time.Hour))
	mock.AddBucket("bucket2", time.Now())

	toolkit := NewToolkit(mock)

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolListBuckets
	req.Params.Arguments = map[string]any{}

	result, err := toolkit.handleListBuckets(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %v", result.Content)
	}
}

func TestListObjects(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddObject("test-bucket", "file1.txt", []byte("content1"), "text/plain")
	mock.AddObject("test-bucket", "file2.txt", []byte("content2"), "text/plain")
	mock.AddObject("test-bucket", "folder/file3.txt", []byte("content3"), "text/plain")

	toolkit := NewToolkit(mock)

	t.Run("list all objects", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolListObjects
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
		}

		result, err := toolkit.handleListObjects(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("list with prefix", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolListObjects
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"prefix": "folder/",
		}

		result, err := toolkit.handleListObjects(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolListObjects
		req.Params.Arguments = map[string]any{}

		result, err := toolkit.handleListObjects(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing bucket")
		}
	})
}

func TestGetObject(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddObject("test-bucket", "text.txt", []byte("Hello, World!"), "text/plain")
	mock.AddObject("test-bucket", "binary.bin", []byte{0x00, 0x01, 0x02, 0x03}, "application/octet-stream")

	toolkit := NewToolkit(mock)

	t.Run("get text object", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolGetObject
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"key":    "text.txt",
		}

		result, err := toolkit.handleGetObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("get binary object", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolGetObject
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"key":    "binary.bin",
		}

		result, err := toolkit.handleGetObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolGetObject
		req.Params.Arguments = map[string]any{
			"key": "text.txt",
		}

		result, err := toolkit.handleGetObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing bucket")
		}
	})
}

func TestPutObject(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	t.Run("put text object", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolPutObject
		req.Params.Arguments = map[string]any{
			"bucket":       "test-bucket",
			"key":          "new-file.txt",
			"content":      "Hello, World!",
			"content_type": "text/plain",
		}

		result, err := toolkit.handlePutObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("put object read-only mode", func(t *testing.T) {
		readOnlyToolkit := NewToolkit(mock, WithReadOnly(true))

		req := mcp.CallToolRequest{}
		req.Params.Name = ToolPutObject
		req.Params.Arguments = map[string]any{
			"bucket":  "test-bucket",
			"key":     "new-file.txt",
			"content": "Hello!",
		}

		result, err := readOnlyToolkit.handlePutObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for read-only mode")
		}
	})
}

func TestDeleteObject(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddObject("test-bucket", "to-delete.txt", []byte("delete me"), "text/plain")

	toolkit := NewToolkit(mock)

	t.Run("delete object", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolDeleteObject
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"key":    "to-delete.txt",
		}

		result, err := toolkit.handleDeleteObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("delete object read-only mode", func(t *testing.T) {
		readOnlyToolkit := NewToolkit(mock, WithReadOnly(true))

		req := mcp.CallToolRequest{}
		req.Params.Name = ToolDeleteObject
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"key":    "to-delete.txt",
		}

		result, err := readOnlyToolkit.handleDeleteObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for read-only mode")
		}
	})
}

func TestCopyObject(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddObject("source-bucket", "source.txt", []byte("copy me"), "text/plain")

	toolkit := NewToolkit(mock)

	t.Run("copy object", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolCopyObject
		req.Params.Arguments = map[string]any{
			"source_bucket": "source-bucket",
			"source_key":    "source.txt",
			"dest_bucket":   "dest-bucket",
			"dest_key":      "dest.txt",
		}

		result, err := toolkit.handleCopyObject(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})
}

func TestPresignURL(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	t.Run("presign GET URL", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolPresignURL
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"key":    "file.txt",
			"method": "GET",
		}

		result, err := toolkit.handlePresignURL(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("presign PUT URL", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Name = ToolPresignURL
		req.Params.Arguments = map[string]any{
			"bucket": "test-bucket",
			"key":    "file.txt",
			"method": "PUT",
		}

		result, err := toolkit.handlePresignURL(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})
}

func TestListConnections(t *testing.T) {
	mock := NewMockS3Client("default")
	toolkit := NewToolkit(mock, WithDefaultConnection("default"))

	req := mcp.CallToolRequest{}
	req.Params.Name = ToolListConnections
	req.Params.Arguments = map[string]any{}

	result, err := toolkit.handleListConnections(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %v", result.Content)
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		expected    bool
	}{
		{
			name:        "text/plain",
			contentType: "text/plain",
			body:        []byte("hello"),
			expected:    true,
		},
		{
			name:        "application/json",
			contentType: "application/json",
			body:        []byte(`{"key": "value"}`),
			expected:    true,
		},
		{
			name:        "binary with null bytes",
			contentType: "application/octet-stream",
			body:        []byte{0x00, 0x01, 0x02},
			expected:    false,
		},
		{
			name:        "empty content",
			contentType: "application/octet-stream",
			body:        []byte{},
			expected:    true,
		},
		{
			name:        "utf-8 text without content type",
			contentType: "",
			body:        []byte("Hello, 世界!"),
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextContent(tt.contentType, tt.body)
			if result != tt.expected {
				t.Errorf("isTextContent(%q, %v) = %v, expected %v", tt.contentType, tt.body, result, tt.expected)
			}
		})
	}
}
