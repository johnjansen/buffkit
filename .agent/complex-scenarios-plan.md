# Complex Scenarios Implementation Plan

## Overview
We have 6 remaining undefined scenarios that require more sophisticated test infrastructure. This document breaks down each scenario and provides a step-by-step implementation approach.

## Priority Order (Easiest to Hardest)

### 1. Development-only Middleware (Difficulty: Medium)
**Location**: `features/development_mode.feature:78`
**Current State**: Generates stub functions

#### Why It's Complex
- Requires inspecting Buffalo's middleware stack
- Need to differentiate dev vs prod middleware
- Middleware order matters

#### Implementation Approach
```go
// Step 1: Create middleware tracking
type middlewareTracker struct {
    names []string
    devOnly []string
}

// Step 2: Hook into Buffalo's middleware registration
// - Intercept app.Use() calls
// - Track middleware names
// - Identify dev-only middleware (logging, debug, etc.)

// Step 3: Implement test steps
func theApplicationIsWiredWithDevelopmentMode() error {
    // Wire app with DevMode: true
    // Track middleware during wiring
}

func iInspectTheMiddlewareStack() error {
    // Get middleware list from app
    // Store in TestSuite
}

func developmentSpecificMiddlewareShouldBePresent() error {
    // Check for debug middleware
    // Check for verbose logging
    // Check for relaxed CORS
}
```

#### Test Data Needed
- List of expected dev middleware names
- List of production middleware to exclude
- Middleware execution order

---

### 2. Development vs Production Mail Behavior (Difficulty: Medium)
**Location**: `features/development_mode.feature:59`
**Current State**: Generates stub functions

#### Why It's Complex
- Requires switching environments mid-test
- Need to verify different mail senders
- Must track mail delivery vs logging

#### Implementation Approach
```go
// Step 1: Create environment switcher
func switchEnvironment(devMode bool) {
    // Re-wire app with new DevMode setting
    // Swap mail sender (SMTP vs Dev)
}

// Step 2: Track mail behavior
type mailTracker struct {
    sent []Message
    logged []Message
    previewed []Message
}

// Step 3: Implement comparison steps
func iHaveTheSameBuffkitConfiguration() error {
    // Store base config
    // Prepare for env switching
}

func devModeIsTrueAndISendAnEmail() error {
    // Set DevMode: true
    // Send test email
    // Track in mailTracker.logged
}

func theEmailShouldBeCapturedForPreview() error {
    // Check DevSender has email
    // Verify preview endpoint shows it
}

func devModeIsFalseAndISendAnEmail() error {
    // Set DevMode: false
    // Send test email
    // Track in mailTracker.sent
}
```

#### Test Data Needed
- Test email templates
- SMTP mock server
- Preview endpoint responses

---

### 3. Development Diagnostics (Difficulty: Medium-Hard)
**Location**: `features/development_mode.feature:67`
**Current State**: Generates stub functions with table

#### Why It's Complex
- Requires creating diagnostic endpoints
- Need to gather runtime statistics
- Table-driven test with multiple components

#### Implementation Approach
```go
// Step 1: Create diagnostic collector
type diagnostics struct {
    SSEConnections int
    AuthUsers      int
    PendingJobs    int
    ImportMaps     []string
    Components     []string
}

// Step 2: Add diagnostic endpoints
func mountDiagnosticEndpoints(app *buffalo.App) {
    app.GET("/__diagnostics/sse", sseStatsHandler)
    app.GET("/__diagnostics/auth", authStatsHandler)
    app.GET("/__diagnostics/jobs", jobStatsHandler)
}

// Step 3: Implement table verification
func iAccessDiagnosticEndpoints() error {
    // Hit each diagnostic endpoint
    // Collect responses
}

func iShouldSeeInformationAbout(table *godog.Table) error {
    // Parse table rows
    // For each component, verify status
    // Match expected vs actual
}
```

#### Test Data Needed
- Expected diagnostic output format
- Component status definitions
- Table parsing logic

---

### 4. Hot Reloading Compatibility (Difficulty: Hard)
**Location**: `features/development_mode.feature:52`
**Current State**: Generates stub functions

#### Why It's Complex
- Requires file system monitoring
- Need to simulate file changes
- Must verify reload without restart

#### Implementation Approach
```go
// Step 1: Create file watcher simulator
type fileWatcher struct {
    watchedPaths []string
    changes      chan fileChange
}

// Step 2: Implement change detection
func iMakeChangesToTemplatesOrAssets() error {
    // Create temp file
    // Modify content
    // Trigger change event
}

func theChangesShouldBeReflectedWithoutRestart() error {
    // Request same endpoint
    // Verify new content served
    // Check app wasn't restarted (same instance)
}

// Step 3: Import map development workflow
func theImportMapsShouldSupportDevelopmentWorkflows() error {
    // Check for dev-friendly import maps
    // Verify no caching headers
    // Check source maps enabled
}
```

