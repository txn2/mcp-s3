package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewToolkit(t *testing.T) {
	mock := NewMockS3Client("test-connection")

	toolkit := NewToolkit(mock)

	if toolkit == nil {
		t.Fatal("expected non-nil toolkit")
	}

	if toolkit.IsReadOnly() {
		t.Error("expected read-only to be false by default")
	}

	if toolkit.MaxGetSize() != DefaultMaxGetSize {
		t.Errorf("expected max get size %d, got %d", DefaultMaxGetSize, toolkit.MaxGetSize())
	}

	if toolkit.MaxPutSize() != DefaultMaxPutSize {
		t.Errorf("expected max put size %d, got %d", DefaultMaxPutSize, toolkit.MaxPutSize())
	}
}

func TestNewToolkit_WithOptions(t *testing.T) {
	mock := NewMockS3Client("test-connection")

	toolkit := NewToolkit(mock,
		WithReadOnly(true),
		WithMaxGetSize(5*1024*1024),
		WithMaxPutSize(50*1024*1024),
		WithDefaultConnection("custom"),
		WithToolPrefix("my_"),
	)

	if !toolkit.IsReadOnly() {
		t.Error("expected read-only to be true")
	}

	if toolkit.MaxGetSize() != 5*1024*1024 {
		t.Errorf("expected max get size %d, got %d", 5*1024*1024, toolkit.MaxGetSize())
	}

	if toolkit.MaxPutSize() != 50*1024*1024 {
		t.Errorf("expected max put size %d, got %d", 50*1024*1024, toolkit.MaxPutSize())
	}
}

func TestToolkit_GetClient(t *testing.T) {
	mock1 := NewMockS3Client("conn1")
	mock2 := NewMockS3Client("conn2")

	toolkit := NewToolkit(mock1,
		WithDefaultConnection("conn1"),
	)
	toolkit.AddClient("conn2", mock2)

	t.Run("get default client", func(t *testing.T) {
		client, err := toolkit.GetClient("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.ConnectionName() != "conn1" {
			t.Errorf("expected connection name 'conn1', got %q", client.ConnectionName())
		}
	})

	t.Run("get named client", func(t *testing.T) {
		client, err := toolkit.GetClient("conn2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.ConnectionName() != "conn2" {
			t.Errorf("expected connection name 'conn2', got %q", client.ConnectionName())
		}
	})

	t.Run("get non-existent client", func(t *testing.T) {
		_, err := toolkit.GetClient("non-existent")
		if err == nil {
			t.Error("expected error for non-existent connection")
		}
	})
}

func TestToolkit_ListConnections(t *testing.T) {
	mock1 := NewMockS3Client("conn1")
	mock2 := NewMockS3Client("conn2")

	toolkit := NewToolkit(mock1)
	toolkit.AddClient("conn1", mock1)
	toolkit.AddClient("conn2", mock2)

	connections := toolkit.ListConnections()

	if len(connections) < 2 {
		t.Errorf("expected at least 2 connections, got %d", len(connections))
	}

	// Check that both connections are present
	found1, found2 := false, false
	for _, conn := range connections {
		if conn == "conn1" {
			found1 = true
		}
		if conn == "conn2" {
			found2 = true
		}
	}

	if !found1 {
		t.Error("expected conn1 in connections list")
	}
	if !found2 {
		t.Error("expected conn2 in connections list")
	}
}

func TestToolkit_DisableTool(t *testing.T) {
	mock := NewMockS3Client("test")

	toolkit := NewToolkit(mock,
		DisableTool(ToolPutObject, ToolDeleteObject),
	)

	if !toolkit.isToolDisabled(ToolPutObject) {
		t.Error("expected PutObject to be disabled")
	}
	if !toolkit.isToolDisabled(ToolDeleteObject) {
		t.Error("expected DeleteObject to be disabled")
	}
	if toolkit.isToolDisabled(ToolGetObject) {
		t.Error("expected GetObject to be enabled")
	}
}

func TestToolkit_EnableOnlyTools(t *testing.T) {
	mock := NewMockS3Client("test")

	toolkit := NewToolkit(mock,
		EnableOnlyTools(ToolListBuckets, ToolListObjects),
	)

	if toolkit.isToolDisabled(ToolListBuckets) {
		t.Error("expected ListBuckets to be enabled")
	}
	if toolkit.isToolDisabled(ToolListObjects) {
		t.Error("expected ListObjects to be enabled")
	}
	if !toolkit.isToolDisabled(ToolGetObject) {
		t.Error("expected GetObject to be disabled")
	}
	if !toolkit.isToolDisabled(ToolPutObject) {
		t.Error("expected PutObject to be disabled")
	}
}

