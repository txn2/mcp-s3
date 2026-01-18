package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestMiddlewareFuncWrapper_Name(t *testing.T) {
	mw := NewMiddlewareFunc("test-middleware", func(next ToolHandler) ToolHandler {
		return next
	})

	if got := mw.Name(); got != "test-middleware" {
		t.Errorf("Name() = %q, want %q", got, "test-middleware")
	}
}

func TestToolMiddlewareRegistry(t *testing.T) {
	t.Run("register and retrieve all", func(t *testing.T) {
		registry := NewToolMiddlewareRegistry()

		mw1 := NewMiddlewareFunc("mw1", func(next ToolHandler) ToolHandler { return next })
		mw2 := NewMiddlewareFunc("mw2", func(next ToolHandler) ToolHandler { return next })

		registry.Register(mw1)
		registry.Register(mw2)

		all := registry.All()
		if len(all) != 2 {
			t.Errorf("All() returned %d items, want 2", len(all))
		}
	})

	t.Run("apply wraps handler", func(t *testing.T) {
		registry := NewToolMiddlewareRegistry()

		called := false
		mw := NewMiddlewareFunc("test", func(next ToolHandler) ToolHandler {
			return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				called = true
				return next(ctx, req)
			}
		})
		registry.Register(mw)

		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return TextResult("done"), nil
		}

		wrapped := registry.Apply(handler)
		_, _ = wrapped(context.Background(), mcp.CallToolRequest{})

		if !called {
			t.Error("middleware was not called")
		}
	})

	t.Run("clear removes all", func(t *testing.T) {
		registry := NewToolMiddlewareRegistry()
		registry.Register(NewMiddlewareFunc("mw", func(next ToolHandler) ToolHandler { return next }))

		registry.Clear()

		if len(registry.All()) != 0 {
			t.Error("Clear() did not remove all middleware")
		}
	})
}
