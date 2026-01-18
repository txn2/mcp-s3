# Architecture

Understanding the internal architecture of mcp-s3.

## Layer Overview

```mermaid
flowchart TB
    subgraph MCP["MCP Server (Tool Registration)"]
    end

    subgraph Toolkit["Toolkit"]
        MW["Middleware Chain<br/>(Logging, Metrics)"]
        INT["Interceptor Chain<br/>(ReadOnly, ACL)"]
        TH["Tool Handlers<br/>(ListBuckets, GetObject, etc.)"]
        TR["Transformer Chain<br/>(Result Processing)"]

        MW --> INT
        INT --> TH
        TH --> TR
    end

    subgraph S3Interface["S3Client Interface"]
        ABS["Abstraction over AWS SDK"]
    end

    subgraph AWS["AWS SDK v2"]
        SDK["S3 Service Client"]
    end

    MCP --> MW
    TR --> ABS
    ABS --> SDK
```

## Key Components

### Toolkit

The `Toolkit` is the central component that:

- Holds references to S3 clients
- Manages middleware, interceptors, and transformers
- Provides tool registration to MCP servers
- Handles multi-connection support

### S3Client Interface

The `S3Client` interface abstracts S3 operations:

```go
type S3Client interface {
    ConnectionName() string
    ListBuckets(ctx context.Context) ([]BucketInfo, error)
    ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int32, continueToken string) (*ListObjectsOutput, error)
    GetObject(ctx context.Context, bucket, key string) (*ObjectContent, error)
    GetObjectMetadata(ctx context.Context, bucket, key string) (*ObjectMetadata, error)
    PutObject(ctx context.Context, input *PutObjectInput) (*PutObjectOutput, error)
    DeleteObject(ctx context.Context, bucket, key string) error
    CopyObject(ctx context.Context, input *CopyObjectInput) (*CopyObjectOutput, error)
    PresignGetURL(ctx context.Context, bucket, key string, expires time.Duration) (*PresignedURL, error)
    PresignPutURL(ctx context.Context, bucket, key string, expires time.Duration) (*PresignedURL, error)
    Close() error
}
```

### ToolContext

The `ToolContext` carries request-scoped data through the processing chain:

```go
type ToolContext struct {
    ToolName       string
    ConnectionName string
    RequestID      string
    // Plus arbitrary key-value storage
}
```

## Request Flow

```mermaid
sequenceDiagram
    participant Client as MCP Client
    participant Server as MCP Server
    participant Int as Interceptors
    participant MW as Middleware
    participant Handler as Tool Handler
    participant Trans as Transformers
    participant S3 as S3Client

    Client->>Server: CallToolRequest
    Server->>Server: Create ToolContext
    Server->>Int: Run interceptors
    alt Blocked
        Int-->>Server: Deny with message
        Server-->>Client: Error response
    else Allowed
        Int->>MW: Pass through
        MW->>Handler: Execute (wrapped)
        Handler->>S3: S3 operation
        S3-->>Handler: Result
        Handler->>Trans: Process result
        Trans-->>MW: Transformed result
        MW-->>Server: Final result
        Server-->>Client: CallToolResult
    end
```

## Configuration Flow

```go
// 1. Create S3 client
s3Client, _ := client.New(ctx, &cfg)

// 2. Create toolkit with options
toolkit := tools.NewToolkit(s3Client,
    tools.WithMiddleware(middleware...),
    tools.WithInterceptor(interceptors...),
    tools.WithTransformer(transformers...),
)

// 3. Register with MCP server
toolkit.RegisterTools(mcpServer)
```
