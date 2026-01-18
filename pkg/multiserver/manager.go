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
	if client, ok := m.clients[name]; ok {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	// Need to create the client
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := m.clients[name]; ok {
		return client, nil
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
	defaultName := m.config.DefaultConnection
	if defaultName == "" && len(m.config.Connections) > 0 {
		defaultName = m.config.Connections[0].Name
	}

	if defaultName == "" {
		return nil, fmt.Errorf("no default connection configured")
	}

	return m.GetClient(ctx, defaultName)
}

// ListConnections returns a list of all available connection names.
func (m *Manager) ListConnections() []string {
	return m.config.ConnectionNames()
}

// DefaultConnectionName returns the name of the default connection.
func (m *Manager) DefaultConnectionName() string {
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

// AddConnection adds a new connection configuration and optionally creates the client.
func (m *Manager) AddConnection(cfg ConnectionConfig, createNow bool) error {
	// Check for duplicate
	if existing := m.config.GetConnection(cfg.Name); existing != nil {
		return fmt.Errorf("connection already exists: %s", cfg.Name)
	}

	// Add to config
	m.config.Connections = append(m.config.Connections, cfg)

	// Optionally create the client now
	if createNow {
		_, err := m.GetClient(context.Background(), cfg.Name)
		return err
	}

	return nil
}

// RemoveConnection removes a connection and closes its client if it exists.
func (m *Manager) RemoveConnection(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close and remove the client if it exists
	if client, ok := m.clients[name]; ok {
		if err := client.Close(); err != nil {
			return fmt.Errorf("failed to close client %s: %w", name, err)
		}
		delete(m.clients, name)
	}

	// Remove from config
	for i, conn := range m.config.Connections {
		if conn.Name == name {
			m.config.Connections = append(m.config.Connections[:i], m.config.Connections[i+1:]...)
			break
		}
	}

	return nil
}

// HasConnection returns true if a connection with the given name exists.
func (m *Manager) HasConnection(name string) bool {
	return m.config.GetConnection(name) != nil
}

// IsClientInitialized returns true if a client for the given connection has been created.
func (m *Manager) IsClientInitialized(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.clients[name]
	return ok
}
