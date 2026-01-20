package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

func makeCallToolRequest(args map[string]any) *mcp.CallToolRequest {
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{},
	}
	if args != nil {
		argsJSON, _ := json.Marshal(args)
		req.Params.Arguments = argsJSON
	}
	return req
}

func TestReadOnlyInterceptor(t *testing.T) {
	t.Run("enabled blocks write tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(true)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		result := interceptor.Intercept(context.Background(), tc, makeCallToolRequest(nil))
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("enabled allows read tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(true)
		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		result := interceptor.Intercept(context.Background(), tc, makeCallToolRequest(nil))
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("disabled allows all tools", func(t *testing.T) {
		interceptor := NewReadOnlyInterceptor(false)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		result := interceptor.Intercept(context.Background(), tc, makeCallToolRequest(nil))
		assertBool(t, "Allow", true, result.Allow)
	})
}

func TestSizeLimitInterceptor(t *testing.T) {
	t.Run("allows small content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(10*1024*1024, 1024)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := makeCallToolRequest(map[string]any{"content": "small content"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("blocks large content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(10*1024*1024, 10)
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := makeCallToolRequest(map[string]any{"content": "this content is way too long"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})
}

func TestPrefixACLInterceptor(t *testing.T) {
	t.Run("denies blocked prefix", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"private/", "secret/"})
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := makeCallToolRequest(map[string]any{"key": "private/file.txt"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("allows non-blocked prefix", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"private/", "secret/"})
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := makeCallToolRequest(map[string]any{"key": "public/file.txt"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("requires allowed prefix when specified", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor([]string{"allowed/"}, nil)
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := makeCallToolRequest(map[string]any{"key": "not-allowed/file.txt"})
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

	t.Run("After logs successful call", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		tc.StartTime = time.Now().Add(-100 * time.Millisecond) // Simulate elapsed time

		result := tools.TextResult("success")
		_, err := mw.After(context.Background(), tc, result, nil)
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
		if entries[0].Tool != "s3_list_buckets" {
			t.Errorf("Tool = %q, want %q", entries[0].Tool, "s3_list_buckets")
		}
	})

	t.Run("After logs error call", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		tc.StartTime = time.Now()

		_, _ = mw.After(context.Background(), tc, nil, context.DeadlineExceeded)

		entries := logger.Entries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Success {
			t.Error("expected success=false")
		}
		if entries[0].Error != "context deadline exceeded" {
			t.Errorf("Error = %q, want %q", entries[0].Error, "context deadline exceeded")
		}
	})

	t.Run("After logs result with IsError", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		tc.StartTime = time.Now()

		result := tools.ErrorResult("result error")
		_, _ = mw.After(context.Background(), tc, result, nil)

		entries := logger.Entries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Success {
			t.Error("expected success=false for error result")
		}
	})

	t.Run("After uses tool context", func(t *testing.T) {
		logger := NewBufferedAuditLogger()
		mw := NewAuditMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "my-connection")
		tc.RequestID = "req-123"
		tc.StartTime = time.Now()

		result := tools.TextResult("ok")
		_, _ = mw.After(context.Background(), tc, result, nil)

		entries := logger.Entries()
		if entries[0].Connection != "my-connection" {
			t.Errorf("Connection = %q, want %q", entries[0].Connection, "my-connection")
		}
		if entries[0].RequestID != "req-123" {
			t.Errorf("RequestID = %q, want %q", entries[0].RequestID, "req-123")
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

func TestLoggingMiddleware(t *testing.T) {
	t.Run("name returns logging", func(t *testing.T) {
		mw := NewLoggingMiddleware(slog.Default())
		if mw.Name() != "logging" {
			t.Errorf("Name() = %q, want %q", mw.Name(), "logging")
		}
	})

	t.Run("Before logs start", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		mw := NewLoggingMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		_, err := mw.Before(context.Background(), tc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logOutput := buf.String()
		if !strings.Contains(logOutput, "s3_list_buckets") {
			t.Error("expected log to contain tool name")
		}
	})

	t.Run("After logs completion", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		mw := NewLoggingMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "my-connection")
		tc.RequestID = "req-456"
		tc.StartTime = time.Now().Add(-100 * time.Millisecond)

		result := tools.TextResult("success")
		_, err := mw.After(context.Background(), tc, result, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logOutput := buf.String()
		if !strings.Contains(logOutput, "s3_list_buckets") {
			t.Error("expected log to contain tool name")
		}
	})

	t.Run("After logs error", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		mw := NewLoggingMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		tc.StartTime = time.Now()

		_, _ = mw.After(context.Background(), tc, nil, context.DeadlineExceeded)

		logOutput := buf.String()
		if !strings.Contains(logOutput, "error") {
			t.Error("expected log to contain error")
		}
	})

	t.Run("After logs result with IsError", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		mw := NewLoggingMiddleware(logger)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		tc.StartTime = time.Now()

		result := tools.ErrorResult("result error")
		_, _ = mw.After(context.Background(), tc, result, nil)

		logOutput := buf.String()
		if !strings.Contains(logOutput, "WARN") {
			t.Error("expected log to contain warning level")
		}
	})
}

func TestMetricsMiddleware(t *testing.T) {
	t.Run("name returns metrics", func(t *testing.T) {
		metrics := NewMetrics()
		mw := NewMetricsMiddleware(metrics)
		if mw.Name() != "metrics" {
			t.Errorf("Name() = %q, want %q", mw.Name(), "metrics")
		}
	})

	t.Run("After tracks successful call", func(t *testing.T) {
		metrics := NewMetrics()
		mw := NewMetricsMiddleware(metrics)

		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		tc.StartTime = time.Now().Add(-100 * time.Millisecond)

		result := tools.TextResult("success")
		_, err := mw.After(context.Background(), tc, result, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stats := metrics.GetToolStats("s3_list_buckets")
		if stats.Calls != 1 {
			t.Errorf("Calls = %d, want 1", stats.Calls)
		}
		if stats.Errors != 0 {
			t.Errorf("Errors = %d, want 0", stats.Errors)
		}
	})

	t.Run("After tracks error call", func(t *testing.T) {
		metrics := NewMetrics()
		mw := NewMetricsMiddleware(metrics)

		tc := tools.NewToolContext(tools.ToolGetObject, "")
		tc.StartTime = time.Now()

		_, _ = mw.After(context.Background(), tc, nil, context.DeadlineExceeded)

		stats := metrics.GetToolStats("s3_get_object")
		if stats.Errors != 1 {
			t.Errorf("Errors = %d, want 1", stats.Errors)
		}
	})

	t.Run("After tracks result with IsError", func(t *testing.T) {
		metrics := NewMetrics()
		mw := NewMetricsMiddleware(metrics)

		tc := tools.NewToolContext(tools.ToolPutObject, "")
		tc.StartTime = time.Now()

		result := tools.ErrorResult("result error")
		_, _ = mw.After(context.Background(), tc, result, nil)

		stats := metrics.GetToolStats("s3_put_object")
		if stats.Errors != 1 {
			t.Errorf("Errors = %d, want 1 for error result", stats.Errors)
		}
	})
}

func TestSizeLimitInterceptor_Intercept(t *testing.T) {
	t.Run("allows get within limit", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(1024, 1024)
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := makeCallToolRequest(map[string]any{"bucket": "b", "key": "k"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("allows non-size-limited tools", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(1024, 1024)
		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		req := makeCallToolRequest(nil)
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})

	t.Run("blocks large put content", func(t *testing.T) {
		interceptor := NewSizeLimitInterceptor(1024, 10) // 10 byte put limit
		tc := tools.NewToolContext(tools.ToolPutObject, "")
		req := makeCallToolRequest(map[string]any{"content": "this is way more than 10 bytes of content"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})
}

func TestPrefixACLInterceptor_ExtractKey(t *testing.T) {
	t.Run("extracts key from get object", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"blocked/"})
		tc := tools.NewToolContext(tools.ToolGetObject, "")
		req := makeCallToolRequest(map[string]any{"key": "blocked/file.txt"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("extracts source_key from copy object", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"blocked/"})
		tc := tools.NewToolContext(tools.ToolCopyObject, "")
		req := makeCallToolRequest(map[string]any{"source_key": "blocked/file.txt", "dest_key": "allowed/file.txt"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("extracts dest_key from copy object when source is empty", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"blocked/"})
		tc := tools.NewToolContext(tools.ToolCopyObject, "")
		req := makeCallToolRequest(map[string]any{"dest_key": "blocked/file.txt"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("extracts prefix from list objects", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"blocked/"})
		tc := tools.NewToolContext(tools.ToolListObjects, "")
		req := makeCallToolRequest(map[string]any{"prefix": "blocked/subdir/"})
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", false, result.Allow)
	})

	t.Run("allows when no prefix in request", func(t *testing.T) {
		interceptor := NewPrefixACLInterceptor(nil, []string{"blocked/"})
		tc := tools.NewToolContext(tools.ToolListBuckets, "")
		req := makeCallToolRequest(nil)
		result := interceptor.Intercept(context.Background(), tc, req)
		assertBool(t, "Allow", true, result.Allow)
	})
}
