package tools

import (
	"testing"
)

// FuzzToolName tests that ToolName type is valid.
func FuzzToolName(f *testing.F) {
	f.Add("s3_list_buckets")
	f.Add("s3_get_object")
	f.Add("unknown_tool")

	f.Fuzz(func(t *testing.T, toolName string) {
		tn := ToolName(toolName)

		// These should never panic
		_ = IsWriteTool(tn)
		_ = string(tn)
	})
}

// FuzzToolContext tests ToolContext operations don't panic.
func FuzzToolContext(f *testing.F) {
	f.Add("s3_list_buckets", "connection-name", "key", "value")
	f.Add("s3_get_object", "", "", "")
	f.Add("unknown", "conn", "test-key", "test-value")

	f.Fuzz(func(t *testing.T, toolName, connName, key, value string) {
		tc := NewToolContext(ToolName(toolName), connName)

		// These should never panic
		tc.Set(key, value)
		_ = tc.Get(key)
		_ = tc.Duration()
	})
}

// FuzzInterceptResult tests InterceptResult construction.
func FuzzInterceptResult(f *testing.F) {
	f.Add("blocked reason")
	f.Add("")
	f.Add("very long reason with special chars: !@#$%^&*()")

	f.Fuzz(func(t *testing.T, reason string) {
		// These should never panic
		_ = Allowed()
		_ = Blocked(reason)
	})
}