#### Test Data Needed
- Template files to modify
- Asset files to change
- Expected reload behavior

---

### 5. Event Filtering and Targeting (Difficulty: Hard)
**Location**: `features/server_sent_events.feature:40`
**Current State**: Generates stub functions

#### Why It's Complex
- Requires client interest/subscription model
- Need selective event routing
- Must track which clients receive what

#### Implementation Approach
```go
// Step 1: Extend Client with interests
type ClientWithInterests struct {
    *ssr.Client
    Interests []string // Topics this client subscribes to
}

// Step 2: Implement targeted broadcasting
func BroadcastToInterested(event Event, topic string) {
    // Filter clients by interest
    // Send only to matching clients
}

// Step 3: Test implementation
func iHaveMultipleClientsWithDifferentInterests() error {
    // Create clients with various interests
    // "user-updates", "system-alerts", "chat-messages"
}

func iBroadcastAnEventToSpecificClients() error {
    // Send event with topic
    // Track distribution
}

func onlyTargetedClientsShouldReceiveTheEvent() error {
    // Verify only interested clients got event
    // Check others didn't receive it
}
```

#### Test Data Needed
- Interest/topic definitions
- Client subscription model
- Event routing logic

---

### 6. SSE with htmx Integration (Difficulty: Very Hard)
**Location**: `features/server_sent_events.feature:46`
**Current State**: Generates stub functions

#### Why It's Complex
- Requires simulating htmx behavior
- Need DOM manipulation simulation
- Must verify automatic updates

#### Implementation Approach
```go
// Step 1: Create htmx simulator
type htmxPage struct {
    DOM        *html.Node
    SSESource  *sse.Client
    AutoSwap   map[string]string // event -> target mapping
}

// Step 2: Simulate htmx SSE handling
func iHaveAnHtmxEnabledPageConnectedToSSE() error {
    // Parse HTML with hx-sse attributes
    // Connect SSE client
    // Setup event listeners
}

func iBroadcastAnUpdateEvent() error {
    // Send SSE event with HTML fragment
    // Include htmx headers (HX-Trigger, etc.)
}

func thePageContentShouldUpdateAutomatically() error {
    // Verify DOM was modified
    // Check correct element was swapped
    // Confirm no page reload occurred
}
```

#### Test Data Needed
- HTML templates with htmx attributes
- SSE event formats for htmx
- DOM parsing/manipulation library

---

## Implementation Strategy

### Phase 1: Foundation (Week 1)
1. **Development-only Middleware** - Establish middleware inspection
2. **Development vs Production Mail** - Build environment switching

### Phase 2: Diagnostics (Week 2)
3. **Development Diagnostics** - Create diagnostic infrastructure
4. **Hot Reloading Compatibility** - Add file watching simulation

### Phase 3: Advanced SSE (Week 3)
5. **Event Filtering and Targeting** - Implement client interests
6. **SSE with htmx Integration** - Build htmx simulation

## Required Dependencies

### Testing Libraries
- `github.com/PuerkitoBio/goquery` - HTML parsing for htmx
- `github.com/fsnotify/fsnotify` - File system watching
- `github.com/stretchr/testify/mock` - Mocking SMTP

### Development Tools
- Mock SMTP server for mail testing
- File system fixtures for hot reload
- HTML templates for htmx testing

## Success Criteria

Each scenario should:
1. Have clear, isolated test steps
2. Not require external services
3. Run in under 100ms
4. Be deterministic (no flaky tests)
5. Document any limitations

## Alternative Approach

If these prove too complex for BDD testing, consider:
1. Moving to integration tests
2. Creating example applications
3. Manual testing documentation
4. Marking as "future enhancement"

## Next Step

Start with **Development-only Middleware** as it's the most straightforward and provides infrastructure for other scenarios.

## Implementation Checklist

- [ ] Development-only Middleware
  - [ ] Middleware tracking system
  - [ ] Dev vs prod comparison
  - [ ] Test step implementation
  
- [ ] Development vs Production Mail
  - [ ] Environment switcher
  - [ ] Mail behavior tracker
  - [ ] Dual-mode testing
  
- [ ] Development Diagnostics
  - [ ] Diagnostic endpoints
  - [ ] Stats collection
  - [ ] Table-driven tests
  
- [ ] Hot Reloading Compatibility
  - [ ] File watcher simulator
  - [ ] Change detection
  - [ ] Reload verification
  
- [ ] Event Filtering and Targeting
  - [ ] Client interests model
  - [ ] Targeted broadcasting
  - [ ] Selective delivery
  
- [ ] SSE with htmx Integration
  - [ ] htmx simulator
  - [ ] DOM manipulation
  - [ ] Auto-update verification