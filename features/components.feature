Feature: Component System
  As a developer using Buffkit
  I want to use server-side components
  So that I can build modular and reusable UI elements

  Background:
    Given I have a Buffalo application with Buffkit wired

  @skip
  Scenario: Register a basic component
    Given I have a component registry
    When I register a component "Button" with template "<button>{{.text}}</button>"
    Then the component should be available in the registry
    And I should be able to render it with attributes

  @skip
  Scenario: Render component with attributes
    Given I have a registered "Button" component
    When I render the component with text "Click me"
    Then the output should contain "<button>Click me</button>"

  @skip
  Scenario: Component with slots
    Given I have a component "Card" with a slot
    When I render the component with slot content
    Then the slot content should be included in the output

  @skip
  Scenario: Nested components
    Given I have multiple registered components
    When I render a component that contains other components
    Then all components should be expanded correctly

  @skip
  Scenario: Component middleware expansion
    Given I have HTML with buffkit:component tags
    When the component middleware processes the HTML
    Then all component tags should be replaced with rendered components
