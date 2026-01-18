// Package server provides a factory for creating the MCP S3 server.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/txn2/mcp-s3/pkg/client"
	"github.com/txn2/mcp-s3/pkg/extensions"
	"github.com/txn2/mcp-s3/pkg/multiserver"
	"github.com/txn2/mcp-s3/pkg/tools"
)

// Version is set at build time.
var Version = "dev"

// Config holds configuration for the MCP S3 server.
type Config struct {
	// Client configuration
	ClientConfig *client.Config

	// Extension configuration
	ExtConfig extensions.Config

	// Multi-server configuration
	MultiConfig *multiserver.MultiConfig

	// Logger
	Logger *slog.Logger
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ClientConfig: nil, // Will use FromEnv()
		ExtConfig:    extensions.DefaultConfig(),
		MultiConfig:  nil,
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}
}

// FromEnv creates a configuration from environment variables.
func FromEnv() Config {
	cfg := DefaultConfig()
	clientCfg := client.FromEnv()
	cfg.ClientConfig = &clientCfg
	cfg.ExtConfig = extensions.FromEnv()

	// Try to load multi-server config from environment
	multiCfg, _ := multiserver.FromEnvJSON()
	cfg.MultiConfig = multiCfg

	return cfg
}

// New creates a new MCP S3 server with the given configuration.
func New(cfg Config) (*server.MCPServer, *tools.Toolkit, error) {
	ctx := context.Background()

	// Create the MCP server
	mcpServer := server.NewMCPServer(
		"mcp-s3",
		Version,
		server.WithLogging(),
	)

	// Create the S3 client
	var s3Client tools.S3Client
	var manager *multiserver.Manager
	var err error

	if cfg.MultiConfig != nil && len(cfg.MultiConfig.Connections) > 0 {
		// Use multi-server manager
		manager = multiserver.NewManager(cfg.MultiConfig)
		s3Client, err = manager.GetDefaultClient(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create default S3 client: %w", err)
		}
	} else if cfg.ClientConfig != nil {
		// Use single client
		s3Client, err = client.New(ctx, cfg.ClientConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create S3 client: %w", err)
		}
	} else {
		// Create from environment
		clientCfg := client.FromEnv()
		s3Client, err = client.New(ctx, &clientCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create S3 client: %w", err)
		}
	}

	// Build toolkit options
	opts := []tools.Option{
		tools.WithReadOnly(cfg.ExtConfig.ReadOnly),
		tools.WithMaxGetSize(cfg.ExtConfig.MaxGetSize),
		tools.WithMaxPutSize(cfg.ExtConfig.MaxPutSize),
		tools.WithLogger(cfg.Logger),
	}

	// Add multi-server support
	if manager != nil {
		opts = append(opts, tools.WithClientProvider(manager.ClientProvider()))
		opts = append(opts, tools.WithDefaultConnection(manager.DefaultConnectionName()))
	} else if s3Client != nil && s3Client.ConnectionName() != "" {
		opts = append(opts, tools.WithDefaultConnection(s3Client.ConnectionName()))
	}

	// Add extensions based on config
	if cfg.ExtConfig.ReadOnly {
		opts = append(opts, tools.WithInterceptor(extensions.NewReadOnlyInterceptor(true)))
	}

	if cfg.ExtConfig.SizeLimit {
		opts = append(opts, tools.WithInterceptor(
			extensions.NewSizeLimitInterceptor(cfg.ExtConfig.MaxGetSize, cfg.ExtConfig.MaxPutSize),
		))
	}

	if cfg.ExtConfig.Logging && cfg.Logger != nil {
		opts = append(opts, tools.WithMiddleware(extensions.NewLoggingMiddleware(cfg.Logger)))
	}

	if cfg.ExtConfig.Audit {
		auditLogger := extensions.NewAuditLogger(os.Stderr)
		opts = append(opts, tools.WithMiddleware(extensions.NewAuditMiddleware(auditLogger)))
	}

	if cfg.ExtConfig.PrefixACL {
		opts = append(opts, tools.WithInterceptor(
			extensions.NewPrefixACLInterceptor(cfg.ExtConfig.AllowedPrefixes, cfg.ExtConfig.DeniedPrefixes),
		))
	}

	// Create the toolkit
	toolkit := tools.NewToolkit(s3Client, opts...)

	// Register tools with the MCP server
	toolkit.RegisterTools(mcpServer)

	return mcpServer, toolkit, nil
}

// NewWithDefaults creates a new MCP S3 server with default configuration from environment.
func NewWithDefaults() (*server.MCPServer, *tools.Toolkit, error) {
	return New(FromEnv())
}
