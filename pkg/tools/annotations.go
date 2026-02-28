package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool { return &b }

// defaultAnnotations holds the default annotations for each built-in tool.
// These are used when no override is provided via WithAnnotation or WithAnnotations.
//
// SDK v1.3.0 field types:
//
//	ReadOnlyHint    bool   (default: false)
//	DestructiveHint *bool  (default: true)
//	IdempotentHint  bool   (default: false)
//	OpenWorldHint   *bool  (default: true)
var defaultAnnotations = map[ToolName]*mcp.ToolAnnotations{
	ToolListBuckets: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolListConnections: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolListObjects: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolGetObject: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolGetObjectMetadata: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolPresignURL: {
		ReadOnlyHint:  true,
		OpenWorldHint: boolPtr(true),
	},
	ToolPutObject: {
		DestructiveHint: boolPtr(false),
		IdempotentHint:  true,
		OpenWorldHint:   boolPtr(true),
	},
	ToolCopyObject: {
		DestructiveHint: boolPtr(false),
		IdempotentHint:  true,
		OpenWorldHint:   boolPtr(true),
	},
	ToolDeleteObject: {
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
}

// DefaultAnnotations returns the default annotations for a tool.
// Returns nil for unknown tool names.
func DefaultAnnotations(name ToolName) *mcp.ToolAnnotations {
	return defaultAnnotations[name]
}

// getAnnotations resolves the annotations for a tool using the priority chain:
// 1. Per-registration override (cfg.annotations) — highest priority
// 2. Toolkit-level override (t.annotations) — medium priority
// 3. Default annotations — lowest priority.
func (t *Toolkit) getAnnotations(name ToolName, cfg *toolConfig) *mcp.ToolAnnotations {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.annotations != nil {
		return cfg.annotations
	}

	// Toolkit-level override (medium priority)
	if ann, ok := t.annotations[name]; ok {
		return ann
	}

	// Default annotations (lowest priority)
	return defaultAnnotations[name]
}
