# Build stage
FROM golang:1.23-alpine@sha256:f5a7ee6abd197fc4a66a9e31f3a7680af65a34a2efc1ec90cddb36ac0a4f5ee6 AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/txn2/mcp-s3/internal/server.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /mcp-s3 \
    ./cmd/mcp-s3

# Final stage
FROM alpine:3.21@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' mcp
USER mcp

# Copy binary from builder
COPY --from=builder /mcp-s3 /usr/local/bin/mcp-s3

# Set default environment
ENV AWS_REGION=us-east-1
ENV MCP_S3_EXT_READONLY=true

# Run the server
ENTRYPOINT ["/usr/local/bin/mcp-s3"]
