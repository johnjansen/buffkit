# Next Steps for Buffkit Development

## Current Status
- **Scenarios Passing**: 23 out of 37 (62.2%)
- **Test Coverage**: Growing from 24% baseline
- **Last Implemented**: Authentication scenarios (protected routes, middleware, user context)

## Immediate Next Step: Implement SSE Client Management

### Target Scenario: "Broadcasting events to all clients"
Location: `features/server_sent_events.feature:15`

### Why This Is Next
1. SSE broker is already implemented and working
2. Just need to implement test steps for multi-client scenarios
3. High value feature for real-time applications
4. Builds on existing SSE infrastructure

### Implementation Checklist

#### Step 1: Implement Multiple Client Connection (10 min)
- [ ] Implement `iHaveMultipleClientsConnectedToSSE()` in `features/steps_test.go`
- [ ] Create multiple SSE client connections
- [ ] Store client references in TestSuite

#### Step 2: Broadcast Event (5 min)
- [ ] Implement `iBroadcastAnEventWithData()`
- [ ] Use broker to send event to all clients
- [ ] Verify event is queued for delivery

#### Step 3: Verify Client Receipt (10 min)
- [ ] Implement `allConnectedClientsShouldReceiveTheEvent()`
- [ ] Check each client received the event
- [ ] Verify event type and data match

### Test Command
```bash
cd buffkit
go test -v ./features -run "TestFeatures/Broadcasting_events_to_all_clients"
```

### Expected Outcome
- Scenario passes with all 3 steps green
- 24/37 scenarios passing (64.9%)
- Multi-client SSE broadcasting validated

## Following Scenarios (Priority Order)

### 1. "Client connection management"
- **Why**: Validates SSE connection lifecycle
- **Effort**: Low (15 min)
- **Impact**: Ensures proper resource management

### 2. "Connection cleanup on disconnect"
- **Why**: Prevents memory leaks
- **Effort**: Low (15 min)
- **Impact**: Production stability

### 3. "Broadcasting HTML fragments"
- **Why**: Core SSR+SSE feature
- **Effort**: Medium (20 min)
- **Impact**: Enables HTMX-style updates

### 4. "Security headers are relaxed in dev mode"
- **Why**: Already have secure middleware
- **Effort**: Low (20 min)
- **Impact**: Validates dev/prod differences

## Quick Wins Available
These scenarios likely already work but need test implementation:
- "Error messages are verbose in dev mode"
- "Development diagnostics"
- "Development-only middleware"

## Progress Tracking
- [x] Mail preview in dev mode
- [x] Mail sender logs emails
- [x] HTML email storage
- [x] Production mode safety
- [x] Protected routes require authentication
- [x] RequireLogin middleware exists
- [x] Authenticated users can access protected routes
- [x] User context is available in protected routes
- [ ] SSE client management (next target)
- [ ] Security headers
- [ ] Component expansion

## Success Metrics
- **Completed**: 23/37 scenarios (62.2%) ✓
- **Today**: Reach 26/37 scenarios (70%)
- **This Week**: Reach 32/37 scenarios (86%)
- **Complete**: All 37 scenarios passing (100%)

## Remember
- Each scenario is a small, testable unit
- Build on what's already working
- Test after each step implementation
- Document any surprises in learnings.md