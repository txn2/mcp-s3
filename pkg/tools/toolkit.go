package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Default size limits.
const (
	// DefaultMaxGetSize is the default maximum size for object retrieval (10MB).
	DefaultMaxGetSize = 10 * 1024 * 1024

	// DefaultMaxPutSize is the default maximum size for object uploads (100MB).
	DefaultMaxPutSize = 100 * 1024 * 1024
)

// toolConfig holds per-tool configuration for registration.
type toolConfig struct {
	middlewares []ToolMiddleware
}

// ToolOption configures a single tool registration.
type ToolOption func(*toolConfig)

// WithPerToolMiddleware adds middleware specific to this tool registration.
func WithPerToolMiddleware(m ...ToolMiddleware) ToolOption {
	return func(cfg *toolConfig) {
		cfg.middlewares = append(cfg.middlewares, m...)
	}
}

// Toolkit provides a collection of S3 MCP tools with extensibility support.
type Toolkit struct {
	// Core components
	client         S3Client
	clientProvider func(name string) (S3Client, error)
	clients        map[string]S3Client
	clientsMu      sync.RWMutex

	// Configuration
	defaultConnection string
	readOnly          bool
	maxGetSize        int64
	maxPutSize        int64
	toolPrefix        string
	disabledTools     map[ToolName]bool

	// Extensibility
	middleware      *MiddlewareChain
	toolMiddlewares map[ToolName][]ToolMiddleware
	interceptors    *InterceptorChain
	transformers    *TransformerChain
	registeredTools map[ToolName]bool

	// Logging
	logger *slog.Logger
}

// NewToolkit creates a new Toolkit with the given S3 client and options.
func NewToolkit(client S3Client, opts ...Option) *Toolkit {
	t := &Toolkit{
		client:          client,
		clients:         make(map[string]S3Client),
		disabledTools:   make(map[ToolName]bool),
		middleware:      NewMiddlewareChain(),
		toolMiddlewares: make(map[ToolName][]ToolMiddleware),
		interceptors:    NewInterceptorChain(),
		transformers:    NewTransformerChain(),
		registeredTools: make(map[ToolName]bool),
		maxGetSize:      DefaultMaxGetSize,
		maxPutSize:      DefaultMaxPutSize,
		readOnly:        false,
		logger:          defaultLogger(),
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	// If client has a connection name, register it
	if client != nil && client.ConnectionName() != "" {
		t.clients[client.ConnectionName()] = client
		if t.defaultConnection == "" {
			t.defaultConnection = client.ConnectionName()
		}
	}

	return t
}

// RegisterAll adds all S3 tools to the given MCP server.
func (t *Toolkit) RegisterAll(server *mcp.Server) {
	t.Register(server, AllTools()...)
}

// Register adds specific tools by name to the MCP server.
func (t *Toolkit) Register(server *mcp.Server, names ...ToolName) {
	for _, name := range names {
		t.registerTool(server, name, nil)
	}
}

// RegisterWith adds a tool with per-registration options.
func (t *Toolkit) RegisterWith(server *mcp.Server, name ToolName, opts ...ToolOption) {
	cfg := &toolConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	t.registerTool(server, name, cfg)
}

// registerTool dispatches to the appropriate tool registration method.
func (t *Toolkit) registerTool(server *mcp.Server, name ToolName, cfg *toolConfig) {
	if t.registeredTools[name] {
		return // Prevent duplicate registration
	}
	if t.isToolDisabled(name) {
		return
	}

	switch name {
	case ToolListBuckets:
		t.registerListBucketsTool(server, cfg)
	case ToolListObjects:
		t.registerListObjectsTool(server, cfg)
	case ToolGetObject:
		t.registerGetObjectTool(server, cfg)
	case ToolGetObjectMetadata:
		t.registerGetObjectMetadataTool(server, cfg)
	case ToolPutObject:
		t.registerPutObjectTool(server, cfg)
	case ToolDeleteObject:
		t.registerDeleteObjectTool(server, cfg)
	case ToolCopyObject:
		t.registerCopyObjectTool(server, cfg)
	case ToolPresignURL:
		t.registerPresignURLTool(server, cfg)
	case ToolListConnections:
		t.registerListConnectionsTool(server, cfg)
	}

	t.registeredTools[name] = true
}

// RegisterTools registers all S3 tools with the MCP server (backward compatibility).
func (t *Toolkit) RegisterTools(s *mcp.Server) {
	t.RegisterAll(s)
}

// AddClient adds an S3 client with the given name to the toolkit.
func (t *Toolkit) AddClient(name string, client S3Client) {
	t.clientsMu.Lock()
	defer t.clientsMu.Unlock()
	t.clients[name] = client
}

// GetClient returns the S3 client for the given connection name.
// If name is empty, returns the default connection.
func (t *Toolkit) GetClient(name string) (S3Client, error) {
	if name == "" {
		name = t.defaultConnection
	}

	// Check cached clients
	t.clientsMu.RLock()
	if client, ok := t.clients[name]; ok {
		t.clientsMu.RUnlock()
		return client, nil
	}
	t.clientsMu.RUnlock()

	// Try to create via provider
	if t.clientProvider != nil {
		t.clientsMu.Lock()
		defer t.clientsMu.Unlock()

		// Double-check after acquiring write lock
		if client, ok := t.clients[name]; ok {
			return client, nil
		}

		client, err := t.clientProvider(name)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrConnectionNotFound, name)
		}

		t.clients[name] = client
		return client, nil
	}

	// Return default client if name matches or no name specified
	if name == "" || name == t.defaultConnection {
		if t.client != nil {
			return t.client, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrConnectionNotFound, name)
}

