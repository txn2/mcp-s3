# Installation

## Binary Installation

Download the latest release for your platform:

=== "Linux (amd64)"

    ```bash
    curl -LO https://github.com/txn2/mcp-s3/releases/latest/download/mcp-s3_linux_amd64.tar.gz
    tar -xzf mcp-s3_linux_amd64.tar.gz
    chmod +x mcp-s3
    sudo mv mcp-s3 /usr/local/bin/
    ```

=== "Linux (arm64)"

    ```bash
    curl -LO https://github.com/txn2/mcp-s3/releases/latest/download/mcp-s3_linux_arm64.tar.gz
    tar -xzf mcp-s3_linux_arm64.tar.gz
    chmod +x mcp-s3
    sudo mv mcp-s3 /usr/local/bin/
    ```

=== "macOS (Intel)"

    ```bash
    curl -LO https://github.com/txn2/mcp-s3/releases/latest/download/mcp-s3_darwin_amd64.tar.gz
    tar -xzf mcp-s3_darwin_amd64.tar.gz
    chmod +x mcp-s3
    sudo mv mcp-s3 /usr/local/bin/
    ```

=== "macOS (Apple Silicon)"

    ```bash
    curl -LO https://github.com/txn2/mcp-s3/releases/latest/download/mcp-s3_darwin_arm64.tar.gz
    tar -xzf mcp-s3_darwin_arm64.tar.gz
    chmod +x mcp-s3
    sudo mv mcp-s3 /usr/local/bin/
    ```

## Go Installation

```bash
go install github.com/txn2/mcp-s3/cmd/mcp-s3@latest
```

## Docker

```bash
docker pull txn2/mcp-s3:latest
```

## Homebrew (macOS/Linux)

```bash
brew tap txn2/tap
brew install mcp-s3
```

## Verify Installation

```bash
mcp-s3 --version
```

## Next Steps

- [Configuration](configuration.md) - Set up AWS credentials and options
- [Tools Reference](tools.md) - Learn about available S3 tools
