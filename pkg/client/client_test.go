package client

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestNew_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Test with invalid credentials that won't actually connect
	cfg := &Config{
		Region:          "us-east-1",
		Endpoint:        "http://invalid-endpoint:9999",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Timeout:         100 * time.Millisecond,
	}

	// Creating the client should succeed (it doesn't connect immediately)
	client, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClient_ConnectionName(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Name:            "my-connection",
	}

	client, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	defer client.Close()

	if got := client.ConnectionName(); got != "my-connection" {
		t.Errorf("ConnectionName() = %q, expected %q", got, "my-connection")
	}
}

func TestClient_Config(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Region:          "eu-west-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
		Timeout:         60 * time.Second,
		Name:            "test-conn",
	}

	client, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	defer client.Close()

	// Get config and verify it's a copy
	returnedCfg := client.Config()
	if returnedCfg == cfg {
		t.Error("Config() should return a copy, not the original")
	}

	// Verify values match
	if returnedCfg.Region != cfg.Region {
		t.Errorf("Region mismatch: got %q, expected %q", returnedCfg.Region, cfg.Region)
	}
	if returnedCfg.Endpoint != cfg.Endpoint {
		t.Errorf("Endpoint mismatch: got %q, expected %q", returnedCfg.Endpoint, cfg.Endpoint)
	}
	if returnedCfg.UsePathStyle != cfg.UsePathStyle {
		t.Errorf("UsePathStyle mismatch: got %v, expected %v", returnedCfg.UsePathStyle, cfg.UsePathStyle)
	}

	// Verify returned config is independent
	returnedCfg.Region = "us-west-2"
	newCfg := client.Config()
	if newCfg.Region != cfg.Region {
		t.Error("modifying returned config should not affect client config")
	}
}

func TestClient_Close(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	}

	client, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	// Close should not error
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v, expected nil", err)
	}
}

func TestBucketInfo(t *testing.T) {
	now := time.Now()
	info := BucketInfo{
		Name:         "test-bucket",
		CreationDate: now,
	}

	if info.Name != "test-bucket" {
		t.Errorf("Name = %q, expected %q", info.Name, "test-bucket")
	}
	if !info.CreationDate.Equal(now) {
		t.Errorf("CreationDate = %v, expected %v", info.CreationDate, now)
	}
}

func TestObjectInfo(t *testing.T) {
	now := time.Now()
	info := ObjectInfo{
		Key:          "path/to/object.txt",
		Size:         1024,
		LastModified: now,
		ETag:         "\"abc123\"",
		StorageClass: "STANDARD",
	}

	if info.Key != "path/to/object.txt" {
		t.Errorf("Key = %q, expected %q", info.Key, "path/to/object.txt")
	}
	if info.Size != 1024 {
		t.Errorf("Size = %d, expected %d", info.Size, 1024)
	}
	if !info.LastModified.Equal(now) {
		t.Errorf("LastModified = %v, expected %v", info.LastModified, now)
	}
	if info.ETag != "\"abc123\"" {
		t.Errorf("ETag = %q, expected %q", info.ETag, "\"abc123\"")
	}
	if info.StorageClass != "STANDARD" {
		t.Errorf("StorageClass = %q, expected %q", info.StorageClass, "STANDARD")
	}
}

func TestObjectMetadata(t *testing.T) {
	now := time.Now()
	meta := ObjectMetadata{
		Key:           "path/to/object.txt",
		Size:          2048,
		LastModified:  now,
		ETag:          "\"def456\"",
		ContentType:   "text/plain",
		ContentLength: 2048,
		Metadata: map[string]string{
			"custom-key": "custom-value",
		},
	}

	if meta.Key != "path/to/object.txt" {
		t.Errorf("Key = %q, expected %q", meta.Key, "path/to/object.txt")
	}
	if meta.Size != 2048 {
		t.Errorf("Size = %d, expected %d", meta.Size, 2048)
	}
	if meta.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, expected %q", meta.ContentType, "text/plain")
	}
	if meta.Metadata["custom-key"] != "custom-value" {
		t.Errorf("Metadata[custom-key] = %q, expected %q", meta.Metadata["custom-key"], "custom-value")
	}
}

func TestObjectContent(t *testing.T) {
	now := time.Now()
	content := ObjectContent{
		Key:          "path/to/object.txt",
		Body:         []byte("Hello, World!"),
		ContentType:  "text/plain",
		Size:         13,
		LastModified: now,
		ETag:         "\"xyz789\"",
		Metadata: map[string]string{
			"author": "test",
		},
	}

	if content.Key != "path/to/object.txt" {
		t.Errorf("Key = %q, expected %q", content.Key, "path/to/object.txt")
	}
	if string(content.Body) != "Hello, World!" {
		t.Errorf("Body = %q, expected %q", string(content.Body), "Hello, World!")
	}
	if content.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, expected %q", content.ContentType, "text/plain")
	}
	if content.Size != 13 {
		t.Errorf("Size = %d, expected %d", content.Size, 13)
	}
}

