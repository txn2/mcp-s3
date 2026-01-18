package multiserver

import (
	"context"
	"testing"
	"time"

	"github.com/txn2/mcp-s3/pkg/client"
	"github.com/txn2/mcp-s3/pkg/tools"
)

// mockClient is a minimal mock for testing.
type mockClient struct {
	name string
}

func (m *mockClient) ConnectionName() string { return m.name }
func (m *mockClient) Config() *client.Config { return &client.Config{Name: m.name} }
func (m *mockClient) ListBuckets(ctx context.Context) ([]client.BucketInfo, error) {
	return nil, nil
}
func (m *mockClient) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string) (*client.ListObjectsOutput, error) {
	return nil, nil
}
func (m *mockClient) GetObject(ctx context.Context, bucket, key string) (*client.ObjectContent, error) {
	return nil, nil
}
func (m *mockClient) GetObjectMetadata(ctx context.Context, bucket, key string) (*client.ObjectMetadata, error) {
	return nil, nil
}
func (m *mockClient) PutObject(ctx context.Context, input *client.PutObjectInput) (*client.PutObjectOutput, error) {
	return nil, nil
}
func (m *mockClient) DeleteObject(ctx context.Context, bucket, key string) error { return nil }
func (m *mockClient) CopyObject(ctx context.Context, input *client.CopyObjectInput) (*client.CopyObjectOutput, error) {
	return nil, nil
}
func (m *mockClient) PresignGetURL(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error) {
	return nil, nil
}
func (m *mockClient) PresignPutURL(ctx context.Context, bucket, key string, expires time.Duration) (*client.PresignedURL, error) {
	return nil, nil
}
func (m *mockClient) Close() error { return nil }

var _ tools.S3Client = (*mockClient)(nil)

func TestConnectionConfig_ToClientConfig(t *testing.T) {
	connCfg := &ConnectionConfig{
		Name:            "test",
		Region:          "us-west-2",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UsePathStyle:    true,
	}

	clientCfg := connCfg.ToClientConfig()

	if clientCfg.Name != "test" {
		t.Errorf("expected Name 'test', got %q", clientCfg.Name)
	}
	if clientCfg.Region != "us-west-2" {
		t.Errorf("expected Region 'us-west-2', got %q", clientCfg.Region)
	}
	if clientCfg.Endpoint != "http://localhost:9000" {
		t.Errorf("expected Endpoint 'http://localhost:9000', got %q", clientCfg.Endpoint)
	}
	if !clientCfg.UsePathStyle {
		t.Error("expected UsePathStyle to be true")
	}
}

func TestMultiConfig_GetConnection(t *testing.T) {
	cfg := &MultiConfig{
		DefaultConnection: "conn1",
		Connections: []ConnectionConfig{
			{Name: "conn1", Region: "us-east-1"},
			{Name: "conn2", Region: "eu-west-1"},
		},
	}

	t.Run("existing connection", func(t *testing.T) {
		conn := cfg.GetConnection("conn1")
		if conn == nil {
			t.Fatal("expected connection to exist")
		}
		if conn.Region != "us-east-1" {
			t.Errorf("expected region 'us-east-1', got %q", conn.Region)
		}
	})

	t.Run("non-existing connection", func(t *testing.T) {
		conn := cfg.GetConnection("nonexistent")
		if conn != nil {
			t.Error("expected nil for non-existing connection")
		}
	})
}

func TestMultiConfig_ConnectionNames(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "conn1"},
			{Name: "conn2"},
			{Name: "conn3"},
		},
	}

	names := cfg.ConnectionNames()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
}

func TestManager_GetClient(t *testing.T) {
	cfg := &MultiConfig{
		DefaultConnection: "test",
		Connections: []ConnectionConfig{
			{Name: "test", Region: "us-east-1"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)

	t.Run("creates client lazily", func(t *testing.T) {
		if manager.IsClientInitialized("test") {
			t.Error("expected client to not be initialized yet")
		}

		client, err := manager.GetClient(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.ConnectionName() != "test" {
			t.Errorf("expected connection name 'test', got %q", client.ConnectionName())
		}

		if !manager.IsClientInitialized("test") {
			t.Error("expected client to be initialized")
		}
	})

	t.Run("returns cached client", func(t *testing.T) {
		client1, _ := manager.GetClient(context.Background(), "test")
		client2, _ := manager.GetClient(context.Background(), "test")

		if client1 != client2 {
			t.Error("expected same client instance")
		}
	})

	t.Run("error for unknown connection", func(t *testing.T) {
		_, err := manager.GetClient(context.Background(), "unknown")
		if err == nil {
			t.Error("expected error for unknown connection")
		}
	})
}

func TestManager_GetDefaultClient(t *testing.T) {
	cfg := &MultiConfig{
		DefaultConnection: "default",
		Connections: []ConnectionConfig{
			{Name: "default", Region: "us-east-1"},
			{Name: "other", Region: "eu-west-1"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)

	client, err := manager.GetDefaultClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.ConnectionName() != "default" {
		t.Errorf("expected connection name 'default', got %q", client.ConnectionName())
	}
}

func TestManager_ListConnections(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "conn1"},
			{Name: "conn2"},
		},
	}

	manager := NewManager(cfg)
	connections := manager.ListConnections()

	if len(connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(connections))
	}
}

func TestManager_AddConnection(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "existing"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)

	t.Run("add new connection", func(t *testing.T) {
		err := manager.AddConnection(ConnectionConfig{Name: "new", Region: "us-west-2"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !manager.HasConnection("new") {
			t.Error("expected new connection to exist")
		}
	})

	t.Run("duplicate connection", func(t *testing.T) {
		err := manager.AddConnection(ConnectionConfig{Name: "existing"}, false)
		if err == nil {
			t.Error("expected error for duplicate connection")
		}
	})
}

func TestManager_RemoveConnection(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "toremove"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)

	// First initialize the client
	_, _ = manager.GetClient(context.Background(), "toremove")

	err := manager.RemoveConnection("toremove")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.HasConnection("toremove") {
		t.Error("expected connection to be removed")
	}
}

func TestManager_Close(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "conn1"},
			{Name: "conn2"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)

	// Initialize both clients
	_, _ = manager.GetClient(context.Background(), "conn1")
	_, _ = manager.GetClient(context.Background(), "conn2")

	err := manager.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.IsClientInitialized("conn1") {
		t.Error("expected conn1 client to be closed")
	}
	if manager.IsClientInitialized("conn2") {
		t.Error("expected conn2 client to be closed")
	}
}

func TestManager_ClientProvider(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "test"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)
	provider := manager.ClientProvider()

	client, err := provider("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.ConnectionName() != "test" {
		t.Errorf("expected connection name 'test', got %q", client.ConnectionName())
	}
}
