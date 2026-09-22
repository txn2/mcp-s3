package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Client wraps the AWS S3 SDK client with convenience methods.
type Client struct {
	s3Client       S3API
	presignClient  PresignAPI
	uploader       ObjectUploader
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
	Key         string
	Body        []byte
	ContentType string
	// Size is the whole object's length. From GetObjectRange it can exceed
	// len(Body).
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

// clearUnresolvedAWSEnvVars removes AWS environment variables that contain
// unresolved template variables. This prevents the AWS SDK from trying to
// use invalid values like "${user_config.aws_profile}" as profile names.
//
// The AWS SDK's LoadDefaultConfig automatically reads these env vars,
// bypassing any application-level configuration.
func clearUnresolvedAWSEnvVars() {
	awsEnvVars := []string{
		"AWS_PROFILE",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
	}
	for _, key := range awsEnvVars {
		if value := os.Getenv(key); isUnresolvedTemplateVar(value) {
			_ = os.Unsetenv(key)
		}
	}
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

	// Clear unresolved template variables from AWS env vars before SDK reads them
	clearUnresolvedAWSEnvVars()

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

	// Create presign client. When a dedicated presign endpoint is configured,
	// sign URLs against a client pointed at that public-facing endpoint so the
	// URLs are reachable outside the cluster; data operations keep using
	// s3Client (the internal Endpoint). With no presign endpoint the two are the
	// same client, preserving prior behavior.
	presignSource := s3Client
	if cfg.PresignEndpoint != "" {
		presignSource = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.PresignEndpoint)
			o.UsePathStyle = cfg.UsePathStyle
		})
	}
	presignClient := s3.NewPresignClient(presignSource)

	// Create the streaming/multipart uploader. It shares the same underlying
	// S3 client so it honors the configured endpoint, credentials, and region.
	uploader := transfermanager.New(s3Client)

	return &Client{
		s3Client:       s3Client,
		presignClient:  presignClient,
		uploader:       uploader,
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
func (c *Client) ListObjects(
	ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string,
) (*ListObjectsOutput, error) {
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
	d := c.startRead(ctx)
	defer d.stop()

	output, err := c.s3Client.GetObject(d.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", d.explain(err))
	}
	defer func() { _ = output.Body.Close() }()

	body, err := io.ReadAll(d.body(output.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", d.explain(err))
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

// ErrInvalidRange is returned by GetObjectRange for an offset or length no
// byte range can be built from. No request is made.
var ErrInvalidRange = errors.New("invalid byte range")

// ErrRangeNotSatisfiable is returned by GetObjectRange when the store answers
// 416: the offset is at or past the end of the object, which includes every
// offset of a zero-byte object.
var ErrRangeNotSatisfiable = errors.New("byte range not satisfiable")

// GetObjectRange reads length bytes of an object starting at offset.
//
// The returned Size is the object's TOTAL size, read from the response's
// Content-Range, not the length of Body: a caller reading the tail of a file
// learns how large the file is from the same call. Size is -1 when the store
// reports no total ("bytes 0-99/*"). Body may be shorter than length when the
// range runs past the end of the object.
//
// A store that ignores Range answers with the whole object. Body is still the
// bytes at offset, at most length of them: the bytes before offset are read
// and discarded, never held.
//
// A zero-byte object has no byte to start a range at, and S3 refuses any
// ranged GET of one with 416 InvalidRange. That is returned as
// ErrRangeNotSatisfiable, the same error as an offset past the end of a
// non-empty object, and not as an empty Body: an empty Body with Size 0 would
// be indistinguishable from a read that failed. A caller that needs to tell
// the two apart asks GetObjectMetadata for the size.
func (c *Client) GetObjectRange(ctx context.Context, bucket, key string, offset, length int64) (*ObjectContent, error) {
	// The last clause keeps offset+length-1 inside int64: a wrapped end is a
	// Range the store cannot parse, and a store ignores one of those and
	// answers with the whole object.
	if offset < 0 || length <= 0 || offset > math.MaxInt64-length {
		return nil, fmt.Errorf("%w: offset %d, length %d", ErrInvalidRange, offset, length)
	}

	d := c.startRead(ctx)
	defer d.stop()

	output, err := c.s3Client.GetObject(d.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String("bytes=" + strconv.FormatInt(offset, 10) + "-" + strconv.FormatInt(offset+length-1, 10)),
	})
	if err != nil {
		// The SDK wraps a non-2xx response in smithy-go's
		// transport/http.ResponseError, which carries the status code.
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusRequestedRangeNotSatisfiable {
			return nil, fmt.Errorf("failed to get object range: %w: %w", ErrRangeNotSatisfiable, err)
		}
		return nil, fmt.Errorf("failed to get object range: %w", d.explain(err))
	}
	defer func() { _ = output.Body.Close() }()

	bodyReader := d.body(output.Body)
	size, err := seekRangeStart(output, bodyReader, offset)
	if err != nil {
		return nil, d.explain(err)
	}

	// Bounded by what was asked for: a store that ignores Range answers with
	// the whole object, and a caller reading 8 bytes of a gigabyte file must
	// not hold the gigabyte.
	body, err := io.ReadAll(io.LimitReader(bodyReader, length))
	if err != nil {
		return nil, fmt.Errorf("failed to read object range: %w", d.explain(err))
	}

	result := &ObjectContent{
		Key:         key,
		Body:        body,
		ContentType: aws.ToString(output.ContentType),
		Size:        size,
		ETag:        aws.ToString(output.ETag),
		Metadata:    output.Metadata,
	}
	if output.LastModified != nil {
		result.LastModified = *output.LastModified
	}
	return result, nil
}

// seekRangeStart leaves body, the reader over output's body, positioned at
// offset and returns the object's total size, -1 when the store does not
// report one.
//
// A response with a Content-Range is a honored range; its start must be the
// offset asked for. A response without one is a store that ignored Range and
// sent the whole object from byte 0, so the bytes before offset are discarded
// as they stream past rather than returned as though they were the range.
func seekRangeStart(output *s3.GetObjectOutput, body io.Reader, offset int64) (int64, error) {
	if output.ContentRange != nil {
		start, total, ok := parseContentRange(*output.ContentRange)
		if !ok || start != offset {
			return 0, fmt.Errorf("unexpected Content-Range %q for a range at offset %d", *output.ContentRange, offset)
		}
		return total, nil
	}

	size := int64(-1)
	if output.ContentLength != nil {
		size = *output.ContentLength
	}
	if size >= 0 && offset >= size {
		return 0, fmt.Errorf("%w: offset %d, object size %d", ErrRangeNotSatisfiable, offset, size)
	}
	if _, err := io.CopyN(io.Discard, body, offset); err != nil {
		if errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("%w: offset %d is past the end of the object", ErrRangeNotSatisfiable, offset)
		}
		return 0, fmt.Errorf("failed to read object range: %w", err)
	}
	return size, nil
}

// parseContentRange reads a Content-Range header of a partial response,
// "bytes <start>-<end>/<total>", reporting a total of -1 when the store sends
// "*" for a length it does not know.
func parseContentRange(header string) (start, total int64, ok bool) {
	spec, found := strings.CutPrefix(header, "bytes ")
	if !found {
		return 0, 0, false
	}
	span, totalText, found := strings.Cut(spec, "/")
	if !found {
		return 0, 0, false
	}
	startText, _, found := strings.Cut(span, "-")
	if !found {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(startText, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	if totalText == "*" {
		return start, -1, true
	}
	total, err = strconv.ParseInt(totalText, 10, 64)
	if err != nil || total < 0 {
		return 0, 0, false
	}
	return start, total, true
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

	return context.WithTimeout(ctx, c.config.Timeout) //#nosec G118 -- cancel func is returned to caller
}

// Close closes the S3 client and releases resources.
// Currently a no-op as the AWS SDK manages its own connection pool.
func (c *Client) Close() error {
	return nil
}
