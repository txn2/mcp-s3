package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func makeRequest(args map[string]any) *mcp.CallToolRequest {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{},
	}
	if args != nil {
		argsJSON, _ := json.Marshal(args)
		req.Params.Arguments = argsJSON
	}
	return req
}

func TestInterceptResult_AllowedWithModification(t *testing.T) {
	req := makeRequest(map[string]any{"key": "value"})

	result := AllowedWithModification(req)

	if !result.Allow {
		t.Error("Allow should be true")
	}
	if result.ModifiedRequest == nil {
		t.Error("ModifiedRequest should not be nil")
	}
}

func TestRequestInterceptorFunc_Name(t *testing.T) {
	interceptor := NewRequestInterceptorFunc("test-interceptor", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
		return Allowed()
	})

	if got := interceptor.Name(); got != "test-interceptor" {
		t.Errorf("Name() = %q, want %q", got, "test-interceptor")
	}
}

func TestInterceptorChain_AllAndClear(t *testing.T) {
	t.Run("all returns interceptors", func(t *testing.T) {
		chain := NewInterceptorChain()
		chain.Add(NewRequestInterceptorFunc("i1", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))
		chain.Add(NewRequestInterceptorFunc("i2", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		all := chain.All()
		if len(all) != 2 {
			t.Errorf("All() returned %d items, want 2", len(all))
		}
	})

	t.Run("clear removes all", func(t *testing.T) {
		chain := NewInterceptorChain()
		chain.Add(NewRequestInterceptorFunc("i1", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		chain.Clear()

		if len(chain.All()) != 0 {
			t.Error("Clear() did not remove all interceptors")
		}
	})

	t.Run("intercept blocks when interceptor blocks", func(t *testing.T) {
		chain := NewInterceptorChain()
		chain.Add(NewRequestInterceptorFunc("blocker", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
			return Blocked("not allowed")
		}))

		tc := NewToolContext("test", "conn")
		req := makeRequest(nil)

		result := chain.Intercept(context.Background(), tc, req)

		if result.Allow {
			t.Error("should block request")
		}
		if result.Reason != "not allowed" {
			t.Errorf("Reason = %q, want %q", result.Reason, "not allowed")
		}
	})
}
