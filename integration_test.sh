#!/bin/bash
# Integration test script for mcp-s3 using SeaweedFS
# Uses named pipes to maintain connection to MCP server

set -e

# Configuration
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=any
export AWS_SECRET_ACCESS_KEY=any
export AWS_REGION=us-east-1
export MCP_S3_EXT_READONLY=false

PIPE_IN=/tmp/mcp_input_$$
PIPE_OUT=/tmp/mcp_output_$$
PASSED=0
FAILED=0

cleanup() {
    exec 3>&- 2>/dev/null || true
    kill $SERVER_PID 2>/dev/null || true
    rm -f "$PIPE_IN" "$PIPE_OUT"
}
trap cleanup EXIT

# Create test bucket using curl
echo "Setting up test environment..."
curl -s -X PUT http://localhost:8333/integration-test-bucket > /dev/null

# Create named pipes
rm -f "$PIPE_IN" "$PIPE_OUT"
mkfifo "$PIPE_IN"

# Start MCP server
./mcp-s3-test < "$PIPE_IN" > "$PIPE_OUT" 2>/dev/null &
SERVER_PID=$!
sleep 1

# Open pipe for writing
exec 3>"$PIPE_IN"

# Send initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' >&3
sleep 0.5

# Read and discard initialization response
head -2 "$PIPE_OUT" > /dev/null

echo ""
echo "=== MCP-S3 Integration Tests with SeaweedFS ==="
echo ""

# Function to send request and check response
test_tool() {
    local name="$1"
    local request="$2"
    local expected="$3"
    local description="$4"

    echo "$request" >&3
    sleep 0.5
    local response=$(head -1 "$PIPE_OUT")

    if echo "$response" | grep -q "$expected"; then
        echo "✅ $name: PASSED - $description"
        ((PASSED++))
    else
        echo "❌ $name: FAILED"
        echo "   Expected to contain: $expected"
        echo "   Response: $response"
        ((FAILED++))
    fi
}

# Test 1: List Buckets
test_tool "s3_list_buckets" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"s3_list_buckets","arguments":{}}}' \
    "integration-test-bucket" \
    "Found test bucket"

# Test 2: Put Object
test_tool "s3_put_object" \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"s3_put_object","arguments":{"bucket":"integration-test-bucket","key":"hello.txt","content":"Hello from MCP-S3 integration test!"}}}' \
    '"key":"hello.txt"' \
    "Object uploaded"

# Test 3: List Objects
test_tool "s3_list_objects" \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"s3_list_objects","arguments":{"bucket":"integration-test-bucket"}}}' \
    "hello.txt" \
    "Object found in listing"

# Test 4: Get Object
test_tool "s3_get_object" \
    '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"s3_get_object","arguments":{"bucket":"integration-test-bucket","key":"hello.txt"}}}' \
    "Hello from MCP-S3 integration test!" \
    "Content retrieved correctly"

# Test 5: Get Object Metadata
test_tool "s3_get_object_metadata" \
    '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"s3_get_object_metadata","arguments":{"bucket":"integration-test-bucket","key":"hello.txt"}}}' \
    '"content_length"' \
    "Metadata retrieved"

# Test 6: Copy Object
test_tool "s3_copy_object" \
    '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"s3_copy_object","arguments":{"source_bucket":"integration-test-bucket","source_key":"hello.txt","dest_bucket":"integration-test-bucket","dest_key":"hello-copy.txt"}}}' \
    "hello-copy.txt" \
    "Object copied"

# Test 7: Verify copy with list
test_tool "s3_list_objects (verify copy)" \
    '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"s3_list_objects","arguments":{"bucket":"integration-test-bucket"}}}' \
    "hello-copy.txt" \
    "Copied object found"

# Test 8: Presign URL (GET)
test_tool "s3_presign_url (GET)" \
    '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"s3_presign_url","arguments":{"bucket":"integration-test-bucket","key":"hello.txt","operation":"get"}}}' \
    "http" \
    "Presigned GET URL generated"

# Test 9: Presign URL (PUT)
test_tool "s3_presign_url (PUT)" \
    '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"s3_presign_url","arguments":{"bucket":"integration-test-bucket","key":"new-upload.txt","operation":"put"}}}' \
    "http" \
    "Presigned PUT URL generated"

# Test 10: List Connections
test_tool "s3_list_connections" \
    '{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"s3_list_connections","arguments":{}}}' \
    "connections" \
    "Connections listed"

# Test 11: Delete Object
test_tool "s3_delete_object" \
    '{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"s3_delete_object","arguments":{"bucket":"integration-test-bucket","key":"hello-copy.txt"}}}' \
    "deleted" \
    "Object deleted"

# Test 12: Verify delete with list
test_tool "s3_list_objects (verify delete)" \
    '{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"s3_list_objects","arguments":{"bucket":"integration-test-bucket"}}}' \
    "hello.txt" \
    "Original object still exists"

# Test 13: Put binary object (base64)
BASE64_CONTENT=$(echo -n "Binary content test" | base64)
test_tool "s3_put_object (base64)" \
    "{\"jsonrpc\":\"2.0\",\"id\":14,\"method\":\"tools/call\",\"params\":{\"name\":\"s3_put_object\",\"arguments\":{\"bucket\":\"integration-test-bucket\",\"key\":\"binary.bin\",\"content\":\"$BASE64_CONTENT\",\"is_base64\":true}}}" \
    '"key":"binary.bin"' \
    "Binary object uploaded"

# Test 14: Put object with content type
test_tool "s3_put_object (with content-type)" \
    '{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"s3_put_object","arguments":{"bucket":"integration-test-bucket","key":"data.json","content":"{\"test\":true}","content_type":"application/json"}}}' \
    '"key":"data.json"' \
    "JSON object uploaded with content type"

# Test 15: List objects with prefix
test_tool "s3_list_objects (with prefix)" \
    '{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"s3_list_objects","arguments":{"bucket":"integration-test-bucket","prefix":"hello"}}}' \
    "hello.txt" \
    "Prefix filter works"

echo ""
echo "=== Results ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "All integration tests PASSED!"
    exit 0
else
    echo "Some tests FAILED!"
    exit 1
fi
