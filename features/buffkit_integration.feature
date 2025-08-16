Feature: Buffkit Integration
  As a developer using Buffalo
  I want to integrate Buffkit into my application
  So that I can get SSR-first features out of the box

  Background:
    Given I have a Buffalo application

  Scenario: Successfully wiring Buffkit with valid configuration
    When I wire Buffkit with a valid configuration
    Then all components should be initialized
    And the Kit should contain a broker
    And the Kit should contain an auth store
    And the Kit should contain a mail sender
    And the Kit should contain an import map manager
    And the Kit should contain a component registry

  Scenario: Rejecting configuration with missing auth secret
    When I wire Buffkit with an empty auth secret
    Then I should get an error "AuthSecret is required"

  Scenario: Rejecting configuration with nil auth secret
    When I wire Buffkit with a nil auth secret
    Then I should get an error "AuthSecret is required"

  Scenario: Handling invalid Redis configuration
    When I wire Buffkit with an invalid Redis URL "redis://invalid:99999/0"
    Then I should get an error containing "failed to initialize jobs"

  Scenario: Providing version information
    When I check the Buffkit version
    Then I should get a non-empty version string
    And the version should contain "alpha"
