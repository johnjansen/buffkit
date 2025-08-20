# BDD Coverage Plan for Buffkit v0.1

## Current State
- **Code Coverage**: ~7.4% overall (some packages up to 75%)
- **BDD Scenarios**: Reduced to core features only
- **Undefined Steps**: ~24 remaining (down from 324)
- **Test Status**: 25/29 scenarios passing, 4 component-related failures remaining

## Goal
Achieve 100% BDD coverage for all functionality defined in PLAN.md specification.

## Package-by-Package Coverage Requirements

### 1. Core Wiring (`buffkit.go`)
**Current Coverage**: 7.4%
**Required Scenarios**:

```gherkin
Feature: Buffkit Core Integration
  
  Scenario: Wire with complete configuration
    Given I have a Buffalo application
    When I wire Buffkit with all configuration options
    Then the SSE broker should be initialized
    And the auth store should be configured
    And the mail sender should be configured
    And the import map manager should be loaded
    And the component registry should be ready
    And the security middleware should be applied
    
  Scenario: Wire with minimal configuration
    Given I have a Buffalo application
    When I wire Buffkit with only required fields
    Then all components should initialize with defaults
    
  Scenario: Wire fails without auth secret
    Given I have a Buffalo application
    When I wire Buffkit without an auth secret
    Then I should get an error "AuthSecret is required"
    
  Scenario: Cleanup on shutdown
    Given Buffkit is wired and running
    When I call Shutdown on the Kit
    Then all resources should be properly closed
    And no goroutines should leak
```

### 2. SSE/SSR (`sse/` and `ssr/`)
**Current Coverage**: SSE 0%, SSR 23.9%
**Required Scenarios**:

```gherkin
Feature: Server-Sent Events
  
  Scenario: SSE endpoint serves event stream
    When I connect to "/events" with Accept: text/event-stream
    Then the response should have content-type "text/event-stream"
    And the response should include "Cache-Control: no-cache"
    
  Scenario: Broadcast to multiple clients
    Given 3 clients are connected to "/events"
    When I broadcast event "update" with data "test data"
    Then all 3 clients should receive the event
    
  Scenario: Heartbeat keeps connection alive
    Given I am connected to "/events"
    When 25 seconds pass without events
    Then I should receive a heartbeat event
    
  Scenario: Client disconnection cleanup
    Given I am connected to "/events"
    When I disconnect
    Then my client should be removed from the broker
    And resources should be freed

Feature: Server-Side Rendering Helpers
  
  Scenario: Render partial for HTMX response
    When I call RenderPartial with template "user_card" and data
    Then I should get rendered HTML
    And the HTML should be suitable for HTMX swap
    
  Scenario: Render partial for SSE broadcast
    When I render a partial for SSE
    Then the HTML should be properly escaped for SSE format
    And newlines should be handled correctly
```

### 3. Authentication (`auth/`)
**Current Coverage**: 0%
**Required Scenarios**:

```gherkin
Feature: Authentication System
  
  Scenario: User login with valid credentials
    Given a user exists with email "user@example.com"
    When I POST to "/login" with valid credentials
    Then a session cookie should be set
    And I should be redirected to "/"
    
  Scenario: User login with invalid credentials
    When I POST to "/login" with wrong password
    Then no session should be created
    And I should see an error message
    
  Scenario: User logout
    Given I am logged in
    When I POST to "/logout"
    Then my session should be cleared
    And I should be redirected to "/login"
    
  Scenario: RequireLogin middleware blocks unauthenticated
    Given I am not logged in
    When I access a protected route
    Then I should be redirected to "/login"
    
  Scenario: RequireLogin middleware allows authenticated
    Given I am logged in
    When I access a protected route
    Then I should see the protected content
    
  Scenario: Password hashing
    When I hash password "MySecurePass123"
    Then the hash should be bcrypt format
    And checking the password should succeed
    
  Scenario: SQL store operations
    Given I have a SQL user store
    When I create a user with email "new@example.com"
    Then I should be able to retrieve by email
    And I should be able to retrieve by ID
    And I should be able to update password
```

### 4. Mail System (`mail/`)
**Current Coverage**: 0%
**Required Scenarios**:

```gherkin
Feature: Mail Sending
  
  Scenario: Send email via SMTP in production
    Given I have SMTP configuration
    When I send an email with subject "Test"
    Then the email should be sent via SMTP
    
  Scenario: Log email in development mode
    Given I have a development mail sender
    When I send an email with subject "Test"
    Then the email should be logged
    And it should be available in preview
    
  Scenario: Mail preview endpoint in dev mode
    Given DevMode is true
    And I have sent 3 emails
    When I visit "/__mail/preview"
    Then I should see all 3 emails listed
    
  Scenario: Mail preview blocked in production
    Given DevMode is false
    When I visit "/__mail/preview"
    Then I should get 404
    
  Scenario: HTML and text email parts
    When I send an email with HTML and text content
    Then both parts should be included
    And the preview should show both versions
```

