package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMiddlewareFuncWrapper_Name(t *testing.T) {
	beforeFn := func(ctx context.Context, tc *ToolContext) (context.Context, error) {
		return ctx, nil
	}
	afterFn := func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
		return result, err
	}

	mw := NewMiddlewareFunc("test-middleware", beforeFn, afterFn)

	if got := mw.Name(); got != "test-middleware" {
		t.Errorf("Name() = %q, want %q", got, "test-middleware")
	}
}

func TestMiddlewareChain(t *testing.T) {
	t.Run("add and retrieve all", func(t *testing.T) {
		chain := NewMiddlewareChain()

		beforeFn := func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			return ctx, nil
		}
		afterFn := func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			return result, err
		}

		mw1 := NewMiddlewareFunc("mw1", beforeFn, afterFn)
		mw2 := NewMiddlewareFunc("mw2", beforeFn, afterFn)

		chain.Add(mw1)
		chain.Add(mw2)

		all := chain.All()
		if len(all) != 2 {
			t.Errorf("All() returned %d items, want 2", len(all))
		}
	})

	t.Run("Before executes in order", func(t *testing.T) {
		chain := NewMiddlewareChain()

		order := []string{}

		mw1 := NewMiddlewareFunc("mw1",
			func(ctx context.Context, tc *ToolContext) (context.Context, error) {
				order = append(order, "mw1-before")
				return ctx, nil
			},
			func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
				order = append(order, "mw1-after")
				return result, err
			},
		)
		mw2 := NewMiddlewareFunc("mw2",
			func(ctx context.Context, tc *ToolContext) (context.Context, error) {
				order = append(order, "mw2-before")
				return ctx, nil
			},
			func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
				order = append(order, "mw2-after")
				return result, err
			},
		)

		chain.Add(mw1)
		chain.Add(mw2)

		tc := NewToolContext(ToolListBuckets, "")
		_, _ = chain.Before(context.Background(), tc)

		if len(order) != 2 {
			t.Errorf("expected 2 before calls, got %d", len(order))
		}
		if order[0] != "mw1-before" {
			t.Errorf("first call should be mw1-before, got %s", order[0])
		}
		if order[1] != "mw2-before" {
			t.Errorf("second call should be mw2-before, got %s", order[1])
		}
	})

	t.Run("After executes in reverse order", func(t *testing.T) {
		chain := NewMiddlewareChain()

		order := []string{}

		mw1 := NewMiddlewareFunc("mw1",
			func(ctx context.Context, tc *ToolContext) (context.Context, error) {
				return ctx, nil
			},
			func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
				order = append(order, "mw1-after")
				return result, err
			},
		)
		mw2 := NewMiddlewareFunc("mw2",
			func(ctx context.Context, tc *ToolContext) (context.Context, error) {
				return ctx, nil
			},
			func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
				order = append(order, "mw2-after")
				return result, err
			},
		)

		chain.Add(mw1)
		chain.Add(mw2)

		tc := NewToolContext(ToolListBuckets, "")
		result := TextResult("test")
		_, _ = chain.After(context.Background(), tc, result, nil)

		if len(order) != 2 {
			t.Errorf("expected 2 after calls, got %d", len(order))
		}
		// After should execute in reverse order
		if order[0] != "mw2-after" {
			t.Errorf("first after call should be mw2-after, got %s", order[0])
		}
		if order[1] != "mw1-after" {
			t.Errorf("second after call should be mw1-after, got %s", order[1])
		}
	})

	t.Run("clear removes all", func(t *testing.T) {
		chain := NewMiddlewareChain()
		chain.Add(NewMiddlewareFunc("mw",
			func(ctx context.Context, tc *ToolContext) (context.Context, error) { return ctx, nil },
			func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
				return result, err
			},
		))

		chain.Clear()

		if len(chain.All()) != 0 {
			t.Error("Clear() did not remove all middleware")
		}
	})
}

func TestBeforeFunc(t *testing.T) {
	called := false
	mw := BeforeFunc(func(ctx context.Context, tc *ToolContext) (context.Context, error) {
		called = true
		return ctx, nil
	})

	tc := NewToolContext(ToolListBuckets, "")
	_, _ = mw.Before(context.Background(), tc)

	if !called {
		t.Error("Before function was not called")
	}
}

func TestAfterFunc(t *testing.T) {
	called := false
	mw := AfterFunc(func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
		called = true
		return result, err
	})

	tc := NewToolContext(ToolListBuckets, "")
	result := TextResult("test")
	_, _ = mw.After(context.Background(), tc, result, nil)

	if !called {
		t.Error("After function was not called")
	}
}
