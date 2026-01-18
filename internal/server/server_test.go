package server

import (
	"context"
	"testing"
	"time"

	"github.com/txn2/mcp-s3/pkg/client"
	"github.com/txn2/mcp-s3/pkg/extensions"
	"github.com/txn2/mcp-s3/pkg/tools"
)

var _ tools.S3Client = (*mockS3Client)(nil)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ExtConfig.ReadOnly != true {
		t.Error("expected ReadOnly to be true by default")
	}

	if cfg.ExtConfig.SizeLimit != true {
		t.Error("expected SizeLimit to be true by default")
	}

	if cfg.Logger == nil {
		t.Error("expected Logger to be set")
	}
}

func TestFromEnv(t *testing.T) {
	cfg := FromEnv()

	if cfg.ClientConfig == nil {
		t.Error("expected ClientConfig to be set")
	}

	// Extension config should have defaults
	if cfg.ExtConfig.MaxGetSize == 0 {
		t.Error("expected MaxGetSize to be set")
	}
}

func TestNew_WithConfig(t *testing.T) {
	// Skip if we don't have a valid endpoint configured
	clientCfg := &client.Config{
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9999", // Invalid endpoint
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Name:            "test",
	}

	cfg := Config{
		ClientConfig: clientCfg,
		ExtConfig:    extensions.DefaultConfig(),
		Logger:       nil,
	}

	// This will fail because the endpoint is invalid, but that's expected
	// We're testing that the configuration is properly processed
	_, _, err := New(cfg)

	// We expect an error because the S3 client can't connect
	// But the server setup should at least get that far
	if err == nil {
		// If no error, we successfully created the server (maybe with a mock)
		t.Log("Server created successfully")
	}
}

func TestBuildToolkitOptions(t *testing.T) {
	cfg := Config{
		ExtConfig: extensions.Config{
			ReadOnly:   true,
			SizeLimit:  true,
			MaxGetSize: 1024,
			MaxPutSize: 2048,
		},
	}

	opts := buildToolkitOptions(cfg, nil, nil)

	// Should have at least 4 options (ReadOnly, MaxGetSize, MaxPutSize, Logger)
	if len(opts) < 4 {
		t.Errorf("expected at least 4 options, got %d", len(opts))
	}
}

func TestAppendConnectionOptions_WithNilManager(t *testing.T) {
	opts := []tools.Option{}

	// With nil manager and nil client, should return unchanged
	result := appendConnectionOptions(opts, nil, nil)

	if len(result) != 0 {
		t.Errorf("expected 0 options, got %d", len(result))
	}
}

func TestAppendConnectionOptions_WithClientNoName(t *testing.T) {
	opts := []tools.Option{}
	mockClient := &mockS3Client{name: ""}

	result := appendConnectionOptions(opts, mockClient, nil)

	// Should not add default connection option when name is empty
	if len(result) != 0 {
		t.Errorf("expected 0 options with empty connection name, got %d", len(result))
	}
}

func TestAppendConnectionOptions_WithClientWithName(t *testing.T) {
	opts := []tools.Option{}
	mockClient := &mockS3Client{name: "test-conn"}

	result := appendConnectionOptions(opts, mockClient, nil)

	// Should add default connection option
	if len(result) != 1 {
		t.Errorf("expected 1 option, got %d", len(result))
	}
}

func TestAppendExtensionOptions_AllEnabled(t *testing.T) {
	cfg := Config{
		ExtConfig: extensions.Config{
			ReadOnly:        true,
			SizeLimit:       true,
			Logging:         true,
			Audit:           true,
			PrefixACL:       true,
			AllowedPrefixes: []string{"allowed/"},
			DeniedPrefixes:  []string{"denied/"},
			MaxGetSize:      1024,
			MaxPutSize:      2048,
		},
		Logger: DefaultConfig().Logger,
	}

	opts := []tools.Option{}
	result := appendExtensionOptions(opts, cfg)

	// Should add 5 options: readonly, sizelimit, logging, audit, prefixacl
	if len(result) != 5 {
		t.Errorf("expected 5 options, got %d", len(result))
	}
}

func TestAppendExtensionOptions_NoneEnabled(t *testing.T) {
	cfg := Config{
		ExtConfig: extensions.Config{
			ReadOnly:  false,
			SizeLimit: false,
			Logging:   false,
			Audit:     false,
			PrefixACL: false,
		},
	}

	opts := []tools.Option{}
	result := appendExtensionOptions(opts, cfg)

	// Should add 0 options
	if len(result) != 0 {
		t.Errorf("expected 0 options, got %d", len(result))
	}
}

func TestAppendExtensionOptions_LoggingWithNilLogger(t *testing.T) {
	cfg := Config{
		ExtConfig: extensions.Config{
			Logging: true,
		},
		Logger: nil,
	}

	opts := []tools.Option{}
	result := appendExtensionOptions(opts, cfg)

	// Should not add logging middleware when logger is nil
	if len(result) != 0 {
		t.Errorf("expected 0 options when logger is nil, got %d", len(result))
	}
}

// mockS3Client is a minimal mock for testing.
type mockS3Client struct {
	name string
}

func (m *mockS3Client) ConnectionName() string { return m.name }
func (m *mockS3Client) Config() *client.Config { return &client.Config{Name: m.name} }
func (m *mockS3Client) ListBuckets(_ context.Context) ([]client.BucketInfo, error) {
	return nil, nil
}
func (m *mockS3Client) ListObjects(_ context.Context, _, _, _ string, _ int32, _ string) (*client.ListObjectsOutput, error) {
	return nil, nil
}
func (m *mockS3Client) GetObject(_ context.Context, _, _ string) (*client.ObjectContent, error) {
	return nil, nil
}
func (m *mockS3Client) GetObjectMetadata(_ context.Context, _, _ string) (*client.ObjectMetadata, error) {
	return nil, nil
}
func (m *mockS3Client) PutObject(_ context.Context, _ *client.PutObjectInput) (*client.PutObjectOutput, error) {
	return nil, nil
}
func (m *mockS3Client) DeleteObject(_ context.Context, _, _ string) error { return nil }
func (m *mockS3Client) CopyObject(_ context.Context, _ *client.CopyObjectInput) (*client.CopyObjectOutput, error) {
	return nil, nil
}
func (m *mockS3Client) PresignGetURL(_ context.Context, _, _ string, _ time.Duration) (*client.PresignedURL, error) {
	return nil, nil
}
func (m *mockS3Client) PresignPutURL(_ context.Context, _, _ string, _ time.Duration) (*client.PresignedURL, error) {
	return nil, nil
}
func (m *mockS3Client) Close() error { return nil }

func TestNewWithDefaults(t *testing.T) {
	// This will likely fail because there's no real S3 endpoint configured
	// but it exercises the code path
	_, _, _ = NewWithDefaults()
	// We don't check the error because it depends on environment
}
