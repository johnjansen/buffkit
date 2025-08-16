# Next Steps for Buffkit Development

## Current Status
- **Scenarios Passing**: 19 out of 37 (51.4%)
- **Test Coverage**: Growing from 24% baseline
- **Last Implemented**: HTML email storage and preview

## Immediate Next Step: Implement Authentication Scenarios

### Target Scenario: "Protected routes require authentication"
Location: `features/authentication.feature:24`

### Why This Is Next
1. Core security feature of any web application
2. Already have auth package with RequireLogin middleware
3. Unlocks multiple related scenarios
4. High value for framework users

### Implementation Checklist

#### Step 1: Implement Protected Handler (10 min)
- [ ] Implement `iHaveAHandlerThatRequiresLogin()` in `features/steps_test.go`
- [ ] Create a test handler that uses auth.RequireLogin middleware
- [ ] Store handler reference in TestSuite

#### Step 2: Access Without Auth (5 min)
- [ ] Implement `iAccessTheProtectedRouteWithoutAuthentication()`
- [ ] Make request to protected route without session
- [ ] Store response for verification

#### Step 3: Verify Redirect (5 min)
- [ ] Implement `iShouldBeRedirectedToLogin()`
- [ ] Check response status is 302 or 303
- [ ] Verify Location header points to login path

### Test Command
```bash
cd buffkit
go test -v ./features -run "TestFeatures/Protected_routes_require_authentication"
```

### Expected Outcome
- Scenario passes with all 3 steps green
- 20/37 scenarios passing (54.1%)
- Authentication flow validated

## Following Scenarios (Priority Order)

### 1. "Mail preview endpoint is not available in production"
- **Why**: Complement to dev mode test, security validation
- **Effort**: Very Low (10 min)
- **Impact**: Validates production safety

### 2. "Protected routes require authentication"
- **Why**: Core security feature
- **Effort**: Medium (30 min)
- **Impact**: Unlocks auth-related scenarios

### 3. "RequireLogin middleware exists"
- **Why**: Validates middleware API
- **Effort**: Low (15 min)
- **Impact**: Documents middleware usage

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
- [x] Production mode safety (already working)
- [ ] Authentication flows (next target)
- [ ] Security headers
- [ ] SSE client management
- [ ] Component expansion

## Success Metrics
- **Completed**: 19/37 scenarios (51.4%) ✓
- **Today**: Reach 22/37 scenarios (59%)
- **This Week**: Reach 28/37 scenarios (75%)
- **Complete**: All 37 scenarios passing (100%)

## Remember
- Each scenario is a small, testable unit
- Build on what's already working
- Test after each step implementation
- Document any surprises in learnings.md