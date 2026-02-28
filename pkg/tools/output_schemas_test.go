package tools

import (
	"fmt"
	"testing"
)

func TestDefaultOutputSchema(t *testing.T) {
	t.Run("returns schema for known tool", func(t *testing.T) {
		schema := DefaultOutputSchema(ToolListBuckets)
		if schema == nil {
			t.Error("expected non-nil schema for ToolListBuckets")
		}
	})

	t.Run("returns nil for unknown tool", func(t *testing.T) {
		schema := DefaultOutputSchema("unknown_tool")
		if schema != nil {
			t.Errorf("expected nil schema for unknown tool, got %v", schema)
		}
	})

	t.Run("all tools have defaults", func(t *testing.T) {
		for _, name := range AllTools() {
			schema := DefaultOutputSchema(name)
			if schema == nil {
				t.Errorf("tool %s has no default output schema", name)
			}
		}
	})

	t.Run("schema is map[string]any with type object", func(t *testing.T) {
		for _, name := range AllTools() {
			schema := DefaultOutputSchema(name)
			m, ok := schema.(map[string]any)
			if !ok {
				t.Errorf("tool %s schema is not map[string]any", name)
				continue
			}
			typ, ok := m["type"]
			if !ok {
				t.Errorf("tool %s schema missing 'type' key", name)
				continue
			}
			if typ != "object" {
				t.Errorf("tool %s schema type = %q, want %q", name, typ, "object")
			}
		}
	})
}

func TestGetOutputSchema(t *testing.T) {
	// Use fmt.Stringer-style sentinel strings as schema values to avoid
	// non-comparable map comparison in tests. The priority chain only
	// cares that the correct value is returned, not the schema structure.
	const sentinelDefault = "default-schema"
	const sentinelToolkit = "toolkit-schema"
	const sentinelReg = "registration-schema"

	t.Run("returns default when no overrides", func(t *testing.T) {
		tk := &Toolkit{}
		schema := tk.getOutputSchema(ToolListBuckets, nil)
		// Should return the real default map, not nil.
		if schema == nil {
			t.Error("expected non-nil default output schema")
		}
	})

	t.Run("toolkit-level override wins over default", func(t *testing.T) {
		tk := &Toolkit{
			outputSchemas: map[ToolName]any{
				ToolListBuckets: sentinelToolkit,
			},
		}
		schema := tk.getOutputSchema(ToolListBuckets, nil)
		if fmt.Sprintf("%v", schema) != sentinelToolkit {
			t.Errorf("expected toolkit override, got %v", schema)
		}
	})

	t.Run("per-registration override wins over toolkit", func(t *testing.T) {
		tk := &Toolkit{
			outputSchemas: map[ToolName]any{
				ToolListBuckets: sentinelToolkit,
			},
		}
		cfg := &toolConfig{outputSchema: sentinelReg}
		schema := tk.getOutputSchema(ToolListBuckets, cfg)
		if fmt.Sprintf("%v", schema) != sentinelReg {
			t.Errorf("expected per-registration override, got %v", schema)
		}
	})

	t.Run("nil config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{
			outputSchemas: map[ToolName]any{
				ToolGetObject: sentinelToolkit,
			},
		}
		schema := tk.getOutputSchema(ToolGetObject, nil)
		if fmt.Sprintf("%v", schema) != sentinelToolkit {
			t.Errorf("expected toolkit override, got %v", schema)
		}
	})

	t.Run("empty toolkit map falls through to default", func(t *testing.T) {
		tk := &Toolkit{
			outputSchemas: map[ToolName]any{},
		}
		schema := tk.getOutputSchema(ToolDeleteObject, nil)
		if schema == nil {
			t.Error("expected non-nil default output schema")
		}
		// Verify it is the real default by checking the type field.
		m, ok := schema.(map[string]any)
		if !ok {
			t.Fatal("expected schema to be map[string]any")
		}
		if m["type"] != "object" {
			t.Errorf("expected schema type 'object', got %v", m["type"])
		}
	})

	t.Run("nil outputSchema in config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{
			outputSchemas: map[ToolName]any{
				ToolListBuckets: sentinelDefault,
			},
		}
		cfg := &toolConfig{outputSchema: nil}
		schema := tk.getOutputSchema(ToolListBuckets, cfg)
		if fmt.Sprintf("%v", schema) != sentinelDefault {
			t.Errorf("expected toolkit override when cfg.outputSchema is nil, got %v", schema)
		}
	})
}
