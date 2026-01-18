# Contributing to mcp-s3

We welcome contributions to mcp-s3! This document provides guidelines for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/mcp-s3.git`
3. Create a branch: `git checkout -b my-feature`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes: `git commit -m "Add my feature"`
8. Push to your fork: `git push origin my-feature`
9. Open a Pull Request

## Development Setup

```bash
# Clone the repository
git clone https://github.com/txn2/mcp-s3.git
cd mcp-s3

# Install dependencies
make mod-download

# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make verify
```

## Testing with SeaweedFS

For local development, you can use SeaweedFS as an S3-compatible backend:

```bash
# Start SeaweedFS with S3 API
docker run -d -p 8333:8333 -p 9333:9333 \
  chrislusf/seaweedfs server -s3

# Configure environment
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=any
export AWS_SECRET_ACCESS_KEY=any

# Run
./build/mcp-s3
```

## Code Style

- Run `make fmt` before committing to format code
- Run `make lint` to check for style issues
- Follow Go best practices and idioms
- Write clear, descriptive commit messages

## Pull Request Guidelines

- Keep PRs focused on a single change
- Include tests for new functionality
- Update documentation as needed
- Ensure all checks pass before requesting review

## Reporting Issues

- Check existing issues before creating a new one
- Include steps to reproduce the issue
- Include relevant environment information (Go version, OS, etc.)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
