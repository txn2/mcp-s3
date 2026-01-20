package tools

import (
	"io"
	"log/slog"
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

// defaultLogger returns a default no-op logger.
func defaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