// ListConnections returns a list of available connection names.
func (t *Toolkit) ListConnections() []string {
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()

	names := make([]string, 0, len(t.clients))
	for name := range t.clients {
		names = append(names, name)
	}

	// Include default if not already in the map
	if t.client != nil && t.defaultConnection != "" {
		found := false
		for _, name := range names {
			if name == t.defaultConnection {
				found = true
				break
			}
		}
		if !found {
			names = append(names, t.defaultConnection)
		}
	}

	return names
}

// IsReadOnly returns true if the toolkit is in read-only mode.
func (t *Toolkit) IsReadOnly() bool {
	return t.readOnly
}

// MaxGetSize returns the maximum size for object retrieval.
func (t *Toolkit) MaxGetSize() int64 {
	return t.maxGetSize
}

// MaxPutSize returns the maximum size for object uploads.
func (t *Toolkit) MaxPutSize() int64 {
	return t.maxPutSize
}

// toolName returns the full tool name with any configured prefix.
func (t *Toolkit) toolName(name ToolName) string {
	if t.toolPrefix != "" {
		return t.toolPrefix + string(name)
	}
	return string(name)
}

// isToolDisabled returns true if the given tool is disabled.
func (t *Toolkit) isToolDisabled(name ToolName) bool {
	return t.disabledTools[name]
}

// wrapHandler wraps a tool handler with middleware, interceptors, and transformers.
func (t *Toolkit) wrapHandler(
	toolName ToolName,
	handler func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error),
	cfg *toolConfig,
) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	// Collect all applicable middlewares
	var allMiddlewares []ToolMiddleware
	allMiddlewares = append(allMiddlewares, t.middleware.All()...)
	if perTool, ok := t.toolMiddlewares[toolName]; ok {
		allMiddlewares = append(allMiddlewares, perTool...)
	}
	if cfg != nil {
		allMiddlewares = append(allMiddlewares, cfg.middlewares...)
	}

	// Zero-overhead optimization: if no middleware/interceptors/transformers, return handler unchanged
	if len(allMiddlewares) == 0 && len(t.interceptors.All()) == 0 && len(t.transformers.All()) == 0 {
		return handler
	}

	return func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		// Create tool context
		connectionName := t.defaultConnection
		tc := NewToolContext(toolName, connectionName)
		tc.StartTime = time.Now()
		ctx = WithToolContext(ctx, tc)

		// Run interceptors
		interceptResult := t.interceptors.Intercept(ctx, tc, req)
		if !interceptResult.Allow {
			t.logger.Warn("request blocked by interceptor",
				"tool", toolName,
				"reason", interceptResult.Reason)
			return ErrorResultf("access denied: %s", interceptResult.Reason), nil, nil
		}

		// Apply any request modifications
		if interceptResult.ModifiedRequest != nil {
			req = interceptResult.ModifiedRequest
		}

		// Run Before hooks (in order)
		var err error
		for _, m := range allMiddlewares {
			ctx, err = m.Before(ctx, tc)
			if err != nil {
				return ErrorResult(err.Error()), nil, nil
			}
		}

		// Execute handler
		result, extra, handlerErr := handler(ctx, req, input)

		// Run After hooks (reverse order - like defer)
		for i := len(allMiddlewares) - 1; i >= 0; i-- {
			result, err = allMiddlewares[i].After(ctx, tc, result, handlerErr)
			if err != nil {
				handlerErr = err
			}
		}

		// Apply transformers
		if result != nil {
			result, err = t.transformers.Transform(ctx, tc, result)
			if err != nil {
				return ErrorResultf("transformer error: %v", err), nil, nil
			}
		}

		return result, extra, nil
	}
}

// Close closes all S3 clients managed by the toolkit.
func (t *Toolkit) Close() error {
	t.clientsMu.Lock()
	defer t.clientsMu.Unlock()

	var lastErr error
	for _, client := range t.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}

	if t.client != nil {
		if err := t.client.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
