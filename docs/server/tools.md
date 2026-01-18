# Tools Reference

Complete reference for all MCP tools provided by mcp-s3.

## s3_list_buckets

List all accessible S3 buckets.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `connection` | string | No | Connection name (default: primary) |

**Example Response:**
```json
{
  "buckets": [
    {"name": "my-bucket", "creation_date": "2024-01-15T10:30:00Z"},
    {"name": "logs-bucket", "creation_date": "2024-02-20T14:45:00Z"}
  ],
  "count": 2
}
```

## s3_list_objects

List objects in a bucket with optional filtering and pagination.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `prefix` | string | No | Filter by key prefix |
| `delimiter` | string | No | Delimiter for folder simulation (usually `/`) |
| `max_keys` | integer | No | Maximum objects to return (1-1000, default: 1000) |
| `continuation_token` | string | No | Token for pagination |
| `connection` | string | No | Connection name |

**Example Response:**
```json
{
  "bucket": "my-bucket",
  "prefix": "data/",
  "objects": [
    {"key": "data/file1.json", "size": 1024, "last_modified": "2024-03-01T12:00:00Z"},
    {"key": "data/file2.json", "size": 2048, "last_modified": "2024-03-02T14:30:00Z"}
  ],
  "common_prefixes": ["data/archive/", "data/logs/"],
  "count": 2,
  "is_truncated": false
}
```

## s3_get_object

Retrieve object content from S3.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `connection` | string | No | Connection name |

**Notes:**

- Text content is returned directly
- Binary content is returned as base64-encoded string with `is_base64: true`
- Subject to `MCP_S3_MAX_GET_SIZE` limit

## s3_get_object_metadata

Get object metadata without downloading content.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `connection` | string | No | Connection name |

**Example Response:**
```json
{
  "bucket": "my-bucket",
  "key": "data/file.json",
  "size": 1024,
  "content_type": "application/json",
  "last_modified": "2024-03-01T12:00:00Z",
  "etag": "\"d41d8cd98f00b204e9800998ecf8427e\""
}
```

## s3_put_object

Upload an object to S3.

!!! warning "Requires Write Access"
    This tool is blocked when `MCP_S3_EXT_READONLY=true` (default).

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `content` | string | Yes | Content to upload |
| `content_type` | string | No | MIME type (default: `application/octet-stream`) |
| `is_base64` | boolean | No | Set true if content is base64-encoded |
| `metadata` | object | No | Custom metadata key-value pairs |
| `connection` | string | No | Connection name |

## s3_delete_object

Delete an object from S3.

!!! warning "Requires Write Access"
    This tool is blocked when `MCP_S3_EXT_READONLY=true` (default).

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `connection` | string | No | Connection name |

## s3_copy_object

Copy an object within or between buckets.

!!! warning "Requires Write Access"
    This tool is blocked when `MCP_S3_EXT_READONLY=true` (default).

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `source_bucket` | string | Yes | Source bucket name |
| `source_key` | string | Yes | Source object key |
| `dest_bucket` | string | Yes | Destination bucket name |
| `dest_key` | string | Yes | Destination object key |
| `metadata` | object | No | New metadata (replaces source metadata) |
| `connection` | string | No | Connection name |

## s3_presign_url

Generate a presigned URL for temporary access.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `method` | string | No | `GET` or `PUT` (default: `GET`) |
| `expires_in` | integer | No | Expiration in seconds (default: 3600, max: 604800) |
| `connection` | string | No | Connection name |

**Example Response:**
```json
{
  "bucket": "my-bucket",
  "key": "data/file.json",
  "url": "https://my-bucket.s3.amazonaws.com/data/file.json?X-Amz-...",
  "method": "GET",
  "expires_in_seconds": 3600,
  "expires_at": "2024-03-15T13:00:00Z"
}
```

## s3_list_connections

List all configured S3 connections.

**Parameters:** None

**Example Response:**
```json
{
  "connections": [
    {"name": "production", "region": "us-east-1"},
    {"name": "staging", "region": "us-west-2"},
    {"name": "local", "region": "us-east-1", "endpoint": "http://localhost:8333"}
  ],
  "default_connection": "production",
  "count": 3
}
```
