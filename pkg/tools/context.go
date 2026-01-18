package tools

import (
	"context"
	"sync"
)

// contextKey is a type for context keys used by this package.
type contextKey string

const (
	// toolContextKey is the context key for ToolContext.
	toolContextKey contextKey = "tool_context"
)

// ToolContext provides contextual information and state for tool execution.
// It allows middleware, interceptors, and tools to share data during a request.
type ToolContext struct {
	// ToolName is the name of the tool being executed.
	ToolName string

	// ConnectionName is the name of the S3 connection being used.
	ConnectionName string

	// RequestID is a unique identifier for this request.
	RequestID string

	// values stores arbitrary key-value pairs for middleware communication.
	values map[string]any
	mu     sync.RWMutex
}

// NewToolContext creates a new ToolContext with the given tool and connection names.
func NewToolContext(toolName, connectionName string) *ToolContext {
	return &ToolContext{
		ToolName:       toolName,
		ConnectionName: connectionName,
		values:         make(map[string]any),
	}
}

// Set stores a value in the context with the given key.
func (tc *ToolContext) Set(key string, value any) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.values[key] = value
}

// Get retrieves a value from the context by key.
// Returns nil if the key doesn't exist.
func (tc *ToolContext) Get(key string) any {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.values[key]
}

// GetString retrieves a string value from the context by key.
// Returns an empty string if the key doesn't exist or isn't a string.
func (tc *ToolContext) GetString(key string) string {
	v := tc.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetBool retrieves a boolean value from the context by key.
// Returns false if the key doesn't exist or isn't a boolean.
func (tc *ToolContext) GetBool(key string) bool {
	v := tc.Get(key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetInt retrieves an integer value from the context by key.
// Returns 0 if the key doesn't exist or isn't an integer.
func (tc *ToolContext) GetInt(key string) int {
	v := tc.Get(key)
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

// Has returns true if the context has a value for the given key.
func (tc *ToolContext) Has(key string) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	_, ok := tc.values[key]
	return ok
}

// Delete removes a value from the context by key.
func (tc *ToolContext) Delete(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.values, key)
}

// Clone creates a shallow copy of the ToolContext.
func (tc *ToolContext) Clone() *ToolContext {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	newTC := &ToolContext{
		ToolName:       tc.ToolName,
		ConnectionName: tc.ConnectionName,
		RequestID:      tc.RequestID,
		values:         make(map[string]any, len(tc.values)),
	}

	for k, v := range tc.values {
		newTC.values[k] = v
	}

	return newTC
}

// WithToolContext returns a new context with the ToolContext attached.
func WithToolContext(ctx context.Context, tc *ToolContext) context.Context {
	return context.WithValue(ctx, toolContextKey, tc)
}

// GetToolContext retrieves the ToolContext from a context.
// Returns nil if no ToolContext is present.
func GetToolContext(ctx context.Context) *ToolContext {
	v := ctx.Value(toolContextKey)
	if tc, ok := v.(*ToolContext); ok {
		return tc
	}
	return nil
}
