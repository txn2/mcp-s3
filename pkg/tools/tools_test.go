package tools

import (
	"context"
	"testing"
	"time"
)

func TestListBuckets(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddBucket("bucket1", time.Now().Add(-24*time.Hour))
	mock.AddBucket("bucket2", time.Now())

	toolkit := NewToolkit(mock)

	input := ListBucketsInput{}
	result, _, err := toolkit.handleListBuckets(context.Background(), nil, input)
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
		input := ListObjectsInput{
			Bucket: "test-bucket",
		}

		result, _, err := toolkit.handleListObjects(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("list with prefix", func(t *testing.T) {
		input := ListObjectsInput{
			Bucket: "test-bucket",
			Prefix: "folder/",
		}

		result, _, err := toolkit.handleListObjects(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		input := ListObjectsInput{}

		result, _, err := toolkit.handleListObjects(context.Background(), nil, input)
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
		input := GetObjectInput{
			Bucket: "test-bucket",
			Key:    "text.txt",
		}

		result, _, err := toolkit.handleGetObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("get binary object", func(t *testing.T) {
		input := GetObjectInput{
			Bucket: "test-bucket",
			Key:    "binary.bin",
		}

		result, _, err := toolkit.handleGetObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		input := GetObjectInput{
			Key: "text.txt",
		}

		result, _, err := toolkit.handleGetObject(context.Background(), nil, input)
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
		input := PutObjectInput{
			Bucket:      "test-bucket",
			Key:         "new-file.txt",
			Content:     "Hello, World!",
			ContentType: "text/plain",
		}

		result, _, err := toolkit.handlePutObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("put object read-only mode", func(t *testing.T) {
		readOnlyToolkit := NewToolkit(mock, WithReadOnly(true))

		input := PutObjectInput{
			Bucket:  "test-bucket",
			Key:     "new-file.txt",
			Content: "Hello!",
		}

		result, _, err := readOnlyToolkit.handlePutObject(context.Background(), nil, input)
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
		input := DeleteObjectInput{
			Bucket: "test-bucket",
			Key:    "to-delete.txt",
		}

		result, _, err := toolkit.handleDeleteObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("delete object read-only mode", func(t *testing.T) {
		readOnlyToolkit := NewToolkit(mock, WithReadOnly(true))

		input := DeleteObjectInput{
			Bucket: "test-bucket",
			Key:    "to-delete.txt",
		}

		result, _, err := readOnlyToolkit.handleDeleteObject(context.Background(), nil, input)
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
		input := CopyObjectInput{
			SourceBucket: "source-bucket",
			SourceKey:    "source.txt",
			DestBucket:   "dest-bucket",
			DestKey:      "dest.txt",
		}

		result, _, err := toolkit.handleCopyObject(context.Background(), nil, input)
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
		input := PresignURLInput{
			Bucket: "test-bucket",
			Key:    "file.txt",
			Method: "GET",
		}

		result, _, err := toolkit.handlePresignURL(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("presign PUT URL", func(t *testing.T) {
		input := PresignURLInput{
			Bucket: "test-bucket",
			Key:    "file.txt",
			Method: "PUT",
		}

		result, _, err := toolkit.handlePresignURL(context.Background(), nil, input)
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

	result, _, err := toolkit.handleListConnections(context.Background(), nil)
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

func TestGetObjectMetadata(t *testing.T) {
	mock := NewMockS3Client("test")
	mock.AddObject("test-bucket", "file.txt", []byte("content"), "text/plain")

	toolkit := NewToolkit(mock)

	t.Run("get metadata success", func(t *testing.T) {
		input := GetObjectMetadataInput{
			Bucket: "test-bucket",
			Key:    "file.txt",
		}

		result, _, err := toolkit.handleGetObjectMetadata(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error result: %v", result.Content)
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		input := GetObjectMetadataInput{
			Key: "file.txt",
		}

		result, _, err := toolkit.handleGetObjectMetadata(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing bucket")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		input := GetObjectMetadataInput{
			Bucket: "test-bucket",
		}

		result, _, err := toolkit.handleGetObjectMetadata(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing key")
		}
	})
}

func TestCopyObject_MissingParams(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	t.Run("missing source_bucket", func(t *testing.T) {
		input := CopyObjectInput{
			SourceKey:  "src.txt",
			DestBucket: "dest",
			DestKey:    "dst.txt",
		}

		result, _, err := toolkit.handleCopyObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing source_bucket")
		}
	})

	t.Run("missing source_key", func(t *testing.T) {
		input := CopyObjectInput{
			SourceBucket: "src",
			DestBucket:   "dest",
			DestKey:      "dst.txt",
		}

		result, _, err := toolkit.handleCopyObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing source_key")
		}
	})
}

func TestPresignURL_InvalidMethod(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	input := PresignURLInput{
		Bucket: "test-bucket",
		Key:    "file.txt",
		Method: "DELETE",
	}

	result, _, err := toolkit.handlePresignURL(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error for invalid method")
	}
}

func TestPutObject_MissingParams(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	t.Run("missing bucket", func(t *testing.T) {
		input := PutObjectInput{
			Key:     "file.txt",
			Content: "hello",
		}

		result, _, err := toolkit.handlePutObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing bucket")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		input := PutObjectInput{
			Bucket:  "test-bucket",
			Content: "hello",
		}

		result, _, err := toolkit.handlePutObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing key")
		}
	})

	t.Run("missing content", func(t *testing.T) {
		input := PutObjectInput{
			Bucket: "test-bucket",
			Key:    "file.txt",
		}

		result, _, err := toolkit.handlePutObject(context.Background(), nil, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for missing content")
		}
	})
}

func TestClampExpiration(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"negative returns default", -1, 3600},
		{"zero returns default", 0, 3600},
		{"valid value unchanged", 7200, 7200},
		{"max value unchanged", 604800, 604800},
		{"over max is clamped", 1000000, 604800},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampExpiration(tt.input)
			if got != tt.expected {
				t.Errorf("clampExpiration(%d) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDecodeContent(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		content := "hello world"
		decoded, err := decodeContent(content, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(decoded) != content {
			t.Errorf("decoded = %q, want %q", decoded, content)
		}
	})

	t.Run("base64 encoded", func(t *testing.T) {
		// "hello" in base64 is "aGVsbG8="
		encoded := "aGVsbG8="
		decoded, err := decodeContent(encoded, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(decoded) != "hello" {
			t.Errorf("decoded = %q, want %q", decoded, "hello")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := decodeContent("not-valid-base64!!!", true)
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

func TestCheckPutSizeLimit(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		mock := NewMockS3Client("test")
		toolkit := NewToolkit(mock, WithMaxPutSize(0))

		err := toolkit.checkPutSizeLimit([]byte("any size content"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("under limit", func(t *testing.T) {
		mock := NewMockS3Client("test")
		toolkit := NewToolkit(mock, WithMaxPutSize(100))

		err := toolkit.checkPutSizeLimit([]byte("small"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("over limit", func(t *testing.T) {
		mock := NewMockS3Client("test")
		toolkit := NewToolkit(mock, WithMaxPutSize(5))

		err := toolkit.checkPutSizeLimit([]byte("this is too long"))
		if err == nil {
			t.Error("expected error for content over limit")
		}
	})
}

func TestCheckGetSizeLimit(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		mock := NewMockS3Client("test")
		mock.AddObject("bucket", "key", []byte("content"), "text/plain")
		toolkit := NewToolkit(mock, WithMaxGetSize(0))

		err := toolkit.checkGetSizeLimit(context.Background(), mock, "bucket", "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("under limit", func(t *testing.T) {
		mock := NewMockS3Client("test")
		mock.AddObject("bucket", "key", []byte("small"), "text/plain")
		toolkit := NewToolkit(mock, WithMaxGetSize(1000))

		err := toolkit.checkGetSizeLimit(context.Background(), mock, "bucket", "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("over limit", func(t *testing.T) {
		mock := NewMockS3Client("test")
		// Add object with 20 bytes
		mock.AddObject("bucket", "key", []byte("this is twenty bytes"), "text/plain")
		toolkit := NewToolkit(mock, WithMaxGetSize(5))

		err := toolkit.checkGetSizeLimit(context.Background(), mock, "bucket", "key")
		if err == nil {
			t.Error("expected error for size over limit")
		}
	})
}

func TestMiddlewareFuncWrapper_NilFunctions(t *testing.T) {
	// Test Before with nil function
	mw := NewMiddlewareFunc("test", nil, nil)

	tc := NewToolContext(ToolListBuckets, "")
	ctx, err := mw.Before(context.Background(), tc)
	if err != nil {
		t.Errorf("Before() with nil function should not error: %v", err)
	}
	if ctx == nil {
		t.Error("Before() should return non-nil context")
	}

	// Test After with nil function
	result := TextResult("test")
	resultOut, errOut := mw.After(context.Background(), tc, result, nil)
	if errOut != nil {
		t.Errorf("After() with nil function should not error: %v", errOut)
	}
	if resultOut != result {
		t.Error("After() should return same result when function is nil")
	}
}
