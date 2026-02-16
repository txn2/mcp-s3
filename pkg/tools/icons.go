package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// defaultIcons holds the default icon for all S3 tools.
// Uses a GitHub-hosted SVG from the mcp-s3 repository.
var defaultIcons = []mcp.Icon{{
	Source:   "https://raw.githubusercontent.com/txn2/mcp-s3/main/icons/s3.svg",
	MIMEType: "image/svg+xml",
}}

// DefaultIcons returns the default icons for S3 tools.
func DefaultIcons() []mcp.Icon {
	return defaultIcons
}

// getIcons resolves the icons for a tool using the priority chain:
// 1. Per-registration override (cfg.icons) — highest priority
// 2. Toolkit-level override (t.icons) — medium priority
// 3. Default icons — lowest priority.
func (t *Toolkit) getIcons(name ToolName, cfg *toolConfig) []mcp.Icon {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.icons != nil {
		return cfg.icons
	}

	// Toolkit-level override (medium priority)
	if t.icons != nil {
		if icons, ok := t.icons[name]; ok {
			return icons
		}
	}

	// Default icons (lowest priority)
	return defaultIcons
}
