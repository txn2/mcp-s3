package tools

import (
	"testing"
)

func TestDefaultDescription(t *testing.T) {
	t.Run("returns description for known tool", func(t *testing.T) {
		desc := DefaultDescription(ToolListBuckets)
		if desc == "" {
			t.Error("expected non-empty description for ToolListBuckets")
		}
	})

	t.Run("returns empty for unknown tool", func(t *testing.T) {
		desc := DefaultDescription("unknown_tool")
		if desc != "" {
			t.Errorf("expected empty description for unknown tool, got %q", desc)
		}
	})

	t.Run("all tools have defaults", func(t *testing.T) {
		for _, name := range AllTools() {
			desc := DefaultDescription(name)
			if desc == "" {
				t.Errorf("tool %s has no default description", name)
			}
		}
	})
}

func TestGetDescription(t *testing.T) {
	t.Run("returns default when no overrides", func(t *testing.T) {
		tk := &Toolkit{}
		desc := tk.getDescription(ToolListBuckets, nil)
		if desc != defaultDescriptions[ToolListBuckets] {
			t.Errorf("expected default description, got %q", desc)
		}
	})

	t.Run("toolkit-level override wins over default", func(t *testing.T) {
		tk := &Toolkit{
			descriptions: map[ToolName]string{
				ToolListBuckets: "custom toolkit desc",
			},
		}
		desc := tk.getDescription(ToolListBuckets, nil)
		if desc != "custom toolkit desc" {
			t.Errorf("expected toolkit override, got %q", desc)
		}
	})

	t.Run("per-registration override wins over toolkit", func(t *testing.T) {
		tk := &Toolkit{
			descriptions: map[ToolName]string{
				ToolListBuckets: "toolkit desc",
			},
		}
		regDesc := "per-registration desc"
		cfg := &toolConfig{description: &regDesc}
		desc := tk.getDescription(ToolListBuckets, cfg)
		if desc != "per-registration desc" {
			t.Errorf("expected per-registration override, got %q", desc)
		}
	})

	t.Run("nil config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{
			descriptions: map[ToolName]string{
				ToolGetObject: "toolkit get desc",
			},
		}
		desc := tk.getDescription(ToolGetObject, nil)
		if desc != "toolkit get desc" {
			t.Errorf("expected toolkit override, got %q", desc)
		}
	})

	t.Run("empty toolkit map falls through to default", func(t *testing.T) {
		tk := &Toolkit{
			descriptions: map[ToolName]string{},
		}
		desc := tk.getDescription(ToolDeleteObject, nil)
		if desc != defaultDescriptions[ToolDeleteObject] {
			t.Errorf("expected default description, got %q", desc)
		}
	})
}
