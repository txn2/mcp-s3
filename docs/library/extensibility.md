# Extensibility

mcp-s3 provides three extension points: middleware, interceptors, and transformers.

## Middleware

Middleware wraps tool execution for cross-cutting concerns:

```go
type ToolMiddleware interface {
    Name() string
    Wrap(next ToolHandler) ToolHandler
}
```

### Example: Timing Middleware

```go
timing := tools.NewMiddlewareFunc("timing", func(next tools.ToolHandler) tools.ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        result, err := next(ctx, req)
        log.Printf("Tool %s took %v", req.Params.Name, time.Since(start))
        return result, err
    }
})

toolkit := tools.NewToolkit(client, tools.WithMiddleware(timing))
```

## Interceptors

Interceptors run before tool execution and can block requests:

```go
type RequestInterceptor interface {
    Name() string
    Intercept(ctx context.Context, tc *ToolContext, request mcp.CallToolRequest) InterceptResult
}
```

### Example: Bucket Whitelist

```go
whitelist := tools.NewRequestInterceptorFunc("bucket-whitelist",
    func(ctx context.Context, tc *tools.ToolContext, req mcp.CallToolRequest) tools.InterceptResult {
        bucket, _ := req.Params.Arguments["bucket"].(string)

        allowed := []string{"public-data", "shared-files"}
        for _, b := range allowed {
            if bucket == b {
                return tools.Allowed()
            }
        }

        return tools.Blocked("bucket not in whitelist")
    },
)

toolkit := tools.NewToolkit(client, tools.WithInterceptor(whitelist))
```

## Transformers

Transformers modify results after tool execution:

```go
type ResultTransformer interface {
    Name() string
    Transform(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error)
}
```

### Example: Add Metadata

```go
addMeta := tools.NewResultTransformerFunc("add-metadata",
    func(ctx context.Context, tc *tools.ToolContext, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
        // Add processing timestamp to context
        tc.Set("processed_at", time.Now())
        return result, nil
    },
)

toolkit := tools.NewToolkit(client, tools.WithTransformer(addMeta))
```

## Built-in Extensions

### ReadOnly Interceptor

Blocks write operations:

```go
import "github.com/txn2/mcp-s3/pkg/extensions"

readonly := extensions.NewReadOnlyInterceptor(true)
toolkit := tools.NewToolkit(client, tools.WithInterceptor(readonly))
```

### Size Limit Interceptor

Enforces object size limits:

```go
sizelimit := extensions.NewSizeLimitInterceptor(
    10*1024*1024,  // 10MB max GET
    100*1024*1024, // 100MB max PUT
)
```

### Prefix ACL Interceptor

Restricts access by key prefix:

```go
prefixACL := extensions.NewPrefixACLInterceptor(
    []string{"public/", "shared/"},  // Allowed prefixes
    []string{"private/", "secret/"}, // Denied prefixes
)
```

### Logging Middleware

Structured request logging:

```go
logging := extensions.NewLoggingMiddleware(slog.Default())
```

### Metrics Middleware

In-memory metrics collection:

```go
metrics := extensions.NewMetrics()
metricsMiddleware := extensions.NewMetricsMiddleware(metrics)

// Later, retrieve stats
stats := metrics.GetToolStats("s3_list_buckets")
```

### Audit Middleware

Audit logging to a writer:

```go
auditLogger := extensions.NewAuditLogger(os.Stderr)
audit := extensions.NewAuditMiddleware(auditLogger)
```

## Combining Extensions

```go
toolkit := tools.NewToolkit(s3Client,
    // Interceptors run first
    tools.WithInterceptor(
        extensions.NewReadOnlyInterceptor(true),
        extensions.NewSizeLimitInterceptor(10*1024*1024, 100*1024*1024),
        extensions.NewPrefixACLInterceptor(allowed, denied),
    ),

    // Middleware wraps execution
    tools.WithMiddleware(
        extensions.NewLoggingMiddleware(logger),
        extensions.NewMetricsMiddleware(metrics),
        extensions.NewAuditMiddleware(auditLogger),
    ),

    // Transformers process results
    tools.WithTransformer(customTransformer),
)
```