func TestToolkit_toolName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		toolName ToolName
		expected string
	}{
		{
			name:     "no prefix",
			prefix:   "",
			toolName: ToolListBuckets,
			expected: "s3_list_buckets",
		},
		{
			name:     "with prefix",
			prefix:   "my_",
			toolName: ToolListBuckets,
			expected: "my_s3_list_buckets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockS3Client("test")
			toolkit := NewToolkit(mock, WithToolPrefix(tt.prefix))

			got := toolkit.toolName(tt.toolName)
			if got != tt.expected {
				t.Errorf("toolName() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestMiddlewareChain_Execution(t *testing.T) {
	callOrder := []string{}

	m1 := NewMiddlewareFunc("m1",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			callOrder = append(callOrder, "m1-before")
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			callOrder = append(callOrder, "m1-after")
			return result, err
		},
	)

	m2 := NewMiddlewareFunc("m2",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			callOrder = append(callOrder, "m2-before")
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			callOrder = append(callOrder, "m2-after")
			return result, err
		},
	)

	chain := NewMiddlewareChain()
	chain.Add(m1)
	chain.Add(m2)

	tc := NewToolContext(ToolListBuckets, "")
	ctx, _ := chain.Before(context.Background(), tc)

	// Simulate handler
	callOrder = append(callOrder, "handler")

	result := TextResult("ok")
	_, _ = chain.After(ctx, tc, result, nil)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(callOrder) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(callOrder), callOrder)
	}

	for i, exp := range expected {
		if callOrder[i] != exp {
			t.Errorf("call %d: expected %q, got %q", i, exp, callOrder[i])
		}
	}
}

