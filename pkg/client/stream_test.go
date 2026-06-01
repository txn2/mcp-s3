package client

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
)

// newStreamClient builds a Client wired with a mock uploader for streaming tests.
func newStreamClient(up *mockUploader) *Client {
	c := newMockClient(nil, nil)
	c.uploader = up
	return c
}

func TestClient_PutObjectStream_Success(t *testing.T) {
	const payload = "streamed content"

	var (
		gotBucket, gotKey, gotContentType string
		gotMetadata                       map[string]string
		gotBody                           string
	)

	up := &mockUploader{
		uploadObjectFunc: func(
			_ context.Context, input *transfermanager.UploadObjectInput, _ ...func(*transfermanager.Options),
		) (*transfermanager.UploadObjectOutput, error) {
			gotBucket = aws.ToString(input.Bucket)
			gotKey = aws.ToString(input.Key)
			gotContentType = aws.ToString(input.ContentType)
			gotMetadata = input.Metadata
			b, err := io.ReadAll(input.Body)
			if err != nil {
				return nil, err
			}
			gotBody = string(b)
			return &transfermanager.UploadObjectOutput{
				ETag:      aws.String("\"streametag\""),
				VersionID: aws.String("v9"),
			}, nil
		},
	}

	result, err := newStreamClient(up).PutObjectStream(context.Background(), &PutObjectStreamInput{
		Bucket:      "my-bucket",
		Key:         "export.csv",
		Body:        strings.NewReader(payload),
		ContentType: "text/csv",
		Metadata:    map[string]string{"author": "test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotBucket != "my-bucket" {
		t.Errorf("bucket: got %q, want my-bucket", gotBucket)
	}
	if gotKey != "export.csv" {
		t.Errorf("key: got %q, want export.csv", gotKey)
	}
	if gotContentType != "text/csv" {
		t.Errorf("content type: got %q, want text/csv", gotContentType)
	}
	if gotMetadata["author"] != "test" {
		t.Errorf("metadata author: got %q, want test", gotMetadata["author"])
	}
	if gotBody != payload {
		t.Errorf("body: got %q, want %q", gotBody, payload)
	}
	if result.ETag != "\"streametag\"" {
		t.Errorf("etag: got %q, want '\"streametag\"'", result.ETag)
	}
	if result.VersionID != "v9" {
		t.Errorf("version: got %q, want v9", result.VersionID)
	}
}

func TestClient_PutObjectStream_Validation(t *testing.T) {
	tests := []struct {
		name  string
		setup func() (*Client, *PutObjectStreamInput)
	}{
		{
			name: "nil input",
			setup: func() (*Client, *PutObjectStreamInput) {
				return newStreamClient(&mockUploader{}), nil
			},
		},
		{
			name: "nil body",
			setup: func() (*Client, *PutObjectStreamInput) {
				return newStreamClient(&mockUploader{}), &PutObjectStreamInput{Bucket: "b", Key: "k"}
			},
		},
		{
			name: "uploader not configured",
			setup: func() (*Client, *PutObjectStreamInput) {
				c := newMockClient(nil, nil) // no uploader wired
				return c, &PutObjectStreamInput{Bucket: "b", Key: "k", Body: strings.NewReader("x")}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, in := tt.setup()
			if _, err := c.PutObjectStream(context.Background(), in); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestClient_PutObjectStream_UploaderError(t *testing.T) {
	up := &mockUploader{
		uploadObjectFunc: func(
			_ context.Context, _ *transfermanager.UploadObjectInput, _ ...func(*transfermanager.Options),
		) (*transfermanager.UploadObjectOutput, error) {
			return nil, errors.New("access denied")
		},
	}

	_, err := newStreamClient(up).PutObjectStream(context.Background(), &PutObjectStreamInput{
		Bucket: "b", Key: "k", Body: strings.NewReader("data"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error should wrap underlying cause, got: %v", err)
	}
}

func TestClient_PutObjectStream_MaxBytes(t *testing.T) {
	// The uploader drains the body so the limit reader is exercised.
	drain := &mockUploader{
		uploadObjectFunc: func(
			_ context.Context, input *transfermanager.UploadObjectInput, _ ...func(*transfermanager.Options),
		) (*transfermanager.UploadObjectOutput, error) {
			if _, err := io.ReadAll(input.Body); err != nil {
				return nil, err
			}
			return &transfermanager.UploadObjectOutput{}, nil
		},
	}

	t.Run("under limit succeeds", func(t *testing.T) {
		_, err := newStreamClient(drain).PutObjectStream(context.Background(), &PutObjectStreamInput{
			Bucket: "b", Key: "k", Body: strings.NewReader("12345"), MaxBytes: 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("over limit aborts", func(t *testing.T) {
		_, err := newStreamClient(drain).PutObjectStream(context.Background(), &PutObjectStreamInput{
			Bucket: "b", Key: "k", Body: strings.NewReader("this body is too large"), MaxBytes: 4,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrStreamTooLarge) {
			t.Errorf("expected ErrStreamTooLarge, got: %v", err)
		}
	})
}

func TestLimitReader(t *testing.T) {
	t.Run("passes through under limit", func(t *testing.T) {
		lr := &limitReader{r: strings.NewReader("hello"), max: 5}
		b, err := io.ReadAll(lr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(b) != "hello" {
			t.Errorf("got %q, want hello", string(b))
		}
	})

	t.Run("errors over limit", func(t *testing.T) {
		lr := &limitReader{r: strings.NewReader("hello world"), max: 5}
		_, err := io.ReadAll(lr)
		if !errors.Is(err, ErrStreamTooLarge) {
			t.Errorf("expected ErrStreamTooLarge, got: %v", err)
		}
	})
}
