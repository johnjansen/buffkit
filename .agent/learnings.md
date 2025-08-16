# Buffkit Implementation Learnings

## Architecture Insights

### 1. Package Organization
- Keeping packages focused and single-purpose makes the system more composable
- Each package should expose minimal interfaces to reduce coupling
- The registry pattern works well for extensible components

### 2. Buffalo Integration
- Buffalo's middleware chain is powerful but needs careful ordering
- Response wrapping for component expansion must happen early in the chain
- Session handling needs to be initialized before auth middleware

### 3. Interface Design
- Small, focused interfaces (like `Sender` for mail) make testing easier
- Providing both interface and default implementation gives flexibility
- Global singletons (like stores) need careful initialization order

## Technical Discoveries

### 1. SSE Implementation
- Heartbeats are critical to keep connections alive through proxies
- Non-blocking channel sends prevent slow clients from blocking the broker
- Need proper cleanup on client disconnect to avoid goroutine leaks

### 2. Component System
- Server-side component expansion needs HTML parsing, not just string replacement
- Slot system requires careful extraction of nested content
- Component middleware must only process HTML responses

### 3. Import Maps
- Browser support is good but needs polyfill for older browsers
- CDN-first approach works well but needs fallback for offline
- Content hashing for vendored files prevents cache issues

## Go-Specific Patterns

### 1. Embedding
- `embed.FS` is perfect for default templates and assets
- Shadowing requires checking app files before embedded files
- Need to handle both development (file system) and production (embedded) modes

### 2. Context Usage
- Buffalo's Context wraps standard context.Context
- Need careful type assertions when retrieving values from context
- Context should flow through all operations for proper cancellation

### 3. Error Handling
- Sentinel errors (like `ErrUserNotFound`) improve error checking
- Wrapping errors with context helps debugging
- Middleware errors need special handling to not break the chain

## Challenges Encountered

### 1. Circular Dependencies
- Auth package referencing mail for welcome emails creates cycles
- Solution: Use interfaces and dependency injection
- Event-driven approach (jobs) decouples packages nicely

### 2. Database Abstraction
- `database/sql` is limiting for complex queries
- Need query builder without full ORM overhead
- Migration system needs dialect-specific SQL handling

### 3. Testing Complexity
- Mocking Buffalo context is non-trivial
- Need test helpers for common scenarios
- Integration tests require full app setup

## Design Decisions

### 1. No ORM by Default
- Keeps the framework lightweight
- Allows users to choose their preferred solution
- SQL knowledge requirement might limit adoption

### 2. SSR-First Approach
- Aligns with modern trends (HTMX, LiveView, etc.)
- Simplifies deployment (no separate API/frontend)
- May need education for SPA-minded developers

### 3. Battery Included Philosophy
- Reduces decision fatigue for developers
- Ensures consistent patterns across apps
- Risk of bloat if not carefully managed

## Performance Considerations

### 1. Component Rendering
- HTML parsing on every request has overhead
- Consider caching expanded components
- Maybe pre-process at build time for production

### 2. SSE Scalability
- In-memory broker doesn't scale horizontally
- Need Redis pub/sub for multi-instance deployments
- Consider using dedicated SSE service

### 3. Job Processing
- Asynq is solid but Redis-only
- Consider supporting other backends (Postgres, SQS)
- Need careful timeout and retry configuration

## Security Observations

### 1. CSRF Protection
- Must be carefully integrated with AJAX requests
- Token rotation strategy needs consideration
- SameSite cookies provide additional protection

### 2. Session Management
- Cookie-based sessions have size limits
- Need secure session store for production
- Consider JWT for stateless alternative

### 3. Content Security Policy
- Import maps require careful CSP configuration
- Inline scripts need nonces or unsafe-inline
- Development vs production CSP needs differ

## Future Improvements

### 1. Developer Experience
- Need better error messages
- Hot reload for templates
- Browser dev tools integration

### 2. Modularity
- Make each package truly optional
- Plugin system for third-party packages
- Configuration presets for common use cases

### 3. Production Readiness
- Add observability (metrics, tracing)
- Implement health checks
- Add deployment templates (Docker, K8s)

## Comparison with Rails

### What Works Well
- Convention over configuration philosophy
- Integrated stack reduces complexity
- Familiar patterns for Rails developers

### What's Different
- Type safety catches errors at compile time
- No ActiveRecord magic (explicit is better)
- Deployment is simpler (single binary)

### What's Missing
- Rich ecosystem of gems
- Mature admin interfaces
- Advanced ORM features

## Key Takeaways

1. **Start simple, iterate** - The stub approach lets us test integration early
2. **Interfaces over implementations** - Allows flexibility without breaking changes
3. **Documentation as code** - Embed docs and examples in the codebase
4. **Test the harness** - Having a working example validates the design
5. **Performance last** - Get it working, then optimize based on real usage

## CI/CD and Project Health

### Badge Implementation
- GitHub Actions status badges provide immediate build feedback
- Codecov integration requires token setup but works without for public repos
- Go Report Card automatically analyzes public repos
- Multiple badge services create comprehensive project health overview

### Testing Evolution
- Moved from unit tests to BDD with Godog
- Feature files serve as living documentation
- Step definitions can be reused across scenarios
- Test suite runs in <1 second maintaining fast feedback

### Linting Insights
- golangci-lint catches deprecated APIs and style issues
- BeforeScenario deprecated in favor of Before with context
- Error checking must be explicit (no ignored returns)
- Boolean comparisons should be simplified

### Documentation Structure
- README badges should be centered and organized by category
- CONTRIBUTING.md essential for open source projects
- SECURITY.md establishes trust and reporting process
- Feature files double as user documentation

### Coverage Strategy
- Run tests with -coverprofile for coverage data
- Upload to Codecov for tracking over time
- Set reasonable targets (70% project, 80% patch)
- Exclude test files and generated code from coverage

## BDD Testing Implementation

### Mail Preview Feature
- PreviewHandler was already implemented but had redundant environment check
- Handler is conditionally mounted based on DevMode flag during wiring
- Global mail sender must be set during Wire() for mail.Send() to work
- Test environment differs from development environment in Buffalo app setup

### Test Step Implementation Patterns
- Missing step definitions cause "pending" status in scenarios
- Step functions can be simple wrappers (e.g., checking DevMode flag)
- HTTP testing uses httptest.NewRecorder() for response capture
- Response body inspection validates feature functionality

### Progressive Scenario Implementation
- Started with 15/37 scenarios passing (24% coverage)
- Implemented mail preview: 17/37 scenarios passing
- Implemented mail logging: 18/37 scenarios passing
- Each scenario builds on existing infrastructure

### Key Testing Discoveries
- Step registration must match exact Gherkin text including "And/Given/When/Then"
- Buffalo app environment ("test" vs "development") affects behavior
- DevSender automatically stores messages for preview functionality
- Test isolation requires resetting suite state between scenarios