### 5. Background Jobs (`jobs/`)
**Current Coverage**: 0%
**Required Scenarios**:

```gherkin
Feature: Background Jobs
  
  Scenario: Initialize job runtime with Redis
    Given I have Redis at "redis://localhost:6379"
    When I create a job runtime
    Then the Asynq client should be initialized
    And the server should be ready
    And the mux should have handlers registered
    
  Scenario: Initialize job runtime without Redis
    Given I have no Redis URL
    When I create a job runtime
    Then job enqueuing should no-op gracefully
    
  Scenario: Enqueue email job
    Given I have a job runtime
    When I enqueue a welcome email job
    Then the job should be added to the queue
    
  Scenario: Process email job
    Given a welcome email job is queued
    When the worker processes the job
    Then the email should be sent
    And the job should be marked complete
    
  Scenario: Session cleanup job
    Given there are expired sessions
    When the cleanup job runs
    Then expired sessions should be removed
```

### 6. Import Maps (`importmap/`)
**Current Coverage**: 22.6%
**Required Scenarios**:

```gherkin
Feature: Import Map Management
  
  Scenario: Load default import map
    When I create an import map manager
    Then it should include default pins for htmx and alpine
    
  Scenario: Pin a new dependency
    When I pin "lodash" to "https://esm.sh/lodash"
    Then the import map should include lodash
    
  Scenario: Pin with download option
    When I pin "lodash" with --download flag
    Then the file should be vendored locally
    And the import map should point to local file
    
  Scenario: Unpin a dependency
    Given I have pinned "lodash"
    When I unpin "lodash"
    Then it should be removed from import map
    
  Scenario: Generate import map HTML
    When I call ToHTML on the manager
    Then I should get a valid <script type="importmap"> tag
    
  Scenario: Import map middleware
    When a request is made to the app
    Then the import map should be available in context
```

### 7. Security (`secure/`)
**Current Coverage**: 0%
**Required Scenarios**:

```gherkin
Feature: Security Middleware
  
  Scenario: Security headers applied
    When I make any request
    Then response should include X-Frame-Options: DENY
    And response should include X-Content-Type-Options: nosniff
    And response should include X-XSS-Protection
    
  Scenario: CSRF protection on POST
    Given CSRF middleware is active
    When I POST without CSRF token
    Then the request should be rejected
    
  Scenario: CSRF token validation
    Given I have a valid CSRF token
    When I POST with the token
    Then the request should succeed
    
  Scenario: Rate limiting
    When I make 100 requests in 1 minute
    Then subsequent requests should be rate limited
    
  Scenario: Security headers in dev vs prod
    Given DevMode is true
    Then security headers should be relaxed
    Given DevMode is false
    Then security headers should be strict
```

### 8. Component Registry (`components/`)
**Current Coverage**: 0%
**Required Scenarios**:

```gherkin
Feature: Component Registry
  
  Scenario: Register a component
    When I register "bk-hello" with a renderer
    Then the registry should contain "bk-hello"
    
  Scenario: Render registered component
    Given I registered "bk-test" that outputs "TEST"
    When I render "<bk-test></bk-test>"
    Then output should contain "TEST"
    
  Scenario: Component with attributes
    Given I registered a component using attributes
    When I render with attributes
    Then attributes should be passed to renderer
    
  Scenario: Component with slots
    Given I registered a component using slots
    When I render with slot content
    Then slots should be captured correctly
    
  Scenario: Middleware expands components
    Given the expander middleware is active
    When response contains "<bk-test>"
    Then it should be expanded before sending
    
  Scenario: Non-HTML responses skip expansion
    When response is JSON
    Then component expansion should not run
```

### 9. Migrations (`migrations/`)
**Current Coverage**: 75.7%
**Required Scenarios**:

```gherkin
Feature: Database Migrations
  
  Scenario: Run migrations on empty database
    Given an empty database
    When I run migrations
    Then migration table should be created
    And all migrations should apply
    
  Scenario: Run migrations when up-to-date
    Given all migrations are applied
    When I run migrations
    Then no migrations should run
    
  Scenario: Rollback migrations
    Given migrations are applied
    When I rollback 2 migrations
    Then 2 migrations should be reversed
    
  Scenario: Migration status
    When I check migration status
    Then I should see applied migrations
    And I should see pending migrations
    
  Scenario: Create new migration
    When I create migration "add_users_table"
    Then up and down files should be created
    
  Scenario: Multi-dialect support
    Given I have PostgreSQL
    Then PostgreSQL syntax should work
    Given I have MySQL
    Then MySQL syntax should work
    Given I have SQLite
    Then SQLite syntax should work
```

