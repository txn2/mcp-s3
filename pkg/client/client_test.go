package client

import (
	"context"
	"testing"
	"time"
)

// Note: Most tests require integration with a real S3 or SeaweedFS instance.
// These tests cover the non-AWS-dependent parts of the client.

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
