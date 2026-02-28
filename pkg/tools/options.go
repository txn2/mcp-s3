package tools

import (
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Option is a functional option for configuring a Toolkit.
type Option func(*Toolkit)

// WithMiddleware adds middleware to the toolkit.
func WithMiddleware(m ...ToolMiddleware) Option {
	return func(t *Toolkit) {
		for _, middleware := range m {
			t.middleware.Register(middleware)
		}
	}
}

// WithInterceptor adds a request interceptor to the toolkit.
func WithInterceptor(i ...RequestInterceptor) Option {
	return func(t *Toolkit) {
		for _, interceptor := range i {
			t.interceptors.Add(interceptor)
		}
	}
}

// WithTransformer adds a result transformer to the toolkit.
func WithTransformer(tr ...ResultTransformer) Option {
	return func(t *Toolkit) {
		for _, transformer := range tr {
			t.transformers.Add(transformer)
		}
	}
}

// WithLogger sets the logger for the toolkit.
func WithLogger(logger *slog.Logger) Option {
	return func(t *Toolkit) {
		t.logger = logger
	}
}

// WithDefaultConnection sets the default connection name.
func WithDefaultConnection(name string) Option {
	return func(t *Toolkit) {
		t.defaultConnection = name
	}
}

// WithReadOnly enables read-only mode, blocking all write operations.
func WithReadOnly(readOnly bool) Option {
	return func(t *Toolkit) {
		t.readOnly = readOnly
	}
}

// WithMaxGetSize sets the maximum size for object retrieval.
func WithMaxGetSize(size int64) Option {
	return func(t *Toolkit) {
		t.maxGetSize = size
	}
}

// WithMaxPutSize sets the maximum size for object uploads.
func WithMaxPutSize(size int64) Option {
	return func(t *Toolkit) {
		t.maxPutSize = size
	}
}

// WithToolPrefix sets a prefix for all tool names.
// This is useful when composing multiple toolkits.
func WithToolPrefix(prefix string) Option {
	return func(t *Toolkit) {
		t.toolPrefix = prefix
	}
}

// WithClientProvider sets a function that provides S3 clients by connection name.
func WithClientProvider(provider func(name string) (S3Client, error)) Option {
	return func(t *Toolkit) {
		t.clientProvider = provider
	}
}

// DisableTool disables specific tools from being registered.
func DisableTool(names ...ToolName) Option {
	return func(t *Toolkit) {
		for _, name := range names {
			t.disabledTools[name] = true
		}
	}
}

// EnableOnlyTools enables only the specified tools, disabling all others.
func EnableOnlyTools(names ...ToolName) Option {
	return func(t *Toolkit) {
		// Disable all tools first
		for _, tool := range AllTools() {
			t.disabledTools[tool] = true
		}
		// Enable only the specified tools
		for _, name := range names {
			delete(t.disabledTools, name)
		}
	}
}

// WithToolMiddleware adds middleware for specific tools.
func WithToolMiddleware(name ToolName, m ...ToolMiddleware) Option {
	return func(t *Toolkit) {
		t.toolMiddlewares[name] = append(t.toolMiddlewares[name], m...)
	}
}

// WithTitles sets toolkit-level human-readable title overrides for tools.
// These take priority over default titles but can be overridden
// by per-registration WithTitle options.
func WithTitles(titles map[ToolName]string) Option {
	return func(t *Toolkit) {
		t.titles = make(map[ToolName]string, len(titles))
		for k, v := range titles {
			t.titles[k] = v
		}
	}
}

// WithTitle sets a per-registration human-readable title override for a single tool.
// This has the highest priority in the title resolution chain.
func WithTitle(title string) ToolOption {
	return func(cfg *toolConfig) {
		cfg.title = &title
	}
}

// WithDescriptions sets toolkit-level description overrides for tools.
// These take priority over default descriptions but can be overridden
// by per-registration WithDescription options.
func WithDescriptions(descs map[ToolName]string) Option {
	return func(t *Toolkit) {
		t.descriptions = make(map[ToolName]string, len(descs))
		for k, v := range descs {
			t.descriptions[k] = v
		}
	}
}

// WithDescription sets a per-registration description override for a single tool.
// This has the highest priority in the description resolution chain.
func WithDescription(desc string) ToolOption {
	return func(cfg *toolConfig) {
		cfg.description = &desc
	}
}

// WithAnnotations sets toolkit-level annotation overrides for tools.
// These take priority over default annotations but can be overridden
// by per-registration WithAnnotation options.
func WithAnnotations(anns map[ToolName]*mcp.ToolAnnotations) Option {
	return func(t *Toolkit) {
		t.annotations = make(map[ToolName]*mcp.ToolAnnotations, len(anns))
		for k, v := range anns {
			t.annotations[k] = v
		}
	}
}

// WithAnnotation sets a per-registration annotation override for a single tool.
// This has the highest priority in the annotation resolution chain.
func WithAnnotation(ann *mcp.ToolAnnotations) ToolOption {
	return func(cfg *toolConfig) {
		cfg.annotations = ann
	}
}

// WithIcons sets toolkit-level icon overrides for tools.
// These take priority over default icons but can be overridden
// by per-registration WithIcon options.
func WithIcons(icons map[ToolName][]mcp.Icon) Option {
	return func(t *Toolkit) {
		if t.icons == nil {
			t.icons = make(map[ToolName][]mcp.Icon, len(icons))
		}
		for k, v := range icons {
			t.icons[k] = v
		}
	}
}

// WithIcon sets a per-registration icon override for a single tool.
// This has the highest priority in the icon resolution chain.
func WithIcon(icons []mcp.Icon) ToolOption {
	return func(cfg *toolConfig) {
		cfg.icons = icons
	}
}

// WithOutputSchemas sets toolkit-level output schema overrides for tools.
// These take priority over default output schemas but can be overridden
// by per-registration WithOutputSchema options.
func WithOutputSchemas(schemas map[ToolName]any) Option {
	return func(t *Toolkit) {
		t.outputSchemas = make(map[ToolName]any, len(schemas))
		for k, v := range schemas {
			t.outputSchemas[k] = v
		}
	}
}

// WithOutputSchema sets a per-registration output schema override for a single tool.
// This has the highest priority in the output schema resolution chain.
func WithOutputSchema(schema any) ToolOption {
	return func(cfg *toolConfig) {
		cfg.outputSchema = schema
	}
}

// defaultLogger returns a default no-op logger.
func defaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
