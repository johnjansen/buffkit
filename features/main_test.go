package features

import (
	"testing"

	"github.com/cucumber/godog"
)

// TestAllFeatures is the main test runner that combines all feature test suites
func TestAllFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge that provides common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize feature-specific scenario suites
			// These work alongside the shared bridge
			InitializeScenario(ctx)                // Basic scenarios from steps_test.go
			InitializeBasicScenario(ctx)           // Basic integration from basic_test.go
			InitializeSSEReconnectionScenario(ctx) // SSE reconnection from sse_reconnection_test.go
			InitializeAuthEnhancedScenario(ctx)    // Enhanced auth from auth_enhanced_steps_test.go
			InitializeComponentsScenario(ctx)      // Components from components_steps_test.go
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"."},
			TestingT: t,
			Tags:     "~@skip", // Skip scenarios marked with @skip
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
