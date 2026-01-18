package tools

import (
	"context"
	"testing"
)

func TestToolContext_GetString(t *testing.T) {
	tc := NewToolContext("test-tool", "test-conn")

	t.Run("returns string value", func(t *testing.T) {
		tc.Set("key", "value")
		if got := tc.GetString("key"); got != "value" {
			t.Errorf("GetString() = %q, want %q", got, "value")
		}
	})

	t.Run("returns empty for non-string", func(t *testing.T) {
		tc.Set("int-key", 42)
		if got := tc.GetString("int-key"); got != "" {
			t.Errorf("GetString() = %q, want empty", got)
		}
	})

	t.Run("returns empty for missing key", func(t *testing.T) {
		if got := tc.GetString("missing"); got != "" {
			t.Errorf("GetString() = %q, want empty", got)
		}
	})
}

func TestToolContext_GetInt(t *testing.T) {
	tc := NewToolContext("test-tool", "test-conn")

	t.Run("returns int value", func(t *testing.T) {
		tc.Set("count", 42)
		if got := tc.GetInt("count"); got != 42 {
			t.Errorf("GetInt() = %d, want %d", got, 42)
		}
	})

	t.Run("returns zero for non-int", func(t *testing.T) {
		tc.Set("str-key", "not an int")
		if got := tc.GetInt("str-key"); got != 0 {
			t.Errorf("GetInt() = %d, want 0", got)
		}
	})

	t.Run("returns zero for missing key", func(t *testing.T) {
		if got := tc.GetInt("missing"); got != 0 {
			t.Errorf("GetInt() = %d, want 0", got)
		}
	})
}

func TestToolContext_Has(t *testing.T) {
	tc := NewToolContext("test-tool", "test-conn")

	t.Run("returns true for existing key", func(t *testing.T) {
		tc.Set("exists", "value")
		if !tc.Has("exists") {
			t.Error("Has() = false, want true")
		}
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		if tc.Has("does-not-exist") {
			t.Error("Has() = true, want false")
		}
	})
}

func TestToolContext_Delete(t *testing.T) {
	tc := NewToolContext("test-tool", "test-conn")

	tc.Set("to-delete", "value")
	if !tc.Has("to-delete") {
		t.Fatal("key should exist before delete")
	}

	tc.Delete("to-delete")
	if tc.Has("to-delete") {
		t.Error("key should not exist after delete")
	}

	// Delete non-existent key should not panic
	tc.Delete("non-existent")
}

func TestToolContext_Clone(t *testing.T) {
	tc := NewToolContext("test-tool", "test-conn")
	tc.RequestID = "req-123"
	tc.Set("key1", "value1")
	tc.Set("key2", 42)

	cloned := tc.Clone()

	// Verify fields are copied
	if cloned.ToolName != tc.ToolName {
		t.Errorf("ToolName = %q, want %q", cloned.ToolName, tc.ToolName)
	}
	if cloned.ConnectionName != tc.ConnectionName {
		t.Errorf("ConnectionName = %q, want %q", cloned.ConnectionName, tc.ConnectionName)
	}
	if cloned.RequestID != tc.RequestID {
		t.Errorf("RequestID = %q, want %q", cloned.RequestID, tc.RequestID)
	}

	// Verify values are copied
	if cloned.GetString("key1") != "value1" {
		t.Error("cloned value mismatch for key1")
	}
	if cloned.GetInt("key2") != 42 {
		t.Error("cloned value mismatch for key2")
	}

	// Verify independence
	cloned.Set("key1", "modified")
	if tc.GetString("key1") != "value1" {
		t.Error("modifying clone affected original")
	}
}

func TestWithToolContext_GetToolContext(t *testing.T) {
	t.Run("stores and retrieves ToolContext", func(t *testing.T) {
		tc := NewToolContext("my-tool", "my-conn")
		ctx := WithToolContext(context.Background(), tc)

		retrieved := GetToolContext(ctx)
		if retrieved != tc {
			t.Error("GetToolContext did not return the same ToolContext")
		}
	})

	t.Run("returns nil for context without ToolContext", func(t *testing.T) {
		ctx := context.Background()
		if got := GetToolContext(ctx); got != nil {
			t.Errorf("GetToolContext() = %v, want nil", got)
		}
	})

	t.Run("returns nil for wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), toolContextKey, "not a ToolContext")
		if got := GetToolContext(ctx); got != nil {
			t.Errorf("GetToolContext() = %v, want nil", got)
		}
	})
}
