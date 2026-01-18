# Quick Start

Add S3 tools to your own MCP server in minutes.

## Installation

```bash
go get github.com/txn2/mcp-s3
```

## Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/mark3labs/mcp-go/server"
    "github.com/txn2/mcp-s3/pkg/client"
    "github.com/txn2/mcp-s3/pkg/tools"
)

func main() {
    ctx := context.Background()

    // Create S3 client from environment
    cfg := client.FromEnv()
    s3Client, err := client.New(ctx, &cfg)
    if err != nil {
        log.Fatalf("Failed to create S3 client: %v", err)
    }

    // Create toolkit with default options
    toolkit := tools.NewToolkit(s3Client)

    // Create MCP server
    mcpServer := server.NewMCPServer("my-server", "1.0.0")

    // Register S3 tools
    toolkit.RegisterTools(mcpServer)

    // Start serving
    if err := server.ServeStdio(mcpServer); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

## With Options

```go
toolkit := tools.NewToolkit(s3Client,
    // Enable read-only mode
    tools.WithReadOnly(true),

    // Set size limits
    tools.WithMaxGetSize(5 * 1024 * 1024),   // 5MB
    tools.WithMaxPutSize(50 * 1024 * 1024),  // 50MB

    // Add logging
    tools.WithLogger(slog.Default()),

    // Disable specific tools
    tools.DisableTool(tools.ToolDeleteObject),
)
```

## Selective Tool Registration

Only register specific tools:

```go
toolkit := tools.NewToolkit(s3Client,
    tools.EnableOnlyTools(
        tools.ToolListBuckets,
        tools.ToolListObjects,
        tools.ToolGetObject,
    ),
)
```

## Custom Client Configuration

```go
cfg := &client.Config{
    Region:          "us-west-2",
    Endpoint:        "http://localhost:8333",
    AccessKeyID:     "seaweedfsadmin",
    SecretAccessKey: "seaweedfsadmin",
    UsePathStyle:    true,
    Timeout:         60 * time.Second,
}

s3Client, err := client.New(ctx, cfg)
```

## Multiple Connections

```go
import "github.com/txn2/mcp-s3/pkg/multiserver"

config := &multiserver.MultiConfig{
    DefaultConnection: "production",
    Connections: []multiserver.ConnectionConfig{
        {Name: "production", Region: "us-east-1"},
        {Name: "staging", Region: "us-west-2"},
    },
}

manager := multiserver.NewManager(config)

toolkit := tools.NewToolkit(nil,
    tools.WithClientProvider(manager.ClientProvider()),
    tools.WithDefaultConnection("production"),
)
```
