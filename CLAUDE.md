# CLAUDE.md

This file provides guidance to Claude Code when working with this project.

## Project Overview

**mcp-s3** is a generic, open-source MCP (Model Context Protocol) server for Amazon S3 and S3-compatible object storage. It enables AI assistants to interact with object storage via the MCP protocol.

**Key Design Goals:**
- Composable: Can be used standalone OR imported as a library
- Generic: No domain-specific logic; suitable for any S3-compatible deployment
- Secure: Read-only by default with configurable limits

## Code Standards

1. **Idiomatic Go**: All code must follow idiomatic Go patterns and conventions. Use `gofmt`, follow Effective Go guidelines, and adhere to Go Code Review Comments.

2. **Test Coverage**: Project must maintain >80% unit test coverage. Build mocks where necessary to achieve this. Use table-driven tests where appropriate.

3. **Testing Definition**: When asked to "test" or "testing" the code, this means running the full CI test suite:
   - Unit tests with race detection (`make test` or `go test -race ./...`)
   - Linting (`make lint` / golangci-lint)
   - Security scanning (`gosec ./...`)
   - All checks that run in CI must pass locally before considering code "tested"

4. **Human Review Required**: A human must review and approve every line of code before it is committed. Therefore, commits are always performed by a human, not by Claude.

5. **Go Report Card**: The project MUST always maintain 100% across all categories on [Go Report Card](https://goreportcard.com/). This includes:
   - **gofmt**: All code must be formatted with `gofmt`
   - **go vet**: No issues from `go vet`
   - **gocyclo**: All functions must have cyclomatic complexity ≤10
   - **golint**: No lint issues (deprecated but still checked)
   - **ineffassign**: No ineffectual assignments
   - **license**: Valid license file present
   - **misspell**: No spelling errors in comments/strings

6. **Diagrams**: Use Mermaid for all diagrams. Never use ASCII art.

## Project Structure

```
mcp-s3/
├── cmd/mcp-s3/main.go         # Standalone server entrypoint
├── pkg/                        # PUBLIC API (importable by other projects)
│   ├── client/                 # S3 client wrapper
│   │   ├── client.go           # Connection and S3 operations
│   │   └── config.go           # Configuration from env/struct
│   ├── tools/                  # MCP tool definitions
│   │   ├── toolkit.go          # NewToolkit() and RegisterAll()
│   │   ├── list_buckets.go     # s3_list_buckets tool
│   │   ├── list_objects.go     # s3_list_objects tool
│   │   ├── get_object.go       # s3_get_object tool
│   │   └── ...                 # Other tools
│   ├── extensions/             # Built-in extensions
│   │   ├── readonly.go         # Block write operations
│   │   ├── sizelimit.go        # Enforce size limits
│   │   └── ...                 # Other extensions
│   └── multiserver/            # Multi-account support
├── internal/server/            # Default server setup (private)
├── go.mod
├── LICENSE                     # Apache 2.0
└── README.md
```

## Key Dependencies

- `github.com/aws/aws-sdk-go-v2` - AWS SDK for Go v2
- `github.com/mark3labs/mcp-go` - MCP SDK for Go

## Building and Running

```bash
# Build
go build -o mcp-s3 ./cmd/mcp-s3

# Run with AWS credentials
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
./mcp-s3
```

## Testing with SeaweedFS

```bash
# Start local SeaweedFS with S3 API
docker run -d -p 8333:8333 chrislusf/seaweedfs server -s3

# Configure for local testing
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=any
export AWS_SECRET_ACCESS_KEY=any
export AWS_REGION=us-east-1

./mcp-s3
```

## Composition Pattern

This package is designed to be imported by other MCP servers:

```go
import (
    "github.com/txn2/mcp-s3/pkg/client"
    "github.com/txn2/mcp-s3/pkg/tools"
)

// Create client
s3Client, _ := client.New(ctx, client.FromEnv())

// Create toolkit and register on your server
toolkit := tools.NewToolkit(s3Client)
toolkit.RegisterAll(yourMCPServer)
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `s3_list_buckets` | List accessible S3 buckets |
| `s3_list_objects` | List objects with prefix/delimiter/pagination |
| `s3_get_object` | Retrieve object content |
| `s3_get_object_metadata` | Get metadata without content (HEAD) |
| `s3_put_object` | Upload object (blocked by default) |
| `s3_delete_object` | Delete object (blocked by default) |
| `s3_copy_object` | Copy object within/between buckets |
| `s3_presign_url` | Generate presigned GET/PUT URLs |
| `s3_list_connections` | List configured S3 connections |

## Configuration Reference

Environment variables:
- `AWS_REGION` - AWS region (default: us-east-1)
- `AWS_ACCESS_KEY_ID` - Access key
- `AWS_SECRET_ACCESS_KEY` - Secret key
- `AWS_SESSION_TOKEN` - Session token (optional)
- `AWS_PROFILE` - Profile name (optional)
- `S3_ENDPOINT` - Custom endpoint for S3-compatible storage
- `S3_USE_PATH_STYLE` - Use path-style URLs (required for most S3-compatible storage)
- `S3_TIMEOUT` - Operation timeout (default: 30s)
- `MCP_S3_EXT_READONLY` - Block write operations (default: true)
- `MCP_S3_EXT_SIZELIMIT` - Enforce size limits (default: true)
- `MCP_S3_MAX_GET_SIZE` - Max bytes for GET (default: 10MB)
- `MCP_S3_MAX_PUT_SIZE` - Max bytes for PUT (default: 100MB)
