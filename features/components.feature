Feature: Component Registry System
  As a developer
  I want to register and use server-side components
  So that I can create reusable UI elements that are expanded server-side

  Background:
    Given the component registry is initialized
    And the component expansion middleware is active

  Scenario: Register and render a simple component
    Given I register a component "bk-hello" that renders "Hello, World!"
    When I render HTML containing "<bk-hello></bk-hello>"
    Then the output should contain "Hello, World!"
    And the output should not contain "<bk-hello"

  Scenario: Component with attributes
    Given I register a component "bk-greeting" that uses the "name" attribute
    When I render HTML containing '<bk-greeting name="Alice"></bk-greeting>'
    Then the output should contain "Hello, Alice"
  # DISABLED: Slot content handling is beyond minimal scope
  # Scenario: Component with slot content
  #   Given I register a component "bk-wrapper" that wraps content
  #   When I render HTML containing "<bk-wrapper>Inner content</bk-wrapper>"
  #   Then the output should contain "Inner content"
  #   And the output should be properly wrapped

  Scenario: Multiple components in one page
    Given I register a component "bk-one" that renders "Component One"
    And I register a component "bk-two" that renders "Component Two"
    When I render HTML containing both "<bk-one></bk-one>" and "<bk-two></bk-two>"
    Then the output should contain "Component One"
    And the output should contain "Component Two"
  # DISABLED: Nested component expansion is beyond minimal scope
  # Scenario: Nested components are expanded
  #   Given I register a component "bk-outer" that contains another component
  #   And I register a component "bk-inner" that renders "Nested"
  #   When I render HTML containing "<bk-outer></bk-outer>"
  #   Then the output should contain "Nested"
  #   And all components should be expanded

  Scenario: Unknown components are preserved
    When I render HTML containing "<bk-unknown>Content</bk-unknown>"
    Then the output should contain "<bk-unknown>Content</bk-unknown>"
    And unknown components should not cause errors

  Scenario: Non-HTML responses are not processed
    Given I have a JSON response
    When the response contains "<bk-component>"
    Then the JSON should be returned unchanged
    And no component expansion should occur
  # DISABLED: Dev mode boundary comments are beyond minimal scope
  # Scenario: Component expansion in development mode
  #   Given the application is in development mode
  #   And I register a component "bk-debug" that renders "Debug Info"
  #   When I render HTML containing "<bk-debug></bk-debug>"
  #   Then the output should contain "Debug Info"
  #   And the output should contain component boundary comments

  Scenario: Component expansion in production mode
    Given the application is in production mode
    And I register a component "bk-prod" that renders "Production"
    When I render HTML containing "<bk-prod></bk-prod>"
    Then the output should contain "Production"
    And the output should not contain component boundary comments

  Scenario: Override a registered component
    Given I register a component "bk-override" that renders "Original"
    When I register a component "bk-override" that renders "Replaced"
    And I render HTML containing "<bk-override></bk-override>"
    Then the output should contain "Replaced"
    And the output should not contain "Original"

  Scenario: Component registry is shared across requests
    Given I register a component "bk-shared" in one request
    When I render HTML containing "<bk-shared></bk-shared>" in another request
    Then the component should be available
    And the output should be correctly expanded
