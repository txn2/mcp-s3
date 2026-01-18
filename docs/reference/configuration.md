# Configuration Reference

Complete configuration reference for mcp-s3.

## Environment Variables

### AWS Connection

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_REGION` | `us-east-1` | AWS region for S3 requests |
| `AWS_ACCESS_KEY_ID` | | AWS access key ID |
| `AWS_SECRET_ACCESS_KEY` | | AWS secret access key |
| `AWS_SESSION_TOKEN` | | Session token for temporary credentials |
| `AWS_PROFILE` | | AWS profile name from credentials file |

### S3 Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `S3_ENDPOINT` | | Custom endpoint URL for S3-compatible storage |
| `S3_USE_PATH_STYLE` | `false` | Use path-style URLs instead of virtual-hosted |
| `S3_TIMEOUT` | `30s` | Timeout for S3 operations |
| `S3_CONNECTION_NAME` | `default` | Name for the primary connection |

### Multi-Connection

| Variable | Description |
|----------|-------------|
| `S3_ADDITIONAL_CONNECTIONS` | JSON object defining additional connections |

Example:
```bash
export S3_ADDITIONAL_CONNECTIONS='{
  "staging": {
    "region": "us-west-2"
  },
  "seaweedfs": {
    "endpoint": "http://localhost:8333",
    "use_path_style": true,
    "access_key_id": "admin",
    "secret_access_key": "admin"
  }
}'
```

### Extensions

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_S3_EXT_READONLY` | `true` | Block write operations (put, delete, copy) |
| `MCP_S3_EXT_SIZELIMIT` | `true` | Enforce object size limits |
| `MCP_S3_MAX_GET_SIZE` | `10MB` | Maximum object size for GET operations |
| `MCP_S3_MAX_PUT_SIZE` | `100MB` | Maximum object size for PUT operations |
| `MCP_S3_EXT_PREFIXACL` | `false` | Enable prefix-based access control |
| `MCP_S3_ALLOWED_PREFIXES` | | Comma-separated allowed prefixes |
| `MCP_S3_DENIED_PREFIXES` | | Comma-separated denied prefixes |
| `MCP_S3_EXT_LOGGING` | `false` | Enable request logging |
| `MCP_S3_EXT_AUDIT` | `false` | Enable audit logging |
| `MCP_S3_EXT_METRICS` | `false` | Enable metrics collection |

## Size Format

Size values support these suffixes:

| Suffix | Multiplier |
|--------|------------|
| `B` | 1 |
| `KB` | 1,024 |
| `MB` | 1,048,576 |
| `GB` | 1,073,741,824 |

Examples:
- `10MB` = 10,485,760 bytes
- `1GB` = 1,073,741,824 bytes
- `500KB` = 512,000 bytes

## Configuration File

For complex configurations, use a YAML file:

```yaml
# config.yaml
connection:
  region: us-east-1
  endpoint: ""
  use_path_style: false
  timeout: 30s

additional_connections:
  staging:
    region: us-west-2
  seaweedfs:
    endpoint: http://localhost:8333
    use_path_style: true
    access_key_id: admin
    secret_access_key: admin

extensions:
  readonly: true
  sizelimit: true
  max_get_size: 10MB
  max_put_size: 100MB
  logging: false
  audit: false
```

Load with:
```bash
mcp-s3 --config config.yaml
```

## Connection Priority

Credentials are resolved in this order:

1. Explicit environment variables (`AWS_ACCESS_KEY_ID`, etc.)
2. AWS profile (`AWS_PROFILE`)
3. Shared credentials file (`~/.aws/credentials`)
4. IAM role (when running on AWS)

## S3-Compatible Storage

### SeaweedFS

```bash
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=any
export AWS_SECRET_ACCESS_KEY=any
export AWS_REGION=us-east-1
```

### LocalStack

```bash
export S3_ENDPOINT=http://localhost:4566
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
```

### Cloudflare R2

```bash
export S3_ENDPOINT=https://accountid.r2.cloudflarestorage.com
export AWS_ACCESS_KEY_ID=your-r2-access-key
export AWS_SECRET_ACCESS_KEY=your-r2-secret-key
export AWS_REGION=auto
```

### Backblaze B2

```bash
export S3_ENDPOINT=https://s3.us-west-004.backblazeb2.com
export AWS_ACCESS_KEY_ID=your-b2-key-id
export AWS_SECRET_ACCESS_KEY=your-b2-application-key
export AWS_REGION=us-west-004
```

## Validation

Environment variables are validated at startup. Invalid configurations produce clear error messages:

```
Error: Invalid configuration
- AWS_REGION: required when not using AWS credential chain
- S3_TIMEOUT: invalid duration format "abc"
```
