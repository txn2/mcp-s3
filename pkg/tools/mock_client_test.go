package tools

import (
	"context"
	"time"

	"github.com/txn2/mcp-s3/pkg/client"
)

// MockS3Client is a mock implementation of S3Client for testing.
type MockS3Client struct {
	connectionName string
	config         *client.Config

	// Mock data
	Buckets  []client.BucketInfo
	Objects  map[string]map[string]*client.ObjectContent // bucket -> key -> content
	Metadata map[string]map[string]*client.ObjectMetadata

	// Mock behaviors
	ListBucketsFunc       func(ctx context.Context) ([]client.BucketInfo, error)
	ListObjectsFunc       func(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string) (*client.ListObjectsOutput, error)
	GetObjectFunc         func(ctx context.Context, bucket, key string) (*client.ObjectContent, error)
	GetObjectMetadataFunc func(ctx context.Context, bucket, key string) (*client.ObjectMetadata, error)
	PutObjectFunc         func(ctx context.Context, input *client.PutObjectInput) (*client.PutObjectOutput, error)
	DeleteObjectFunc      func(ctx context.Context, bucket, key string) error
	CopyObjectFunc        func(ctx context.Context, input *client.CopyObjectInput) (*client.CopyObjectOutput, error)
	PresignGetURLFunc     func(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error)
	PresignPutURLFunc     func(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error)
}

// NewMockS3Client creates a new mock S3 client with the given connection name.
func NewMockS3Client(connectionName string) *MockS3Client {
	return &MockS3Client{
		connectionName: connectionName,
		config: &client.Config{
			Region: "us-east-1",
			Name:   connectionName,
		},
		Buckets:  make([]client.BucketInfo, 0),
		Objects:  make(map[string]map[string]*client.ObjectContent),
		Metadata: make(map[string]map[string]*client.ObjectMetadata),
	}
}

// ConnectionName returns the configured connection name.
func (m *MockS3Client) ConnectionName() string {
	return m.connectionName
}

// Config returns a copy of the client configuration.
func (m *MockS3Client) Config() *client.Config {
	return m.config.Clone()
}

// ListBuckets returns a list of all buckets accessible to the client.
func (m *MockS3Client) ListBuckets(ctx context.Context) ([]client.BucketInfo, error) {
	if m.ListBucketsFunc != nil {
		return m.ListBucketsFunc(ctx)
	}
	return m.Buckets, nil
}

// ListObjects lists objects in a bucket with optional prefix, delimiter, and pagination.
func (m *MockS3Client) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string) (*client.ListObjectsOutput, error) {
	if m.ListObjectsFunc != nil {
		return m.ListObjectsFunc(ctx, bucket, prefix, delimiter, maxKeys, continueToken)
	}

	bucketObjects, ok := m.Objects[bucket]
	if !ok {
		return &client.ListObjectsOutput{
			Objects: []client.ObjectInfo{},
		}, nil
	}

	objects := make([]client.ObjectInfo, 0)
	for key, content := range bucketObjects {
		// Apply prefix filter
		if prefix != "" && len(key) < len(prefix) {
			continue
		}
		if prefix != "" && key[:len(prefix)] != prefix {
			continue
		}

		objects = append(objects, client.ObjectInfo{
			Key:          key,
			Size:         content.Size,
			LastModified: content.LastModified,
			ETag:         content.ETag,
		})
	}

	return &client.ListObjectsOutput{
		Objects:  objects,
		KeyCount: int32(len(objects)),
	}, nil
}

// GetObject retrieves an object's content from S3.
func (m *MockS3Client) GetObject(ctx context.Context, bucket, key string) (*client.ObjectContent, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(ctx, bucket, key)
	}

	bucketObjects, ok := m.Objects[bucket]
	if !ok {
		return nil, ErrNotFound
	}

	content, ok := bucketObjects[key]
	if !ok {
		return nil, ErrNotFound
	}

	return content, nil
}

// GetObjectMetadata retrieves an object's metadata without downloading the content.
func (m *MockS3Client) GetObjectMetadata(ctx context.Context, bucket, key string) (*client.ObjectMetadata, error) {
	if m.GetObjectMetadataFunc != nil {
		return m.GetObjectMetadataFunc(ctx, bucket, key)
	}

	bucketMeta, ok := m.Metadata[bucket]
	if !ok {
		return nil, ErrNotFound
	}

	meta, ok := bucketMeta[key]
	if !ok {
		return nil, ErrNotFound
	}

	return meta, nil
}

