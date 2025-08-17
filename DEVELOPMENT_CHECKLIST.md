# Buffkit Development Checklist

## Pre-Development Checklist

### 1. BDD/TDD First
- [ ] Write feature file scenarios BEFORE coding
- [ ] Define step definitions 
- [ ] Run tests to see them fail
- [ ] Implement only enough to make tests pass
- [ ] Refactor if needed

### 2. Architecture Check
- [ ] One concern per file/function
- [ ] Methods under 20 lines
- [ ] Cyclomatic complexity under 8
- [ ] Proper separation of concerns
- [ ] No premature optimization

## Pre-Commit Checklist

### 1. Compilation
```bash
go build ./...
```
- [ ] All packages compile without errors

### 2. Vetting
```bash
go vet ./...
```
- [ ] No vet issues

### 3. Linting
```bash
golangci-lint run ./...
```
- [ ] 0 linting issues
- [ ] errcheck: All errors handled (`_ = functionCall()` or proper error handling)
- [ ] ineffassign: No unused assignments
- [ ] staticcheck: No empty branches or unnecessary type declarations

### 4. Formatting
```bash
gofmt -l .
```
- [ ] No unformatted files

### 5. Testing
```bash
go test ./...
```
- [ ] All tests pass
- [ ] BDD scenarios pass
- [ ] Test coverage maintained or improved

### 6. Dependencies
```bash
go mod verify
go mod tidy
```
- [ ] Dependencies verified
- [ ] go.mod and go.sum are tidy

## Code Quality Checklist

### Comments & Documentation
- [ ] Aggressive but clear commenting
- [ ] Explain what, how, and why
- [ ] No redundant comments
- [ ] Public APIs documented

### Error Handling
- [ ] All errors handled appropriately
- [ ] No exceptions for control flow
- [ ] Consistent error messages
- [ ] Proper error wrapping with context

### Type Safety
- [ ] Type hints used where they improve clarity
- [ ] Avoid type hints when they add noise
- [ ] Return consistent types
- [ ] No interface{} abuse

### Security
- [ ] No hardcoded secrets
- [ ] Input validation
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection where needed

## Buffalo/Buffkit Specific

### Routing
- [ ] Routes properly registered in Wire()
- [ ] Middleware applied correctly
- [ ] Protected routes use RequireLogin
- [ ] Rate limiting on sensitive endpoints

### Templates
- [ ] Templates in correct directories
- [ ] Shadowable design maintained
- [ ] No inline scripts (use import maps)
- [ ] Proper escaping of user content

### Database
- [ ] Migrations created for schema changes
- [ ] Down migrations provided
- [ ] SQL is dialect-aware when needed
- [ ] Prepared statements used

### Background Jobs
- [ ] Jobs registered with mux
- [ ] Proper error handling in job handlers
- [ ] Idempotent job design
- [ ] Appropriate queue priorities

## CI/CD Checklist

### GitHub Actions
- [ ] Check `.github/workflows/*.yml` for requirements
- [ ] Ensure local checks match CI
- [ ] All required checks pass

### Performance
- [ ] No N+1 queries
- [ ] Efficient algorithms
- [ ] Proper caching where beneficial
- [ ] Resource cleanup (defer close)

## Platform Compatibility

### Operating Systems
- [ ] Works on macOS (development)
- [ ] Works on Linux (production)
- [ ] No Windows-specific code

### Dependencies
- [ ] Go 1.21+ compatible
- [ ] Buffalo v0.18+ compatible
- [ ] All dependencies in go.mod

## Git Hygiene

### Commits
- [ ] Clear, descriptive commit messages
- [ ] One logical change per commit
- [ ] No debug code committed
- [ ] No commented-out code

### Branch Management
- [ ] Feature branches for new work
- [ ] Branch names describe the change
- [ ] Rebase before merging
- [ ] No merge commits in feature branches

## Post-Development

### Documentation
- [ ] README updated if needed
- [ ] API changes documented
- [ ] Breaking changes noted
- [ ] Examples updated

### Review
- [ ] Self-review completed
- [ ] Edge cases considered
- [ ] Performance implications reviewed
- [ ] Security implications reviewed

## Quick Commands Reference

```bash
# Install pre-commit hook
git config core.hooksPath .githooks

# Run all checks
go build ./... && \
go vet ./... && \
golangci-lint run ./... && \
go test ./... && \
go mod verify && \
go mod tidy

# Run BDD tests only
go test ./features -v

# Check formatting
gofmt -l .

# Format all files
gofmt -w .

# Run with race detection
go test -race ./...

# Check test coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Remember

> **The user's rules:**
> - TDD or BDD, always!
> - Ship-it ... we go straight to production
> - Be pedantic
> - Be critical but constructive
> - Harsh is better than kind
> - Don't keep telling me I'm right, just listen
> - Focus beats overreach
> - Clarity beats cleverness
> - Simplicity beats complexity

## Setup Instructions

1. **Enable pre-commit hooks:**
   ```bash
   git config core.hooksPath .githooks
   ```

2. **Install required tools:**
   ```bash
   # Install golangci-lint
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Verify installation
   golangci-lint version
   ```

3. **Run checks before committing:**
   ```bash
   ./.githooks/pre-commit
   ```

4. **For CI debugging:**
   - Check GitHub Actions logs
   - Run same commands locally that CI runs
   - Ensure environment matches (Go version, etc.)

---

*This checklist is mandatory. No exceptions. Quality over speed.*