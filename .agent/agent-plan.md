# Component Test Steps Implementation Plan

## Current Status
- 12/41 scenarios passing in component tests
- 52 undefined steps blocking full test coverage
- Component registry working for basic components
- SharedContext properly expands components

## Implementation Strategy

### Phase 1: Core Component Registration (Priority 1)
**Goal**: Implement missing component renderers and registration steps

1. **Add new component renderers to registry.go**:
   - [ ] renderTextComponent - Safe HTML escaping
   - [ ] renderModalComponent - Dialog with ARIA attributes
   - [ ] renderTabsComponent - Tab container with state
   - [ ] renderCodeComponent - Code block with syntax highlighting
   - [ ] renderFormFieldComponent - Form field wrapper
   - [ ] renderFeatureFlagComponent - Conditional rendering
   - [ ] renderIconComponent - Icon rendering
   - [ ] renderUserAvatarComponent - Avatar with user data

2. **Update RegisterDefaults() in registry.go**:
   - [ ] Add all new components to default registration
   - [ ] Ensure proper naming conventions (bk-component-name)

3. **Implement registration steps in components_steps_test.go**:
   - [ ] iHaveRegisteredATextComponent
   - [ ] iHaveRegisteredAModalComponent
   - [ ] iHaveRegisteredMultipleComponents
   - [ ] iHaveRegisteredAButtonComponentWithVariants
   - [ ] iHaveRegisteredACodeComponent
   - [ ] iHaveRegisteredAFormFieldComponent
   - [ ] iHaveRegisteredAFeatureFlagComponent
   - [ ] iHaveRegisteredATabsComponent

### Phase 2: Output Validation Steps (Priority 2)
**Goal**: Implement HTML validation and attribute checking

1. **Attribute preservation steps**:
   - [ ] theOutputShouldContainId
   - [ ] theOutputShouldContainClass (fix existing)
   - [ ] theOutputShouldContainDataTestId
   - [ ] theOutputShouldContainDataTurbo
   - [ ] theOutputShouldContainDataTrackEvent
   - [ ] theOutputShouldContainDataComponent

2. **ARIA attribute steps**:
   - [ ] theOutputShouldContainAriaExpanded
   - [ ] theOutputShouldContainRole
   - [ ] theOutputShouldContainAriaModal
   - [ ] theOutputShouldContainAriaLabelledby
   - [ ] outputShouldHaveProperARIA

3. **Framework integration steps**:
   - [ ] theOutputShouldContainHxPost
   - [ ] theOutputShouldContainHxTarget
   - [ ] theOutputShouldContainXData
   - [ ] outputShouldWorkWithHTMX
   - [ ] outputShouldWorkWithAlpine

4. **Component state steps**:
   - [ ] theOutputShouldContainDataState
   - [ ] theOutputShouldContainDataInitialTab
   - [ ] ariaExpandedShouldReflectState

### Phase 3: Complex Rendering Scenarios (Priority 3)
**Goal**: Handle special rendering cases and performance

1. **Performance and scale**:
   - [ ] iRenderHTMLWithComponentInstances
   - [ ] theExpansionShouldCompleteWithinMs
   - [ ] allComponentsShouldBeExpandedCorrectly
   - [ ] iRenderPageWithManyComponents
   - [ ] renderingShouldCompleteQuickly

2. **Conditional rendering**:
   - [ ] theComponentShouldNotBeExpanded
   - [ ] theOriginalHTMLShouldBePreserved
   - [ ] componentTagsShouldNotBeExpanded
   - [ ] componentExpansionShouldBeSkipped

3. **Content handling**:
   - [ ] theOutputShouldNotContainAnActualScriptTag
   - [ ] outputShouldBeSafelyEscaped
   - [ ] whitespaceShouldBePreserved
   - [ ] outputShouldContainSanitizedContent

### Phase 4: Advanced Features (Priority 4)
**Goal**: Feature flags, JSON handling, and development mode

1. **Feature flag system**:
   - [ ] theFlagIsEnabled
   - [ ] theOutputShouldConditionallyShow
   - [ ] Implement feature flag state management

2. **JSON API handling**:
   - [ ] iHaveAJSONAPIEndpoint
   - [ ] theResponseContentTypeIs
   - [ ] iRenderJSONContainingComponents
   - [ ] jsonShouldBeUnchanged

3. **Development mode**:
   - [ ] applicationIsInDevelopmentMode
   - [ ] applicationIsInProductionMode
   - [ ] outputShouldContainComponentComments
   - [ ] commentsShouldIncludeComponentName

4. **Component registry**:
   - [ ] iQueryComponentRegistry
   - [ ] iShouldGetListContaining
   - [ ] iRegisterCustomButtonComponent
   - [ ] customButtonShouldBeUsed

5. **Accessibility**:
   - [ ] eachInputShouldHaveUniqueID
   - [ ] eachLabelShouldHaveMatchingFor
   - [ ] outputShouldContainAppropriateARIALabels

## File Modifications

### 1. components/registry.go
- Add 8 new component renderers
- Update RegisterDefaults() method
- Add feature flag support
- Add development mode support

### 2. features/components_steps_test.go
- Implement ~52 undefined step definitions
- Add helper functions for validation
- Add feature flag state management
- Add development mode toggle

### 3. features/components.feature
- No changes needed (tests are well-defined)

## Testing Order

1. Run tests after each phase to verify progress
2. Use `go test ./features -run TestCoreFeatures -v` for detailed output
3. Track undefined steps with `go test ./features -run TestCoreFeatures 2>&1 | grep "ctx.Step"`
4. Run benchmarks after performance steps implemented

## Success Metrics

- Phase 1: ~20 scenarios passing
- Phase 2: ~30 scenarios passing
- Phase 3: ~35 scenarios passing
- Phase 4: 40+ scenarios passing (>95% coverage)

## Timeline

- Phase 1: 2-3 hours
- Phase 2: 3-4 hours
- Phase 3: 2-3 hours
- Phase 4: 2-3 hours
- Total: ~10-13 hours

## Next Steps

1. Start with Phase 1 - implement core component renderers
2. Test each component individually before moving to next
3. Implement validation steps after components are working
4. Add performance and advanced features last