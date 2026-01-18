package extensions

import (
	"context"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

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

// Wrap wraps the handler with logging.
func (m *LoggingMiddleware) Wrap(next tools.ToolHandler) tools.ToolHandler {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tc := tools.GetToolContext(ctx)

		// Build log attributes
		attrs := []any{
			"tool", request.Params.Name,
		}

		if tc != nil {
			if tc.ConnectionName != "" {
				attrs = append(attrs, "connection", tc.ConnectionName)
			}
			if tc.RequestID != "" {
				attrs = append(attrs, "request_id", tc.RequestID)
			}
		}

		// Add relevant arguments (excluding sensitive data)
		if args, ok := request.Params.Arguments.(map[string]any); ok {
			if bucket, ok := args["bucket"].(string); ok {
				attrs = append(attrs, "bucket", bucket)
			}
			if key, ok := args["key"].(string); ok {
				attrs = append(attrs, "key", key)
			}
			if prefix, ok := args["prefix"].(string); ok {
				attrs = append(attrs, "prefix", prefix)
			}
		}

		// Log request start
		m.logger.Info("tool request started", attrs...)

		// Track timing
		start := time.Now()

		// Execute handler
		result, err := next(ctx, request)

		// Calculate duration
		duration := time.Since(start)
		attrs = append(attrs, "duration_ms", duration.Milliseconds())

		// Log result
		if err != nil {
			m.logger.Error("tool request failed", append(attrs, "error", err.Error())...)
		} else if result.IsError {
			m.logger.Warn("tool request returned error", attrs...)
		} else {
			m.logger.Info("tool request completed", attrs...)
		}

		return result, err
	}
}

// Ensure LoggingMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*LoggingMiddleware)(nil)
