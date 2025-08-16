# Buffkit Cleanup Tasks

## High Priority
- [ ] Fix import issues - auth.r{} renderer struct conflicts with package imports
- [ ] Add proper Buffalo renderer integration instead of stub renderers
- [ ] Implement actual template embedding and loading from embed.FS
- [ ] Fix the formatInt function in secure/middleware.go (currently broken)
- [ ] Implement proper CSRF token generation using crypto/rand
- [ ] Add proper error handling in all middleware functions

## Medium Priority
- [ ] Replace simple hash functions with crypto/sha256 for content hashing
- [ ] Implement proper rate limiting with Redis backend
- [ ] Add connection pooling for database connections
- [ ] Implement proper context handling in all async operations
- [ ] Add graceful shutdown for SSE broker
- [ ] Implement actual HTML parsing for component expansion (currently stubbed)

## Low Priority
- [ ] Add comprehensive logging with levels
- [ ] Implement metrics collection for monitoring
- [ ] Add request ID tracking through middleware chain
- [ ] Create proper test fixtures for harness
- [ ] Add benchmark tests for critical paths
- [ ] Document all public APIs with godoc comments

## Technical Debt
- [ ] Remove duplicate renderer structs (r{}) across packages
- [ ] Consolidate error types into single errors package
- [ ] Standardize configuration loading across all packages
- [ ] Add validation for all user inputs
- [ ] Implement proper session storage backend
- [ ] Add migration versioning and rollback support

## Missing Features from Plan
- [ ] Implement actual migration runner logic
- [ ] Add grift tasks for migrations and import maps
- [ ] Create embedded default templates
- [ ] Implement template shadowing mechanism
- [ ] Add proper SSE client JavaScript
- [ ] Implement job retry logic with backoff
- [ ] Add email template support
- [ ] Create more default components

## Testing Gaps
- [ ] Unit tests for all packages
- [ ] Integration tests for middleware chain
- [ ] SSE connection tests
- [ ] Component rendering tests
- [ ] Auth flow tests
- [ ] Job processing tests
- [ ] Email sending tests

## Documentation Needs
- [ ] API documentation for each package
- [ ] Migration guide from Rails/Loco
- [ ] Component authoring guide
- [ ] Deployment guide
- [ ] Performance tuning guide
- [ ] Security best practices