# Configuration

mcp-s3 is configured entirely through environment variables.

## AWS Credentials

### Standard AWS Credentials

| Variable | Description | Default |
|----------|-------------|---------|
| `AWS_REGION` | AWS region | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | Access key ID | (credential chain) |
| `AWS_SECRET_ACCESS_KEY` | Secret access key | (credential chain) |
| `AWS_SESSION_TOKEN` | Session token for temporary credentials | (optional) |
| `AWS_PROFILE` | AWS profile name from ~/.aws/credentials | (optional) |

### S3-Specific Options

| Variable | Description | Default |
|----------|-------------|---------|
| `S3_ENDPOINT` | Custom endpoint URL (for SeaweedFS, LocalStack) | (AWS default) |
| `S3_USE_PATH_STYLE` | Use path-style URLs instead of virtual-hosted | `false` |
| `S3_TIMEOUT` | Operation timeout | `30s` |
| `S3_CONNECTION_NAME` | Name for the default connection | (none) |

## Extension Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_S3_EXT_READONLY` | `true` | Block all write operations (put, delete, copy) |
| `MCP_S3_EXT_SIZELIMIT` | `true` | Enforce object size limits |
| `MCP_S3_MAX_GET_SIZE` | `10MB` | Maximum size for GET operations |
| `MCP_S3_MAX_PUT_SIZE` | `100MB` | Maximum size for PUT operations |
| `MCP_S3_EXT_LOGGING` | `false` | Enable structured request logging |
| `MCP_S3_EXT_AUDIT` | `false` | Enable audit logging |

## Examples

### AWS S3

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1
mcp-s3
```

### SeaweedFS

```bash
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=seaweedfsadmin
export AWS_SECRET_ACCESS_KEY=seaweedfsadmin
mcp-s3
```

### LocalStack

```bash
export S3_ENDPOINT=http://localhost:4566
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
mcp-s3
```

### Enable Write Operations

```bash
export MCP_S3_EXT_READONLY=false
mcp-s3
```

## Multi-Connection Setup

For multiple AWS accounts or regions, use `S3_ADDITIONAL_CONNECTIONS`:

```bash
export S3_ADDITIONAL_CONNECTIONS='{
  "production": {
    "region": "us-east-1",
    "access_key_id": "AKIA...",
    "secret_access_key": "..."
  },
  "staging": {
    "region": "us-west-2",
    "access_key_id": "AKIA...",
    "secret_access_key": "..."
  },
  "local": {
    "endpoint": "http://localhost:8333",
    "use_path_style": true,
    "access_key_id": "seaweedfsadmin",
    "secret_access_key": "seaweedfsadmin"
  }
}'
export S3_CONNECTION_NAME=production
```

Then use the `connection` parameter in tool calls to select which connection to use.
