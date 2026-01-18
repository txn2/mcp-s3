# Library Overview

mcp-s3 can be used as a Go library to add S3 tools to your own MCP server.

## Why Use as a Library?

- **Compose with other tools**: Add S3 capabilities alongside your own custom MCP tools
- **Custom configuration**: Full control over client creation and options
- **Extensibility**: Add custom middleware, interceptors, and transformers
- **Integration**: Use the integration interfaces to connect with other systems

## Package Structure

| Package | Description |
|---------|-------------|
| `pkg/client` | S3 client wrapper around AWS SDK v2 |
| `pkg/tools` | MCP tool implementations and Toolkit |
| `pkg/extensions` | Built-in middleware and interceptors |
| `pkg/multiserver` | Multi-connection management |
| `pkg/integration` | Cross-system integration interfaces |
| `internal/server` | Server factory (for reference) |

## Key Packages

### pkg/client

Low-level S3 client wrapper around AWS SDK v2:

```go
import "github.com/txn2/mcp-s3/pkg/client"

cfg := client.FromEnv()
s3Client, err := client.New(ctx, &cfg)

buckets, err := s3Client.ListBuckets(ctx)
content, err := s3Client.GetObject(ctx, "bucket", "key")
```

### pkg/tools

MCP toolkit with all S3 tools:

```go
import "github.com/txn2/mcp-s3/pkg/tools"

toolkit := tools.NewToolkit(s3Client,
    tools.WithReadOnly(true),
    tools.WithMaxGetSize(10*1024*1024),
)

toolkit.RegisterTools(mcpServer)
```

### pkg/extensions

Built-in extensions for common needs:

```go
import "github.com/txn2/mcp-s3/pkg/extensions"

// Read-only interceptor
readonly := extensions.NewReadOnlyInterceptor(true)

// Size limit interceptor
sizelimit := extensions.NewSizeLimitInterceptor(10*1024*1024, 100*1024*1024)

// Logging middleware
logging := extensions.NewLoggingMiddleware(logger)
```

### pkg/integration

Interfaces for cross-system integration:

```go
import "github.com/txn2/mcp-s3/pkg/integration"

resolver := integration.NewDefaultResolver("default", "my-bucket")
ref, err := resolver.ParseURI("s3://bucket/key")
```
