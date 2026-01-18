package tools

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestErrorResultf(t *testing.T) {
	result := ErrorResultf("error %d: %s", 42, "something went wrong")

	if !result.IsError {
		t.Error("ErrorResultf should set IsError to true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	expected := "Error: error 42: something went wrong"
	if textContent.Text != expected {
		t.Errorf("text = %q, want %q", textContent.Text, expected)
	}
}

func TestOptionalMetadata(t *testing.T) {
	t.Run("returns metadata map", func(t *testing.T) {
		args := map[string]any{
			"metadata": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		}

		result := OptionalMetadata(args, "metadata")
		if result == nil {
			t.Fatal("expected non-nil metadata")
		}
		if result["key1"] != "value1" {
			t.Errorf("key1 = %q, want %q", result["key1"], "value1")
		}
		if result["key2"] != "value2" {
			t.Errorf("key2 = %q, want %q", result["key2"], "value2")
		}
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		args := map[string]any{}
		result := OptionalMetadata(args, "metadata")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("returns nil for wrong type", func(t *testing.T) {
		args := map[string]any{
			"metadata": "not a map",
		}
		result := OptionalMetadata(args, "metadata")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("filters non-string values", func(t *testing.T) {
		args := map[string]any{
			"metadata": map[string]any{
				"string": "value",
				"int":    42,
				"bool":   true,
			},
		}

		result := OptionalMetadata(args, "metadata")
		if result == nil {
			t.Fatal("expected non-nil metadata")
		}
		if result["string"] != "value" {
			t.Error("expected string key to be present")
		}
		if _, ok := result["int"]; ok {
			t.Error("int key should be filtered out")
		}
		if _, ok := result["bool"]; ok {
			t.Error("bool key should be filtered out")
		}
	})
}

func TestOptionalInt_TypeConversions(t *testing.T) {
	t.Run("handles float64", func(t *testing.T) {
		args := map[string]any{"num": float64(42.0)}
		result := OptionalInt(args, "num", 0)
		if result != 42 {
			t.Errorf("OptionalInt() = %d, want 42", result)
		}
	})

	t.Run("handles int64", func(t *testing.T) {
		args := map[string]any{"num": int64(42)}
		result := OptionalInt(args, "num", 0)
		if result != 42 {
			t.Errorf("OptionalInt() = %d, want 42", result)
		}
	})

	t.Run("returns default for unsupported type", func(t *testing.T) {
		args := map[string]any{"num": "not a number"}
		result := OptionalInt(args, "num", 99)
		if result != 99 {
			t.Errorf("OptionalInt() = %d, want 99", result)
		}
	})
}

func TestOptionalInt32_BoundsCheck(t *testing.T) {
	t.Run("clamps to MaxInt32", func(t *testing.T) {
		args := map[string]any{"num": int(1 << 40)} // larger than int32
		result := OptionalInt32(args, "num", 0)
		if result != 2147483647 { // math.MaxInt32
			t.Errorf("OptionalInt32() = %d, want MaxInt32", result)
		}
	})
}

func TestRequireString_EdgeCases(t *testing.T) {
	t.Run("returns error for nil value", func(t *testing.T) {
		args := map[string]any{"key": nil}
		_, err := RequireString(args, "key")
		if err == nil {
			t.Error("expected error for nil value")
		}
	})

	t.Run("returns error for wrong type", func(t *testing.T) {
		args := map[string]any{"key": 42}
		_, err := RequireString(args, "key")
		if err == nil {
			t.Error("expected error for wrong type")
		}
	})
}

func TestJSONResult_Error(t *testing.T) {
	// Test with value that can't be marshaled
	ch := make(chan int)
	_, err := JSONResult(ch)

	if err == nil {
		t.Error("JSONResult with unmarshalable value should return error")
	}
}
