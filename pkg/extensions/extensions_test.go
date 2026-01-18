package extensions

import (
	"bytes"
	"context"
	"errors"
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

// Additional tests for improved coverage

func TestNewAuditLogger(t *testing.T) {
	t.Run("writes to writer", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewAuditLogger(&buf)

		err := logger.Log(AuditEntry{Tool: "test_tool", Success: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if buf.Len() == 0 {
			t.Error("expected data written to buffer")
		}
	})

	t.Run("nil writer does not panic", func(t *testing.T) {
		logger := NewAuditLogger(nil)
		err := logger.Log(AuditEntry{Tool: "test", Success: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAuditMiddleware(t *testing.T) {
	t.Run("name returns audit", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)
		if mw.Name() != "audit" {
			t.Errorf("Name() = %q, want %q", mw.Name(), "audit")
		}
	})

	t.Run("wrap logs successful call", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.TextResult("success"), nil
		}

		wrapped := mw.Wrap(handler)
		req := mcp.CallToolRequest{}
		req.Params.Name = "test_tool"
		req.Params.Arguments = map[string]any{"bucket": "test-bucket"}

		_, err := wrapped(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		entries := logger.Entries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if !entries[0].Success {
			t.Error("expected success=true")
		}
		if entries[0].Tool != "test_tool" {
			t.Errorf("Tool = %q, want %q", entries[0].Tool, "test_tool")
		}
	})

	t.Run("wrap logs error call", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, errors.New("test error")
		}

		wrapped := mw.Wrap(handler)
		req := mcp.CallToolRequest{}
		req.Params.Name = "test_tool"

		_, _ = wrapped(context.Background(), req)

		entries := logger.Entries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Success {
			t.Error("expected success=false")
		}
		if entries[0].Error != "test error" {
			t.Errorf("Error = %q, want %q", entries[0].Error, "test error")
		}
	})

	t.Run("wrap logs result with IsError", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.ErrorResult(errors.New("result error")), nil
		}

		wrapped := mw.Wrap(handler)
		req := mcp.CallToolRequest{}
		req.Params.Name = "test_tool"

		_, _ = wrapped(context.Background(), req)

		entries := logger.Entries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Success {
			t.Error("expected success=false for error result")
		}
	})

	t.Run("wrap uses tool context", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return tools.TextResult("ok"), nil
		}

		wrapped := mw.Wrap(handler)
		req := mcp.CallToolRequest{}
		req.Params.Name = "test_tool"

		tc := tools.NewToolContext("test_tool", "my-connection")
		tc.RequestID = "req-123"
		ctx := tools.WithToolContext(context.Background(), tc)

		_, _ = wrapped(ctx, req)

		entries := logger.Entries()
		if entries[0].Connection != "my-connection" {
			t.Errorf("Connection = %q, want %q", entries[0].Connection, "my-connection")
		}
		if entries[0].RequestID != "req-123" {
			t.Errorf("RequestID = %q, want %q", entries[0].RequestID, "req-123")
		}
	})
}

func TestSanitizeArguments(t *testing.T) {
	t.Run("nil args returns nil", func(t *testing.T) {
		result := sanitizeArguments(nil)
		if result != nil {
			t.Error("expected nil")
		}
	})

	t.Run("sanitizes content field", func(t *testing.T) {
		args := map[string]any{
			"content": "secret data here",
			"bucket":  "my-bucket",
		}
		result := sanitizeArguments(args)

		// content should be replaced with length
		if result["content"] != 16 {
			t.Errorf("content = %v, want 16", result["content"])
		}
		// bucket should be preserved
		if result["bucket"] != "my-bucket" {
			t.Errorf("bucket = %v, want my-bucket", result["bucket"])
		}
	})

	t.Run("sanitizes metadata field", func(t *testing.T) {
		args := map[string]any{
			"metadata": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		}
		result := sanitizeArguments(args)

		// metadata should be replaced with keys only
		keys, ok := result["metadata"].([]string)
		if !ok {
			t.Fatalf("metadata should be []string, got %T", result["metadata"])
		}
		if len(keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(keys))
		}
	})

	t.Run("handles non-string content", func(t *testing.T) {
		args := map[string]any{
			"content": 12345,
		}
		result := sanitizeArguments(args)
		if result["content"] != "[content]" {
			t.Errorf("content = %v, want [content]", result["content"])
		}
	})

	t.Run("handles non-map metadata", func(t *testing.T) {
		args := map[string]any{
			"metadata": "not a map",
		}
		result := sanitizeArguments(args)
		if result["metadata"] != "[metadata]" {
			t.Errorf("metadata = %v, want [metadata]", result["metadata"])
		}
	})
}

func TestReadOnlyInterceptor_Name(t *testing.T) {
	interceptor := NewReadOnlyInterceptor(true)
	if interceptor.Name() != "readonly" {
		t.Errorf("Name() = %q, want %q", interceptor.Name(), "readonly")
	}
}

func TestSizeLimitInterceptor_Name(t *testing.T) {
	interceptor := NewSizeLimitInterceptor(1024, 1024)
	if interceptor.Name() != "sizelimit" {
		t.Errorf("Name() = %q, want %q", interceptor.Name(), "sizelimit")
	}
}

func TestPrefixACLInterceptor_Name(t *testing.T) {
	interceptor := NewPrefixACLInterceptor(nil, nil)
	if interceptor.Name() != "prefixacl" {
		t.Errorf("Name() = %q, want %q", interceptor.Name(), "prefixacl")
	}
}

func TestMetrics_GetAllStats(t *testing.T) {
	metrics := NewMetrics()
	metrics.RecordCall("tool1", 100*1e6, false)
	metrics.RecordCall("tool2", 200*1e6, false)
	metrics.RecordCall("tool1", 150*1e6, true)

	allStats := metrics.GetAllStats()

	if len(allStats) != 2 {
		t.Errorf("expected 2 tools, got %d", len(allStats))
	}

	if stats, ok := allStats["tool1"]; ok {
		if stats.Calls != 2 {
			t.Errorf("tool1 calls = %d, want 2", stats.Calls)
		}
	} else {
		t.Error("tool1 not found in stats")
	}
}
