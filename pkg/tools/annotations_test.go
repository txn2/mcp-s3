package tools

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDefaultAnnotations(t *testing.T) {
	t.Run("returns annotations for known tool", func(t *testing.T) {
		ann := DefaultAnnotations(ToolListBuckets)
		if ann == nil {
			t.Fatal("expected non-nil annotations for ToolListBuckets")
		}
		if !ann.ReadOnlyHint {
			t.Error("expected ReadOnlyHint=true for ToolListBuckets")
		}
	})

	t.Run("returns nil for unknown tool", func(t *testing.T) {
		ann := DefaultAnnotations("unknown_tool")
		if ann != nil {
			t.Errorf("expected nil annotations for unknown tool, got %+v", ann)
		}
	})

	t.Run("all tools have defaults", func(t *testing.T) {
		for _, name := range AllTools() {
			ann := DefaultAnnotations(name)
			if ann == nil {
				t.Errorf("tool %s has no default annotations", name)
			}
		}
	})

	t.Run("read tools are read-only", func(t *testing.T) {
		readTools := ReadTools()
		for _, name := range readTools {
			ann := DefaultAnnotations(name)
			if ann == nil {
				t.Fatalf("tool %s has no default annotations", name)
			}
			if !ann.ReadOnlyHint {
				t.Errorf("tool %s should be read-only", name)
			}
		}
	})

	t.Run("write tools are not read-only", func(t *testing.T) {
		writeTools := WriteTools()
		for _, name := range writeTools {
			ann := DefaultAnnotations(name)
			if ann == nil {
				t.Fatalf("tool %s has no default annotations", name)
			}
			if ann.ReadOnlyHint {
				t.Errorf("tool %s should not be read-only", name)
			}
		}
	})

	t.Run("delete is destructive by default", func(t *testing.T) {
		ann := DefaultAnnotations(ToolDeleteObject)
		if ann == nil {
			t.Fatal("expected non-nil annotations for ToolDeleteObject")
		}
		// DestructiveHint defaults to true (nil means true)
		if ann.DestructiveHint != nil && !*ann.DestructiveHint {
			t.Error("expected DestructiveHint to be true (default) for ToolDeleteObject")
		}
	})

	t.Run("put and copy are non-destructive", func(t *testing.T) {
		for _, name := range []ToolName{ToolPutObject, ToolCopyObject} {
			ann := DefaultAnnotations(name)
			if ann == nil {
				t.Fatalf("tool %s has no default annotations", name)
			}
			if ann.DestructiveHint == nil || *ann.DestructiveHint {
				t.Errorf("tool %s should have DestructiveHint=false", name)
			}
		}
	})

	t.Run("all tools are open-world", func(t *testing.T) {
		for _, name := range AllTools() {
			ann := DefaultAnnotations(name)
			if ann == nil {
				t.Fatalf("tool %s has no default annotations", name)
			}
			if ann.OpenWorldHint == nil || !*ann.OpenWorldHint {
				t.Errorf("tool %s should have OpenWorldHint=true", name)
			}
		}
	})
}

func TestGetAnnotations(t *testing.T) {
	t.Run("returns default when no overrides", func(t *testing.T) {
		tk := &Toolkit{}
		ann := tk.getAnnotations(ToolListBuckets, nil)
		if ann != defaultAnnotations[ToolListBuckets] {
			t.Error("expected default annotations")
		}
	})

	t.Run("toolkit-level override wins over default", func(t *testing.T) {
		custom := &mcp.ToolAnnotations{ReadOnlyHint: false}
		tk := &Toolkit{
			annotations: map[ToolName]*mcp.ToolAnnotations{
				ToolListBuckets: custom,
			},
		}
		ann := tk.getAnnotations(ToolListBuckets, nil)
		if ann != custom {
			t.Error("expected toolkit override")
		}
	})

	t.Run("per-registration override wins over toolkit", func(t *testing.T) {
		tkAnns := &mcp.ToolAnnotations{ReadOnlyHint: true}
		regAnns := &mcp.ToolAnnotations{ReadOnlyHint: false}
		tk := &Toolkit{
			annotations: map[ToolName]*mcp.ToolAnnotations{
				ToolListBuckets: tkAnns,
			},
		}
		cfg := &toolConfig{annotations: regAnns}
		ann := tk.getAnnotations(ToolListBuckets, cfg)
		if ann != regAnns {
			t.Error("expected per-registration override")
		}
	})
}
