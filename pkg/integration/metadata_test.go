package integration

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSimpleMetadataProvider_GetMetadata(t *testing.T) {
	expectedMeta := &ObjectMetadata{
		Size:        1024,
		ContentType: "text/plain",
		ETag:        "abc123",
	}

	provider := NewSimpleMetadataProvider(
		func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
			return expectedMeta, nil
		},
	)

	ref := &ObjectReference{Bucket: "bucket", Key: "key"}
	meta, err := provider.GetMetadata(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Size != expectedMeta.Size {
		t.Errorf("Size = %d, want %d", meta.Size, expectedMeta.Size)
	}
	if meta.ContentType != expectedMeta.ContentType {
		t.Errorf("ContentType = %q, want %q", meta.ContentType, expectedMeta.ContentType)
	}
	if meta.ETag != expectedMeta.ETag {
		t.Errorf("ETag = %q, want %q", meta.ETag, expectedMeta.ETag)
	}
}

func TestSimpleMetadataProvider_Exists(t *testing.T) {
	t.Run("returns true when object exists", func(t *testing.T) {
		provider := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				return &ObjectMetadata{}, nil
			},
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		exists, err := provider.Exists(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !exists {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("returns false when error", func(t *testing.T) {
		provider := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				return nil, errors.New("not found")
			},
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		exists, err := provider.Exists(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if exists {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestSimpleMetadataProvider_GetETag(t *testing.T) {
	t.Run("returns ETag on success", func(t *testing.T) {
		provider := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				return &ObjectMetadata{ETag: "etag-123"}, nil
			},
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		etag, err := provider.GetETag(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if etag != "etag-123" {
			t.Errorf("GetETag() = %q, want %q", etag, "etag-123")
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		expectedErr := errors.New("metadata error")
		provider := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				return nil, expectedErr
			},
		)

		ref := &ObjectReference{Bucket: "bucket", Key: "key"}
		_, err := provider.GetETag(context.Background(), ref)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestCachedMetadataProvider(t *testing.T) {
	t.Run("caches metadata", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				callCount++
				return &ObjectMetadata{Size: int64(callCount)}, nil
			},
		)

		cached := NewCachedMetadataProvider(inner, time.Hour)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		// First call
		meta1, err := cached.GetMetadata(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Second call should use cache
		meta2, err := cached.GetMetadata(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if callCount != 1 {
			t.Errorf("inner provider called %d times, want 1", callCount)
		}

		// Both should have same value from first call
		if meta1.Size != meta2.Size {
			t.Errorf("cached values differ: %d vs %d", meta1.Size, meta2.Size)
		}
	})

	t.Run("expires after TTL", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				callCount++
				return &ObjectMetadata{Size: int64(callCount)}, nil
			},
		)

		// Use a very short TTL
		cached := NewCachedMetadataProvider(inner, 10*time.Millisecond)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		// First call
		_, err := cached.GetMetadata(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Wait for TTL to expire
		time.Sleep(20 * time.Millisecond)

		// Second call should fetch fresh data
		_, err = cached.GetMetadata(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if callCount != 2 {
			t.Errorf("inner provider called %d times after TTL, want 2", callCount)
		}
	})

	t.Run("different keys are cached separately", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				callCount++
				return &ObjectMetadata{}, nil
			},
		)

		cached := NewCachedMetadataProvider(inner, time.Hour)

		ref1 := &ObjectReference{Bucket: "bucket", Key: "key1"}
		ref2 := &ObjectReference{Bucket: "bucket", Key: "key2"}

		_, _ = cached.GetMetadata(context.Background(), ref1)
		_, _ = cached.GetMetadata(context.Background(), ref2)

		if callCount != 2 {
			t.Errorf("inner provider called %d times, want 2", callCount)
		}
	})

	t.Run("ClearCache clears the cache", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				callCount++
				return &ObjectMetadata{}, nil
			},
		)

		cached := NewCachedMetadataProvider(inner, time.Hour)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		_, _ = cached.GetMetadata(context.Background(), ref)
		cached.ClearCache()
		_, _ = cached.GetMetadata(context.Background(), ref)

		if callCount != 2 {
			t.Errorf("inner provider called %d times after clear, want 2", callCount)
		}
	})

	t.Run("Exists uses cache", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				callCount++
				return &ObjectMetadata{}, nil
			},
		)

		cached := NewCachedMetadataProvider(inner, time.Hour)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		exists, err := cached.Exists(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}

		// Call again - should use cache
		_, _ = cached.Exists(context.Background(), ref)

		if callCount != 1 {
			t.Errorf("inner provider called %d times, want 1", callCount)
		}
	})

	t.Run("GetETag uses cache", func(t *testing.T) {
		callCount := 0
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				callCount++
				return &ObjectMetadata{ETag: "etag"}, nil
			},
		)

		cached := NewCachedMetadataProvider(inner, time.Hour)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		etag, err := cached.GetETag(context.Background(), ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if etag != "etag" {
			t.Errorf("GetETag() = %q, want %q", etag, "etag")
		}

		// Call again - should use cache
		_, _ = cached.GetETag(context.Background(), ref)

		if callCount != 1 {
			t.Errorf("inner provider called %d times, want 1", callCount)
		}
	})

	t.Run("propagates errors", func(t *testing.T) {
		expectedErr := errors.New("metadata error")
		inner := NewSimpleMetadataProvider(
			func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
				return nil, expectedErr
			},
		)

		cached := NewCachedMetadataProvider(inner, time.Hour)
		ref := &ObjectReference{Bucket: "bucket", Key: "key"}

		_, err := cached.GetMetadata(context.Background(), ref)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestObjectMetadata(t *testing.T) {
	now := time.Now()
	meta := &ObjectMetadata{
		Reference: &ObjectReference{
			Connection: "conn",
			Bucket:     "bucket",
			Key:        "key",
		},
		Size:         1024,
		ContentType:  "application/json",
		LastModified: now,
		ETag:         "abc123",
		CustomMetadata: map[string]string{
			"author": "test",
		},
		StorageClass: "STANDARD",
		VersionID:    "v1",
	}

	if meta.Size != 1024 {
		t.Errorf("Size = %d, want %d", meta.Size, 1024)
	}
	if meta.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want %q", meta.ContentType, "application/json")
	}
	if meta.ETag != "abc123" {
		t.Errorf("ETag = %q, want %q", meta.ETag, "abc123")
	}
	if meta.CustomMetadata["author"] != "test" {
		t.Errorf("CustomMetadata[author] = %q, want %q", meta.CustomMetadata["author"], "test")
	}
	if meta.StorageClass != "STANDARD" {
		t.Errorf("StorageClass = %q, want %q", meta.StorageClass, "STANDARD")
	}
	if meta.VersionID != "v1" {
		t.Errorf("VersionID = %q, want %q", meta.VersionID, "v1")
	}
}

