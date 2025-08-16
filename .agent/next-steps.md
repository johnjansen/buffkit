# Next Steps for Buffkit Development

## Current Status
- **Scenarios Passing**: 27 out of 37 (73.0%)
- **Test Coverage**: Growing from 24% baseline
- **Last Implemented**: SSE client management scenarios (broadcasting, connections, cleanup, HTML fragments)

## Immediate Next Step: Implement Remaining Pending Scenarios

### Target Scenarios: Development Mode Features
Location: `features/development_mode.feature`

### Why This Is Next
1. Only 4 pending scenarios remain (plus 8 undefined)
2. Development mode features are partially implemented
3. Most infrastructure already exists
4. Quick wins to reach 80%+ completion

### Pending Scenarios to Complete

#### 1. "Security headers are relaxed in dev mode" (10 min)
- Implement `theApplicationIsRunningInDevelopmentMode()`
- Implement `iMakeARequestToAnyEndpoint()`
- Implement `theSecurityHeadersShouldBePresentButRelaxed()`

#### 2. "Error messages are verbose in dev mode" (10 min)
- Implement `anErrorOccursDuringRequestProcessing()`
- Implement `iShouldSeeDetailedErrorMessages()`
- Implement `stackTracesShouldBeIncluded()`

#### 3. "Hot reloading compatibility" (15 min)
- Implement `iMakeChangesToTemplatesOrAssets()`
- Implement `theChangesShouldBeReflectedWithoutRestart()`
- Implement `theImportMapsShouldSupportDevelopmentWorkflows()`

#### 4. "Development diagnostics" (15 min)
- Implement `iAccessDiagnosticEndpoints()`
- Implement `iShouldSeeInformationAbout()` with table parsing

### Test Command
```bash
cd buffkit
go test -v ./features -run "TestFeatures/Security_headers" 
```

### Expected Outcome
- 4+ more scenarios passing
- 31/37 scenarios passing (83.8%)
- Development mode features complete

## Following Scenarios (Priority Order)

### 1. "Development vs production mail behavior"
- **Why**: Validates environment-specific behavior
- **Effort**: Medium (20 min)
- **Impact**: Ensures proper mail handling

### 2. "Development-only middleware"
- **Why**: Validates middleware stack differences
- **Effort**: Low (15 min)
- **Impact**: Documents dev mode features

### 3. "Event filtering and targeting"
- **Why**: Advanced SSE feature
- **Effort**: Medium (25 min)
- **Impact**: Enables targeted updates

### 4. "SSE with htmx integration"
- **Why**: Key integration point
- **Effort**: Medium (30 min)
- **Impact**: Validates real-world usage

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
- [x] SSE broadcasting to all clients
- [x] SSE client connection management
- [x] SSE connection cleanup on disconnect
- [x] SSE broadcasting HTML fragments
- [ ] Security headers (next target)
- [ ] Development diagnostics
- [ ] Component expansion

## Success Metrics
- **Completed**: 27/37 scenarios (73.0%) ✓✓
- **Today**: Reach 31/37 scenarios (83%)
- **This Week**: Reach 35/37 scenarios (94%)
- **Complete**: All 37 scenarios passing (100%)

## Remember
- Each scenario is a small, testable unit
- Build on what's already working
- Test after each step implementation
- Document any surprises in learnings.md