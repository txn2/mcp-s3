# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability within mcp-s3, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **GitHub Security Advisories** (Preferred): Use [GitHub's private vulnerability reporting](https://github.com/txn2/mcp-s3/security/advisories/new) to report the vulnerability directly.

2. **Email**: Send an email to security@txn2.com with:
   - A description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact of the vulnerability
   - Any suggested fixes (optional)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours.
- **Communication**: We will keep you informed about the progress of fixing the vulnerability.
- **Timeline**: We aim to release a fix within 90 days of the initial report, depending on complexity.
- **Credit**: We will credit you in the release notes (unless you prefer to remain anonymous).

### Security Best Practices for Users

When deploying mcp-s3:

1. **Credentials Management**
   - Never commit credentials to version control
   - Use environment variables or secret managers for sensitive configuration
   - Rotate credentials regularly
   - Use IAM roles when running on AWS infrastructure

2. **Network Security**
   - Use HTTPS endpoints when connecting to S3
   - Deploy behind a firewall or VPN when possible
   - Use VPC endpoints for AWS S3 access in production

3. **Access Control**
   - Use IAM policies with minimal necessary permissions
   - Consider using read-only credentials for exploration
   - Use bucket policies to restrict access

4. **Data Safety**
   - mcp-s3 defaults to read-only mode (`MCP_S3_EXT_READONLY=true`)
   - Size limits prevent excessive data transfer (default: 10MB GET, 100MB PUT)
   - Use prefix ACLs to restrict accessible paths

5. **Logging and Monitoring**
   - Enable audit logging to track all operations
   - Monitor access patterns for unusual activity
   - Set up alerts for failed authentication attempts

## Security Features

mcp-s3 includes several security features by default:

- **Read-Only Mode**: Write operations blocked by default
- **Size Limits**: Configurable limits prevent excessive data transfer
- **Prefix ACLs**: Restrict access to specific object prefixes
- **Audit Logging**: Optional logging of all operations
- **Multi-Account**: Separate credentials per connection

## Security Updates

Security updates are released as patch versions and announced via:

- GitHub Security Advisories
- Release notes
- The project README

We recommend always running the latest version of mcp-s3.
