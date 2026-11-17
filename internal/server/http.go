package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const shutdownTimeout = 5 * time.Second

// ServeHTTP serves mcpServer over the MCP streamable HTTP transport on ln
// until ctx is cancelled, then shuts down gracefully.
//
// The endpoint has no authentication; bind to loopback or put an
// authenticating proxy in front of it.
func ServeHTTP(ctx context.Context, mcpServer *mcp.Server, ln net.Listener) error {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, nil)
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	stop := context.AfterFunc(ctx, func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	})
	defer stop()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
