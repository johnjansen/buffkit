# Buffkit Implementation Plan

## Phase 1: Core Structure & Wire Function
1. Create main `buffkit.go` with Wire() function and Config struct
2. Define Kit struct to hold references to all subsystems
3. Implement basic error handling and initialization

## Phase 2: Package Stubs (Minimal Working Implementation)
### 2.1 SSR Package (`ssr/`)
- [ ] Create Broker struct with channels for clients
- [ ] Implement SSE endpoint handler
- [ ] Add heartbeat ticker (25s default)
- [ ] Create RenderPartial helper
- [ ] Stub SSE client JavaScript

### 2.2 Auth Package (`auth/`)
- [ ] Define User struct and UserStore interface
- [ ] Create default SQL store implementation
- [ ] Implement RequireLogin middleware
- [ ] Add login/logout handlers
- [ ] Create login template

### 2.3 Jobs Package (`jobs/`)
- [ ] Wrap Asynq client/server/mux
- [ ] Create Runtime struct
- [ ] Add basic enqueue helpers
- [ ] Implement worker task

### 2.4 Mail Package (`mail/`)
- [ ] Define Message struct and Sender interface
- [ ] Implement SMTPSender (stub)
- [ ] Add dev preview handler
- [ ] Create basic mail templates

### 2.5 Import Maps Package (`importmap/`)
- [ ] Create importmap.json structure
- [ ] Implement pin/unpin tasks
- [ ] Add print functionality
- [ ] Generate script tags

### 2.6 Secure Package (`secure/`)
- [ ] Add security headers middleware
- [ ] Integrate CSRF protection
- [ ] Set secure defaults

### 2.7 Components Package (`components/`)
- [ ] Create component registry
- [ ] Implement expansion middleware
- [ ] Add basic components (button, card, dropdown)
- [ ] Handle slots parsing

## Phase 3: Database & Migrations
- [ ] Create migration runner
- [ ] Add migration tracking table
- [ ] Implement up/down/status commands
- [ ] Create initial migration files

## Phase 4: Harness Application
- [ ] Create `harness/` directory with minimal Buffalo app
- [ ] Wire in Buffkit
- [ ] Add example routes showing each feature
- [ ] Create templates demonstrating components
- [ ] Add Makefile for easy testing

## Phase 5: Testing & Documentation
- [ ] Unit tests for critical paths
- [ ] Integration test for harness
- [ ] Update README with actual examples
- [ ] Add inline documentation

## Current Focus: Phase 1 & 2 Stubs
Start with minimal implementations that compile and can be wired together, even if they don't fully function yet.

## Success Criteria
- [ ] `go build` succeeds
- [ ] Harness app starts without errors
- [ ] Can navigate to login page
- [ ] SSE endpoint responds (even if empty)
- [ ] Components render (even if not expanded)
- [ ] Import map script tag appears in HTML

## Architecture Decisions
- Use embedded files for default templates
- Keep interfaces minimal and clear
- Avoid external dependencies where possible
- Make everything overridable/shadowable
- Focus on composability over configuration