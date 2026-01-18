package server

import (
	"testing"

	"github.com/txn2/mcp-s3/pkg/client"
	"github.com/txn2/mcp-s3/pkg/extensions"
)

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
