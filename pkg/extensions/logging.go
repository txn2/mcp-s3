package extensions

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-s3/pkg/tools"
)

// LoggingMiddleware provides structured logging for tool operations.
type LoggingMiddleware struct {
	logger *slog.Logger
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(logger *slog.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Name returns the middleware name.
func (m *LoggingMiddleware) Name() string {
	return "logging"
}

// Before logs the start of a tool request.
func (m *LoggingMiddleware) Before(ctx context.Context, tc *tools.ToolContext) (context.Context, error) {
	// Build log attributes
	attrs := []any{
		"tool", tc.ToolName,
	}

	if tc.ConnectionName != "" {
		attrs = append(attrs, "connection", tc.ConnectionName)
	}
	if tc.RequestID != "" {
		attrs = append(attrs, "request_id", tc.RequestID)
	}

	// Log request start
	m.logger.Info("tool request started", attrs...)

	return ctx, nil
}

// After logs the completion of a tool request.
func (m *LoggingMiddleware) After(ctx context.Context, tc *tools.ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error) {
	// Build log attributes
	attrs := []any{
		"tool", tc.ToolName,
	}

	if tc.ConnectionName != "" {
		attrs = append(attrs, "connection", tc.ConnectionName)
	}
	if tc.RequestID != "" {
		attrs = append(attrs, "request_id", tc.RequestID)
	}

	// Calculate duration
	duration := time.Since(tc.StartTime)
	attrs = append(attrs, "duration_ms", duration.Milliseconds())

	// Log result
	if handlerErr != nil {
		m.logger.Error("tool request failed", append(attrs, "error", handlerErr.Error())...)
	} else if result != nil && result.IsError {
		m.logger.Warn("tool request returned error", attrs...)
	} else {
		m.logger.Info("tool request completed", attrs...)
	}

	return result, handlerErr
}

// Ensure LoggingMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*LoggingMiddleware)(nil)