func TestInterceptorChain(t *testing.T) {
	t.Run("all allow", func(t *testing.T) {
		chain := NewInterceptorChain()

		chain.Add(NewRequestInterceptorFunc("i1", func(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		chain.Add(NewRequestInterceptorFunc("i2", func(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		result := chain.Intercept(context.Background(), NewToolContext("test", ""), &mcp.CallToolRequest{})
		if !result.Allow {
			t.Error("expected request to be allowed")
		}
	})

	t.Run("one blocks", func(t *testing.T) {
		chain := NewInterceptorChain()

		chain.Add(NewRequestInterceptorFunc("i1", func(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		chain.Add(NewRequestInterceptorFunc("i2", func(ctx context.Context, tc *ToolContext, request *mcp.CallToolRequest) InterceptResult {
			return Blocked("test block")
		}))

		result := chain.Intercept(context.Background(), NewToolContext("test", ""), &mcp.CallToolRequest{})
		if result.Allow {
			t.Error("expected request to be blocked")
		}
		if result.Reason != "test block" {
			t.Errorf("expected reason 'test block', got %q", result.Reason)
		}
	})
}

func TestTransformerChain(t *testing.T) {
	chain := NewTransformerChain()

	chain.Add(NewResultTransformerFunc("t1", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		tc.Set("t1-ran", true)
		return result, nil
	}))

	chain.Add(NewResultTransformerFunc("t2", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		tc.Set("t2-ran", true)
		return result, nil
	}))

	tc := NewToolContext("test", "")
	result := TextResult("original")

	transformed, err := chain.Transform(context.Background(), tc, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transformed == nil {
		t.Fatal("expected non-nil result")
	}

	if !tc.GetBool("t1-ran") {
		t.Error("expected t1 to have run")
	}
	if !tc.GetBool("t2-ran") {
		t.Error("expected t2 to have run")
	}
}

func TestToolkit_RegisterAll(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	// Create a mock MCP server
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// This should not panic
	toolkit.RegisterAll(s)
}

func TestToolkit_RegisterAll_WithDisabled(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock,
		DisableTool(ToolPutObject, ToolDeleteObject),
	)

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// Should register tools without the disabled ones
	toolkit.RegisterAll(s)
}

func TestToolkit_Close(t *testing.T) {
	mock1 := NewMockS3Client("conn1")
	mock2 := NewMockS3Client("conn2")

	toolkit := NewToolkit(mock1)
	toolkit.AddClient("conn2", mock2)

	err := toolkit.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolkit_Close_NoClients(t *testing.T) {
	toolkit := NewToolkit(nil)

	err := toolkit.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolkit_GetClient_WithProvider(t *testing.T) {
	mock := NewMockS3Client("default")

	providerCalled := false
	toolkit := NewToolkit(mock,
		WithClientProvider(func(name string) (S3Client, error) {
			providerCalled = true
			return NewMockS3Client(name), nil
		}),
	)

	client, err := toolkit.GetClient("dynamic-conn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !providerCalled {
		t.Error("expected provider to be called")
	}

	if client.ConnectionName() != "dynamic-conn" {
		t.Errorf("expected connection name 'dynamic-conn', got %q", client.ConnectionName())
	}

	// Second call should use cache
	providerCalled = false
	client2, err := toolkit.GetClient("dynamic-conn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if providerCalled {
		t.Error("expected provider NOT to be called for cached client")
	}

	if client != client2 {
		t.Error("expected same client instance from cache")
	}
}

func makeTestRequest(args map[string]any) *mcp.CallToolRequest {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{},
	}
	if args != nil {
		argsJSON, _ := json.Marshal(args)
		req.Params.Arguments = argsJSON
	}
	return req
}

func TestToolkit_wrapHandler_BlockedByInterceptor(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock,
		WithInterceptor(NewRequestInterceptorFunc("blocker", func(ctx context.Context, tc *ToolContext, req *mcp.CallToolRequest) InterceptResult {
			return Blocked("test block reason")
		})),
	)

	handlerCalled := false
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		handlerCalled = true
		return TextResult("success"), nil, nil
	}

	wrapped := toolkit.wrapHandler(ToolListBuckets, handler, nil)

	req := makeTestRequest(nil)
	result, _, err := wrapped(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if handlerCalled {
		t.Error("expected handler NOT to be called when blocked")
	}

	// Result should indicate access denied
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check that the result contains the block reason
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
}

func TestToolkit_wrapHandler_WithMiddleware(t *testing.T) {
	mock := NewMockS3Client("test")

	beforeCalled := false
	afterCalled := false
	toolkit := NewToolkit(mock,
		WithMiddleware(NewMiddlewareFunc("test-mw",
			func(ctx context.Context, tc *ToolContext) (context.Context, error) {
				beforeCalled = true
				return ctx, nil
			},
			func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
				afterCalled = true
				return result, err
			},
		)),
	)

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		return TextResult("success"), nil, nil
	}

	wrapped := toolkit.wrapHandler(ToolListBuckets, handler, nil)

	req := makeTestRequest(nil)
	_, _, err := wrapped(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !beforeCalled {
		t.Error("expected middleware Before to be called")
	}
	if !afterCalled {
		t.Error("expected middleware After to be called")
	}
}

func TestToolkit_wrapHandler_WithTransformer(t *testing.T) {
	mock := NewMockS3Client("test")

	transformerCalled := false
	toolkit := NewToolkit(mock,
		WithTransformer(NewResultTransformerFunc("test-transformer", func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
			transformerCalled = true
			return result, nil
		})),
	)

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		return TextResult("success"), nil, nil
	}

	wrapped := toolkit.wrapHandler(ToolListBuckets, handler, nil)

	req := makeTestRequest(nil)
	_, _, err := wrapped(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !transformerCalled {
		t.Error("expected transformer to be called")
	}
}

func TestToolkit_RegisterWith(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// Register a tool with per-registration options
	mw := NewMiddlewareFunc("test-mw",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			return result, err
		},
	)

	toolkit.RegisterWith(s, ToolListBuckets, WithPerToolMiddleware(mw))

	// Tool should be registered
	if !toolkit.registeredTools[ToolListBuckets] {
		t.Error("expected ToolListBuckets to be registered")
	}
}

func TestToolkit_RegisterTools(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// RegisterTools is an alias for RegisterAll
	toolkit.RegisterTools(s)

	// All tools should be registered
	for _, name := range AllTools() {
		if !toolkit.registeredTools[name] {
			t.Errorf("expected %s to be registered", name)
		}
	}
}

func TestWithToolMiddleware(t *testing.T) {
	mock := NewMockS3Client("test")

	mw := NewMiddlewareFunc("test-mw",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			return result, err
		},
	)

	toolkit := NewToolkit(mock, WithToolMiddleware(ToolListBuckets, mw))

	if len(toolkit.toolMiddlewares[ToolListBuckets]) != 1 {
		t.Errorf("expected 1 middleware for ToolListBuckets, got %d", len(toolkit.toolMiddlewares[ToolListBuckets]))
	}
}

func TestWithPerToolMiddleware(t *testing.T) {
	mw := NewMiddlewareFunc("test-mw",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			return result, err
		},
	)

	cfg := &toolConfig{}
	opt := WithPerToolMiddleware(mw)
	opt(cfg)

	if len(cfg.middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(cfg.middlewares))
	}
}

func TestNewToolMiddlewareRegistry(t *testing.T) {
	registry := NewToolMiddlewareRegistry()

	if registry == nil {
		t.Error("expected non-nil registry")
	}

	// Test Register method (alias for Add)
	mw := NewMiddlewareFunc("test-mw",
		func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			return ctx, nil
		},
		func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, err error) (*mcp.CallToolResult, error) {
			return result, err
		},
	)

	registry.Register(mw)

	if len(registry.All()) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(registry.All()))
	}
}

func TestToolkit_RegisterTool_PreventsDuplicates(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock)

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// Register the same tool twice
	toolkit.Register(s, ToolListBuckets)
	toolkit.Register(s, ToolListBuckets)

	// Should only be registered once (no panic)
	if !toolkit.registeredTools[ToolListBuckets] {
		t.Error("expected ToolListBuckets to be registered")
	}
}

func TestToolkit_RegisterTool_SkipsDisabled(t *testing.T) {
	mock := NewMockS3Client("test")
	toolkit := NewToolkit(mock, DisableTool(ToolPutObject))

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// Try to register disabled tool
	toolkit.Register(s, ToolPutObject)

	// Should not be registered
	if toolkit.registeredTools[ToolPutObject] {
		t.Error("expected ToolPutObject NOT to be registered when disabled")
	}
}
