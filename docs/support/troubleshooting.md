# Troubleshooting

Solutions to common issues with mcp-s3.

## Connection Issues

### Connection Refused

**Symptoms:**
```
Error: dial tcp 127.0.0.1:8333: connect: connection refused
```

**Causes:**
- S3-compatible server not running
- Wrong endpoint or port
- Firewall blocking connection

**Solutions:**

1. Verify the server is running:
   ```bash
   curl http://localhost:8333/
   ```

2. Check endpoint configuration:
   ```bash
   echo $S3_ENDPOINT
   ```

3. Test network connectivity:
   ```bash
   nc -zv localhost 8333
   ```

### Connection Timeout

**Symptoms:**
```
Error: connection timeout after 30s
```

**Causes:**
- Network latency
- Firewall timeout
- DNS resolution issues

**Solutions:**

1. Check DNS resolution:
   ```bash
   nslookup s3.amazonaws.com
   ```

2. Increase timeout:
   ```bash
   export S3_TIMEOUT=60s
   ```

3. Test with AWS CLI:
   ```bash
   aws s3 ls
   ```

## Authentication Errors

### Access Denied

**Symptoms:**
```
Error: AccessDenied: Access Denied
```

**Solutions:**

1. Verify credentials are set:
   ```bash
   echo "Key: $AWS_ACCESS_KEY_ID"
   [ -n "$AWS_SECRET_ACCESS_KEY" ] && echo "Secret is set"
   ```

2. Test credentials directly:
   ```bash
   aws sts get-caller-identity
   ```

3. Check IAM permissions for the bucket:
   ```bash
   aws s3 ls s3://bucket-name/
   ```

### Invalid Credentials

**Symptoms:**
```
Error: InvalidAccessKeyId: The AWS Access Key Id you provided does not exist
```

**Solutions:**

1. Verify the access key ID is correct
2. Check that the credentials haven't been rotated
3. Ensure you're using the right AWS account

### Signature Mismatch

**Symptoms:**
```
Error: SignatureDoesNotMatch: The request signature we calculated does not match
```

**Solutions:**

1. Check for trailing whitespace in credentials:
   ```bash
   export AWS_SECRET_ACCESS_KEY=$(echo "$AWS_SECRET_ACCESS_KEY" | tr -d ' \n')
   ```

2. Verify clock synchronization:
   ```bash
   date
   ```

3. For S3-compatible storage, check signature version compatibility

## Bucket and Object Errors

### Bucket Not Found

**Symptoms:**
```
Error: NoSuchBucket: The specified bucket does not exist
```

**Solutions:**

1. Verify bucket name:
   ```bash
   aws s3 ls
   ```

2. Check region configuration:
   ```bash
   aws s3api get-bucket-location --bucket bucket-name
   ```

3. Ensure the bucket exists in the correct account

### Object Not Found

**Symptoms:**
```
Error: NoSuchKey: The specified key does not exist
```

**Solutions:**

1. List objects to verify the key:
   ```bash
   aws s3 ls s3://bucket-name/prefix/
   ```

2. Check for typos in the key name
3. Verify object hasn't been deleted

## Size Limit Errors

### Object Too Large (GET)

**Symptoms:**
```
Error: Object size (52428800 bytes) exceeds maximum allowed size (10485760 bytes)
```

**Solutions:**

1. Increase the GET size limit:
   ```bash
   export MCP_S3_MAX_GET_SIZE=100MB
   ```

2. Use presigned URLs for large objects:
   > "Generate a presigned URL for this file"

### Object Too Large (PUT)

**Symptoms:**
```
Error: Object size exceeds maximum allowed size for PUT operations
```

**Solutions:**

1. Increase the PUT size limit:
   ```bash
   export MCP_S3_MAX_PUT_SIZE=500MB
   ```

2. Consider using multipart upload for very large files

## Read-Only Mode

**Symptoms:**
```
Error: Write operation blocked: PUT operations are not allowed in read-only mode
```

**Solutions:**

1. If write access is needed, disable read-only mode:
   ```bash
   export MCP_S3_EXT_READONLY=false
   ```

2. Or use presigned URLs for uploads:
   > "Generate a presigned PUT URL"

## S3-Compatible Storage Issues

### Path Style Required

**Symptoms:**
```
Error: The bucket you are attempting to access must be addressed using the specified endpoint
```

**Solutions:**

1. Enable path-style URLs:
   ```bash
   export S3_USE_PATH_STYLE=true
   ```

### SeaweedFS Specific

**Symptoms:**
```
Error: dial tcp: lookup bucket.localhost: no such host
```

**Solutions:**

1. Enable path-style URLs:
   ```bash
   export S3_USE_PATH_STYLE=true
   ```

2. Use the correct endpoint:
   ```bash
   export S3_ENDPOINT=http://localhost:8333
   ```

## Verbose Logging

Enable detailed logging for debugging:

```bash
# Enable request logging
export MCP_S3_EXT_LOGGING=true

# Run with logs to stderr
mcp-s3 2>mcp-s3.log
```

## Debug Mode

### Test Connection

```bash
# Verify configuration
env | grep -E "AWS_|S3_|MCP_S3"

# Test with AWS CLI
aws s3 ls --endpoint-url $S3_ENDPOINT
```

### Check Effective Configuration

```bash
# Print all relevant environment variables
printenv | grep -E "^(AWS_|S3_|MCP_S3)" | sort
```

## Docker Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs mcp-s3

# Run interactively
docker run -it --rm \
  -e AWS_REGION=us-east-1 \
  -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
  ghcr.io/txn2/mcp-s3:latest
```

### Network Issues in Docker

```bash
# Use host network for debugging
docker run --network=host \
  -e S3_ENDPOINT=http://localhost:8333 \
  ghcr.io/txn2/mcp-s3:latest
```

### Connect to Local S3-Compatible Storage

```bash
# Use host.docker.internal on macOS/Windows
docker run \
  -e S3_ENDPOINT=http://host.docker.internal:8333 \
  -e S3_USE_PATH_STYLE=true \
  ghcr.io/txn2/mcp-s3:latest
```

## Getting Help

If you can't resolve an issue:

1. Check the [GitHub Issues](https://github.com/txn2/mcp-s3/issues)
2. Open a new issue with:
   - mcp-s3 version
   - Configuration (without credentials)
   - Error message
   - Steps to reproduce
