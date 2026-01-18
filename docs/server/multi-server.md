# Multi-Server

Connect to multiple S3-compatible storage services from a single mcp-s3 installation.

## Use Cases

- **Environment Separation** - Access production, staging, and development buckets
- **Multi-Cloud** - Query data across AWS S3, SeaweedFS, and other providers
- **Regional Access** - Connect to buckets in different AWS regions

## Configuration

### Primary Connection

Configure with standard environment variables:

```bash
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### Additional Connections

Add more connections with `S3_ADDITIONAL_CONNECTIONS`:

```bash
export S3_ADDITIONAL_CONNECTIONS='{
  "staging": {
    "region": "us-west-2",
    "access_key_id": "AKIAI...",
    "secret_access_key": "..."
  },
  "seaweedfs": {
    "endpoint": "http://localhost:8333",
    "use_path_style": true,
    "access_key_id": "any",
    "secret_access_key": "any"
  }
}'
```

### Connection Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `region` | No | AWS region (inherits from primary) |
| `endpoint` | No | Custom endpoint for S3-compatible storage |
| `access_key_id` | No | Access key (inherits from primary) |
| `secret_access_key` | No | Secret key (inherits from primary) |
| `session_token` | No | Session token for temporary credentials |
| `use_path_style` | No | Use path-style URLs (required for most S3-compatible storage) |

### Credential Inheritance

Additional connections inherit credentials from the primary connection unless overridden:

```bash
# Primary credentials
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# us-west connection inherits credentials
export S3_ADDITIONAL_CONNECTIONS='{
  "us-west": {
    "region": "us-west-2"
  }
}'
```

To use different credentials:

```bash
export S3_ADDITIONAL_CONNECTIONS='{
  "seaweedfs": {
    "endpoint": "http://localhost:8333",
    "use_path_style": true,
    "access_key_id": "admin",
    "secret_access_key": "admin"
  }
}'
```

## Usage

### Connection Parameter

All tools accept an optional `connection` parameter:

> "List buckets in the staging account"

The AI assistant uses `connection: "staging"`.

### List Connections

Use `s3_list_connections` to see available connections:

> "What S3 connections are configured?"

Response:
```json
{
  "connections": [
    {"name": "default", "region": "us-east-1"},
    {"name": "staging", "region": "us-west-2"},
    {"name": "seaweedfs", "endpoint": "http://localhost:8333"}
  ],
  "default": "default"
}
```

### Natural Language Examples

| Prompt | Connection Used |
|--------|-----------------|
| "List buckets in staging" | `staging` |
| "Get file from SeaweedFS" | `seaweedfs` |
| "Copy between prod and staging" | Both connections |
| "Show me the buckets" | `default` (primary) |

## Architecture

```mermaid
flowchart LR
    subgraph MCP["mcp-s3"]
        M["Connection Manager"]
    end

    M -->|"default"| P["AWS S3 us-east-1"]
    M -->|"staging"| S["AWS S3 us-west-2"]
    M -->|"seaweedfs"| D["SeaweedFS localhost:8333"]
```

## Best Practices

### Naming

Use clear, consistent names:

```bash
# Good
"production", "staging", "development"
"us-east", "us-west", "eu-central"
"aws", "seaweedfs", "minio"

# Avoid
"bucket1", "bucket2"
"s3-new", "s3-old"
```

### Security

Use separate credentials per environment when possible:

```yaml
# config.yaml
additional_connections:
  production:
    region: us-east-1
    access_key_id: ${PROD_AWS_ACCESS_KEY_ID}
    secret_access_key: ${PROD_AWS_SECRET_ACCESS_KEY}
  staging:
    region: us-west-2
    access_key_id: ${STAGING_AWS_ACCESS_KEY_ID}
    secret_access_key: ${STAGING_AWS_SECRET_ACCESS_KEY}
```

### Default Connection

Set the safest connection as default (usually development or read-only).

## Error Handling

### Unknown Connection

```
Error: Connection 'invalid' not found
Available: default, staging, seaweedfs
```

### Connection Failed

```
Error: Failed to connect to 'staging': access denied
Check that the credentials are correct and have appropriate permissions.
```

## Next Steps

- [Configuration Reference](../reference/configuration.md) - All configuration options
- [Security](../reference/security.md) - Security considerations
