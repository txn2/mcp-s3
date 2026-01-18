package tools

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func FuzzRequireString(f *testing.F) {
	f.Add("key", "value")
	f.Add("key", "")
	f.Add("", "value")
	f.Add("bucket", "my-bucket")
	f.Add("key", "path/to/object.txt")

	f.Fuzz(func(t *testing.T, key, value string) {
		args := map[string]any{key: value}

		result, err := RequireString(args, key)
		if value == "" {
			// Empty string should return error
			if err == nil {
				t.Errorf("RequireString should error on empty value")
			}
		} else {
			// Non-empty should succeed
			if err != nil {
				t.Errorf("RequireString(%q, %q) unexpected error: %v", key, value, err)
			}
			if result != value {
				t.Errorf("RequireString(%q, %q) = %q, expected %q", key, value, result, value)
			}
		}
	})
}

func FuzzOptionalString(f *testing.F) {
	f.Add("key", "value", "default")
	f.Add("key", "", "default")
	f.Add("missing", "value", "default")

	f.Fuzz(func(t *testing.T, key, value, defaultValue string) {
		args := map[string]any{"key": value}

		result := OptionalString(args, key, defaultValue)

		// Should never panic and should return either value or default
		if key == "key" {
			if result != value {
				t.Errorf("OptionalString returned %q, expected %q", result, value)
			}
		} else {
			if result != defaultValue {
				t.Errorf("OptionalString returned %q, expected default %q", result, defaultValue)
			}
		}
	})
}

func FuzzOptionalInt(f *testing.F) {
	f.Add("key", 42, 0)
	f.Add("key", -1, 100)
	f.Add("key", 0, 50)
	f.Add("missing", 10, 99)

	f.Fuzz(func(t *testing.T, key string, value, defaultValue int) {
		args := map[string]any{"key": value}

		result := OptionalInt(args, key, defaultValue)

		// Should never panic
		if key == "key" {
			if result != value {
				t.Errorf("OptionalInt returned %d, expected %d", result, value)
			}
		} else {
			if result != defaultValue {
				t.Errorf("OptionalInt returned %d, expected default %d", result, defaultValue)
			}
		}
	})
}

func FuzzOptionalBool(f *testing.F) {
	f.Add("key", true, false)
	f.Add("key", false, true)
	f.Add("missing", true, false)

	f.Fuzz(func(t *testing.T, key string, value, defaultValue bool) {
		args := map[string]any{"key": value}

		result := OptionalBool(args, key, defaultValue)

		// Should never panic
		if key == "key" {
			if result != value {
				t.Errorf("OptionalBool returned %v, expected %v", result, value)
			}
		} else {
			if result != defaultValue {
				t.Errorf("OptionalBool returned %v, expected default %v", result, defaultValue)
			}
		}
	})
}

func FuzzGetArgs(f *testing.F) {
	f.Add("bucket", "key")

	f.Fuzz(func(t *testing.T, bucket, key string) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"bucket": bucket,
			"key":    key,
		}

		args, err := GetArgs(req)
		if err != nil {
			t.Errorf("GetArgs failed: %v", err)
		}
		if args == nil {
			t.Error("GetArgs returned nil args")
		}
	})
}
