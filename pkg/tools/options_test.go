package tools

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestWithMiddleware(t *testing.T) {
	tk := &Toolkit{
		middleware: NewToolMiddlewareRegistry(),
	}

	called := false
	mw := NewMiddlewareFunc("test", func(next ToolHandler) ToolHandler {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			called = true
			return next(ctx, req)
		}
	})

	opt := WithMiddleware(mw)
	opt(tk)

	if len(tk.middleware.All()) != 1 {
		t.Error("middleware not added")
	}

	// Verify middleware works
	handler := tk.middleware.Apply(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return TextResult("done"), nil
	})
	_, _ = handler(context.Background(), mcp.CallToolRequest{})

	if !called {
		t.Error("middleware was not invoked")
	}
}

func TestWithInterceptor(t *testing.T) {
	tk := &Toolkit{
		interceptors: NewInterceptorChain(),
	}

	interceptor := NewRequestInterceptorFunc("test", func(ctx context.Context, tc *ToolContext, req mcp.CallToolRequest) InterceptResult {
		return Blocked("blocked")
	})

	opt := WithInterceptor(interceptor)
	opt(tk)

	if len(tk.interceptors.All()) != 1 {
		t.Error("interceptor not added")
	}
}

func TestWithTransformer(t *testing.T) {
	tk := &Toolkit{
		transformers: NewTransformerChain(),
	}

	transformer := NewResultTransformerFunc("test", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		return result, nil
	})

	opt := WithTransformer(transformer)
	opt(tk)

	if len(tk.transformers.All()) != 1 {
		t.Error("transformer not added")
	}
}

func TestWithLogger(t *testing.T) {
	tk := &Toolkit{}

	logger := slog.Default()
	opt := WithLogger(logger)
	opt(tk)

	if tk.logger != logger {
		t.Error("logger not set")
	}
}

func TestWithClientProvider(t *testing.T) {
	tk := &Toolkit{}

	provider := func(name string) (S3Client, error) {
		return nil, nil
	}

	opt := WithClientProvider(provider)
	opt(tk)

	if tk.clientProvider == nil {
		t.Error("client provider not set")
	}
}
