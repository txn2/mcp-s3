# Tools API

Complete reference for all mcp-s3 tools.

## s3_list_buckets

List all accessible S3 buckets.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `connection` | string | No | Connection name (default: primary) |

### Response

```json
{
  "buckets": [
    {
      "name": "my-bucket",
      "creation_date": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

---

## s3_list_objects

List objects in a bucket with optional filtering.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `prefix` | string | No | Filter by prefix |
| `delimiter` | string | No | Delimiter for hierarchy (usually `/`) |
| `max_keys` | integer | No | Maximum objects to return (default: 1000) |
| `continuation_token` | string | No | Token for pagination |
| `connection` | string | No | Connection name |

### Response

```json
{
  "objects": [
    {
      "key": "path/to/file.txt",
      "size": 1024,
      "last_modified": "2024-01-15T10:30:00Z",
      "etag": "\"d41d8cd98f00b204e9800998ecf8427e\"",
      "storage_class": "STANDARD"
    }
  ],
  "common_prefixes": ["path/to/folder/"],
  "is_truncated": false,
  "next_continuation_token": null,
  "key_count": 1
}
```

---

## s3_get_object

Retrieve object content.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `connection` | string | No | Connection name |

### Response

```json
{
  "key": "path/to/file.txt",
  "content_type": "text/plain",
  "size": 1024,
  "last_modified": "2024-01-15T10:30:00Z",
  "etag": "\"d41d8cd98f00b204e9800998ecf8427e\"",
  "body": "File content here...",
  "encoding": "text",
  "metadata": {
    "custom-key": "custom-value"
  }
}
```

### Notes

- Text content is returned as-is
- Binary content is returned as base64 with `encoding: "base64"`
- Objects larger than `MCP_S3_MAX_GET_SIZE` are rejected

---

## s3_get_object_metadata

Get object metadata without downloading content (HEAD request).

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `connection` | string | No | Connection name |

### Response

```json
{
  "key": "path/to/file.txt",
  "content_type": "text/plain",
  "content_length": 1024,
  "size": 1024,
  "last_modified": "2024-01-15T10:30:00Z",
  "etag": "\"d41d8cd98f00b204e9800998ecf8427e\"",
  "metadata": {
    "custom-key": "custom-value"
  }
}
```

---

## s3_put_object

Upload an object to S3.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `body` | string | Yes | Object content |
| `content_type` | string | No | Content-Type header |
| `metadata` | object | No | Custom metadata |
| `connection` | string | No | Connection name |

### Response

```json
{
  "bucket": "my-bucket",
  "key": "path/to/file.txt",
  "etag": "\"d41d8cd98f00b204e9800998ecf8427e\"",
  "size": 1024
}
```

### Notes

- Blocked by default when `MCP_S3_EXT_READONLY=true`
- Objects larger than `MCP_S3_MAX_PUT_SIZE` are rejected

---

## s3_delete_object

Delete an object from S3.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `connection` | string | No | Connection name |

### Response

```json
{
  "bucket": "my-bucket",
  "key": "path/to/file.txt",
  "deleted": true
}
```

### Notes

- Blocked by default when `MCP_S3_EXT_READONLY=true`

---

## s3_copy_object

Copy an object within or between buckets.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `source_bucket` | string | Yes | Source bucket name |
| `source_key` | string | Yes | Source object key |
| `dest_bucket` | string | Yes | Destination bucket name |
| `dest_key` | string | Yes | Destination object key |
| `metadata` | object | No | New metadata (replaces source) |
| `connection` | string | No | Connection name |

### Response

```json
{
  "source_bucket": "source-bucket",
  "source_key": "path/to/source.txt",
  "dest_bucket": "dest-bucket",
  "dest_key": "path/to/dest.txt",
  "etag": "\"d41d8cd98f00b204e9800998ecf8427e\""
}
```

### Notes

- Blocked by default when `MCP_S3_EXT_READONLY=true`

---

## s3_presign_url

Generate a presigned URL for direct access.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `bucket` | string | Yes | Bucket name |
| `key` | string | Yes | Object key |
| `method` | string | No | HTTP method: `GET` or `PUT` (default: `GET`) |
| `expires_in` | integer | No | URL validity in seconds (default: 3600) |
| `connection` | string | No | Connection name |

### Response

```json
{
  "url": "https://bucket.s3.amazonaws.com/key?X-Amz-Algorithm=...",
  "method": "GET",
  "expires_at": "2024-01-15T11:30:00Z"
}
```

---

## s3_list_connections

List all configured S3 connections.

### Parameters

None.

### Response

```json
{
  "connections": [
    {
      "name": "default",
      "region": "us-east-1"
    },
    {
      "name": "seaweedfs",
      "endpoint": "http://localhost:8333"
    }
  ],
  "default": "default"
}
```

---

## Error Responses

All tools return errors in this format:

```json
{
  "error": "AccessDenied",
  "message": "Access Denied",
  "hint": "Check that the credentials have appropriate permissions for this bucket."
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `NoSuchBucket` | Bucket does not exist |
| `NoSuchKey` | Object does not exist |
| `AccessDenied` | Insufficient permissions |
| `InvalidBucketName` | Invalid bucket name format |
| `EntityTooLarge` | Object exceeds size limit |
| `ReadOnlyMode` | Write operation blocked by read-only mode |
| `ConnectionNotFound` | Unknown connection name |
