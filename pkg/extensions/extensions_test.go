package extensions

import (
	"context"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/txn2/mcp-s3/pkg/tools"
)

func TestFromEnv(t *testing.T) {
	envVars := []string{
		"MCP_S3_EXT_READONLY", "MCP_S3_EXT_SIZELIMIT",
		"MCP_S3_MAX_GET_SIZE", "MCP_S3_MAX_PUT_SIZE",
		"MCP_S3_EXT_LOGGING", "MCP_S3_EXT_AUDIT",
	}

	saved := saveEnv(envVars)
	defer restoreEnv(saved)
	clearEnv(envVars)

	t.Run("defaults", func(t *testing.T) {
		cfg := FromEnv()
		assertBool(t, "ReadOnly", true, cfg.ReadOnly)
		assertBool(t, "SizeLimit", true, cfg.SizeLimit)
		assertBool(t, "Logging", false, cfg.Logging)
		assertBool(t, "Audit", false, cfg.Audit)
	})

	t.Run("custom values", func(t *testing.T) {
		setEnvVars(map[string]string{
			"MCP_S3_EXT_READONLY":  "false",
			"MCP_S3_EXT_SIZELIMIT": "true",
			"MCP_S3_MAX_GET_SIZE":  "5MB",
			"MCP_S3_MAX_PUT_SIZE":  "50MB",
			"MCP_S3_EXT_LOGGING":   "true",
			"MCP_S3_EXT_AUDIT":     "true",
		})
		defer clearEnv(envVars)

		cfg := FromEnv()
		assertBool(t, "ReadOnly", false, cfg.ReadOnly)
		assertBool(t, "SizeLimit", true, cfg.SizeLimit)
		assertInt64(t, "MaxGetSize", 5*1024*1024, cfg.MaxGetSize)
		assertInt64(t, "MaxPutSize", 50*1024*1024, cfg.MaxPutSize)
		assertBool(t, "Logging", true, cfg.Logging)
		assertBool(t, "Audit", true, cfg.Audit)
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
		result := interceptor.Intercept(context.Background(), tc, mcp.CallToolRequest{})
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("enabled allows read tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(true)
		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		result := interceptor.Intercept(context.Background(), tc, mcp.CallToolRequest{})
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("disabled allows all tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(false)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		result := interceptor.Intercept(context.Background(), tc, mcp.CallToolRequest{})
		assertBool(t, "Allow", true, result.Allow)
	})
}

func TestSizeLimitInterceptor(t *testing.T) {
	t.Run("allows small content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(10*1024*1024, 1024)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"content": "small content"}
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("blocks large content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(10*1024*1024, 10)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"content": "this content is way too long"}
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})
}

func TestPrefixACLInterceptor(t *testing.T) {
	t.Run("denies blocked prefix", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"private/", "secret/"})
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"key": "private/file.txt"}
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("allows non-blocked prefix", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"private/", "secret/"})
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"key": "public/file.txt"}
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("requires allowed prefix when specified", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor([]string{"allowed/"}, nil)
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"key": "not-allowed/file.txt"}
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})
}

func TestAuditLogger(t *testing.T) {
	t.Run("buffered logger stores entries", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		if err := logger.Log(AuditEntry{Tool: "s3_list_buckets", Success: true}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(logger.Entries()) != 1 {
			t.Errorf("expected 1 entry, got %d", len(logger.Entries()))
		}
	})

	t.Run("clear removes entries", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		_ = logger.Log(AuditEntry{Tool: "test"})
		logger.Clear()
		if len(logger.Entries()) != 0 {
			t.Errorf("expected 0 entries after clear, got %d", len(logger.Entries()))
		}
	})
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()
	metrics.RecordCall("s3_list_buckets", 100*1e6, false)
	metrics.RecordCall("s3_list_buckets", 200*1e6, false)
	metrics.RecordCall("s3_list_buckets", 50*1e6, true)

	stats := metrics.GetToolStats("s3_list_buckets")
	if stats.Calls != 3 {
		t.Errorf("expected 3 calls, got %d", stats.Calls)
	}
	if stats.Errors != 1 {
		t.Errorf("expected 1 error, got %d", stats.Errors)
	}
	if stats.ErrorRate < 0.3 || stats.ErrorRate > 0.4 {
		t.Errorf("expected error rate ~0.33, got %f", stats.ErrorRate)
	}
}

// Test helpers

func saveEnv(vars []string) map[string]string {
	saved := make(map[string]string)
	for _, v := range vars {
		saved[v] = os.Getenv(v)
	}
	return saved
}

func restoreEnv(saved map[string]string) {
	for k, v := range saved {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
}

func clearEnv(vars []string) {
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func setEnvVars(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func assertBool(t *testing.T, field string, expected, got bool) {
	t.Helper()
	if got != expected {
		t.Errorf("%s: expected %v, got %v", field, expected, got)
	}
}

func assertInt64(t *testing.T, field string, expected, got int64) {
	t.Helper()
	if got != expected {
		t.Errorf("%s: expected %d, got %d", field, expected, got)
	}
}
