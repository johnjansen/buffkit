package features

import (
	"testing"

	"github.com/cucumber/godog"
)

// TestGenerators tests the generator functionality
func TestGenerators(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeGeneratorScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths: []string{
				"generators.feature",
			},
			TestingT: t,
			Tags:     "@generators",
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run generator feature tests")
	}
}
