# Next Steps for Buffkit Development

## Current Status
- **Scenarios Passing**: 18 out of 37 (48.6%)
- **Test Coverage**: Growing from 24% baseline
- **Last Implemented**: Development mail sender (logs and preview)

## Immediate Next Step: Implement HTML Email Storage

### Target Scenario: "Development mail sender stores email content"
Location: `features/development_mode.feature:31`

### Why This Is Next
1. Builds directly on the mail sender we just implemented
2. Small, focused change (add HTML handling)
3. Already have the infrastructure in place
4. Natural progression from text-only emails

### Implementation Checklist

#### Step 1: Implement HTML Email Sending (5 min)
- [ ] Update `iSendAnHTMLEmailWithContent()` in `features/steps_test.go`
- [ ] Create mail.Message with HTML content
- [ ] Send via ts.kit.Mail

#### Step 2: Verify HTML Storage (5 min)
- [ ] Implement `theEmailShouldBeStoredWithHTMLContent()`
- [ ] Cast to DevSender and check GetMessages()
- [ ] Verify HTML field is populated

#### Step 3: Preview Rendering Check (5 min)
- [ ] Implement `iShouldBeAbleToPreviewTheRenderedHTML()`
- [ ] Visit /__mail/preview endpoint
- [ ] Check response contains the HTML content

#### Step 4: Dual Format Verification (5 min)
- [ ] Implement `theEmailShouldIncludeBothHTMLAndTextVersions()`
- [ ] Verify message has both Text and HTML fields
- [ ] Confirm preview shows both versions

### Test Command
```bash
cd buffkit
go test -v ./features -run "TestFeatures/Development_mail_sender_stores_email_content"
```

### Expected Outcome
- Scenario passes with all 4 steps green
- 19/37 scenarios passing (51.4%)
- HTML emails visible in preview interface

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
- [ ] HTML email storage
- [ ] Production mode safety
- [ ] Authentication flows
- [ ] Security headers
- [ ] SSE client management
- [ ] Component expansion

## Success Metrics
- **Today**: Reach 20/37 scenarios (54%)
- **This Week**: Reach 25/37 scenarios (67%)
- **Complete**: All 37 scenarios passing (100%)

## Remember
- Each scenario is a small, testable unit
- Build on what's already working
- Test after each step implementation
- Document any surprises in learnings.md