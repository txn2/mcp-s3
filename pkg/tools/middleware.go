package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandler is a function that handles a tool request.
type ToolHandler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)

// ToolMiddleware wraps a tool handler to add behavior before or after execution.
// Middleware can modify the request, short-circuit execution, or modify the result.
type ToolMiddleware interface {
	// Name returns a unique identifier for this middleware.
	Name() string

	// Wrap wraps the given handler with additional behavior.
	Wrap(next ToolHandler) ToolHandler
}

// MiddlewareFunc is a function type that implements ToolMiddleware.
type MiddlewareFunc struct {
	name   string
	wrapFn func(ToolHandler) ToolHandler
}

// NewMiddlewareFunc creates a new MiddlewareFunc with the given name and wrap function.
func NewMiddlewareFunc(name string, wrapFn func(ToolHandler) ToolHandler) *MiddlewareFunc {
	return &MiddlewareFunc{
		name:   name,
		wrapFn: wrapFn,
	}
}

// Name returns the middleware name.
func (m *MiddlewareFunc) Name() string {
	return m.name
}

// Wrap wraps the given handler.
func (m *MiddlewareFunc) Wrap(next ToolHandler) ToolHandler {
	return m.wrapFn(next)
}

// ChainMiddleware chains multiple middleware together.
// Middleware is applied in reverse order so that the first middleware
// in the list is the outermost (first to execute).
func ChainMiddleware(handler ToolHandler, middleware ...ToolMiddleware) ToolHandler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i].Wrap(handler)
	}
	return handler
}

// ToolMiddlewareRegistry manages a collection of middleware.
type ToolMiddlewareRegistry struct {
	middleware []ToolMiddleware
}

// NewToolMiddlewareRegistry creates a new middleware registry.
func NewToolMiddlewareRegistry() *ToolMiddlewareRegistry {
	return &ToolMiddlewareRegistry{
		middleware: make([]ToolMiddleware, 0),
	}
}

// Register adds middleware to the registry.
func (r *ToolMiddlewareRegistry) Register(m ToolMiddleware) {
	r.middleware = append(r.middleware, m)
}

// All returns all registered middleware.
func (r *ToolMiddlewareRegistry) All() []ToolMiddleware {
	result := make([]ToolMiddleware, len(r.middleware))
	copy(result, r.middleware)
	return result
}

// Apply applies all registered middleware to a handler.
func (r *ToolMiddlewareRegistry) Apply(handler ToolHandler) ToolHandler {
	return ChainMiddleware(handler, r.middleware...)
}

// Clear removes all middleware from the registry.
func (r *ToolMiddlewareRegistry) Clear() {
	r.middleware = make([]ToolMiddleware, 0)
}
