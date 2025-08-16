Feature: Basic Buffkit Functionality
  As a developer
  I want to verify core Buffkit functionality works
  So that I can build upon a solid foundation

  Background:
    Given I have a Buffalo application

  Scenario: Wire Buffkit successfully
    When I wire Buffkit with a valid configuration
    Then all components should be initialized

  Scenario: Get version information
    When I check the Buffkit version
    Then I should get a non-empty version string
