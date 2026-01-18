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
	mcpServer := server.NewMCPServer("mcp-s3", Version, server.WithLogging())

	s3Client, manager, err := createS3Client(cfg)
	if err != nil {
		return nil, nil, err
	}

	opts := buildToolkitOptions(cfg, s3Client, manager)
	toolkit := tools.NewToolkit(s3Client, opts...)
	toolkit.RegisterTools(mcpServer)

	return mcpServer, toolkit, nil
}

func createS3Client(cfg Config) (tools.S3Client, *multiserver.Manager, error) {
	ctx := context.Background()

	if cfg.MultiConfig != nil && len(cfg.MultiConfig.Connections) > 0 {
		manager := multiserver.NewManager(cfg.MultiConfig)
		s3Client, err := manager.GetDefaultClient(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create default S3 client: %w", err)
		}
		return s3Client, manager, nil
	}

	clientCfg := cfg.ClientConfig
	if clientCfg == nil {
		envCfg := client.FromEnv()
		clientCfg = &envCfg
	}

	s3Client, err := client.New(ctx, clientCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create S3 client: %w", err)
	}
	return s3Client, nil, nil
}

func buildToolkitOptions(cfg Config, s3Client tools.S3Client, manager *multiserver.Manager) []tools.Option {
	opts := []tools.Option{
		tools.WithReadOnly(cfg.ExtConfig.ReadOnly),
		tools.WithMaxGetSize(cfg.ExtConfig.MaxGetSize),
		tools.WithMaxPutSize(cfg.ExtConfig.MaxPutSize),
		tools.WithLogger(cfg.Logger),
	}

	opts = appendConnectionOptions(opts, s3Client, manager)
	opts = appendExtensionOptions(opts, cfg)
	return opts
}

func appendConnectionOptions(opts []tools.Option, s3Client tools.S3Client, manager *multiserver.Manager) []tools.Option {
	if manager != nil {
		opts = append(opts, tools.WithClientProvider(manager.ClientProvider()))
		opts = append(opts, tools.WithDefaultConnection(manager.DefaultConnectionName()))
	} else if s3Client != nil && s3Client.ConnectionName() != "" {
		opts = append(opts, tools.WithDefaultConnection(s3Client.ConnectionName()))
	}
	return opts
}

func appendExtensionOptions(opts []tools.Option, cfg Config) []tools.Option {
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
	return opts
}

// NewWithDefaults creates a new MCP S3 server with default configuration from environment.
func NewWithDefaults() (*server.MCPServer, *tools.Toolkit, error) {
	return New(FromEnv())
}
