package tools

// defaultTitles holds the default human-readable title for each built-in tool.
// These are used when no override is provided via WithTitle or WithTitles.
var defaultTitles = map[ToolName]string{
	ToolListBuckets:       "List Buckets",
	ToolListConnections:   "List Connections",
	ToolListObjects:       "List Objects",
	ToolGetObject:         "Get Object",
	ToolGetObjectMetadata: "Get Object Metadata",
	ToolPresignURL:        "Generate Presigned URL",
	ToolPutObject:         "Put Object",
	ToolCopyObject:        "Copy Object",
	ToolDeleteObject:      "Delete Object",
}

// DefaultTitle returns the default human-readable title for a tool.
// Returns an empty string for unknown tool names.
func DefaultTitle(name ToolName) string {
	return defaultTitles[name]
}

// getTitle resolves the title for a tool using the priority chain:
// 1. Per-registration override (cfg.title) — highest priority
// 2. Toolkit-level override (t.titles) — medium priority
// 3. Default title — lowest priority.
func (t *Toolkit) getTitle(name ToolName, cfg *toolConfig) string {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.title != nil {
		return *cfg.title
	}

	// Toolkit-level override (medium priority)
	if title, ok := t.titles[name]; ok {
		return title
	}

	// Default title (lowest priority)
	return defaultTitles[name]
}
