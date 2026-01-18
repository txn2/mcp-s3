package extensions

import (
	"context"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/txn2/mcp-s3/pkg/tools"
)

func TestFromEnv(t *testing.T) {
	// Save and restore environment
	envVars := []string{
		"MCP_S3_EXT_READONLY",
		"MCP_S3_EXT_SIZELIMIT",
		"MCP_S3_MAX_GET_SIZE",
		"MCP_S3_MAX_PUT_SIZE",
		"MCP_S3_EXT_LOGGING",
		"MCP_S3_EXT_AUDIT",
	}

	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range saved {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Clear all env vars
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	t.Run("defaults", func(t *testing.T) {
		cfg := FromEnv()

		if !cfg.ReadOnly {
			t.Error("expected ReadOnly to be true by default")
		}
		if !cfg.SizeLimit {
			t.Error("expected SizeLimit to be true by default")
		}
		if cfg.Logging {
			t.Error("expected Logging to be false by default")
		}
		if cfg.Audit {
			t.Error("expected Audit to be false by default")
		}
	})

	t.Run("custom values", func(t *testing.T) {
		os.Setenv("MCP_S3_EXT_READONLY", "false")
		os.Setenv("MCP_S3_EXT_SIZELIMIT", "true")
		os.Setenv("MCP_S3_MAX_GET_SIZE", "5MB")
		os.Setenv("MCP_S3_MAX_PUT_SIZE", "50MB")
		os.Setenv("MCP_S3_EXT_LOGGING", "true")
		os.Setenv("MCP_S3_EXT_AUDIT", "true")
		defer func() {
			for _, v := range envVars {
				os.Unsetenv(v)
			}
		}()

		cfg := FromEnv()

		if cfg.ReadOnly {
			t.Error("expected ReadOnly to be false")
		}
		if !cfg.SizeLimit {
			t.Error("expected SizeLimit to be true")
		}
		if cfg.MaxGetSize != 5*1024*1024 {
			t.Errorf("expected MaxGetSize 5MB, got %d", cfg.MaxGetSize)
		}
		if cfg.MaxPutSize != 50*1024*1024 {
			t.Errorf("expected MaxPutSize 50MB, got %d", cfg.MaxPutSize)
		}
		if !cfg.Logging {
			t.Error("expected Logging to be true")
		}
		if !cfg.Audit {
			t.Error("expected Audit to be true")
		}
	})
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"10KB", 10 * 1024},
		{"10kb", 10 * 1024},
		{"10K", 10 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"10M", 10 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSize(tt.input, 0)
			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReadOnlyInterceptor(t *testing.T) {
	t.Run("enabled blocks write tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(true)

		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := mcp.CallToolRequest{}

		result := interceptor.Intercept(context.Background(), tc, req)
		if result.Allow {
			t.Error("expected write tool to be blocked")
		}
	})

	t.Run("enabled allows read tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(true)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		req := mcp.CallToolRequest{}

		result := interceptor.Intercept(context.Background(), tc, req)
		if !result.Allow {
			t.Error("expected read tool to be allowed")
		}
	})

	t.Run("disabled allows all tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(false)

		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := mcp.CallToolRequest{}

		result := interceptor.Intercept(context.Background(), tc, req)
		if !result.Allow {
			t.Error("expected write tool to be allowed when disabled")
		}
	})
}

func TestSizeLimitInterceptor(t *testing.T) {
	t.Run("allows small content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(10*1024*1024, 1024)

		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"content": "small content",
		}

		result := interceptor.Intercept(context.Background(), tc, req)
		if !result.Allow {
			t.Error("expected small content to be allowed")
		}
	})

	t.Run("blocks large content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(10*1024*1024, 10) // 10 byte limit

		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"content": "this content is way too long",
		}

		result := interceptor.Intercept(context.Background(), tc, req)
		if result.Allow {
			t.Error("expected large content to be blocked")
		}
	})
}

func TestPrefixACLInterceptor(t *testing.T) {
	t.Run("denies blocked prefix", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"private/", "secret/"})

		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"key": "private/file.txt",
		}

		result := interceptor.Intercept(context.Background(), tc, req)
		if result.Allow {
			t.Error("expected denied prefix to be blocked")
		}
	})

	t.Run("allows non-blocked prefix", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"private/", "secret/"})

		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"key": "public/file.txt",
		}

		result := interceptor.Intercept(context.Background(), tc, req)
		if !result.Allow {
			t.Error("expected non-blocked prefix to be allowed")
		}
	})

	t.Run("requires allowed prefix when specified", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor([]string{"allowed/"}, nil)

		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"key": "not-allowed/file.txt",
		}

		result := interceptor.Intercept(context.Background(), tc, req)
		if result.Allow {
			t.Error("expected non-allowed prefix to be blocked")
		}
	})
}

func TestAuditLogger(t *testing.T) {
	t.Run("buffered logger stores entries", func(t *testing.T) {
		logger := NewBufferedAuditLogger()

		entry := AuditEntry{
			Tool:    "s3_list_buckets",
			Success: true,
		}

		err := logger.Log(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		entries := logger.Entries()
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("clear removes entries", func(t *testing.T) {
		logger := NewBufferedAuditLogger()

		_ = logger.Log(AuditEntry{Tool: "test"})
		logger.Clear()

		entries := logger.Entries()
		if len(entries) != 0 {
			t.Errorf("expected 0 entries after clear, got %d", len(entries))
		}
	})
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()

	// Record some calls
	metrics.RecordCall("s3_list_buckets", 100*1e6, false) // 100ms
	metrics.RecordCall("s3_list_buckets", 200*1e6, false) // 200ms
	metrics.RecordCall("s3_list_buckets", 50*1e6, true)   // 50ms, error

	stats := metrics.GetToolStats("s3_list_buckets")

	if stats.Calls != 3 {
		t.Errorf("expected 3 calls, got %d", stats.Calls)
	}

	if stats.Errors != 1 {
		t.Errorf("expected 1 error, got %d", stats.Errors)
	}

	// Check error rate (1/3 ≈ 0.33)
	if stats.ErrorRate < 0.3 || stats.ErrorRate > 0.4 {
		t.Errorf("expected error rate ~0.33, got %f", stats.ErrorRate)
	}
}
