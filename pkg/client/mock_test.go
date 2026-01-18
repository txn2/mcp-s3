package client

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// mockS3API is a mock implementation of S3API for testing.
type mockS3API struct {
	listBucketsFunc   func(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	listObjectsV2Func func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	getObjectFunc     func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	headObjectFunc    func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	putObjectFunc     func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	deleteObjectFunc  func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	copyObjectFunc    func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

func (m *mockS3API) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	if m.listBucketsFunc != nil {
		return m.listBucketsFunc(ctx, params, optFns...)
	}
	return &s3.ListBucketsOutput{}, nil
}

func (m *mockS3API) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.listObjectsV2Func != nil {
		return m.listObjectsV2Func(ctx, params, optFns...)
	}
	return &s3.ListObjectsV2Output{}, nil
}

func (m *mockS3API) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader("")),
	}, nil
}

func (m *mockS3API) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

func (m *mockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3API) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3API) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	if m.copyObjectFunc != nil {
		return m.copyObjectFunc(ctx, params, optFns...)
	}
	return &s3.CopyObjectOutput{}, nil
}

// mockPresignAPI is a mock implementation of PresignAPI for testing.
type mockPresignAPI struct {
	presignGetObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error)
	presignPutObjectFunc func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error)
}

func (m *mockPresignAPI) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error) {
	if m.presignGetObjectFunc != nil {
		return m.presignGetObjectFunc(ctx, params, optFns...)
	}
	return &s3.PresignedHTTPRequest{
		URL:    "https://example.com/presigned",
		Method: "GET",
	}, nil
}

func (m *mockPresignAPI) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedHTTPRequest, error) {
	if m.presignPutObjectFunc != nil {
		return m.presignPutObjectFunc(ctx, params, optFns...)
	}
	return &s3.PresignedHTTPRequest{
		URL:    "https://example.com/presigned",
		Method: "PUT",
	}, nil
}

// newMockClient creates a Client with mock S3 and presign APIs for testing.
func newMockClient(s3api *mockS3API, presignAPI *mockPresignAPI) *Client {
	if s3api == nil {
		s3api = &mockS3API{}
	}
	if presignAPI == nil {
		presignAPI = &mockPresignAPI{}
	}
	return &Client{
		s3Client:       s3api,
		presignClient:  presignAPI,
		config:         &Config{Timeout: 30 * time.Second},
		connectionName: "mock",
	}
}

// Helper functions for creating test data

func ptrTime(t time.Time) *time.Time {
	return &t
}

func ptrString(s string) *string {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrInt32(i int32) *int32 {
	return &i
}

func ptrBool(b bool) *bool {
	return &b
}

func testBuckets() []types.Bucket {
	now := time.Now()
	return []types.Bucket{
		{Name: aws.String("bucket-1"), CreationDate: &now},
		{Name: aws.String("bucket-2"), CreationDate: &now},
	}
}

func testObjects() []types.Object {
	now := time.Now()
	return []types.Object{
		{
			Key:          aws.String("file1.txt"),
			Size:         aws.Int64(1024),
			LastModified: &now,
			ETag:         aws.String("\"etag1\""),
			StorageClass: types.ObjectStorageClassStandard,
		},
		{
			Key:          aws.String("file2.txt"),
			Size:         aws.Int64(2048),
			LastModified: &now,
			ETag:         aws.String("\"etag2\""),
			StorageClass: types.ObjectStorageClassStandard,
		},
	}
}
