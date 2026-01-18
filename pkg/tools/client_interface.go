package tools

import (
	"context"
	"time"

	"github.com/txn2/mcp-s3/pkg/client"
)

// S3Client defines the interface for S3 operations used by MCP tools.
// This interface allows for easy mocking in tests.
type S3Client interface {
	// ConnectionName returns the configured connection name.
	ConnectionName() string

	// Config returns a copy of the client configuration.
	Config() *client.Config

	// ListBuckets returns a list of all buckets accessible to the client.
	ListBuckets(ctx context.Context) ([]client.BucketInfo, error)

	// ListObjects lists objects in a bucket with optional prefix, delimiter, and pagination.
	ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string) (*client.ListObjectsOutput, error)

	// GetObject retrieves an object's content from S3.
	GetObject(ctx context.Context, bucket, key string) (*client.ObjectContent, error)

	// GetObjectMetadata retrieves an object's metadata without downloading the content.
	GetObjectMetadata(ctx context.Context, bucket, key string) (*client.ObjectMetadata, error)

	// PutObject uploads an object to S3.
	PutObject(ctx context.Context, input *client.PutObjectInput) (*client.PutObjectOutput, error)

	// DeleteObject deletes an object from S3.
	DeleteObject(ctx context.Context, bucket, key string) error

	// CopyObject copies an object within or between buckets.
	CopyObject(ctx context.Context, input *client.CopyObjectInput) (*client.CopyObjectOutput, error)

	// PresignGetURL generates a presigned URL for downloading an object.
	PresignGetURL(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error)

	// PresignPutURL generates a presigned URL for uploading an object.
	PresignPutURL(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error)

	// Close closes the S3 client and releases resources.
	Close() error
}

// Ensure client.Client implements S3Client.
var _ S3Client = (*client.Client)(nil)
