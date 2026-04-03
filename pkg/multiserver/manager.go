package multiserver

import (
	"context"
	"fmt"
	"sync"

	"github.com/txn2/mcp-s3/pkg/client"
	"github.com/txn2/mcp-s3/pkg/tools"
)

// Manager manages multiple S3 connections with lazy initialization.
type Manager struct {
	config  *MultiConfig
	clients map[string]tools.S3Client
	mu      sync.RWMutex

	// Factory function for creating clients (allows for mocking in tests)
	clientFactory func(ctx context.Context, cfg *client.Config) (tools.S3Client, error)
}

// NewManager creates a new connection manager with the given configuration.
func NewManager(config *MultiConfig) *Manager {
	return &Manager{
		config:  config,
		clients: make(map[string]tools.S3Client),
		clientFactory: func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
			return client.New(ctx, cfg)
		},
	}
}

// NewManagerWithFactory creates a new connection manager with a custom client factory.
// Useful for testing.
func NewManagerWithFactory(config *MultiConfig, factory func(ctx context.Context, cfg *client.Config) (tools.S3Client, error)) *Manager {
	return &Manager{
		config:        config,
		clients:       make(map[string]tools.S3Client),
		clientFactory: factory,
	}
}

// GetClient returns a client for the given connection name.
// If the client doesn't exist, it is created lazily.
func (m *Manager) GetClient(ctx context.Context, name string) (tools.S3Client, error) {
	// Check if we already have the client
	m.mu.RLock()
	if cached, ok := m.clients[name]; ok {
		m.mu.RUnlock()
		return cached, nil
	}
	m.mu.RUnlock()

	// Need to create the client
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := m.clients[name]; ok {
		return cached, nil
	}

	// Get the connection config
	connCfg := m.config.GetConnection(name)
	if connCfg == nil {
		return nil, fmt.Errorf("connection not found: %s", name)
	}

	// Create the client
	clientCfg := connCfg.ToClientConfig()
	newClient, err := m.clientFactory(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for %s: %w", name, err)
	}

	m.clients[name] = newClient
	return newClient, nil
}

// GetDefaultClient returns the client for the default connection.
func (m *Manager) GetDefaultClient(ctx context.Context) (tools.S3Client, error) {
	defaultName := m.DefaultConnectionName()
	if defaultName == "" {
		return nil, fmt.Errorf("no default connection configured")
	}

	return m.GetClient(ctx, defaultName)
}

// ListConnections returns a list of all available connection names.
func (m *Manager) ListConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.ConnectionNames()
}

// DefaultConnectionName returns the name of the default connection.
func (m *Manager) DefaultConnectionName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultConnectionName()
}

// defaultConnectionName returns the default connection name without locking.
// Caller must hold m.mu.
//
// When DefaultConnection is not explicitly set, the first connection in the
// list is used as the implicit default. Because AddConnection/RemoveConnection
// protect the default from mutation, the implicit default is stable as long
// as the caller sets DefaultConnection explicitly. If relying on the implicit
// first-element default, be aware that it cannot be removed or replaced.
func (m *Manager) defaultConnectionName() string {
	if m.config.DefaultConnection != "" {
		return m.config.DefaultConnection
	}
	if len(m.config.Connections) > 0 {
		return m.config.Connections[0].Name
	}
	return ""
}

// ClientProvider returns a function that can be used as a client provider for the toolkit.
func (m *Manager) ClientProvider() func(name string) (tools.S3Client, error) {
	return func(name string) (tools.S3Client, error) {
		return m.GetClient(context.Background(), name)
	}
}

// Close closes all managed clients.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close client %s: %w", name, err)
		}
	}

	m.clients = make(map[string]tools.S3Client)
	return lastErr
}

// AddConnection adds or replaces a connection configuration.
// If a connection with the same name already exists, the cached client is closed
// and the configuration is replaced. The default connection cannot be replaced.
//
// Note: when createNow is true, the write lock is held during client creation
// (which may involve network I/O). This ensures atomicity but blocks concurrent
// reads for the duration. Use createNow=false and let GetClient lazily create
// the client if lock contention is a concern.
func (m *Manager) AddConnection(cfg ConnectionConfig, createNow bool) error {
	if cfg.Name == "" {
		return fmt.Errorf("connection name must not be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Name == m.defaultConnectionName() {
		return fmt.Errorf("cannot replace the default connection %q via AddConnection", cfg.Name)
	}

	// Save previous config in case we need to roll back
	previousCfg := m.config.GetConnection(cfg.Name)

	// Close existing cached client if replacing.
	// Close errors are intentionally ignored: the client is being replaced
	// and S3 clients have no server-side session to clean up.
	if existing, ok := m.clients[cfg.Name]; ok {
		_ = existing.Close()
		delete(m.clients, cfg.Name)
	}

	// Add or replace in config
	m.config.addOrReplace(cfg)

	// Optionally create the client now
	if createNow {
		clientCfg := cfg.ToClientConfig()
		newClient, err := m.clientFactory(context.Background(), clientCfg)
		if err != nil {
			// Roll back: restore previous config or remove the new entry
			if previousCfg != nil {
				m.config.addOrReplace(*previousCfg)
			} else {
				m.config.remove(cfg.Name)
			}
			return fmt.Errorf("failed to create client for %s: %w", cfg.Name, err)
		}
		m.clients[cfg.Name] = newClient
	}

	return nil
}

// RemoveConnection removes a connection and closes its cached client.
// The default connection cannot be removed.
func (m *Manager) RemoveConnection(name string) error {
	if name == "" {
		return fmt.Errorf("connection name must not be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if name == m.defaultConnectionName() {
		return fmt.Errorf("cannot remove the default connection %q", name)
	}

	if !m.config.hasConnection(name) {
		return fmt.Errorf("connection %q not found", name)
	}

	// Close and remove the cached client if it exists.
	// Close errors are intentionally ignored: the connection is being
	// removed and S3 clients have no server-side session to clean up.
	if existing, ok := m.clients[name]; ok {
		_ = existing.Close()
		delete(m.clients, name)
	}

	// Remove from config
	m.config.remove(name)
	return nil
}

// HasConnection returns true if a connection with the given name exists.
func (m *Manager) HasConnection(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.hasConnection(name)
}

// IsClientInitialized returns true if a client for the given connection has been created.
func (m *Manager) IsClientInitialized(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.clients[name]
	return ok
}
