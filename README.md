# txn2/mcp-s3

[![GitHub license](https://img.shields.io/github/license/txn2/mcp-s3.svg)](https://github.com/txn2/mcp-s3/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/txn2/mcp-s3.svg)](https://pkg.go.dev/github.com/txn2/mcp-s3)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/mcp-s3)](https://goreportcard.com/report/github.com/txn2/mcp-s3)
[![CI](https://github.com/txn2/mcp-s3/actions/workflows/ci.yml/badge.svg)](https://github.com/txn2/mcp-s3/actions/workflows/ci.yml)

**Full documentation at [mcp-s3.txn2.com](https://mcp-s3.txn2.com)**

A Model Context Protocol (MCP) server for [Amazon S3](https://aws.amazon.com/s3/) and S3-compatible object storage, enabling AI assistants to browse buckets, read and write objects, and generate presigned URLs.

AI assistants can help with file organization, data migration, and content management, but they need secure access to storage systems. mcp-s3 bridges this gap by connecting S3-compatible storage to AI assistants through the MCP protocol, with configurable safety controls and multi-account support.

## Core Capabilities

**Composable Architecture**
- Import as a Go library to build custom MCP servers
- Add authentication, tenant isolation, audit logging without forking
- Middleware and interceptor patterns for enterprise requirements
- Combine with other MCP servers (mcp-trino, mcp-datahub) for unified data access

**Multi-Provider Support**
- Works with AWS S3, SeaweedFS, LocalStack, and any S3-compatible storage
- Connect to multiple accounts/regions from a single MCP installation
- Unified interface across production, staging, and development environments

**Secure Defaults**
- Read-only mode prevents accidental data modification
- Size limits prevent large file transfers
- Prefix-based ACLs restrict access to specific paths
- Audit logging for compliance requirements

## Features

- **List Buckets**: Browse accessible S3 buckets across connections
- **List Objects**: Navigate bucket contents with prefix/delimiter/pagination
- **Read Objects**: Retrieve object content with automatic text/binary detection
- **Write Objects**: Upload content (disabled by default in read-only mode)
- **Delete Objects**: Remove objects (disabled by default in read-only mode)
- **Copy Objects**: Copy within or between buckets
- **Presigned URLs**: Generate temporary access URLs for GET/PUT operations
- **Compose Custom Servers**: Import as a Go library with middleware and interceptors

## Installation

### Go Install

```bash
go install github.com/txn2/mcp-s3/cmd/mcp-s3@latest
```

### From Source

```bash
git clone https://github.com/txn2/mcp-s3.git
cd mcp-s3
make build
```

### Docker

```bash
docker run -e AWS_ACCESS_KEY_ID=... -e AWS_SECRET_ACCESS_KEY=... ghcr.io/txn2/mcp-s3
```

## Quick Start

### Claude Code CLI

Claude Code is the terminal-based coding assistant. Add mcp-s3 as an MCP server:

```bash
# For AWS S3
claude mcp add s3 \
  -e AWS_ACCESS_KEY_ID=your-access-key \
  -e AWS_SECRET_ACCESS_KEY=your-secret-key \
  -e AWS_REGION=us-east-1 \
  -- mcp-s3

# For SeaweedFS
claude mcp add seaweedfs \
  -e S3_ENDPOINT=http://localhost:8333 \
  -e S3_USE_PATH_STYLE=true \
  -e AWS_ACCESS_KEY_ID=any \
  -e AWS_SECRET_ACCESS_KEY=any \
  -- mcp-s3
```

### Claude Desktop

Add to your `claude_desktop_config.json` (find via Claude Desktop → Settings → Developer):

```json
{
  "mcpServers": {
    "s3": {
      "command": "mcp-s3",
      "env": {
        "AWS_ACCESS_KEY_ID": "your-access-key",
        "AWS_SECRET_ACCESS_KEY": "your-secret-key",
        "AWS_REGION": "us-east-1"
      }
    }
  }
}
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `s3_list_buckets` | List all accessible S3 buckets |
| `s3_list_objects` | List objects with prefix/delimiter/pagination |
| `s3_get_object` | Retrieve object content |
| `s3_get_object_metadata` | Get object metadata without content |
| `s3_put_object` | Upload object (disabled in read-only mode) |
| `s3_delete_object` | Delete object (disabled in read-only mode) |
| `s3_copy_object` | Copy object within/between buckets |
| `s3_presign_url` | Generate presigned GET/PUT URLs |
| `s3_list_connections` | List configured S3 connections |

## Configuration

### Environment Variables

**Primary Connection:**

| Variable | Description | Default |
|----------|-------------|---------|
| `AWS_REGION` | AWS region | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | Access key | (credential chain) |
| `AWS_SECRET_ACCESS_KEY` | Secret key | (credential chain) |
| `AWS_SESSION_TOKEN` | Session token | (optional) |
| `AWS_PROFILE` | Profile name | (optional) |
| `S3_ENDPOINT` | Custom endpoint (SeaweedFS) | (AWS default) |
| `S3_USE_PATH_STYLE` | Path-style URLs | `false` |
| `S3_TIMEOUT` | Operation timeout | `30s` |

**Extensions:**

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_S3_EXT_READONLY` | `true` | Block write operations |
| `MCP_S3_EXT_SIZELIMIT` | `true` | Enforce size limits |
| `MCP_S3_MAX_GET_SIZE` | `10MB` | Max bytes for GET |
| `MCP_S3_MAX_PUT_SIZE` | `100MB` | Max bytes for PUT |
| `MCP_S3_EXT_LOGGING` | `false` | Enable structured logging |
| `MCP_S3_EXT_AUDIT` | `false` | Enable audit logging |

### Multi-Connection Setup

Set `S3_ADDITIONAL_CONNECTIONS` with a JSON object:

```bash
export S3_ADDITIONAL_CONNECTIONS='{
  "production": {
    "region": "us-east-1",
    "access_key_id": "prod-key",
    "secret_access_key": "prod-secret"
  },
  "seaweedfs": {
    "region": "us-east-1",
    "endpoint": "http://seaweedfs:8333",
    "use_path_style": true
  }
}'
export S3_CONNECTION_NAME=production
```

## Library Usage

mcp-s3 is designed as a composable Go library. Import the packages to build custom MCP servers with S3 capabilities:

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
        log.Fatal(err)
    }
    defer s3Client.Close()

    // Create toolkit with options
    toolkit := tools.NewToolkit(s3Client,
        tools.WithReadOnly(true),
        tools.WithMaxGetSize(10*1024*1024),
        tools.WithToolPrefix("myapp_"),
    )
    defer toolkit.Close()

    // Create MCP server and register tools
    mcpServer := server.NewMCPServer("my-server", "1.0.0")
    toolkit.RegisterTools(mcpServer)

    // Add custom middleware
    toolkit.Use(myLoggingMiddleware)
    toolkit.AddInterceptor(myAuthInterceptor)

    // Serve
    if err := server.ServeStdio(mcpServer); err != nil {
        log.Fatal(err)
    }
}
```

### Extensibility Patterns

**Middleware** wraps tool execution for cross-cutting concerns:

```go
func loggingMiddleware(next tools.ToolHandler) tools.ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        log.Printf("Tool called: %s", req.Params.Name)
        return next(ctx, req)
    }
}
```

**Interceptors** can block or modify requests before execution:

```go
type authInterceptor struct{}

func (a *authInterceptor) Intercept(ctx context.Context, tc *tools.ToolContext, req mcp.CallToolRequest) tools.InterceptResult {
    if !isAuthorized(ctx, tc.ToolName) {
        return tools.InterceptResult{Allow: false, Message: "Unauthorized"}
    }
    return tools.InterceptResult{Allow: true}
}
```

**Transformers** modify results after execution:

```go
type redactTransformer struct{}

func (r *redactTransformer) Transform(ctx context.Context, tc *tools.ToolContext, result *mcp.CallToolResult) *mcp.CallToolResult {
    // Redact sensitive content from results
    return redactedResult
}
```

## Security Considerations

- **Read-Only Mode**: Enabled by default, blocks PUT and DELETE operations
- **Size Limits**: Default 10MB for GET, 100MB for PUT to prevent abuse
- **Prefix ACLs**: Restrict access to specific bucket prefixes
- **Audit Logging**: Optional logging of all operations for compliance

## Development

```bash
# Clone the repository
git clone https://github.com/txn2/mcp-s3.git
cd mcp-s3

# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make verify

# Serve documentation locally
make docs-serve
```

### Testing with SeaweedFS

```bash
# Start SeaweedFS with S3 API
docker run -d -p 8333:8333 -p 9333:9333 \
  chrislusf/seaweedfs server -s3

# Configure environment
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=any
export AWS_SECRET_ACCESS_KEY=any

# Run
./build/mcp-s3
```

## Contributing

We welcome contributions for bug fixes, tests, and documentation. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[Apache License 2.0](LICENSE)

## Related Projects

- [Model Context Protocol](https://modelcontextprotocol.io/) - The MCP specification
- [mcp-trino](https://github.com/txn2/mcp-trino) - MCP server for Trino SQL queries
- [mcp-datahub](https://github.com/txn2/mcp-datahub) - MCP server for DataHub metadata
- [Amazon S3](https://aws.amazon.com/s3/) - Object storage service
- [SeaweedFS](https://seaweedfs.io/) - S3-compatible object storage

---

Open source by [Craig Johnston](https://twitter.com/cjimti), sponsored by [Deasil Works, Inc.](https://deasil.works/)
