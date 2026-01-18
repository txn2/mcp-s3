package extensions

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/txn2/mcp-s3/pkg/tools"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp  time.Time      `json:"timestamp"`
	Tool       string         `json:"tool"`
	Connection string         `json:"connection,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	Arguments  map[string]any `json:"arguments,omitempty"`
	Success    bool           `json:"success"`
	Error      string         `json:"error,omitempty"`
	Duration   time.Duration  `json:"duration_ns"`
	DurationMs float64        `json:"duration_ms"`
}

// AuditLogger logs audit entries.
type AuditLogger struct {
	writer  io.Writer
	mu      sync.Mutex
	entries []AuditEntry // In-memory buffer for testing
	buffer  bool
}

// NewAuditLogger creates a new audit logger that writes to the given writer.
func NewAuditLogger(writer io.Writer) *AuditLogger {
	return &AuditLogger{
		writer:  writer,
		entries: make([]AuditEntry, 0),
		buffer:  false,
	}
}

// NewBufferedAuditLogger creates an audit logger that buffers entries in memory.
// Useful for testing.
func NewBufferedAuditLogger() *AuditLogger {
	return &AuditLogger{
		entries: make([]AuditEntry, 0),
		buffer:  true,
	}
}

// Log records an audit entry.
func (l *AuditLogger) Log(entry AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.buffer {
		l.entries = append(l.entries, entry)
		return nil
	}

	if l.writer == nil {
		return nil
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = l.writer.Write(append(data, '\n'))
	return err
}

// Entries returns all buffered entries (only for buffered loggers).
func (l *AuditLogger) Entries() []AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := make([]AuditEntry, len(l.entries))
	copy(result, l.entries)
	return result
}

// Clear clears all buffered entries.
func (l *AuditLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]AuditEntry, 0)
}

// AuditMiddleware logs audit entries for tool operations.
type AuditMiddleware struct {
	logger *AuditLogger
}

// NewAuditMiddleware creates a new audit middleware.
func NewAuditMiddleware(logger *AuditLogger) *AuditMiddleware {
	return &AuditMiddleware{
		logger: logger,
	}
}

// Name returns the middleware name.
func (m *AuditMiddleware) Name() string {
	return "audit"
}

// Wrap wraps the handler with audit logging.
func (m *AuditMiddleware) Wrap(next tools.ToolHandler) tools.ToolHandler {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tc := tools.GetToolContext(ctx)

		entry := AuditEntry{
			Timestamp: time.Now().UTC(),
			Tool:      request.Params.Name,
			Arguments: sanitizeArguments(request.Params.Arguments),
		}

		if tc != nil {
			entry.Connection = tc.ConnectionName
			entry.RequestID = tc.RequestID
		}

		start := time.Now()
		result, err := next(ctx, request)
		entry.Duration = time.Since(start)
		entry.DurationMs = float64(entry.Duration) / float64(time.Millisecond)

		if err != nil {
			entry.Success = false
			entry.Error = err.Error()
		} else if result != nil && result.IsError {
			entry.Success = false
			// Try to extract error message from result
			if len(result.Content) > 0 {
				if text, ok := result.Content[0].(mcp.TextContent); ok {
					entry.Error = text.Text
				}
			}
		} else {
			entry.Success = true
		}

		// Log the entry (ignore errors for now)
		_ = m.logger.Log(entry)

		return result, err
	}
}

// sanitizeArguments removes sensitive data from arguments for logging.
func sanitizeArguments(args map[string]any) map[string]any {
	if args == nil {
		return nil
	}

	result := make(map[string]any, len(args))
	for k, v := range args {
		switch k {
		case "content":
			// Don't log full content, just indicate presence
			if s, ok := v.(string); ok {
				result[k] = len(s)
			} else {
				result[k] = "[content]"
			}
		case "metadata":
			// Include metadata keys but not values
			if m, ok := v.(map[string]any); ok {
				keys := make([]string, 0, len(m))
				for key := range m {
					keys = append(keys, key)
				}
				result[k] = keys
			} else {
				result[k] = "[metadata]"
			}
		default:
			result[k] = v
		}
	}

	return result
}

// Ensure AuditMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*AuditMiddleware)(nil)
