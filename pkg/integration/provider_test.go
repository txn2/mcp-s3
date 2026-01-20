package integration

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestSimpleContentProvider_GetContent(t *testing.T) {
	expectedContent := []byte("hello world")
	provider := NewSimpleContentProvider(
		func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
			return expectedContent, nil
		},
		nil,
		nil,
	)

	ref := &ObjectReference{Bucket: "bucket", Key: "key"}
	content, err := provider.GetContent(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(content) != string(expectedContent) {
		t.Errorf("GetContent() = %q, want %q", content, expectedContent)
	}
}

func TestSimpleContentProvider_GetContentStream(t *testing.T) {
	expectedContent := []byte("stream content")
	provider := NewSimpleContentProvider(
		func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
			return expectedContent, nil
		},
		nil,
		nil,
	)

	ref := &ObjectReference{Bucket: "bucket", Key: "key"}
	stream, err := provider.GetContentStream(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()

	content, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("failed to read stream: %v", err)
	}

	if string(content) != string(expectedContent) {
		t.Errorf("GetContentStream() = %q, want %q", content, expectedContent)
	}
}

func TestSimpleContentProvider_GetContentType(t *testing.T) {
	t.Run("with content type function", func(t *testing.T) {
		provider := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return nil, nil
			},
			func(ctx context.Context, ref *ObjectReference) (string, error) {
				return "application/json", nil
			},
			nil,
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		contentType, err := provider.GetContentType(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if contentType != "application/json" {
			t.Errorf("GetContentType() = %q, want %q", contentType, "application/json")
		}
	})

	t.Run("without content type function returns default", func(t *testing.T) {
		provider := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return nil, nil
			},
			nil,
			nil,
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		contentType, err := provider.GetContentType(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if contentType != "application/octet-stream" {
			t.Errorf("GetContentType() = %q, want %q", contentType, "application/octet-stream")
		}
	})
}

func TestSimpleContentProvider_GetSize(t *testing.T) {
	t.Run("with size function", func(t *testing.T) {
		provider := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return nil, nil
			},
			nil,
			func(ctx context.Context, ref *ObjectReference) (int64, error) {
				return 12345, nil
			},
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		size, err := provider.GetSize(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != 12345 {
			t.Errorf("GetSize() = %d, want %d", size, 12345)
		}
	})

	t.Run("without size function calculates from content", func(t *testing.T) {
		content := []byte("hello world")
		provider := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return content, nil
			},
			nil,
			nil,
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		size, err := provider.GetSize(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != int64(len(content)) {
			t.Errorf("GetSize() = %d, want %d", size, len(content))
		}
	})

	t.Run("without size function propagates error", func(t *testing.T) {
		expectedErr := errors.New("content error")
		provider := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return nil, expectedErr
			},
			nil,
			nil,
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		_, err := provider.GetSize(context.Background(), ref)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestCachedContentProvider(t *testing.T) {
	t.Run("caches content", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				callCount++
				return []byte("content"), nil
			},
			nil,
			nil,
		)

		cached := NewCachedContentProvider(inner)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		// First call
		_, err := cached.GetContent(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Second call should use cache
		_, err = cached.GetContent(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if callCount != 1 {
			t.Errorf("inner provider called %d times, want 1", callCount)
		}
	})

	t.Run("different keys are cached separately", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				callCount++
				return []byte(ref.Key), nil
			},
			nil,
			nil,
		)

		cached := NewCachedContentProvider(inner)

		ref1 := &ObjectReference{Bucket: "bucket", Key: "key1"}
		ref2 := &ObjectReference{Bucket: "bucket", Key: "key2"}

		_, _ = cached.GetContent(context.Background(), ref1)
		_, _ = cached.GetContent(context.Background(), ref2)

		if callCount != 2 {
			t.Errorf("inner provider called %d times, want 2", callCount)
		}
	})

	t.Run("ClearCache clears the cache", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				callCount++
				return []byte("content"), nil
			},
			nil,
			nil,
		)

		cached := NewCachedContentProvider(inner)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		_, _ = cached.GetContent(context.Background(), ref)
		cached.ClearCache()
		_, _ = cached.GetContent(context.Background(), ref)

		if callCount != 2 {
			t.Errorf("inner provider called %d times after clear, want 2", callCount)
		}
	})

	t.Run("GetContentStream uses cache", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				callCount++
				return []byte("stream content"), nil
			},
			nil,
			nil,
		)

		cached := NewCachedContentProvider(inner)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		stream, err := cached.GetContentStream(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, _ := io.ReadAll(stream)
		stream.Close()

		if string(content) != "stream content" {
			t.Errorf("GetContentStream() = %q, want %q", content, "stream content")
		}
	})

	t.Run("GetContentType delegates to inner", func(t *testing.T) {
		inner := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return nil, nil
			},
			func(ctx context.Context, ref *ObjectReference) (string, error) {
				return "text/plain", nil
			},
			nil,
		)

		cached := NewCachedContentProvider(inner)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		contentType, err := cached.GetContentType(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if contentType != "text/plain" {
			t.Errorf("GetContentType() = %q, want %q", contentType, "text/plain")
		}
	})

	t.Run("GetSize delegates to inner", func(t *testing.T) {
		inner := NewSimpleContentProvider(
			func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
				return nil, nil
			},
			nil,
			func(ctx context.Context, ref *ObjectReference) (int64, error) {
				return 999, nil
			},
		)

		cached := NewCachedContentProvider(inner)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		size, err := cached.GetSize(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != 999 {
			t.Errorf("GetSize() = %d, want %d", size, 999)
		}
	})
}

func TestBytesReader(t *testing.T) {
	t.Run("reads all data", func(t *testing.T) {
		data := []byte("hello world")
		reader := newBytesReader(data)

		result, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if string(result) != string(data) {
			t.Errorf("Read() = %q, want %q", result, data)
		}
	})

	t.Run("returns EOF when exhausted", func(t *testing.T) {
		reader := newBytesReader([]byte("short"))

		buf := make([]byte, 100)
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("first read error: %v", err)
		}
		if n != 5 {
			t.Errorf("first read n = %d, want 5", n)
		}

		n, err = reader.Read(buf)
		if err != io.EOF {
			t.Errorf("second read error = %v, want EOF", err)
		}
		if n != 0 {
			t.Errorf("second read n = %d, want 0", n)
		}
	})

	t.Run("handles empty data", func(t *testing.T) {
		reader := newBytesReader([]byte{})

		buf := make([]byte, 10)
		n, err := reader.Read(buf)

		if err != io.EOF {
			t.Errorf("error = %v, want EOF", err)
		}
		if n != 0 {
			t.Errorf("n = %d, want 0", n)
		}
	})

	t.Run("reads in chunks", func(t *testing.T) {
		data := []byte("hello world")
		reader := newBytesReader(data)

		buf := make([]byte, 5)
		var result []byte

		for {
			n, err := reader.Read(buf)
			result = append(result, buf[:n]...)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		if string(result) != string(data) {
			t.Errorf("chunked read = %q, want %q", result, data)
		}
	})
}
