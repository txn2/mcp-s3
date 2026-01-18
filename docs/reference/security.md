# Security Reference

Security features, best practices, and verification methods for mcp-s3.

## Default Security

mcp-s3 is secure by default:

| Feature | Default | Description |
|---------|---------|-------------|
| Read-Only Mode | Enabled | Blocks PUT and DELETE operations |
| Size Limits | Enabled | Prevents excessive data transfer |
| Credential Isolation | Yes | Each connection uses separate credentials |

## Read-Only Mode

By default, write operations are blocked:

```bash
# Default - write operations blocked
export MCP_S3_EXT_READONLY=true

# Enable write operations (use with caution)
export MCP_S3_EXT_READONLY=false
```

### Blocked Operations

When read-only mode is enabled:
- `s3_put_object` - Returns error
- `s3_delete_object` - Returns error
- `s3_copy_object` - Returns error

## Size Limits

Prevent excessive data transfer:

```bash
# Maximum size for GET operations
export MCP_S3_MAX_GET_SIZE=10MB

# Maximum size for PUT operations
export MCP_S3_MAX_PUT_SIZE=100MB
```

Objects exceeding these limits are rejected with a clear error message.

## Prefix ACLs

Restrict access to specific object paths:

```bash
# Enable prefix ACLs
export MCP_S3_EXT_PREFIXACL=true

# Only allow access to these prefixes
export MCP_S3_ALLOWED_PREFIXES=data/,reports/,exports/

# Block access to these prefixes
export MCP_S3_DENIED_PREFIXES=secrets/,internal/,_private/
```

### Evaluation Order

1. Denied prefixes are checked first (deny wins)
2. If allowed prefixes are set, access requires a match
3. If no allowed prefixes, all non-denied prefixes are allowed

## Audit Logging

Track all operations:

```bash
export MCP_S3_EXT_AUDIT=true
```

Audit log format:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "tool": "s3_get_object",
  "bucket": "my-bucket",
  "key": "path/to/file.txt",
  "connection": "default",
  "status": "success",
  "duration_ms": 45
}
```

## Credential Security

### Environment Variables

Never commit credentials to version control:

```bash
# Bad - in .env file committed to git
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE

# Good - use a secret manager or CI/CD secrets
# GitHub Actions
${{ secrets.AWS_ACCESS_KEY_ID }}

# Kubernetes
secretKeyRef:
  name: aws-credentials
  key: access-key-id
```

### IAM Roles

When running on AWS, prefer IAM roles over static credentials:

```bash
# On EC2, ECS, or Lambda - no credentials needed
# IAM role attached to the instance/task/function provides access
unset AWS_ACCESS_KEY_ID
unset AWS_SECRET_ACCESS_KEY
```

### Least Privilege

Create IAM policies with minimal permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-bucket",
        "arn:aws:s3:::my-bucket/*"
      ]
    }
  ]
}
```

## Network Security

### HTTPS

Always use HTTPS for AWS S3 (default). For local development with S3-compatible storage:

```bash
# Development only - HTTP
export S3_ENDPOINT=http://localhost:8333

# Production - always HTTPS
export S3_ENDPOINT=https://s3.example.com
```

### VPC Endpoints

For AWS, use VPC endpoints to keep traffic private:

```hcl
resource "aws_vpc_endpoint" "s3" {
  vpc_id       = aws_vpc.main.id
  service_name = "com.amazonaws.us-east-1.s3"
}
```

## Release Verification

### Checksums

Verify download integrity:

```bash
# Download checksum file
curl -LO https://github.com/txn2/mcp-s3/releases/download/v0.1.0/checksums.txt

# Verify
sha256sum -c checksums.txt
```

### SLSA Provenance

Verify build provenance (SLSA Level 3):

```bash
# Download provenance
curl -LO https://github.com/txn2/mcp-s3/releases/download/v0.1.0/mcp-s3_0.1.0_linux_amd64.tar.gz.intoto.jsonl

# Verify with slsa-verifier
slsa-verifier verify-artifact \
  --provenance-path mcp-s3_0.1.0_linux_amd64.tar.gz.intoto.jsonl \
  --source-uri github.com/txn2/mcp-s3 \
  mcp-s3_0.1.0_linux_amd64.tar.gz
```

### Cosign Signatures

Verify with Sigstore Cosign:

```bash
cosign verify-blob \
  --bundle mcp-s3_0.1.0_linux_amd64.tar.gz.sigstore.json \
  mcp-s3_0.1.0_linux_amd64.tar.gz
```

## Container Security

### Non-Root User

The Docker image runs as a non-root user:

```dockerfile
RUN adduser -D -g '' mcp
USER mcp
```

### Read-Only Filesystem

Run with read-only root filesystem:

```bash
docker run --read-only \
  -e AWS_REGION=us-east-1 \
  ghcr.io/txn2/mcp-s3:latest
```

### Resource Limits

Apply resource limits:

```bash
docker run \
  --memory=256m \
  --cpus=0.5 \
  ghcr.io/txn2/mcp-s3:latest
```

## Security Checklist

- [ ] Read-only mode enabled for production
- [ ] Size limits configured appropriately
- [ ] Prefix ACLs restrict access to necessary paths
- [ ] Audit logging enabled for compliance
- [ ] IAM roles used instead of static credentials
- [ ] Minimal IAM permissions configured
- [ ] HTTPS used for all connections
- [ ] Release artifacts verified before deployment
- [ ] Container runs as non-root user
- [ ] Resource limits applied to containers

## Reporting Vulnerabilities

Report security issues via [GitHub Security Advisories](https://github.com/txn2/mcp-s3/security/advisories/new) or email security@txn2.com.

See [SECURITY.md](https://github.com/txn2/mcp-s3/blob/main/SECURITY.md) for details.