// PutObject uploads an object to S3.
func (m *MockS3Client) PutObject(ctx context.Context, input *client.PutObjectInput) (*client.PutObjectOutput, error) {
	if m.PutObjectFunc != nil {
		return m.PutObjectFunc(ctx, input)
	}

	if m.Objects[input.Bucket] == nil {
		m.Objects[input.Bucket] = make(map[string]*client.ObjectContent)
	}

	m.Objects[input.Bucket][input.Key] = &client.ObjectContent{
		Key:          input.Key,
		Body:         input.Body,
		ContentType:  input.ContentType,
		Size:         int64(len(input.Body)),
		LastModified: time.Now(),
		ETag:         "\"mock-etag\"",
		Metadata:     input.Metadata,
	}

	return &client.PutObjectOutput{
		ETag: "\"mock-etag\"",
	}, nil
}

// DeleteObject deletes an object from S3.
func (m *MockS3Client) DeleteObject(ctx context.Context, bucket, key string) error {
	if m.DeleteObjectFunc != nil {
		return m.DeleteObjectFunc(ctx, bucket, key)
	}

	if bucketObjects, ok := m.Objects[bucket]; ok {
		delete(bucketObjects, key)
	}

	return nil
}

// CopyObject copies an object within or between buckets.
func (m *MockS3Client) CopyObject(ctx context.Context, input *client.CopyObjectInput) (*client.CopyObjectOutput, error) {
	if m.CopyObjectFunc != nil {
		return m.CopyObjectFunc(ctx, input)
	}

	sourceBucket, ok := m.Objects[input.SourceBucket]
	if !ok {
		return nil, ErrNotFound
	}

	sourceObj, ok := sourceBucket[input.SourceKey]
	if !ok {
		return nil, ErrNotFound
	}

	if m.Objects[input.DestBucket] == nil {
		m.Objects[input.DestBucket] = make(map[string]*client.ObjectContent)
	}

	m.Objects[input.DestBucket][input.DestKey] = &client.ObjectContent{
		Key:          input.DestKey,
		Body:         sourceObj.Body,
		ContentType:  sourceObj.ContentType,
		Size:         sourceObj.Size,
		LastModified: time.Now(),
		ETag:         "\"mock-copy-etag\"",
		Metadata:     input.Metadata,
	}

	return &client.CopyObjectOutput{
		ETag:         "\"mock-copy-etag\"",
		LastModified: time.Now(),
	}, nil
}

// PresignGetURL generates a presigned URL for downloading an object.
func (m *MockS3Client) PresignGetURL(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error) {
	if m.PresignGetURLFunc != nil {
		return m.PresignGetURLFunc(ctx, bucket, key, expires)
	}

	return &client.PresignedURL{
		URL:       "https://mock-bucket.s3.amazonaws.com/" + key + "?presigned=get",
		Method:    "GET",
		ExpiresAt: time.Now().Add(expires),
	}, nil
}

// PresignPutURL generates a presigned URL for uploading an object.
func (m *MockS3Client) PresignPutURL(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error) {
	if m.PresignPutURLFunc != nil {
		return m.PresignPutURLFunc(ctx, bucket, key, expires)
	}

	return &client.PresignedURL{
		URL:       "https://mock-bucket.s3.amazonaws.com/" + key + "?presigned=put",
		Method:    "PUT",
		ExpiresAt: time.Now().Add(expires),
	}, nil
}

// Close closes the S3 client and releases resources.
func (m *MockS3Client) Close() error {
	return nil
}

// AddBucket adds a bucket to the mock client.
func (m *MockS3Client) AddBucket(name string, creationDate time.Time) {
	m.Buckets = append(m.Buckets, client.BucketInfo{
		Name:         name,
		CreationDate: creationDate,
	})
}

// AddObject adds an object to the mock client.
func (m *MockS3Client) AddObject(bucket, key string, content []byte, contentType string) {
	if m.Objects[bucket] == nil {
		m.Objects[bucket] = make(map[string]*client.ObjectContent)
	}

	m.Objects[bucket][key] = &client.ObjectContent{
		Key:          key,
		Body:         content,
		ContentType:  contentType,
		Size:         int64(len(content)),
		LastModified: time.Now(),
		ETag:         "\"mock-etag\"",
	}

	if m.Metadata[bucket] == nil {
		m.Metadata[bucket] = make(map[string]*client.ObjectMetadata)
	}

	m.Metadata[bucket][key] = &client.ObjectMetadata{
		Key:           key,
		Size:          int64(len(content)),
		ContentType:   contentType,
		ContentLength: int64(len(content)),
		LastModified:  time.Now(),
		ETag:          "\"mock-etag\"",
	}
}

// Ensure MockS3Client implements S3Client.
var _ S3Client = (*MockS3Client)(nil)
