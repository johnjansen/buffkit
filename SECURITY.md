# Security Policy

## Supported Versions

As Buffkit is currently in early development (pre-1.0), we only provide security updates for the latest release.

| Version | Supported          |
| ------- | ------------------ |
| < 1.0   | :white_check_mark: Latest only |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

### 1. Do NOT Create a Public Issue

Security vulnerabilities should not be reported through public GitHub issues as this could put users at risk.

### 2. Report Privately

Please report security vulnerabilities by emailing the maintainer directly through their GitHub profile or by using GitHub's private vulnerability reporting feature:

1. Go to the [Security tab](https://github.com/johnjansen/buffkit/security) of the repository
2. Click on "Report a vulnerability"
3. Fill out the form with details

### 3. Include Details

When reporting, please include:

- **Description**: Clear explanation of the vulnerability
- **Impact**: What can an attacker do with this vulnerability?
- **Steps to Reproduce**: Detailed steps to trigger the vulnerability
- **Affected Versions**: Which versions of Buffkit are affected
- **Suggested Fix**: If you have ideas on how to fix it

### 4. Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity:
  - Critical: 1-3 days
  - High: 1 week
  - Medium: 2-4 weeks
  - Low: Next release

## Security Best Practices

When using Buffkit in your applications:

### Authentication & Sessions

- Always use strong, random `AuthSecret` values
- Rotate session secrets regularly
- Never commit secrets to version control
- Use environment variables for sensitive configuration

```go
// Good - Using environment variable
config := buffkit.Config{
    AuthSecret: os.Getenv("BUFFKIT_AUTH_SECRET"),
}

// Bad - Hardcoded secret
config := buffkit.Config{
    AuthSecret: "hardcoded-secret-bad", // NEVER DO THIS
}
```

### CSRF Protection

- Always enable CSRF protection for state-changing operations
- Verify CSRF tokens on all POST/PUT/DELETE requests
- Use the built-in CSRF middleware

### Content Security

- Keep Content-Security-Policy headers enabled in production
- Only relax security headers in development mode
- Sanitize all user input before rendering

### Redis Security

- Use strong Redis passwords in production
- Enable Redis ACLs where possible
- Use TLS for Redis connections in production
- Limit Redis access to application servers only

```go
// Production Redis configuration
config := buffkit.Config{
    RedisURL: "rediss://user:password@redis.example.com:6380/0", // TLS enabled
}
```

### Development vs Production

- Never use `DevMode: true` in production
- Development mail preview should never be accessible in production
- Ensure debug endpoints are disabled in production

```go
// Ensure this is controlled by environment
config := buffkit.Config{
    DevMode: os.Getenv("GO_ENV") == "development",
}
```

## Known Security Features

Buffkit includes several built-in security features:

### Automatic Security Headers

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy` (configurable)
- `Strict-Transport-Security` (when served over HTTPS)

### CSRF Protection

- Automatic CSRF token generation and validation
- Double-submit cookie pattern
- SameSite cookie attributes

### Session Security

- Secure session cookies (when over HTTPS)
- HttpOnly cookies by default
- Configurable session timeout
- Session rotation on privilege escalation

## Security Checklist

Before deploying a Buffkit application:

- [ ] Strong `AuthSecret` configured via environment variable
- [ ] Redis password set and connection secured
- [ ] `DevMode` is false
- [ ] HTTPS enabled with valid certificates
- [ ] Security headers verified in production
- [ ] CSRF protection enabled for all forms
- [ ] Input validation implemented
- [ ] Output properly escaped
- [ ] Error messages don't leak sensitive information
- [ ] Logging doesn't include sensitive data
- [ ] Dependencies up to date (`go mod tidy && go get -u`)
- [ ] No debug endpoints exposed

## Dependency Security

Keep dependencies updated:

```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Update dependencies
go get -u ./...
go mod tidy

# Verify dependencies
go mod verify
```

## Contact

For urgent security issues, you can also reach out via:
- GitHub Security Advisory feature
- Direct message to maintainers

## Acknowledgments

We appreciate responsible disclosure and will acknowledge security researchers who:
- Follow responsible disclosure practices
- Give us reasonable time to fix issues
- Don't exploit vulnerabilities beyond POC

Thank you for helping keep Buffkit and its users safe!