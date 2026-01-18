package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Client wraps the AWS S3 SDK client with convenience methods.
type Client struct {
	s3Client       S3API
	presignClient  PresignAPI
	config         *Config
	connectionName string
}

// BucketInfo contains information about an S3 bucket.
type BucketInfo struct {
	Name         string
	CreationDate time.Time
}

// ObjectInfo contains information about an S3 object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	StorageClass string
}

// ObjectMetadata contains metadata about an S3 object (from HEAD request).
type ObjectMetadata struct {
	Key           string
	Size          int64
	LastModified  time.Time
	ETag          string
	ContentType   string
	ContentLength int64
	Metadata      map[string]string
}

// ObjectContent contains the content and metadata of an S3 object.
type ObjectContent struct {
	Key          string
	Body         []byte
	ContentType  string
	Size         int64
	LastModified time.Time
	ETag         string
	Metadata     map[string]string
}

// ListObjectsOutput contains the result of listing objects.
type ListObjectsOutput struct {
	Objects           []ObjectInfo
	CommonPrefixes    []string
	IsTruncated       bool
	NextContinueToken string
	KeyCount          int32
}

// PutObjectInput contains the parameters for uploading an object.
type PutObjectInput struct {
	Bucket      string
	Key         string
	Body        []byte
	ContentType string
	Metadata    map[string]string
}

// PutObjectOutput contains the result of uploading an object.
type PutObjectOutput struct {
	ETag      string
	VersionID string
}

// CopyObjectInput contains the parameters for copying an object.
type CopyObjectInput struct {
	SourceBucket string
	SourceKey    string
	DestBucket   string
	DestKey      string
	Metadata     map[string]string
}

// CopyObjectOutput contains the result of copying an object.
type CopyObjectOutput struct {
	ETag         string
	LastModified time.Time
	VersionID    string
}

// PresignedURL contains information about a presigned URL.
type PresignedURL struct {
	URL       string
	Method    string
	ExpiresAt time.Time
}

// New creates a new S3 client with the given configuration.
func New(ctx context.Context, cfg *Config) (*Client, error) {
	// Sanitize AWS environment variables before the SDK reads them.
	// This must happen before config.LoadDefaultConfig() as the AWS SDK
	// reads environment variables directly, bypassing our sanitization.
	SanitizeAWSEnvVars()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Build AWS config options
	var opts []func(*config.LoadOptions) error

	// Set region
	opts = append(opts, config.WithRegion(cfg.Region))

	// Set profile if specified
	if cfg.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Set explicit credentials if provided
	if cfg.HasCredentials() {
		staticCreds := credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			cfg.SessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(staticCreds))
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Build S3 client options
	var s3Opts []func(*s3.Options)

	// Set custom endpoint if specified
	if cfg.HasEndpoint() {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.UsePathStyle
		})
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg, s3Opts...)

	// Create presign client
	presignClient := s3.NewPresignClient(s3Client)

	return &Client{
		s3Client:       s3Client,
		presignClient:  presignClient,
		config:         cfg.Clone(),
		connectionName: cfg.Name,
	}, nil
}

// ConnectionName returns the configured connection name.
func (c *Client) ConnectionName() string {
	return c.connectionName
}

// Config returns a copy of the client configuration.
func (c *Client) Config() *Config {
	return c.config.Clone()
}

// ListBuckets returns a list of all buckets accessible to the client.
func (c *Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	output, err := c.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	buckets := make([]BucketInfo, 0, len(output.Buckets))
	for _, b := range output.Buckets {
		bucket := BucketInfo{
			Name: aws.ToString(b.Name),
		}
		if b.CreationDate != nil {
			bucket.CreationDate = *b.CreationDate
		}
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// ListObjects lists objects in a bucket with optional prefix, delimiter, and pagination.
func (c *Client) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string) (*ListObjectsOutput, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if delimiter != "" {
		input.Delimiter = aws.String(delimiter)
	}
	if maxKeys > 0 {
		input.MaxKeys = aws.Int32(maxKeys)
	}
	if continueToken != "" {
		input.ContinuationToken = aws.String(continueToken)
	}

	output, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	result := &ListObjectsOutput{
		Objects:        make([]ObjectInfo, 0, len(output.Contents)),
		CommonPrefixes: make([]string, 0, len(output.CommonPrefixes)),
		IsTruncated:    aws.ToBool(output.IsTruncated),
		KeyCount:       aws.ToInt32(output.KeyCount),
	}

	if output.NextContinuationToken != nil {
		result.NextContinueToken = *output.NextContinuationToken
	}

	for _, obj := range output.Contents {
		info := ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
		}
		if obj.LastModified != nil {
			info.LastModified = *obj.LastModified
		}
		result.Objects = append(result.Objects, info)
	}

	for _, cp := range output.CommonPrefixes {
		result.CommonPrefixes = append(result.CommonPrefixes, aws.ToString(cp.Prefix))
	}

	return result, nil
}

