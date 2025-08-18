package features

import (
	"testing"

	"github.com/cucumber/godog"
)

// TestCoreFeatures tests authentication, basic features, and components
// This is a focused test suite that avoids the hanging issue from TestAllFeatures
func TestCoreFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge for common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize core feature scenarios
			InitializeScenario(ctx, bridge)           // Basic scenarios from steps_test.go
			InitializeBasicScenario(ctx)              // Basic integration from basic_test.go
			InitializeComponentsScenario(ctx, bridge) // Components from components_steps_test.go - pass bridge for shared context
		},
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"basic.feature",
				"buffkit_integration.feature",
				"components.feature",
			},
			TestingT: t,
			Tags:     "~@skip", // Skip scenarios marked with @skip
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run core feature tests")
	}
}

// TestAuthenticationFeatures tests authentication-related scenarios
func TestAuthenticationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge for common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize authentication scenarios
			InitializeScenario(ctx, bridge)             // Basic scenarios (includes auth steps)
			InitializeAuthEnhancedScenario(ctx, bridge) // Enhanced auth from auth_enhanced_steps_test.go - pass bridge
		},
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"authentication.feature",
				"authentication_enhanced.feature",
			},
			TestingT: t,
			Tags:     "~@skip",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run authentication feature tests")
	}
}

// TestSSEFeatures tests Server-Sent Events scenarios
func TestSSEFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge for common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize SSE scenarios
			InitializeScenario(ctx, bridge)                // Basic scenarios (includes SSE steps) - pass bridge
			InitializeSSEReconnectionScenario(ctx, bridge) // SSE reconnection from sse_reconnection_test.go - pass bridge
		},
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"server_sent_events.feature",
				"sse_reconnection.feature",
			},
			TestingT: t,
			Tags:     "~@skip",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run SSE feature tests")
	}
}

// TestDevelopmentFeatures tests development mode and test patterns
func TestDevelopmentFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge for common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize development scenarios
			InitializeScenario(ctx, bridge) // Basic scenarios
		},
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"development_mode.feature",
				"test_patterns.feature",
			},
			TestingT: t,
			Tags:     "~@skip",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run development feature tests")
	}
}

// TestCLIFeatures tests CLI and Grift tasks
func TestCLIFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge for common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize CLI scenarios
			InitializeScenario(ctx, bridge)      // Basic scenarios - pass bridge
			InitializeGriftScenario(ctx, bridge) // Grift tasks from grift_tasks_test.go - pass bridge
		},
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"cli_tasks.feature",
				"grift_tasks.feature",
			},
			TestingT: t,
			Tags:     "~@skip",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run CLI feature tests")
	}
}

// TestAllFeaturesSequential runs all test suites sequentially
// This is a safer alternative to the original TestAllFeatures that was hanging
func TestAllFeaturesSequential(t *testing.T) {
	t.Run("CoreFeatures", TestCoreFeatures)
	t.Run("AuthenticationFeatures", TestAuthenticationFeatures)
	t.Run("SSEFeatures", TestSSEFeatures)
	t.Run("DevelopmentFeatures", TestDevelopmentFeatures)
	t.Run("CLIFeatures", TestCLIFeatures)

	// Also run the existing individual tests that have their own test functions
	t.Run("BasicFeatures", TestBasicFeatures)
	t.Run("GriftTasks", TestGriftTasks)
}

// TestAllFeatures - DEPRECATED: This function hangs due to resource contention
// when initializing all test suites simultaneously. Use TestAllFeaturesSequential instead.
// Keeping this function commented out for reference and documentation.
/*
func TestAllFeatures(t *testing.T) {
	t.Skip("DEPRECATED: This test hangs. Use TestAllFeaturesSequential or individual test suites instead")

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge that provides common step definitions
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize feature-specific scenario suites
			// These work alongside the shared bridge
			InitializeScenario(ctx, bridge) // Basic scenarios from steps_test.go
			InitializeBasicScenario(ctx)           // Basic integration from basic_test.go
			InitializeSSEReconnectionScenario(ctx, bridge) // SSE reconnection from sse_reconnection_test.go
			InitializeAuthEnhancedScenario(ctx, bridge)    // Enhanced auth from auth_enhanced_steps_test.go
			InitializeComponentsScenario(ctx, bridge)      // Components from components_steps_test.go
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
*/
