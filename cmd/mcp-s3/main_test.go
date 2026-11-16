package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServe_Errors(t *testing.T) {
	srv := mcp.NewServer(&mcp.Implementation{Name: "x", Version: "x"}, nil)

	tests := []struct{ name, transport, addr string }{
		{"unknown transport", "sse", ""},
		{"unbindable address", "http", "not-an-address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := serve(context.Background(), srv, tt.transport, tt.addr); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestServe_HTTPStopsOnCancel(t *testing.T) {
	srv := mcp.NewServer(&mcp.Implementation{Name: "x", Version: "x"}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := serve(ctx, srv, "http", "127.0.0.1:0"); err != nil {
		t.Errorf("serve: %v", err)
	}
}