func TestListObjectsOutput(t *testing.T) {
	output := ListObjectsOutput{
		Objects: []ObjectInfo{
			{Key: "file1.txt", Size: 100},
			{Key: "file2.txt", Size: 200},
		},
		CommonPrefixes:    []string{"folder1/", "folder2/"},
		IsTruncated:       true,
		NextContinueToken: "token123",
		KeyCount:          2,
	}

	if len(output.Objects) != 2 {
		t.Errorf("len(Objects) = %d, expected %d", len(output.Objects), 2)
	}
	if len(output.CommonPrefixes) != 2 {
		t.Errorf("len(CommonPrefixes) = %d, expected %d", len(output.CommonPrefixes), 2)
	}
	if !output.IsTruncated {
		t.Error("IsTruncated should be true")
	}
	if output.NextContinueToken != "token123" {
		t.Errorf("NextContinueToken = %q, expected %q", output.NextContinueToken, "token123")
	}
	if output.KeyCount != 2 {
		t.Errorf("KeyCount = %d, expected %d", output.KeyCount, 2)
	}
}

func TestPutObjectInput(t *testing.T) {
	input := PutObjectInput{
		Bucket:      "my-bucket",
		Key:         "path/to/file.txt",
		Body:        []byte("content"),
		ContentType: "text/plain",
		Metadata: map[string]string{
			"version": "1.0",
		},
	}

	if input.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q, expected %q", input.Bucket, "my-bucket")
	}
	if input.Key != "path/to/file.txt" {
		t.Errorf("Key = %q, expected %q", input.Key, "path/to/file.txt")
	}
	if string(input.Body) != "content" {
		t.Errorf("Body = %q, expected %q", string(input.Body), "content")
	}
	if input.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, expected %q", input.ContentType, "text/plain")
	}
}

func TestCopyObjectInput(t *testing.T) {
	input := CopyObjectInput{
		SourceBucket: "source-bucket",
		SourceKey:    "source/key.txt",
		DestBucket:   "dest-bucket",
		DestKey:      "dest/key.txt",
		Metadata: map[string]string{
			"copied": "true",
		},
	}

	if input.SourceBucket != "source-bucket" {
		t.Errorf("SourceBucket = %q, expected %q", input.SourceBucket, "source-bucket")
	}
	if input.SourceKey != "source/key.txt" {
		t.Errorf("SourceKey = %q, expected %q", input.SourceKey, "source/key.txt")
	}
	if input.DestBucket != "dest-bucket" {
		t.Errorf("DestBucket = %q, expected %q", input.DestBucket, "dest-bucket")
	}
	if input.DestKey != "dest/key.txt" {
		t.Errorf("DestKey = %q, expected %q", input.DestKey, "dest/key.txt")
	}
}

func TestPresignedURL(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	url := PresignedURL{
		URL:       "https://bucket.s3.amazonaws.com/key?signature=...",
		Method:    "GET",
		ExpiresAt: expiresAt,
	}

	if url.URL == "" {
		t.Error("URL should not be empty")
	}
	if url.Method != "GET" {
		t.Errorf("Method = %q, expected %q", url.Method, "GET")
	}
	if !url.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, expected %v", url.ExpiresAt, expiresAt)
	}
}

// Tests using mocks

