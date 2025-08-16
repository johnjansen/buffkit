package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/cucumber/godog"
	"github.com/gobuffalo/buffalo"
	"github.com/johnjansen/buffkit"
)

// BasicTestSuite holds minimal test context
type BasicTestSuite struct {
	app    *buffalo.App
	kit    *buffkit.Kit
	config buffkit.Config
	err    error
}

// Reset clears the test state
func (bts *BasicTestSuite) Reset() {
	bts.app = nil
	bts.kit = nil
	bts.config = buffkit.Config{}
	bts.err = nil
}

// Given I have a Buffalo application
func (bts *BasicTestSuite) iHaveABuffaloApplication() error {
	bts.app = buffalo.New(buffalo.Options{
		Env: "test",
	})
	return nil
}

// When I wire Buffkit with a valid configuration
func (bts *BasicTestSuite) iWireBuffkitWithAValidConfiguration() error {
	bts.config = buffkit.Config{
		AuthSecret: []byte("test-secret-key-32-chars-long-enough"),
		DevMode:    true,
	}

	kit, err := buffkit.Wire(bts.app, bts.config)
	bts.kit = kit
	bts.err = err
	return nil
}

// When I check the Buffkit version
func (bts *BasicTestSuite) iCheckTheBuffkitVersion() error {
	version := buffkit.Version()
	if version == "" {
		bts.err = fmt.Errorf("version is empty")
	}
	return nil
}

// Then all components should be initialized
func (bts *BasicTestSuite) allComponentsShouldBeInitialized() error {
	if bts.err != nil {
		return bts.err
	}
	if bts.kit == nil {
		return fmt.Errorf("kit is nil")
	}
	return nil
}

// Then I should get a non-empty version string
func (bts *BasicTestSuite) iShouldGetANonEmptyVersionString() error {
	if bts.err != nil {
		return bts.err
	}
	return nil
}

// InitializeBasicScenario sets up the basic test scenarios
func InitializeBasicScenario(ctx *godog.ScenarioContext) {
	bts := &BasicTestSuite{}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		bts.Reset()
		return ctx, nil
	})

	ctx.Step(`^I have a Buffalo application$`, bts.iHaveABuffaloApplication)
	ctx.Step(`^I wire Buffkit with a valid configuration$`, bts.iWireBuffkitWithAValidConfiguration)
	ctx.Step(`^I check the Buffkit version$`, bts.iCheckTheBuffkitVersion)
	ctx.Step(`^all components should be initialized$`, bts.allComponentsShouldBeInitialized)
	ctx.Step(`^I should get a non-empty version string$`, bts.iShouldGetANonEmptyVersionString)
}

// TestBasicFeatures runs only the most basic scenarios
func TestBasicFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeBasicScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"basic.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("basic feature tests failed")
	}
}
