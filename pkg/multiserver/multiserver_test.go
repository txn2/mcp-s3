package multiserver

import (
	"context"
	"os"
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
	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	t.Run("add new connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "existing",
			Connections: []ConnectionConfig{
				{Name: "existing"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.AddConnection(ConnectionConfig{Name: "new", Region: "us-west-2"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !manager.HasConnection("new") {
			t.Error("expected new connection to exist")
		}
	})

	t.Run("replace existing connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "default",
			Connections: []ConnectionConfig{
				{Name: "default"},
				{Name: "replaceable", Region: "us-east-1"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		// Initialize the client first
		_, _ = manager.GetClient(context.Background(), "replaceable")
		if !manager.IsClientInitialized("replaceable") {
			t.Fatal("expected client to be initialized")
		}

		// Replace with new config
		err := manager.AddConnection(ConnectionConfig{Name: "replaceable", Region: "eu-west-1"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Old cached client should be evicted
		if manager.IsClientInitialized("replaceable") {
			t.Error("expected cached client to be evicted after replace")
		}

		// Connection should still exist with new config
		if !manager.HasConnection("replaceable") {
			t.Error("expected connection to still exist")
		}
	})

	t.Run("add with createNow", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "existing",
			Connections: []ConnectionConfig{
				{Name: "existing"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.AddConnection(ConnectionConfig{Name: "eager", Region: "us-west-2"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !manager.IsClientInitialized("eager") {
			t.Error("expected client to be initialized when createNow is true")
		}
	})

	t.Run("cannot replace default connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "default",
			Connections: []ConnectionConfig{
				{Name: "default"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.AddConnection(ConnectionConfig{Name: "default", Region: "eu-west-1"}, false)
		if err == nil {
			t.Error("expected error when replacing default connection")
		}
	})

	t.Run("empty name returns error", func(t *testing.T) {
		cfg := &MultiConfig{
			Connections: []ConnectionConfig{},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.AddConnection(ConnectionConfig{Name: ""}, false)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}

func TestManager_RemoveConnection(t *testing.T) {
	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	t.Run("remove existing connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "default",
			Connections: []ConnectionConfig{
				{Name: "default"},
				{Name: "toremove"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		// Initialize the client
		_, _ = manager.GetClient(context.Background(), "toremove")

		err := manager.RemoveConnection("toremove")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if manager.HasConnection("toremove") {
			t.Error("expected connection to be removed")
		}
		if manager.IsClientInitialized("toremove") {
			t.Error("expected client to be closed")
		}
	})

	t.Run("cannot remove default connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "default",
			Connections: []ConnectionConfig{
				{Name: "default"},
				{Name: "other"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.RemoveConnection("default")
		if err == nil {
			t.Error("expected error when removing default connection")
		}
	})

	t.Run("remove non-existent connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "default",
			Connections: []ConnectionConfig{
				{Name: "default"},
			},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.RemoveConnection("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent connection")
		}
	})

	t.Run("empty name returns error", func(t *testing.T) {
		cfg := &MultiConfig{
			Connections: []ConnectionConfig{},
		}
		manager := NewManagerWithFactory(cfg, factory)

		err := manager.RemoveConnection("")
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
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

func TestManager_DefaultConnectionName(t *testing.T) {
	t.Run("explicit default", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "my-default",
			Connections: []ConnectionConfig{
				{Name: "my-default"},
			},
		}

		manager := NewManager(cfg)

		if manager.DefaultConnectionName() != "my-default" {
			t.Errorf("DefaultConnectionName() = %q, want %q", manager.DefaultConnectionName(), "my-default")
		}
	})

	t.Run("no default uses first connection", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "",
			Connections: []ConnectionConfig{
				{Name: "first-conn"},
				{Name: "second-conn"},
			},
		}

		manager := NewManager(cfg)

		if manager.DefaultConnectionName() != "first-conn" {
			t.Errorf("DefaultConnectionName() = %q, want %q", manager.DefaultConnectionName(), "first-conn")
		}
	})

	t.Run("no default no connections returns empty", func(t *testing.T) {
		cfg := &MultiConfig{
			DefaultConnection: "",
			Connections:       []ConnectionConfig{},
		}

		manager := NewManager(cfg)

		if manager.DefaultConnectionName() != "" {
			t.Errorf("DefaultConnectionName() = %q, want empty string", manager.DefaultConnectionName())
		}
	})
}

func TestFromEnvJSON(t *testing.T) {
	t.Run("empty env returns nil", func(t *testing.T) {
		os.Unsetenv("S3_ADDITIONAL_CONNECTIONS")
		os.Unsetenv("S3_CONNECTION_NAME")

		cfg, err := FromEnvJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Error("expected nil config for empty env")
		}
	})

	t.Run("valid JSON", func(t *testing.T) {
		os.Setenv("S3_ADDITIONAL_CONNECTIONS", `{"prod":{"region":"us-east-1"},"staging":{"region":"us-west-2"}}`)
		os.Setenv("S3_CONNECTION_NAME", "prod")
		defer os.Unsetenv("S3_ADDITIONAL_CONNECTIONS")
		defer os.Unsetenv("S3_CONNECTION_NAME")

		cfg, err := FromEnvJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
		if cfg.DefaultConnection != "prod" {
			t.Errorf("DefaultConnection = %q, want %q", cfg.DefaultConnection, "prod")
		}
		if len(cfg.Connections) != 2 {
			t.Errorf("expected 2 connections, got %d", len(cfg.Connections))
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		os.Setenv("S3_ADDITIONAL_CONNECTIONS", `invalid json`)
		defer os.Unsetenv("S3_ADDITIONAL_CONNECTIONS")

		_, err := FromEnvJSON()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestFromYAMLFile(t *testing.T) {
	t.Run("valid YAML file", func(t *testing.T) {
		content := `default_connection: test
connections:
  - name: test
    region: us-east-1
`
		tmpfile, err := os.CreateTemp("", "config*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, writeErr := tmpfile.WriteString(content); writeErr != nil {
			t.Fatal(writeErr)
		}
		tmpfile.Close()

		cfg, err := FromYAMLFile(tmpfile.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.DefaultConnection != "test" {
			t.Errorf("DefaultConnection = %q, want %q", cfg.DefaultConnection, "test")
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := FromYAMLFile("/nonexistent/path/config.yaml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "config*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, writeErr := tmpfile.WriteString("invalid: [yaml: content"); writeErr != nil {
			t.Fatal(writeErr)
		}
		tmpfile.Close()

		_, err = FromYAMLFile(tmpfile.Name())
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

func TestFromJSONFile(t *testing.T) {
	t.Run("valid JSON file", func(t *testing.T) {
		content := `{"default_connection":"test","connections":[{"name":"test","region":"us-east-1"}]}`
		tmpfile, err := os.CreateTemp("", "config*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, writeErr := tmpfile.WriteString(content); writeErr != nil {
			t.Fatal(writeErr)
		}
		tmpfile.Close()

		cfg, err := FromJSONFile(tmpfile.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.DefaultConnection != "test" {
			t.Errorf("DefaultConnection = %q, want %q", cfg.DefaultConnection, "test")
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := FromJSONFile("/nonexistent/path/config.json")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "config*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, writeErr := tmpfile.WriteString("not valid json"); writeErr != nil {
			t.Fatal(writeErr)
		}
		tmpfile.Close()

		_, err = FromJSONFile(tmpfile.Name())
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestManager_GetDefaultClient_NoDefault(t *testing.T) {
	// No default and no connections - should error
	cfg := &MultiConfig{
		DefaultConnection: "",
		Connections:       []ConnectionConfig{},
	}

	manager := NewManager(cfg)

	_, err := manager.GetDefaultClient(context.Background())
	if err == nil {
		t.Error("expected error when no default connection set and no connections")
	}
}

func TestManager_GetDefaultClient_FallbackToFirst(t *testing.T) {
	// No explicit default but has connections - should use first
	cfg := &MultiConfig{
		DefaultConnection: "",
		Connections: []ConnectionConfig{
			{Name: "first-conn"},
			{Name: "second-conn"},
		},
	}

	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	manager := NewManagerWithFactory(cfg, factory)

	c, err := manager.GetDefaultClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ConnectionName() != "first-conn" {
		t.Errorf("expected first-conn, got %q", c.ConnectionName())
	}
}

func TestManager_ConcurrentAddRemove(t *testing.T) {
	factory := func(ctx context.Context, cfg *client.Config) (tools.S3Client, error) {
		return &mockClient{name: cfg.Name}, nil
	}

	cfg := &MultiConfig{
		DefaultConnection: "default",
		Connections: []ConnectionConfig{
			{Name: "default"},
		},
	}
	manager := NewManagerWithFactory(cfg, factory)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_ = manager.AddConnection(ConnectionConfig{Name: "dynamic", Region: "us-east-1"}, false)
			_ = manager.RemoveConnection("dynamic")
		}
	}()

	// Concurrent reads while add/remove is happening
	for i := 0; i < 100; i++ {
		_ = manager.ListConnections()
		_ = manager.HasConnection("dynamic")
		_ = manager.DefaultConnectionName()
	}

	<-done
}

func TestMultiConfig_addOrReplace(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "existing", Region: "us-east-1"},
		},
	}

	t.Run("replace existing", func(t *testing.T) {
		cfg.addOrReplace(ConnectionConfig{Name: "existing", Region: "eu-west-1"})
		conn := cfg.GetConnection("existing")
		if conn == nil {
			t.Fatal("expected connection to exist")
		}
		if conn.Region != "eu-west-1" {
			t.Errorf("expected region 'eu-west-1', got %q", conn.Region)
		}
		if len(cfg.Connections) != 1 {
			t.Errorf("expected 1 connection, got %d", len(cfg.Connections))
		}
	})

	t.Run("add new", func(t *testing.T) {
		cfg.addOrReplace(ConnectionConfig{Name: "new", Region: "ap-southeast-1"})
		if len(cfg.Connections) != 2 {
			t.Errorf("expected 2 connections, got %d", len(cfg.Connections))
		}
	})
}

func TestMultiConfig_remove(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "keep"},
			{Name: "remove"},
		},
	}

	cfg.remove("remove")
	if len(cfg.Connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(cfg.Connections))
	}
	if cfg.Connections[0].Name != "keep" {
		t.Errorf("expected 'keep', got %q", cfg.Connections[0].Name)
	}

	// Removing non-existent is a no-op
	cfg.remove("nonexistent")
	if len(cfg.Connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(cfg.Connections))
	}
}

func TestMultiConfig_hasConnection(t *testing.T) {
	cfg := &MultiConfig{
		Connections: []ConnectionConfig{
			{Name: "exists"},
		},
	}

	if !cfg.hasConnection("exists") {
		t.Error("expected hasConnection to return true")
	}
	if cfg.hasConnection("nope") {
		t.Error("expected hasConnection to return false")
	}
}
