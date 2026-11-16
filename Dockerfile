# syntax=docker/dockerfile:1

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Install ca-certificates for TLS connections
RUN apk add --no-cache ca-certificates

# Copy the binary from goreleaser (multi-arch build context)
ARG TARGETARCH
COPY linux/${TARGETARCH}/mcp-s3 /usr/local/bin/mcp-s3

# Run as non-root user
RUN adduser -D -u 1000 mcp
USER mcp

# Only used with MCP_S3_TRANSPORT=http. The binary defaults to loopback, which is
# unreachable through a published port, so bind all interfaces inside the container.
# Restrict exposure on the host side with `-p 127.0.0.1:8080:8080`.
ENV MCP_S3_HTTP_ADDR=0.0.0.0:8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/mcp-s3"]
