package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Default size limits.
const (
	// DefaultMaxGetSize is the default maximum size for object retrieval (10MB).
	DefaultMaxGetSize = 10 * 1024 * 1024

	// DefaultMaxPutSize is the default maximum size for object uploads (100MB).
	DefaultMaxPutSize = 100 * 1024 * 1024
)

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
	disabledTools     map[string]bool

	// Extensibility
	middleware   *ToolMiddlewareRegistry
	interceptors *InterceptorChain
	transformers *TransformerChain

	// Logging
	logger *slog.Logger
}

// NewToolkit creates a new Toolkit with the given S3 client and options.
func NewToolkit(client S3Client, opts ...Option) *Toolkit {
	t := &Toolkit{
		client:        client,
		clients:       make(map[string]S3Client),
		disabledTools: make(map[string]bool),
		middleware:    NewToolMiddlewareRegistry(),
		interceptors:  NewInterceptorChain(),
		transformers:  NewTransformerChain(),
		maxGetSize:    DefaultMaxGetSize,
		maxPutSize:    DefaultMaxPutSize,
		readOnly:      false,
		logger:        defaultLogger(),
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

// RegisterTools registers all S3 tools with the MCP server.
func (t *Toolkit) RegisterTools(s *server.MCPServer) {
	// Register each tool if not disabled
	if !t.isToolDisabled(ToolListBuckets) {
		t.registerListBuckets(s)
	}
	if !t.isToolDisabled(ToolListObjects) {
		t.registerListObjects(s)
	}
	if !t.isToolDisabled(ToolGetObject) {
		t.registerGetObject(s)
	}
	if !t.isToolDisabled(ToolGetObjectMetadata) {
		t.registerGetObjectMetadata(s)
	}
	if !t.isToolDisabled(ToolPutObject) {
		t.registerPutObject(s)
	}
	if !t.isToolDisabled(ToolDeleteObject) {
		t.registerDeleteObject(s)
	}
	if !t.isToolDisabled(ToolCopyObject) {
		t.registerCopyObject(s)
	}
	if !t.isToolDisabled(ToolPresignURL) {
		t.registerPresignURL(s)
	}
	if !t.isToolDisabled(ToolListConnections) {
		t.registerListConnections(s)
	}
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
func (t *Toolkit) toolName(name string) string {
	if t.toolPrefix != "" {
		return t.toolPrefix + name
	}
	return name
}

// isToolDisabled returns true if the given tool is disabled.
func (t *Toolkit) isToolDisabled(name string) bool {
	return t.disabledTools[name]
}

// wrapHandler wraps a tool handler with middleware, interceptors, and transformers.
func (t *Toolkit) wrapHandler(toolName string, handler ToolHandler) ToolHandler {
	// Create the full processing chain
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Create tool context
		connectionName := OptionalString(request.Params.Arguments, "connection", t.defaultConnection)
		tc := NewToolContext(toolName, connectionName)
		ctx = WithToolContext(ctx, tc)

		// Run interceptors
		interceptResult := t.interceptors.Intercept(ctx, tc, request)
		if !interceptResult.Allow {
			t.logger.Warn("request blocked by interceptor",
				"tool", toolName,
				"reason", interceptResult.Reason)
			return ErrorResultf("access denied: %s", interceptResult.Reason), nil
		}

		// Apply any request modifications
		if interceptResult.ModifiedRequest != nil {
			request = *interceptResult.ModifiedRequest
		}

		// Apply middleware and execute handler
		wrappedHandler := t.middleware.Apply(handler)
		result, err := wrappedHandler(ctx, request)
		if err != nil {
			return nil, err
		}

		// Apply transformers
		result, err = t.transformers.Transform(ctx, tc, result)
		if err != nil {
			return nil, fmt.Errorf("transformer error: %w", err)
		}

		return result, nil
	}
}

// registerTool is a helper to register a tool with the server.
func (t *Toolkit) registerTool(s *server.MCPServer, tool mcp.Tool, handler ToolHandler) {
	wrappedHandler := t.wrapHandler(tool.Name, handler)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return wrappedHandler(ctx, request)
	})
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