func TestClient_ListBuckets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Now()
		mock := &mockS3API{
			listBucketsFunc: func(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
				return &s3.ListBucketsOutput{
					Buckets: []types.Bucket{
						{Name: aws.String("bucket-1"), CreationDate: &now},
						{Name: aws.String("bucket-2"), CreationDate: &now},
					},
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		buckets, err := client.ListBuckets(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(buckets) != 2 {
			t.Errorf("expected 2 buckets, got %d", len(buckets))
		}
		if buckets[0].Name != "bucket-1" {
			t.Errorf("expected bucket-1, got %s", buckets[0].Name)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			listBucketsFunc: func(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
				return nil, errors.New("access denied")
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.ListBuckets(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to list buckets") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestClient_ListObjects(t *testing.T) {
	t.Run("success with objects", func(t *testing.T) {
		now := time.Now()
		mock := &mockS3API{
			listObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{Key: aws.String("file1.txt"), Size: aws.Int64(1024), LastModified: &now, ETag: aws.String("\"etag1\"")},
						{Key: aws.String("file2.txt"), Size: aws.Int64(2048), LastModified: &now, ETag: aws.String("\"etag2\"")},
					},
					CommonPrefixes:        []types.CommonPrefix{{Prefix: aws.String("folder/")}},
					IsTruncated:           aws.Bool(true),
					NextContinuationToken: aws.String("token123"),
					KeyCount:              aws.Int32(2),
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		result, err := client.ListObjects(context.Background(), "my-bucket", "prefix/", "/", 100, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Objects) != 2 {
			t.Errorf("expected 2 objects, got %d", len(result.Objects))
		}
		if len(result.CommonPrefixes) != 1 {
			t.Errorf("expected 1 common prefix, got %d", len(result.CommonPrefixes))
		}
		if !result.IsTruncated {
			t.Error("expected IsTruncated to be true")
		}
		if result.NextContinueToken != "token123" {
			t.Errorf("expected token123, got %s", result.NextContinueToken)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			listObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return nil, errors.New("bucket not found")
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.ListObjects(context.Background(), "my-bucket", "", "", 0, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_GetObject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Now()
		mock := &mockS3API{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body:          io.NopCloser(strings.NewReader("Hello, World!")),
					ContentType:   aws.String("text/plain"),
					ContentLength: aws.Int64(13),
					ETag:          aws.String("\"etag123\""),
					LastModified:  &now,
					Metadata:      map[string]string{"custom": "value"},
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		result, err := client.GetObject(context.Background(), "my-bucket", "file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result.Body) != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %s", string(result.Body))
		}
		if result.ContentType != "text/plain" {
			t.Errorf("expected text/plain, got %s", result.ContentType)
		}
		if result.Size != 13 {
			t.Errorf("expected size 13, got %d", result.Size)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("object not found")
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.GetObject(context.Background(), "my-bucket", "missing.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_GetObjectMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Now()
		mock := &mockS3API{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{
					ContentType:   aws.String("application/json"),
					ContentLength: aws.Int64(256),
					ETag:          aws.String("\"etag456\""),
					LastModified:  &now,
					Metadata:      map[string]string{"version": "1.0"},
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		result, err := client.GetObjectMetadata(context.Background(), "my-bucket", "data.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ContentType != "application/json" {
			t.Errorf("expected application/json, got %s", result.ContentType)
		}
		if result.Size != 256 {
			t.Errorf("expected size 256, got %d", result.Size)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("not found")
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.GetObjectMetadata(context.Background(), "my-bucket", "missing.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_PutObject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockS3API{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				// Verify input parameters
				if aws.ToString(params.Bucket) != "my-bucket" {
					t.Errorf("expected bucket my-bucket, got %s", aws.ToString(params.Bucket))
				}
				if aws.ToString(params.Key) != "new-file.txt" {
					t.Errorf("expected key new-file.txt, got %s", aws.ToString(params.Key))
				}
				return &s3.PutObjectOutput{
					ETag:      aws.String("\"newetag\""),
					VersionId: aws.String("v1"),
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		result, err := client.PutObject(context.Background(), &PutObjectInput{
			Bucket:      "my-bucket",
			Key:         "new-file.txt",
			Body:        []byte("new content"),
			ContentType: "text/plain",
			Metadata:    map[string]string{"author": "test"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ETag != "\"newetag\"" {
			t.Errorf("expected etag '\"newetag\"', got %s", result.ETag)
		}
		if result.VersionID != "v1" {
			t.Errorf("expected version v1, got %s", result.VersionID)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("access denied")
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.PutObject(context.Background(), &PutObjectInput{
			Bucket: "my-bucket",
			Key:    "file.txt",
			Body:   []byte("content"),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_DeleteObject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockS3API{
			deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				if aws.ToString(params.Bucket) != "my-bucket" {
					t.Errorf("expected bucket my-bucket, got %s", aws.ToString(params.Bucket))
				}
				if aws.ToString(params.Key) != "delete-me.txt" {
					t.Errorf("expected key delete-me.txt, got %s", aws.ToString(params.Key))
				}
				return &s3.DeleteObjectOutput{}, nil
			},
		}
		client := newMockClient(mock, nil)

		err := client.DeleteObject(context.Background(), "my-bucket", "delete-me.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("access denied")
			},
		}
		client := newMockClient(mock, nil)

		err := client.DeleteObject(context.Background(), "my-bucket", "file.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_CopyObject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Now()
		mock := &mockS3API{
			copyObjectFunc: func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
				return &s3.CopyObjectOutput{
					CopyObjectResult: &types.CopyObjectResult{
						ETag:         aws.String("\"copyetag\""),
						LastModified: &now,
					},
					VersionId: aws.String("v2"),
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		result, err := client.CopyObject(context.Background(), &CopyObjectInput{
			SourceBucket: "source-bucket",
			SourceKey:    "source.txt",
			DestBucket:   "dest-bucket",
			DestKey:      "dest.txt",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ETag != "\"copyetag\"" {
			t.Errorf("expected etag '\"copyetag\"', got %s", result.ETag)
		}
		if result.VersionID != "v2" {
			t.Errorf("expected version v2, got %s", result.VersionID)
		}
	})

	t.Run("with metadata", func(t *testing.T) {
		mock := &mockS3API{
			copyObjectFunc: func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
				if params.MetadataDirective != types.MetadataDirectiveReplace {
					t.Error("expected MetadataDirectiveReplace when metadata provided")
				}
				return &s3.CopyObjectOutput{
					CopyObjectResult: &types.CopyObjectResult{},
				}, nil
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.CopyObject(context.Background(), &CopyObjectInput{
			SourceBucket: "bucket",
			SourceKey:    "source.txt",
			DestBucket:   "bucket",
			DestKey:      "dest.txt",
			Metadata:     map[string]string{"new-key": "new-value"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockS3API{
			copyObjectFunc: func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
				return nil, errors.New("source not found")
			},
		}
		client := newMockClient(mock, nil)

		_, err := client.CopyObject(context.Background(), &CopyObjectInput{
			SourceBucket: "bucket",
			SourceKey:    "missing.txt",
			DestBucket:   "bucket",
			DestKey:      "dest.txt",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_PresignGetURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		presignMock := &mockPresignAPI{
			presignGetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error) {
				return &s3.PresignedHTTPRequest{
					URL:    "https://bucket.s3.amazonaws.com/key?X-Amz-Signature=...",
					Method: "GET",
				}, nil
			},
		}
		client := newMockClient(nil, presignMock)

		result, err := client.PresignGetURL(context.Background(), "my-bucket", "file.txt", time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.URL == "" {
			t.Error("expected non-empty URL")
		}
		if result.Method != "GET" {
			t.Errorf("expected GET, got %s", result.Method)
		}
	})

	t.Run("error", func(t *testing.T) {
		presignMock := &mockPresignAPI{
			presignGetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error) {
				return nil, errors.New("presign failed")
			},
		}
		client := newMockClient(nil, presignMock)

		_, err := client.PresignGetURL(context.Background(), "bucket", "key", time.Hour)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_PresignPutURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		presignMock := &mockPresignAPI{
			presignPutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error) {
				return &s3.PresignedHTTPRequest{
					URL:    "https://bucket.s3.amazonaws.com/key?X-Amz-Signature=...",
					Method: "PUT",
				}, nil
			},
		}
		client := newMockClient(nil, presignMock)

		result, err := client.PresignPutURL(context.Background(), "my-bucket", "upload.txt", time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.URL == "" {
			t.Error("expected non-empty URL")
		}
		if result.Method != "PUT" {
			t.Errorf("expected PUT, got %s", result.Method)
		}
	})

	t.Run("error", func(t *testing.T) {
		presignMock := &mockPresignAPI{
			presignPutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error) {
				return nil, errors.New("presign failed")
			},
		}
		client := newMockClient(nil, presignMock)

		_, err := client.PresignPutURL(context.Background(), "bucket", "key", time.Hour)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_contextWithTimeout(t *testing.T) {
	t.Run("applies timeout", func(t *testing.T) {
		client := &Client{
			config: &Config{Timeout: 5 * time.Second},
		}

		ctx, cancel := client.contextWithTimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected context to have deadline")
		}
		if time.Until(deadline) > 5*time.Second {
			t.Error("deadline should be within 5 seconds")
		}
	})

	t.Run("no timeout when zero", func(t *testing.T) {
		client := &Client{
			config: &Config{Timeout: 0},
		}

		ctx, cancel := client.contextWithTimeout(context.Background())
		defer cancel()

		_, ok := ctx.Deadline()
		if ok {
			t.Error("expected no deadline when timeout is 0")
		}
	})

	t.Run("respects existing shorter deadline", func(t *testing.T) {
		client := &Client{
			config: &Config{Timeout: 10 * time.Second},
		}

		parentCtx, parentCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer parentCancel()

		ctx, cancel := client.contextWithTimeout(parentCtx)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected context to have deadline")
		}
		if time.Until(deadline) > 2*time.Second {
			t.Error("should respect parent's shorter deadline")
		}
	})
}
