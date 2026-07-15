# Security Policy

## Supported Versions

We provide security updates for the following versions of git-herd:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in git-herd, please follow these steps:

### 1. Do NOT Create a Public Issue

Please do not create a public GitHub issue for security vulnerabilities. This helps protect users until a fix is available.

### 2. Report Privately

Send your security report via email to:
- **Email**: [security@entro314-labs.com]
- **Subject**: `[SECURITY] git-herd Vulnerability Report`

### 3. Include Detailed Information

Please include the following information in your report:

- **Description**: A detailed description of the vulnerability
- **Impact**: What could an attacker accomplish by exploiting this vulnerability?
- **Reproduction**: Step-by-step instructions to reproduce the vulnerability
- **Environment**: Operating system, git-herd version, Go version
- **Supporting Material**: Screenshots, logs, or proof-of-concept code (if applicable)

### 4. Response Timeline

We are committed to responding to security reports promptly:

- **Acknowledgment**: We will acknowledge your report within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 5 business days
- **Status Updates**: We will provide regular updates throughout the investigation
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

### 5. Coordinated Disclosure

We follow responsible disclosure practices:

1. We will work with you to understand and validate the vulnerability
2. We will develop and test a fix
3. We will coordinate the public disclosure timeline with you
4. We will credit you in our security advisory (if desired)

## Security Considerations

### General Security Practices

When using git-herd, consider the following security best practices:

#### 1. File System Access
- git-herd requires read access to directories and git repositories
- Ensure you only run git-herd on directories you trust
- Be cautious when running with elevated privileges

#### 2. Git Operations
- git-herd performs `git fetch` and `git pull` operations
- These operations can trigger git hooks in repositories
- Review git hooks in repositories before running git-herd
- Use the `--dry-run` flag to preview operations

#### 3. Network Operations
- Git operations may involve network requests to remote repositories
- Ensure your network environment is secure
- Consider using SSH keys for authentication instead of passwords

#### 4. Configuration Files
- git-herd reads configuration from files in your home directory
- Ensure configuration files have appropriate permissions (600 or 644)
- Do not include sensitive information in configuration files

### Known Security Limitations

#### 1. Git Hook Execution
- git-herd does not prevent execution of git hooks
- Malicious hooks could potentially execute arbitrary code
- **Mitigation**: Review repositories and their hooks before processing

#### 2. Path Traversal
- git-herd processes directory structures
- While we implement path validation, use caution with untrusted directories
- **Mitigation**: Use absolute paths and avoid processing untrusted directories

#### 3. Resource Consumption
- Processing many repositories can consume significant resources
- Large repositories may cause high memory or disk usage
- **Mitigation**: Use the `--workers` flag to limit concurrent operations

## Security Features

### Input Validation
- Path validation to prevent directory traversal
- Repository validation before processing
- Configuration parameter validation

### Error Handling
- Secure error messages that don't leak sensitive information
- Proper cleanup of temporary resources
- Graceful handling of interrupted operations

### Process Isolation
- Each git operation runs in a separate process context
- Timeout mechanisms to prevent hung operations
- Signal handling for graceful shutdown

## Security Updates

### Notification Channels
Stay informed about security updates through:

- **GitHub Security Advisories**: We publish security advisories on our GitHub repository
- **GitHub Releases**: Security updates are clearly marked in release notes
- **Email Notifications**: If you've reported a vulnerability, we'll notify you of the fix

### Update Recommendations
- **Critical Vulnerabilities**: Update immediately
- **High Severity**: Update within 1 week
- **Medium/Low Severity**: Update with next scheduled maintenance

### Verification
All releases are:
- Signed with GPG keys
- Published with checksums for verification
- Available through official distribution channels

## Dependencies

### Dependency Management
We actively monitor our dependencies for security vulnerabilities:

- Regular dependency updates through automated tools
- Security scanning of dependencies
- Prompt response to vulnerability disclosures in dependencies

### Third-Party Components
git-herd uses the following main dependencies:
- Go standard library
- Cobra CLI framework
- Bubble Tea TUI framework
- Viper configuration library
- go-git library

## Compliance and Standards

### Security Standards
git-herd follows these security practices:

- **OWASP Guidelines**: We follow OWASP secure coding practices
- **Go Security**: We follow Go-specific security recommendations
- **Supply Chain Security**: We verify dependencies and use pinned versions

### Security Testing
Our security testing includes:

- **Static Analysis**: Automated code security scanning
- **Dependency Scanning**: Vulnerability scanning of dependencies
- **Manual Review**: Security-focused code reviews
- **Penetration Testing**: Periodic security assessments

## Contact Information

For security-related questions or concerns:

- **Security Team**: security@entro314-labs.com
- **General Contact**: support@entro314-labs.com
- **GitHub Issues**: For non-security bugs and features only

## Acknowledgments

We appreciate the security research community's efforts to improve software security. Security researchers who responsibly disclose vulnerabilities will be:

- Credited in our security advisories (if desired)
- Listed in our hall of fame (if desired)
- Kept informed throughout the resolution process

---

**Note**: This security policy is subject to change. Please check this document periodically for updates. Last updated: January 2025.