// GetObject retrieves an object's content from S3.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (*ObjectContent, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer func() { _ = output.Body.Close() }()

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	result := &ObjectContent{
		Key:         key,
		Body:        body,
		ContentType: aws.ToString(output.ContentType),
		Size:        aws.ToInt64(output.ContentLength),
		ETag:        aws.ToString(output.ETag),
		Metadata:    output.Metadata,
	}
	if output.LastModified != nil {
		result.LastModified = *output.LastModified
	}

	return result, nil
}

// GetObjectMetadata retrieves an object's metadata without downloading the content.
func (c *Client) GetObjectMetadata(ctx context.Context, bucket, key string) (*ObjectMetadata, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	output, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	result := &ObjectMetadata{
		Key:           key,
		Size:          aws.ToInt64(output.ContentLength),
		ETag:          aws.ToString(output.ETag),
		ContentType:   aws.ToString(output.ContentType),
		ContentLength: aws.ToInt64(output.ContentLength),
		Metadata:      output.Metadata,
	}
	if output.LastModified != nil {
		result.LastModified = *output.LastModified
	}

	return result, nil
}

// PutObject uploads an object to S3.
func (c *Client) PutObject(ctx context.Context, input *PutObjectInput) (*PutObjectOutput, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	s3Input := &s3.PutObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
		Body:   bytes.NewReader(input.Body),
	}

	if input.ContentType != "" {
		s3Input.ContentType = aws.String(input.ContentType)
	}
	if len(input.Metadata) > 0 {
		s3Input.Metadata = input.Metadata
	}

	output, err := c.s3Client.PutObject(ctx, s3Input)
	if err != nil {
		return nil, fmt.Errorf("failed to put object: %w", err)
	}

	return &PutObjectOutput{
		ETag:      aws.ToString(output.ETag),
		VersionID: aws.ToString(output.VersionId),
	}, nil
}

// DeleteObject deletes an object from S3.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// CopyObject copies an object within or between buckets.
func (c *Client) CopyObject(ctx context.Context, input *CopyObjectInput) (*CopyObjectOutput, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	copySource := fmt.Sprintf("%s/%s", input.SourceBucket, input.SourceKey)

	s3Input := &s3.CopyObjectInput{
		Bucket:     aws.String(input.DestBucket),
		Key:        aws.String(input.DestKey),
		CopySource: aws.String(copySource),
	}

	if len(input.Metadata) > 0 {
		s3Input.Metadata = input.Metadata
		s3Input.MetadataDirective = types.MetadataDirectiveReplace
	}

	output, err := c.s3Client.CopyObject(ctx, s3Input)
	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	result := &CopyObjectOutput{
		VersionID: aws.ToString(output.VersionId),
	}

	if output.CopyObjectResult != nil {
		result.ETag = aws.ToString(output.CopyObjectResult.ETag)
		if output.CopyObjectResult.LastModified != nil {
			result.LastModified = *output.CopyObjectResult.LastModified
		}
	}

	return result, nil
}

// PresignGetURL generates a presigned URL for downloading an object.
func (c *Client) PresignGetURL(ctx context.Context, bucket, key string, expires time.Duration) (*PresignedURL, error) {
	presignedReq, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return nil, fmt.Errorf("failed to presign GET URL: %w", err)
	}

	return &PresignedURL{
		URL:       presignedReq.URL,
		Method:    presignedReq.Method,
		ExpiresAt: time.Now().Add(expires),
	}, nil
}

// PresignPutURL generates a presigned URL for uploading an object.
func (c *Client) PresignPutURL(ctx context.Context, bucket, key string, expires time.Duration) (*PresignedURL, error) {
	presignedReq, err := c.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return nil, fmt.Errorf("failed to presign PUT URL: %w", err)
	}

	return &PresignedURL{
		URL:       presignedReq.URL,
		Method:    presignedReq.Method,
		ExpiresAt: time.Now().Add(expires),
	}, nil
}

// contextWithTimeout returns a context with the configured timeout.
// If the parent context already has a deadline that is sooner, it uses that instead.
func (c *Client) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.config.Timeout <= 0 {
		return ctx, func() {}
	}

	// Check if parent context already has a sooner deadline
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) < c.config.Timeout {
			return ctx, func() {}
		}
	}

	return context.WithTimeout(ctx, c.config.Timeout)
}

// Close closes the S3 client and releases resources.
// Currently a no-op as the AWS SDK manages its own connection pool.
func (c *Client) Close() error {
	return nil
}