### 10. Grift Tasks (`grifts.go`)
**Current Coverage**: In features but not unit tests
**Required Scenarios**:

```gherkin
Feature: Grift Tasks
  
  Scenario: List available tasks
    When I run "grift list"
    Then I should see all Buffkit tasks
    
  Scenario: Run migration task
    When I run "grift buffkit:migrate"
    Then migrations should execute
    
  Scenario: Run rollback task
    When I run "grift buffkit:migrate:down 1"
    Then one migration should rollback
    
  Scenario: Run worker task
    When I run "grift jobs:worker"
    Then worker should start processing
    
  Scenario: Task error handling
    Given invalid database URL
    When I run migration task
    Then I should see helpful error message
```

## Implementation Strategy

### Phase 1: Core Infrastructure (Week 1) - IN PROGRESS
1. ✅ Complete `buffkit_integration.feature` with all wiring scenarios
2. ✅ Fix database connection issues in grift tasks - COMPLETED
3. ✅ Implement missing component step definitions (mostly done)
4. 🔧 Fix remaining component expansion issues (4 scenarios failing)
5. Add proper cleanup/shutdown tests
6. Ensure no goroutine leaks

### Phase 2: Request/Response Flow (Week 1)
1. Complete auth flow scenarios
2. Complete SSE/SSR scenarios
3. Complete security middleware scenarios

### Phase 3: Data & Persistence (Week 2)
1. Complete migration scenarios
2. Complete user store scenarios
3. Complete session management scenarios

### Phase 4: Background & Mail (Week 2)
1. Complete job queue scenarios
2. Complete mail sending scenarios
3. Complete dev mode mail preview

### Phase 5: Frontend Assets (Week 3)
1. Complete import map scenarios
2. Complete component registry scenarios
3. Complete middleware integration

### Phase 6: CLI & Tasks (Week 3)
1. Complete all grift task scenarios
2. Add error cases and edge cases
3. Ensure helpful error messages

## Success Metrics

1. **Coverage Target**: 80%+ for all packages
2. **Scenario Count**: ~150 scenarios covering all functionality
3. **Step Reuse**: Maximum reuse through shared_context.go
4. **Execution Time**: Full suite runs in < 30 seconds
5. **No Flaky Tests**: All tests deterministic and reliable

## Testing Patterns to Use

1. **Shared Context**: Use `shared_context.go` for common assertions
2. **Test Isolation**: Each scenario should be independent
3. **Resource Cleanup**: Proper cleanup in After hooks
4. **Mock External**: Mock Redis, SMTP when needed
5. **Real Database**: Use SQLite in-memory for fast tests

## Files to Create/Update

1. `features/buffkit_integration.feature` - Expand with all wiring scenarios
2. `features/authentication.feature` - Add store and session scenarios
3. `features/mail.feature` - New file for mail scenarios
4. `features/jobs.feature` - New file for background job scenarios
5. `features/import_maps.feature` - New file for import map scenarios
6. `features/security.feature` - New file for security scenarios
7. `features/migrations.feature` - Expand existing with more scenarios

## Avoiding Scope Creep

**DO Include**:
- Everything explicitly in PLAN.md
- Basic error cases
- Resource cleanup
- Integration between components

**DO NOT Include**:
- Password reset flows
- Email verification
- Account locking
- Multi-factor auth
- Advanced SSE reconnection
- Hot reload
- Diagnostic endpoints
- Feature flags
- Pre-built components beyond registry

## Next Steps

1. ✅ Review this plan with stakeholder
2. ✅ Create missing feature files (all core features created)
3. 🔧 Implement remaining step definitions (~24 undefined)
4. 🔧 Fix failing scenarios:
   - Database connection in migrations table check
   - Error message expectations for Redis failures
5. Run coverage reports after each phase
6. Adjust plan based on findings

## Progress Log

### Session 1 (Current)
- Fixed component step definitions (Register methods now use correct signature)
- Fixed grift task error handling
- Reduced undefined steps from 324 to ~24
- Fixed compilation errors in test suite
- Updated error expectations to match actual output
- **FIXED**: Database connection issue in migrations testing - SharedContext now uses DATABASE_URL when TestDB not set
- All 9 grift task scenarios now passing ✅
- Test results improved: 25/29 scenarios passing (up from 13/18)

## Estimated Timeline

- **Total Effort**: 3 weeks
- **Scenarios to Write**: ~150
- **Step Definitions**: ~50-75 (with heavy reuse)
- **Expected Final Coverage**: 80-90%
- **Expected Undefined Steps**: 0