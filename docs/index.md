---
hide:
  - toc
---

# txn2/mcp-s3

An MCP server that connects AI assistants to Amazon S3 and S3-compatible object storage. Browse buckets, read and write objects, and generate presigned URLs for secure file sharing.

Unlike other MCP servers, mcp-s3 is designed as a composable Go library. Import it into your own MCP server to add S3 capabilities with custom authentication, tenant isolation, and audit logging. The standalone server works out of the box; the library lets you build exactly what your organization needs.

[Get Started](server/installation.md){ .md-button .md-button--primary }
[View on GitHub](https://github.com/txn2/mcp-s3){ .md-button }

---

## Two Ways to Use

<div class="grid cards" markdown>

-   :material-server:{ .lg .middle } **Use the Server**

    ---

    Connect Claude, Cursor, or any MCP client to S3 with secure defaults.

    - Read-only mode by default
    - Size limits enforced
    - Multi-account support

    [:octicons-arrow-right-24: Install in 5 minutes](server/installation.md)

-   :material-code-braces:{ .lg .middle } **Build Custom MCP**

    ---

    Import the Go library for enterprise servers with auth, tenancy, and compliance.

    - OAuth, API keys, SSO
    - Bucket/prefix isolation
    - SOC2 / HIPAA audit logs

    [:octicons-arrow-right-24: View library docs](library/index.md)

</div>

---

## Core Capabilities

<div class="grid cards" markdown>

-   :material-puzzle:{ .lg .middle } **Composable Architecture**

    ---

    Import as a Go library to build custom MCP servers with authentication,
    tenant isolation, and audit logging without forking.

    [:octicons-arrow-right-24: Library docs](library/index.md)

-   :material-cloud:{ .lg .middle } **Multi-Provider Support**

    ---

    Works with AWS S3, SeaweedFS, LocalStack, and any S3-compatible storage.
    Connect to multiple accounts from a single installation.

    [:octicons-arrow-right-24: Configuration](server/configuration.md)

-   :material-server-network:{ .lg .middle } **Multi-Account**

    ---

    Query production, staging, and development S3 buckets from a single
    MCP installation with unified credentials.

    [:octicons-arrow-right-24: Multi-connection setup](server/configuration.md#multi-connection-setup)

-   :material-shield-check:{ .lg .middle } **Secure Defaults**

    ---

    Read-only mode prevents accidental writes. Size limits prevent abuse.
    Prefix ACLs restrict access to specific paths.

    [:octicons-arrow-right-24: Security](library/extensibility.md)

</div>

---

## Available Tools

| Tool | Description |
|------|-------------|
| `s3_list_buckets` | List all accessible S3 buckets |
| `s3_list_objects` | List objects with prefix/delimiter filtering |
| `s3_get_object` | Retrieve object content |
| `s3_get_object_metadata` | Get object metadata without downloading |
| `s3_put_object` | Upload an object (disabled in read-only mode) |
| `s3_delete_object` | Delete an object (disabled in read-only mode) |
| `s3_copy_object` | Copy an object within or between buckets |
| `s3_presign_url` | Generate presigned GET or PUT URLs |
| `s3_list_connections` | List configured S3 connections |

---

## Quick Start

=== "Claude Code"

    ```bash
    claude mcp add s3 \
      -e AWS_ACCESS_KEY_ID=your-key \
      -e AWS_SECRET_ACCESS_KEY=your-secret \
      -e AWS_REGION=us-east-1 \
      -- mcp-s3
    ```

=== "SeaweedFS"

    ```bash
    claude mcp add seaweedfs \
      -e S3_ENDPOINT=http://localhost:8333 \
      -e S3_USE_PATH_STYLE=true \
      -e AWS_ACCESS_KEY_ID=any \
      -e AWS_SECRET_ACCESS_KEY=any \
      -- mcp-s3
    ```

=== "Go Install"

    ```bash
    go install github.com/txn2/mcp-s3/cmd/mcp-s3@latest
    ```

---

## Next Steps

- [Installation Guide](server/installation.md) - Detailed installation instructions
- [Configuration](server/configuration.md) - Environment variables and setup options
- [Tools Reference](server/tools.md) - Complete tool documentation
- [Library Usage](library/quickstart.md) - Use mcp-s3 as a Go library
