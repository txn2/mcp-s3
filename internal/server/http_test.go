package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-s3/pkg/tools"
)

func TestServeHTTP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "mcp-s3", Version: "test"}, nil)
	tools.NewToolkit(&mockS3Client{}).RegisterAll(mcpServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ServeHTTP(ctx, mcpServer, ln) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://" + ln.Addr().String()}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	found := false
	for _, tool := range res.Tools {
		found = found || tool.Name == "s3_list_buckets"
	}
	if !found {
		t.Errorf("s3_list_buckets not served over HTTP; got %d tools", len(res.Tools))
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ServeHTTP after cancel: %v", err)
		}
	case <-time.After(2 * shutdownTimeout):
		t.Fatal("ServeHTTP did not return after context cancel")
	}
}

func TestServeHTTP_ServeError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_ = ln.Close()

	err = ServeHTTP(context.Background(), mcp.NewServer(&mcp.Implementation{Name: "x", Version: "x"}, nil), ln)
	if err == nil {
		t.Error("expected error serving on a closed listener")
	}
}
