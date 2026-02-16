package extensions

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

// Before stores the start time in context for duration calculation.
func (m *AuditMiddleware) Before(ctx context.Context, _ *tools.ToolContext) (context.Context, error) {
	// StartTime is already set in ToolContext
	return ctx, nil
}

// After logs the audit entry after handler execution.
func (m *AuditMiddleware) After(
	_ context.Context, tc *tools.ToolContext, result *mcp.CallToolResult, handlerErr error,
) (*mcp.CallToolResult, error) {
	entry := AuditEntry{
		Timestamp:  time.Now().UTC(),
		Tool:       string(tc.ToolName),
		Connection: tc.ConnectionName,
		RequestID:  tc.RequestID,
	}

	entry.Duration = time.Since(tc.StartTime)
	entry.DurationMs = float64(entry.Duration) / float64(time.Millisecond)

	switch {
	case handlerErr != nil:
		entry.Success = false
		entry.Error = handlerErr.Error()
	case result != nil && result.IsError:
		entry.Success = false
		// Try to extract error message from result
		if len(result.Content) > 0 {
			if text, ok := result.Content[0].(*mcp.TextContent); ok {
				entry.Error = text.Text
			}
		}
	default:
		entry.Success = true
	}

	// Log the entry (best-effort; audit should not block tool execution)
	_ = m.logger.Log(entry) //nolint:errcheck // audit logging is best-effort

	return result, handlerErr
}

// Ensure AuditMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*AuditMiddleware)(nil)
