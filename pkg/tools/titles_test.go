package tools

import "testing"

func TestDefaultTitle(t *testing.T) {
	t.Run("returns title for known tool", func(t *testing.T) {
		title := DefaultTitle(ToolListBuckets)
		if title == "" {
			t.Error("expected non-empty title for ToolListBuckets")
		}
	})

	t.Run("returns empty for unknown tool", func(t *testing.T) {
		title := DefaultTitle("unknown_tool")
		if title != "" {
			t.Errorf("expected empty title for unknown tool, got %q", title)
		}
	})

	t.Run("all tools have defaults", func(t *testing.T) {
		for _, name := range AllTools() {
			title := DefaultTitle(name)
			if title == "" {
				t.Errorf("tool %s has no default title", name)
			}
		}
	})

	t.Run("ListBuckets title is correct", func(t *testing.T) {
		if got := DefaultTitle(ToolListBuckets); got != "List Buckets" {
			t.Errorf("expected %q, got %q", "List Buckets", got)
		}
	})
}

func TestGetTitle(t *testing.T) {
	t.Run("returns default when no overrides", func(t *testing.T) {
		tk := &Toolkit{}
		title := tk.getTitle(ToolListBuckets, nil)
		if title != defaultTitles[ToolListBuckets] {
			t.Errorf("expected default title, got %q", title)
		}
	})

	t.Run("toolkit-level override wins over default", func(t *testing.T) {
		tk := &Toolkit{
			titles: map[ToolName]string{
				ToolListBuckets: "custom toolkit title",
			},
		}
		title := tk.getTitle(ToolListBuckets, nil)
		if title != "custom toolkit title" {
			t.Errorf("expected toolkit override, got %q", title)
		}
	})

	t.Run("per-registration override wins over toolkit", func(t *testing.T) {
		tk := &Toolkit{
			titles: map[ToolName]string{
				ToolListBuckets: "toolkit title",
			},
		}
		regTitle := "per-registration title"
		cfg := &toolConfig{title: &regTitle}
		title := tk.getTitle(ToolListBuckets, cfg)
		if title != "per-registration title" {
			t.Errorf("expected per-registration override, got %q", title)
		}
	})

	t.Run("nil config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{
			titles: map[ToolName]string{
				ToolGetObject: "toolkit get title",
			},
		}
		title := tk.getTitle(ToolGetObject, nil)
		if title != "toolkit get title" {
			t.Errorf("expected toolkit override, got %q", title)
		}
	})

	t.Run("empty toolkit map falls through to default", func(t *testing.T) {
		tk := &Toolkit{
			titles: map[ToolName]string{},
		}
		title := tk.getTitle(ToolDeleteObject, nil)
		if title != defaultTitles[ToolDeleteObject] {
			t.Errorf("expected default title, got %q", title)
		}
	})

	t.Run("nil title in config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{
			titles: map[ToolName]string{
				ToolListBuckets: "toolkit title",
			},
		}
		cfg := &toolConfig{title: nil}
		title := tk.getTitle(ToolListBuckets, cfg)
		if title != "toolkit title" {
			t.Errorf("expected toolkit override, got %q", title)
		}
	})
}