func TestCompositeProvider(t *testing.T) {
	contentProvider := NewSimpleContentProvider(
		func(ctx context.Context, ref *ObjectReference) ([]byte, error) {
			return []byte("content"), nil
		},
		nil,
		nil,
	)

	metadataProvider := NewSimpleMetadataProvider(
		func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
			return &ObjectMetadata{Size: 100}, nil
		},
	)

	resolver := NewDefaultResolver("default", "bucket")

	composite := NewCompositeProvider(contentProvider, metadataProvider, nil, resolver)

	ref := &ObjectReference{Bucket: "bucket", Key: "key"}

	// Test content access
	content, err := composite.GetContent(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetContent error: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("GetContent() = %q, want %q", content, "content")
	}

	// Test metadata access
	meta, err := composite.GetMetadata(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetMetadata error: %v", err)
	}
	if meta.Size != 100 {
		t.Errorf("GetMetadata().Size = %d, want %d", meta.Size, 100)
	}

	// Test resolver access
	resolved, err := composite.Resolve("s3://bucket/key.txt")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if resolved.Bucket != "bucket" {
		t.Errorf("Resolve().Bucket = %q, want %q", resolved.Bucket, "bucket")
	}
}

func TestNewCachedMetadataProvider(t *testing.T) {
	inner := NewSimpleMetadataProvider(
		func(ctx context.Context, ref *ObjectReference) (*ObjectMetadata, error) {
			return &ObjectMetadata{}, nil
		},
	)

	cached := NewCachedMetadataProvider(inner, 5*time.Minute)

	if cached.inner != inner {
		t.Error("inner provider not set correctly")
	}
	if cached.ttl != 5*time.Minute {
		t.Errorf("ttl = %v, want %v", cached.ttl, 5*time.Minute)
	}
	if cached.cache == nil {
		t.Error("cache not initialized")
	}
	if cached.times == nil {
		t.Error("times not initialized")
	}
}
