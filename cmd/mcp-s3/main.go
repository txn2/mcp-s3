// Package main provides the entry point for the mcp-s3 server.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcps3 "github.com/txn2/mcp-s3/internal/server"
)

// defaultHTTPAddr binds loopback only: the HTTP endpoint has no authentication.
const defaultHTTPAddr = "127.0.0.1:8080"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Create the server with defaults from environment
	mcpServer, toolkit, err := mcps3.NewWithDefaults()
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	defer func() { _ = toolkit.Close() }()

	if err := serve(ctx, mcpServer, os.Getenv("MCP_S3_TRANSPORT"), os.Getenv("MCP_S3_HTTP_ADDR")); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// serve runs mcpServer on the selected transport: "stdio" (default) or "http" (streamable HTTP).
func serve(ctx context.Context, mcpServer *mcp.Server, transport, addr string) error {
	switch transport {
	case "", "stdio":
		return mcpServer.Run(ctx, &mcp.StdioTransport{})
	case "http":
		if addr == "" {
			addr = defaultHTTPAddr
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("listening on %s: %w", addr, err)
		}
		return mcps3.ServeHTTP(ctx, mcpServer, ln)
	default:
		return fmt.Errorf("unknown MCP_S3_TRANSPORT %q (want stdio or http)", transport)
	}
}
