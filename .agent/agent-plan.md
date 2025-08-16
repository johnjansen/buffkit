# Buffkit Implementation Plan

## Current Sprint: Implement Simplest Pending BDD Scenario

### Target: "Mail preview endpoint is available in dev mode"

This is the simplest pending scenario because:
- Clear, single responsibility (show mail preview in dev mode)
- No complex dependencies (just needs basic routing and template)
- Testable in isolation
- Foundation for other mail features

### Implementation Steps

#### Step 1: Create Mail Preview Handler
**What**: Create a handler that serves the mail preview interface at `/__mail/preview`
**How**: 
1. Create `mail/preview_handler.go`
2. Define `PreviewHandler` function that returns `buffalo.Handler`
3. Check if DevMode is enabled, return 404 if not
4. For now, return a simple HTML template with mock data

**Why**: This establishes the endpoint and conditional dev-mode behavior

#### Step 2: Create Mail Preview Template
**What**: Create an embedded HTML template for the preview interface
**How**:
1. Create `mail/templates/preview.html`
2. Use Go's `embed` package to embed the template
3. Create a basic HTML page with:
   - Title: "Mail Preview"
   - Empty list placeholder for emails
   - Basic styling

**Why**: Provides visual feedback that the endpoint works

#### Step 3: Wire Preview Handler into Buffkit
**What**: Register the preview handler when DevMode is true
**How**:
1. In `buffkit.go`, add preview handler registration in `Wire()`
2. Only register if `config.DevMode == true`
3. Mount at `GET /__mail/preview`

**Why**: Integrates the feature into the main wiring flow

#### Step 4: Update Test Implementation
**What**: Implement the pending test steps
**How**:
1. Update `features/steps_test.go`
2. Implement `iVisit()` to make HTTP GET request
3. Implement `iShouldSeeTheMailPreviewInterface()` to check response body
4. Implement `theResponseStatusShouldBe()` to verify status code
5. Implement `iShouldSeeAListOfSentEmails()` to check for list element

**Why**: Makes the BDD scenario pass and validates our implementation

#### Step 5: Run and Verify
**What**: Execute the specific BDD scenario
**How**:
```bash
cd buffkit
go test -v ./features -run "TestFeatures/Mail_preview_endpoint_is_available_in_dev_mode"
```

**Why**: Confirms implementation meets requirements

### Success Criteria
- [x] Scenario "Mail preview endpoint is available in dev mode" passes
- [x] Endpoint returns 200 in dev mode
- [x] Endpoint returns 404 in production mode
- [x] Basic HTML interface is displayed

### Code Locations
- `mail/preview_handler.go` - Handler implementation
- `mail/templates/preview.html` - Embedded template
- `buffkit.go` - Wiring logic
- `features/steps_test.go` - Test step implementations

### Estimated Time: 30 minutes
- 10 min: Handler and template
- 10 min: Wiring and integration
- 10 min: Test implementation and verification

## Next Simplest Scenarios (Priority Order)
1. "Mail preview endpoint is not available in production" - Complement to current
2. "Development mail sender logs emails" - Build on preview foundation
3. "Development mail sender stores email content" - Extend mail storage

## Previous Implementation Plan (Archive)

### Phase 1: Core Structure & Wire Function ✅
- Created main `buffkit.go` with Wire() function and Config struct
- Defined Kit struct to hold references to all subsystems
- Implemented basic error handling and initialization

### Phase 2: Package Stubs (In Progress)
- SSR Package (`ssr/`) - Partially complete
- Auth Package (`auth/`) - Partially complete
- Jobs Package (`jobs/`) - Pending
- Mail Package (`mail/`) - Current focus
- Import Maps Package (`importmap/`) - Pending
- Secure Package (`secure/`) - Partially complete
- Components Package (`components/`) - Pending

### Architecture Decisions
- Use embedded files for default templates
- Keep interfaces minimal and clear
- Avoid external dependencies where possible
- Make everything overridable/shadowable
- Focus on composability over configuration