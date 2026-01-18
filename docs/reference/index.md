# Reference

Complete technical reference for mcp-s3.

## Quick Links

| Section | Description |
|---------|-------------|
| [Tools API](tools-api.md) | All tool parameters, responses, and error codes |
| [Configuration](configuration.md) | Environment variables and configuration options |
| [Security](security.md) | Limits, authentication, and verification |

## Tools

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

## Environment Variables

### Connection

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_REGION` | `us-east-1` | AWS region |
| `AWS_ACCESS_KEY_ID` | | Access key |
| `AWS_SECRET_ACCESS_KEY` | | Secret key |
| `AWS_SESSION_TOKEN` | | Session token (optional) |
| `AWS_PROFILE` | | Profile name (optional) |
| `S3_ENDPOINT` | | Custom endpoint for S3-compatible storage |
| `S3_USE_PATH_STYLE` | `false` | Use path-style URLs |
| `S3_TIMEOUT` | `30s` | Operation timeout |

### Extensions

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_S3_EXT_READONLY` | `true` | Block write operations |
| `MCP_S3_EXT_SIZELIMIT` | `true` | Enforce size limits |
| `MCP_S3_MAX_GET_SIZE` | `10MB` | Max bytes for GET |
| `MCP_S3_MAX_PUT_SIZE` | `100MB` | Max bytes for PUT |
| `MCP_S3_EXT_LOGGING` | `false` | Enable request logging |
| `MCP_S3_EXT_AUDIT` | `false` | Enable audit logging |

## Limits

| Limit | Default | Maximum |
|-------|---------|---------|
| GET size | 10MB | Configurable |
| PUT size | 100MB | Configurable |
| Operation timeout | 30s | 300s |

## Blocked Operations (Read-Only Mode)

- `s3_put_object`
- `s3_delete_object`

## Release Verification

All releases include:

- **Checksums** - SHA256 verification
- **SLSA Provenance** - Level 3 build attestation
- **Cosign Signatures** - Keyless verification

```bash
# Verify with Cosign
cosign verify-blob \
  --bundle mcp-s3_*.tar.gz.sigstore.json \
  mcp-s3_*.tar.gz
```
