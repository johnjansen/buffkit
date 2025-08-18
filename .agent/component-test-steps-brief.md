# Component Test Steps Implementation Brief

## Executive Summary
We need to implement ~52 undefined component-specific test steps to achieve full test coverage for the Buffkit component system. These steps validate component rendering, attribute handling, slot management, and framework integration (HTMX, Alpine.js).

## Current Status
- ✅ **12/41 scenarios passing** in component tests
- ✅ **Component registry working** - basic components render correctly
- ✅ **Test infrastructure fixed** - SharedContext properly expands components
- ❌ **52 undefined steps** blocking full test coverage
- ❌ **Missing component types**: modal, tabs, feature flags, text, code blocks

## Priority Implementation Groups

### 🔴 Priority 1: Core Component Registration (8 steps)
These enable basic component testing functionality.

#### Steps to Implement:
```go
// components_steps_test.go additions

func (s *ComponentsTestSuite) iHaveRegisteredATextComponent() error {
    // Register a text component that safely renders content
    s.registry.Register("bk-text", renderTextComponent)
}

func (s *ComponentsTestSuite) iHaveRegisteredAModalComponent() error {
    // Register modal with title attribute and slot content
    s.registry.Register("bk-modal", renderModalComponent)
}

func (s *ComponentsTestSuite) iHaveRegisteredMultipleComponents() error {
    // Register button, card, modal, dropdown all at once
    s.registry.RegisterDefaults()
    s.registry.Register("bk-modal", renderModalComponent)
    s.registry.Register("bk-tabs", renderTabsComponent)
}

func (s *ComponentsTestSuite) iHaveRegisteredAButtonComponentWithVariants() error {
    // Ensure button supports variant="primary|secondary|danger"
}

func (s *ComponentsTestSuite) iHaveRegisteredACodeComponent() error {
    // Register code block component with syntax highlighting support
}

func (s *ComponentsTestSuite) iHaveRegisteredAFormFieldComponent() error {
    // Register form field wrapper with label, input, error slots
}

func (s *ComponentsTestSuite) iHaveRegisteredAFeatureFlagComponent() error {
    // Register component that conditionally renders based on flag
}

func (s *ComponentsTestSuite) iHaveRegisteredATabsComponent() error {
    // Register tabs with default-tab attribute
}
```

### 🟡 Priority 2: Rendering Validation (15 steps)
These verify specific HTML patterns in rendered output.

#### Pattern Categories:
1. **Attribute Preservation**
   - `theOutputShouldContainId(id string)`
   - `theOutputShouldContainClass()`
   - `theOutputShouldContainDataTestId(id string)`
   - `theOutputShouldContainDataTurbo(value string)`

2. **ARIA Attributes**
   - `theOutputShouldContainAriaExpanded(value string)`
   - `theOutputShouldContainRole(role string)`

3. **HTMX Integration**
   - `theOutputShouldContainHxTarget(target string)`
   - Handle hx-post, hx-get, hx-trigger preservation

4. **Alpine.js Integration**
   - `theOutputShouldContainXData(data string)`
   - Preserve x-show, x-if directives

5. **Component State**
   - `theOutputShouldContainDataState(state string)`
   - `theOutputShouldContainDataInitialTab(tab string)`

#### Implementation Template:
```go
func (s *ComponentsTestSuite) theOutputShouldContainId(id string) error {
    expected := fmt.Sprintf(`id="%s"`, id)
    if !strings.Contains(s.shared.Output, expected) {
        return fmt.Errorf("output does not contain %s\nActual: %s", expected, s.shared.Output)
    }
    return nil
}
```

### 🟢 Priority 3: Complex Rendering Scenarios (12 steps)
These handle special rendering cases.

#### Steps to Implement:
```go
func (s *ComponentsTestSuite) iRenderHTMLWithComponentInstances(count int) error {
    // Generate HTML with N component instances for performance testing
    var html strings.Builder
    for i := 0; i < count; i++ {
        html.WriteString(fmt.Sprintf(`<bk-button>Button %d</bk-button>`, i))
    }
    return s.shared.IRenderHTMLContaining(html.String())
}

func (s *ComponentsTestSuite) theExpansionShouldCompleteWithinMs(ms int) error {
    // Measure rendering time and verify it's under threshold
    start := time.Now()
    // ... rendering already done in previous step
    elapsed := time.Since(start).Milliseconds()
    if elapsed > int64(ms) {
        return fmt.Errorf("expansion took %dms, expected under %dms", elapsed, ms)
    }
}

func (s *ComponentsTestSuite) theComponentShouldNotBeExpanded() error {
    // Verify component tag is still present (not expanded)
    if !strings.Contains(s.shared.Output, "<bk-") {
        return fmt.Errorf("component was unexpectedly expanded")
    }
}

func (s *ComponentsTestSuite) theOriginalHTMLShouldBePreserved() error {
    // For malformed HTML, verify no corruption occurred
}
```

### 🔵 Priority 4: Advanced Features (17 steps)
These test special component behaviors.

#### Feature Flag Component:
```go
func (s *ComponentsTestSuite) theFlagIsEnabled(flag string) error {
    // Set feature flag state for testing
    s.featureFlags[flag] = true
}

func renderFeatureFlagComponent(attrs map[string]string, slots map[string]string) ([]byte, error) {
    flag := attrs["flag"]
    if featureFlags[flag] {
        return []byte(slots["default"]), nil
    }
    return []byte(""), nil
}
```

#### Security & Safety:
```go
func (s *ComponentsTestSuite) theOutputShouldNotContainAnActualScriptTag() error {
    // Verify <script> is escaped as &lt;script&gt;
    if strings.Contains(s.shared.Output, "<script") {
        return fmt.Errorf("output contains unescaped script tag")
    }
    if !strings.Contains(s.shared.Output, "&lt;script") {
        return fmt.Errorf("script tag was not properly escaped")
    }
}
```

