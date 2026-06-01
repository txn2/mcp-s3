package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
)

// ErrStreamTooLarge indicates that a streaming upload was aborted because the
// body exceeded the caller-supplied size limit. Callers can test for it with
// errors.Is.
var ErrStreamTooLarge = errors.New("stream exceeds maximum allowed size")

// PutObjectStreamInput contains the parameters for a streaming/multipart upload.
//
// Unlike PutObjectInput, the body is an io.Reader rather than a []byte, so the
// payload is never fully buffered in memory. This makes it suitable for large
// or unbounded sources such as query exports.
type PutObjectStreamInput struct {
	Bucket      string
	Key         string
	Body        io.Reader
	ContentType string
	Metadata    map[string]string

	// MaxBytes, when greater than zero, aborts the upload once more than
	// MaxBytes have been read from Body, returning an error that wraps
	// ErrStreamTooLarge. A value of zero means no limit is enforced here.
	MaxBytes int64
}

// PutObjectStream uploads an object from an io.Reader using the AWS SDK transfer
// manager, which splits the body into parts and uploads them without buffering
// the full payload in memory.
//
// Unlike the buffered operations on Client, the per-operation timeout
// (S3_TIMEOUT) is intentionally NOT applied here: a streaming upload of a large
// object can legitimately run far longer than an ordinary request. Callers
// control the deadline through ctx.
//
// Like the other write methods on Client, PutObjectStream performs the upload
// directly; the read-only and size-limit MCP extensions guard the tool layer,
// not direct library calls. Use MaxBytes to bound a stream at the library level.
func (c *Client) PutObjectStream(ctx context.Context, input *PutObjectStreamInput) (*PutObjectOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("put object stream: input is required")
	}
	if input.Body == nil {
		return nil, fmt.Errorf("put object stream: body is required")
	}
	if c.uploader == nil {
		return nil, fmt.Errorf("put object stream: uploader is not configured")
	}

	body := input.Body
	if input.MaxBytes > 0 {
		body = &limitReader{r: body, max: input.MaxBytes}
	}

	uploadInput := &transfermanager.UploadObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
		Body:   body,
	}
	if input.ContentType != "" {
		uploadInput.ContentType = aws.String(input.ContentType)
	}
	if len(input.Metadata) > 0 {
		uploadInput.Metadata = input.Metadata
	}

	output, err := c.uploader.UploadObject(ctx, uploadInput)
	if err != nil {
		return nil, fmt.Errorf("failed to stream object: %w", err)
	}

	return &PutObjectOutput{
		ETag:      aws.ToString(output.ETag),
		VersionID: aws.ToString(output.VersionID),
	}, nil
}

// limitReader wraps an io.Reader and returns an error wrapping ErrStreamTooLarge
// once more than max bytes have been read. It enforces an upper bound on a
// stream whose length is not known in advance.
type limitReader struct {
	r    io.Reader
	max  int64
	read int64
}

func (l *limitReader) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.read += int64(n)
	if l.read > l.max {
		return n, fmt.Errorf("read %d bytes: %w of %d bytes", l.read, ErrStreamTooLarge, l.max)
	}
	return n, err
}
