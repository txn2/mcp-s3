package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolMiddleware provides hooks for tool execution.
// Before is called before the handler, After is called after (in reverse order).
type ToolMiddleware interface {
	// Name returns a unique identifier for this middleware.
	Name() string

	// Before is called before the tool handler executes.
	// It can modify the context or return an error to stop execution.
	Before(ctx context.Context, tc *ToolContext) (context.Context, error)

	// After is called after the tool handler executes (in reverse order).
	// It can modify the result or handle errors.
	After(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error)
}

// MiddlewareFunc is a function-based implementation of ToolMiddleware.
type MiddlewareFunc struct {
	name     string
	beforeFn func(context.Context, *ToolContext) (context.Context, error)
	afterFn  func(context.Context, *ToolContext, *mcp.CallToolResult, error) (*mcp.CallToolResult, error)
}

// NewMiddlewareFunc creates a new MiddlewareFunc with the given name and functions.
func NewMiddlewareFunc(name string, beforeFn func(context.Context, *ToolContext) (context.Context, error), afterFn func(context.Context, *ToolContext, *mcp.CallToolResult, error) (*mcp.CallToolResult, error)) *MiddlewareFunc {
	return &MiddlewareFunc{
		name:     name,
		beforeFn: beforeFn,
		afterFn:  afterFn,
	}
}

// Name returns the middleware name.
func (m *MiddlewareFunc) Name() string {
	return m.name
}

// Before calls the before function if set.
func (m *MiddlewareFunc) Before(ctx context.Context, tc *ToolContext) (context.Context, error) {
	if m.beforeFn != nil {
		return m.beforeFn(ctx, tc)
	}
	return ctx, nil
}

// After calls the after function if set.
func (m *MiddlewareFunc) After(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error) {
	if m.afterFn != nil {
		return m.afterFn(ctx, tc, result, handlerErr)
	}
	return result, handlerErr
}

// BeforeFunc creates a middleware that only runs before the handler.
func BeforeFunc(fn func(context.Context, *ToolContext) (context.Context, error)) *MiddlewareFunc {
	return &MiddlewareFunc{
		name:     "before",
		beforeFn: fn,
	}
}

// AfterFunc creates a middleware that only runs after the handler.
func AfterFunc(fn func(context.Context, *ToolContext, *mcp.CallToolResult, error) (*mcp.CallToolResult, error)) *MiddlewareFunc {
	return &MiddlewareFunc{
		name:    "after",
		afterFn: fn,
	}
}

// MiddlewareChain manages a collection of middleware.
type MiddlewareChain struct {
	middlewares []ToolMiddleware
}

// NewMiddlewareChain creates a new middleware chain.
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]ToolMiddleware, 0),
	}
}

// Add adds middleware to the chain.
func (c *MiddlewareChain) Add(m ToolMiddleware) {
	c.middlewares = append(c.middlewares, m)
}

// Before runs all Before hooks in order.
func (c *MiddlewareChain) Before(ctx context.Context, tc *ToolContext) (context.Context, error) {
	var err error
	for _, m := range c.middlewares {
		ctx, err = m.Before(ctx, tc)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// After runs all After hooks in reverse order (like defer).
func (c *MiddlewareChain) After(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error) {
	var err error
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		result, err = c.middlewares[i].After(ctx, tc, result, handlerErr)
		if err != nil {
			handlerErr = err
		}
	}
	return result, handlerErr
}

// All returns all registered middleware.
func (c *MiddlewareChain) All() []ToolMiddleware {
	result := make([]ToolMiddleware, len(c.middlewares))
	copy(result, c.middlewares)
	return result
}

// Clear removes all middleware from the chain.
func (c *MiddlewareChain) Clear() {
	c.middlewares = make([]ToolMiddleware, 0)
}

// ToolMiddlewareRegistry manages a collection of middleware (alias for backward compatibility).
type ToolMiddlewareRegistry = MiddlewareChain

// NewToolMiddlewareRegistry creates a new middleware registry.
func NewToolMiddlewareRegistry() *ToolMiddlewareRegistry {
	return NewMiddlewareChain()
}

// Register adds middleware to the registry.
func (c *MiddlewareChain) Register(m ToolMiddleware) {
	c.Add(m)
}
