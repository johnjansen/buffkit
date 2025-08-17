# CRITICAL ISSUE: TestAllFeatures Hanging

## Issue Summary
**Status:** 🔴 BLOCKING v0.1-alpha release  
**Impact:** Cannot run full test suite, blocking verification of ~80 remaining undefined steps  
**First Detected:** During BDD consolidation work  
**Last Verified:** Current (after goroutine leak fixes)  

## Problem Description
The `TestAllFeatures` test suite hangs indefinitely when run, requiring timeout termination. Individual test suites work correctly when run in isolation.

### Symptoms
- `go test ./features -run TestAllFeatures` hangs after starting first scenario
- Test output shows it starts "Accessing login form" scenario then freezes
- Timeout reveals goroutines still running (SSR broker despite shutdown attempts)
- Individual tests pass: `TestBasicFeatures`, `TestGriftTasks` work fine
- Only affects the combined test runner that runs all features together

### Test Output Before Hang
```
=== RUN   TestAllFeatures
Feature: Authentication System
  As a web application user
  I want to authenticate with the system
  So that I can access protected resources
=== RUN   TestAllFeatures/Accessing_login_form

  Background:
    Given I have a Buffalo application with Buffkit wired # steps_test.go:1897 -> *TestSuite

  Scenario: Accessing login form                          # authentication.feature:9
    When I visit "/login"                                 # shared_bridge.go:60 -> *SharedContext
[HANGS HERE]
```

### Goroutine Leak Stack Trace
```
goroutine 23 [select]:
github.com/johnjansen/buffkit/ssr.(*Broker).run(0x1400059a4b0)
	/Users/johnjansen/Documents/GitHub/buffkit/ssr/broker.go:139 +0x98
created by github.com/johnjansen/buffkit/ssr.NewBroker in goroutine 9

goroutine 24 [select]:
github.com/johnjansen/buffkit/ssr.(*Broker).heartbeat(0x1400059a4b0)
	/Users/johnjansen/Documents/GitHub/buffkit/ssr/broker.go:192 +0xdc
created by github.com/johnjansen/buffkit/ssr.NewBroker in goroutine 9
```

## What We've Tried

### ✅ Fixes Attempted
1. **Added Shutdown() to SSR Broker** - Created proper cleanup mechanism
2. **Added Kit.Shutdown()** - Graceful shutdown for all subsystems
3. **Added After hooks** - Cleanup in all test scenarios
4. **Fixed duplicate methods** - Resolved naming conflicts
5. **Added broker cleanup in Reset()** - Explicitly shutdown brokers

### ❌ Still Not Working
- TestAllFeatures continues to hang despite all cleanup attempts
- Broker goroutines created early (goroutine 9) persist
- Suggests initialization issue rather than cleanup issue

## Root Cause Analysis

### Likely Causes
1. **Multiple test suite initialization conflict** - TestAllFeatures initializes ALL scenario suites simultaneously
2. **Shared state between test suites** - Possible resource contention
3. **Blocking operation in test setup** - Something in the combined initialization blocks
4. **Pattern matcher conflict** - Multiple regex patterns might be conflicting

### Evidence
- Individual tests work → Issue is in combination
- Hangs on first HTTP operation → Likely Buffalo app or context issue
- Broker created in goroutine 9 → Early initialization problem

## Working Tests vs Broken Test

### ✅ Working
```go
// These work fine individually:
go test ./features -run TestBasicFeatures    // 2 scenarios pass
go test ./features -run TestGriftTasks       // 5+ scenarios pass
go test ./features -run TestBasicFeatures -timeout=5s  // Completes quickly
```

### ❌ Broken
```go
// This hangs:
go test ./features -run TestAllFeatures      // Hangs indefinitely
go test ./features -v                        // Hangs (runs TestAllFeatures)
```

## Impact on Project

### Blocked Work
1. Cannot verify the ~200+ pattern implementations work correctly
2. Cannot run full test suite in CI/CD
3. Cannot identify which of the 80 remaining undefined steps are critical
4. Cannot create comprehensive test report for v0.1-alpha

### Risk Assessment
- **High Risk:** Shipping without full test verification
- **Medium Risk:** Missing critical bug in pattern matching
- **Low Risk:** Individual components are tested and working

## Proposed Solutions

### Option 1: Split TestAllFeatures (Recommended)
```go
// Instead of one massive test, create focused test suites:
func TestCoreFeatures(t *testing.T) {
    // Auth, Basic, Components
}

func TestAdvancedFeatures(t *testing.T) {
    // SSE, Development Mode
}

func TestCLIFeatures(t *testing.T) {
    // Grift tasks, migrations
}
```

### Option 2: Sequential Test Execution
```go
// Run each scenario suite in sequence, not parallel
func TestAllFeaturesSequential(t *testing.T) {
    tests := []func(*testing.T){
        TestBasicFeatures,
        TestAuthentication,
        TestComponents,
        // etc.
    }
    for _, test := range tests {
        test(t)
    }
}
```

### Option 3: Debug the Initialization
1. Add extensive logging to track initialization order
2. Use delve debugger to step through the hang
3. Identify exact blocking operation
4. Fix the root cause

### Option 4: Bypass for v0.1-alpha
1. Document known limitation
2. Use individual test results as verification
3. Fix in v0.2 with more time
4. Ship with "experimental" label

## Decision Required

### Questions to Answer
1. Is full test suite verification required for v0.1-alpha?
2. Can we ship with individual test verification only?
3. Should we invest time debugging or work around it?
4. Is this a Buffalo framework issue or our implementation?

### Recommended Action
**Short term (for v0.1-alpha):**
- Use Option 1: Split into smaller test suites
- Run each suite separately in CI/CD
- Document the limitation

**Long term (for v0.2):**
- Investigate root cause with Option 3
- Potentially refactor test architecture
- Consider switching test framework if needed

## Related Files
- `features/main_test.go` - The problematic TestAllFeatures
- `features/shared_bridge.go` - Where it hangs on line 60
- `ssr/broker.go` - Goroutine leak source
- `buffkit.go` - Wire() creates the broker

## Next Steps
1. **Immediate:** Try Option 1 (split test suites)
2. **Today:** Verify split suites cover all scenarios
3. **Tomorrow:** Update CI/CD to run split suites
4. **This Week:** Document workaround in README
5. **Future:** Add to v0.2 roadmap for proper fix

## Update Log
- **2024-XX-XX:** Issue first detected during BDD consolidation
- **2024-XX-XX:** Added Shutdown() methods - didn't fix
- **2024-XX-XX:** Split into individual tests - works
- **2024-XX-XX:** Created this document to track issue