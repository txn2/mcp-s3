package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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
		toolName string
		expected string
	}{
		{
			name:     "no prefix",
			prefix:   "",
			toolName: "s3_list_buckets",
			expected: "s3_list_buckets",
		},
		{
			name:     "with prefix",
			prefix:   "my_",
			toolName: "s3_list_buckets",
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

func TestMiddleware(t *testing.T) {
	callOrder := []string{}

	m1 := NewMiddlewareFunc("m1", func(next ToolHandler) ToolHandler {
		return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			callOrder = append(callOrder, "m1-before")
			result, err := next(ctx, request)
			callOrder = append(callOrder, "m1-after")
			return result, err
		}
	})

	m2 := NewMiddlewareFunc("m2", func(next ToolHandler) ToolHandler {
		return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			callOrder = append(callOrder, "m2-before")
			result, err := next(ctx, request)
			callOrder = append(callOrder, "m2-after")
			return result, err
		}
	})

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		callOrder = append(callOrder, "handler")
		return TextResult("ok"), nil
	}

	chained := ChainMiddleware(handler, m1, m2)

	_, err := chained(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

		chain.Add(NewRequestInterceptorFunc("i1", func(ctx context.Context, tc *ToolContext, request mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		chain.Add(NewRequestInterceptorFunc("i2", func(ctx context.Context, tc *ToolContext, request mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		result := chain.Intercept(context.Background(), NewToolContext("test", ""), mcp.CallToolRequest{})
		if !result.Allow {
			t.Error("expected request to be allowed")
		}
	})

	t.Run("one blocks", func(t *testing.T) {
		chain := NewInterceptorChain()

		chain.Add(NewRequestInterceptorFunc("i1", func(ctx context.Context, tc *ToolContext, request mcp.CallToolRequest) InterceptResult {
			return Allowed()
		}))

		chain.Add(NewRequestInterceptorFunc("i2", func(ctx context.Context, tc *ToolContext, request mcp.CallToolRequest) InterceptResult {
			return Blocked("test block")
		}))

		result := chain.Intercept(context.Background(), NewToolContext("test", ""), mcp.CallToolRequest{})
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
		// Add a marker to the result
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