#### JSON API Handling:
```go
func (s *ComponentsTestSuite) iHaveAJSONAPIEndpoint() error {
    // Set up test endpoint that returns JSON
}

func (s *ComponentsTestSuite) theResponseContentTypeIs(contentType string) error {
    // Set response content type for testing
}

func (s *ComponentsTestSuite) theComponentExpansionShouldBeSkipped() error {
    // Verify output is unchanged when content-type != text/html
}
```

## New Component Implementations Needed

### 1. Modal Component
```go
func renderModalComponent(attrs map[string]string, slots map[string]string) ([]byte, error) {
    title := attrs["title"]
    if title == "" {
        title = "Modal"
    }
    
    return []byte(fmt.Sprintf(`
        <div class="bk-modal" role="dialog" aria-labelledby="modal-title">
            <div class="bk-modal-header">
                <h2 id="modal-title">%s</h2>
            </div>
            <div class="bk-modal-body">%s</div>
        </div>
    `, html.EscapeString(title), slots["default"])), nil
}
```

### 2. Tabs Component
```go
func renderTabsComponent(attrs map[string]string, slots map[string]string) ([]byte, error) {
    defaultTab := attrs["default-tab"]
    if defaultTab == "" {
        defaultTab = "tab1"
    }
    
    return []byte(fmt.Sprintf(`
        <div class="bk-tabs" data-initial-tab="%s">
            <div class="bk-tabs-content">%s</div>
        </div>
    `, defaultTab, slots["default"])), nil
}
```

### 3. Text Component (Safe Rendering)
```go
func renderTextComponent(attrs map[string]string, slots map[string]string) ([]byte, error) {
    // Escapes all HTML to prevent XSS
    content := html.EscapeString(slots["default"])
    return []byte(fmt.Sprintf(`<span class="bk-text">%s</span>`, content)), nil
}
```

### 4. Code Component
```go
func renderCodeComponent(attrs map[string]string, slots map[string]string) ([]byte, error) {
    lang := attrs["language"]
    if lang == "" {
        lang = "plaintext"
    }
    
    content := html.EscapeString(slots["default"])
    return []byte(fmt.Sprintf(`
        <pre class="bk-code"><code class="language-%s">%s</code></pre>
    `, lang, content)), nil
}
```

## Implementation Strategy

### Phase 1: Core Components (Day 1)
1. Implement missing component renderers (modal, tabs, text, code)
2. Add to registry.RegisterDefaults()
3. Implement registration step definitions
4. Run tests to verify ~20 scenarios pass

### Phase 2: Output Validation (Day 1-2)
1. Implement attribute checking steps
2. Add ARIA attribute validation
3. Add framework integration checks (HTMX, Alpine)
4. Target ~30 scenarios passing

### Phase 3: Advanced Features (Day 2)
1. Implement performance testing steps
2. Add security validation (XSS prevention)
3. Implement feature flag system
4. Add JSON API handling
5. Target 35+ scenarios passing

### Phase 4: Polish & Edge Cases (Day 2-3)
1. Handle malformed HTML gracefully
2. Add component boundary markers
3. Implement custom component override
4. Target 40+ scenarios passing (>95% coverage)

## Testing Approach

### For Each Component:
1. **Unit Test**: Test renderer function directly
2. **Integration Test**: Test via BDD scenario
3. **Edge Cases**: Malformed input, missing attributes, XSS attempts
4. **Performance**: Verify renders under 10ms

### Validation Pattern:
```go
// Consistent error messages for debugging
func validateOutput(output, expected, description string) error {
    if !strings.Contains(output, expected) {
        return fmt.Errorf("%s\nExpected: %s\nActual output:\n%s", 
            description, expected, output)
    }
    return nil
}
```

## Success Metrics

### Immediate Goals (v0.1-alpha):
- ✅ 35+ scenarios passing (85% coverage)
- ✅ All core components implemented
- ✅ Security features validated
- ✅ Performance benchmarks met

### Future Goals (v0.2):
- ✅ 40+ scenarios passing (>95% coverage)
- ✅ Custom component system
- ✅ Advanced slot management
- ✅ Full framework integration

## Files to Modify

1. **components/registry.go**
   - Add new component renderers
   - Update RegisterDefaults()

2. **features/components_steps_test.go**
   - Implement all undefined step definitions
   - Add helper functions for validation

3. **features/components.feature**
   - Review and adjust expectations if needed
   - Add comments for complex scenarios

## Quick Start Commands

```bash
# Run component tests only
go test ./features -run TestCoreFeatures -v

# Check undefined steps
go test ./features -run TestCoreFeatures 2>&1 | grep "ctx.Step"

# Run with coverage
go test ./features -run TestCoreFeatures -cover

# Benchmark component rendering
go test -bench=BenchmarkComponentRender ./components
```

## Dependencies

- `golang.org/x/net/html` - Already used for HTML parsing
- No new dependencies required
- All implementations use standard library

## Risk Mitigation

1. **Performance**: Pre-compile templates if rendering is slow
2. **Security**: Always escape user content, validate attributes
3. **Compatibility**: Test with real Buffalo apps
4. **Maintenance**: Keep components simple and focused

## Estimated Timeline

- **Day 1**: Implement Priority 1 & 2 (23 steps) - 8 hours
- **Day 2**: Implement Priority 3 & 4 (29 steps) - 8 hours  
- **Day 3**: Testing, debugging, documentation - 4 hours

**Total: ~20 hours for full implementation**

## Next Steps

1. Review this brief with the team
2. Start with Priority 1 components
3. Implement in TDD style - write test, then implementation
4. Run full test suite after each phase
5. Document any deviations from plan