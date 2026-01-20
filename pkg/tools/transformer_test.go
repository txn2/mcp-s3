package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResultTransformerFunc_Name(t *testing.T) {
	transformer := NewResultTransformerFunc("test-transformer", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		return result, nil
	})

	if got := transformer.Name(); got != "test-transformer" {
		t.Errorf("Name() = %q, want %q", got, "test-transformer")
	}
}

func TestTransformerChain_AllClearNil(t *testing.T) {
	t.Run("all returns transformers", func(t *testing.T) {
		chain := NewTransformerChain()
		chain.Add(NewResultTransformerFunc("t1", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
			return result, nil
		}))
		chain.Add(NewResultTransformerFunc("t2", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
			return result, nil
		}))

		all := chain.All()
		if len(all) != 2 {
			t.Errorf("All() returned %d items, want 2", len(all))
		}
	})

	t.Run("clear removes all", func(t *testing.T) {
		chain := NewTransformerChain()
		chain.Add(NewResultTransformerFunc("t1", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
			return result, nil
		}))

		chain.Clear()

		if len(chain.All()) != 0 {
			t.Error("Clear() did not remove all transformers")
		}
	})

	t.Run("transform handles nil result", func(t *testing.T) {
		chain := NewTransformerChain()
		chain.Add(NewResultTransformerFunc("t1", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
			return result, nil
		}))

		tc := NewToolContext("test", "conn")
		result, err := chain.Transform(context.Background(), tc, nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result to pass through")
		}
	})
}
