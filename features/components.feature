Feature: Component System
  As a developer
  I want to use server-side components that can be progressively enhanced
  So that I can build rich, accessible UIs that work without JavaScript

  Background:
    Given the component registry is initialized
    And the component expansion middleware is active
  # Basic Component Rendering

  Scenario: Render a simple button component
    Given I have registered a button component
    When I render HTML containing "<bk-button>Click me</bk-button>"
    Then the output should contain "<button"
    And the output should contain "Click me</button>"
    And the output should not contain "<bk-button"

  Scenario: Render component with attributes
    Given I have registered a button component
    When I render HTML containing '<bk-button variant="primary" size="large">Submit</bk-button>'
    Then the output should contain 'class="'
    And the output should contain "primary"
    And the output should contain "large"
    And the output should contain "Submit</button>"

  Scenario: Pass through data attributes
    Given I have registered a dropdown component
    When I render HTML containing '<bk-dropdown data-test-id="menu" data-track-event="open">Menu</bk-dropdown>'
    Then the output should contain 'data-test-id="menu"'
    And the output should contain 'data-track-event="open"'
    And the output should contain 'data-component="dropdown"'
  # Slot Content Projection

  Scenario: Component with named slots
    Given I have registered a card component with named slots
    When I render HTML containing:
      """
      <bk-card>
        <div slot="header">Card Title</div>
        <div slot="body">Card content goes here</div>
        <div slot="footer">Card actions</div>
      </bk-card>
      """
    Then the output should contain "Card Title"
    And the output should contain "Card content goes here"
    And the output should contain "Card actions"
    And the output should be properly structured HTML

  Scenario: Component with default slot
    Given I have registered an alert component
    When I render HTML containing "<bk-alert>This is a warning message</bk-alert>"
    Then the output should contain "This is a warning message"
    And the output should contain appropriate alert styling
  # Nested Components

  Scenario: Render nested components
    Given I have registered button and card components
    When I render HTML containing:
      """
      <bk-card>
        <div slot="body">
          <p>Card content</p>
          <bk-button>Action</bk-button>
        </div>
      </bk-card>
      """
    Then the output should contain expanded card HTML
    And the output should contain expanded button HTML
    And the output should not contain "bk-card"
    And the output should not contain "bk-button"
  # Progressive Enhancement Support

  Scenario: Interactive component includes enhancement data
    Given I have registered a dropdown component
    When I render HTML containing '<bk-dropdown>Menu</bk-dropdown>'
    Then the output should contain 'data-component="dropdown"'
    And the output should contain 'data-state="closed"'
    And the output should contain 'aria-expanded="false"'
    And the output should be accessible without JavaScript

  Scenario: Form component with validation attributes
    Given I have registered an input component
    When I render HTML containing '<bk-input type="email" required name="user_email" />'
    Then the output should contain 'type="email"'
    And the output should contain "required"
    And the output should contain 'name="user_email"'
    And the output should contain appropriate ARIA labels
  # Error Handling

  Scenario: Handle unregistered component gracefully
    When I render HTML containing "<bk-unknown>Content</bk-unknown>"
    Then the output should contain "<bk-unknown>Content</bk-unknown>"
    And no error should be raised

  Scenario: Handle malformed component HTML
    Given I have registered a button component
    When I render HTML containing "<bk-button>Unclosed tag"
    Then the component should not be expanded
    And the original HTML should be preserved
  # Security

  Scenario: Prevent XSS in component attributes
    Given I have registered a button component
    When I render HTML containing '<bk-button onclick="alert(1)">Click</bk-button>'
    Then the output should not contain "onclick"
    And the output should contain sanitized content

  Scenario: Escape user content properly
    Given I have registered a text component
    When I render HTML containing '<bk-text><script>alert("XSS")</script></bk-text>'
    Then the output should contain "&lt;script&gt;"
    And the output should not contain an actual script tag
  # Performance

  Scenario: Handle large pages efficiently
    Given I have registered multiple components
    When I render HTML with 100 component instances
    Then the expansion should complete within 100ms
    And all components should be properly expanded

  Scenario: Skip expansion for non-HTML responses
    Given I have a JSON API endpoint
    When the response content-type is "application/json"
    Then the component expansion should be skipped
    And the JSON should be returned unchanged
  # Custom Attributes

  Scenario: Preserve custom HTML attributes
    Given I have registered a button component
    When I render HTML containing '<bk-button id="submit-btn" data-turbo="false">Submit</bk-button>'
    Then the output should contain 'id="submit-btn"'
    And the output should contain 'data-turbo="false"'

  Scenario: Handle boolean attributes
    Given I have registered an input component
    When I render HTML containing '<bk-input disabled readonly checked />'
    Then the output should contain "disabled"
    And the output should contain "readonly"
    And the output should contain "checked"
  # Component Variants

  Scenario Outline: Render component variants
    Given I have registered a button component with variants
    When I render HTML containing '<bk-button variant="<variant>">Click</bk-button>'
    Then the output should contain appropriate classes for "<variant>"

    Examples:
      | variant   |
      | primary   |
      | secondary |
      | danger    |
      | ghost     |
  # Accessibility
# Edge Cases

  Scenario: Generate unique IDs for accessibility
    Given I have registered a form field component
    When I render HTML containing multiple '<bk-input label="Email" />' components
    Then each input should have a unique ID
    And each label should have a matching "for" attribute
  # Integration

  Scenario: Work with HTMX attributes
    Given I have registered a button component
    When I render HTML containing '<bk-button hx-post="/api/save" hx-target="#result">Save</bk-button>'
    Then the output should contain 'hx-post="/api/save"'
    And the output should contain 'hx-target="#result"'

  Scenario: Work with Alpine.js directives
    Given I have registered a dropdown component
    When I render HTML containing '<bk-dropdown x-data="{ open: false }">Menu</bk-dropdown>'
    Then the output should contain 'x-data="{ open: false }"'
    And the output should contain data attributes for progressive enhancement
  # Component Registry

  Scenario: List all registered components
    Given I have registered button, card, and modal components
    When I query the component registry
    Then I should get a list containing "button", "card", and "modal"

  Scenario: Override default component
    Given I have registered a default button component
    When I register a custom button component
    Then the custom component should be used for rendering
    And the default component should be replaced
  # Development Mode

  Scenario: Show component boundaries in development
    Given the application is in development mode
    And I have registered a card component
    When I render HTML containing "<bk-card>Content</bk-card>"
    Then the output should contain HTML comments marking component boundaries
    And the comments should include the component name

  Scenario: Hide component boundaries in production
    Given the application is in production mode
    And I have registered a card component
    When I render HTML containing "<bk-card>Content</bk-card>"
    Then the output should not contain component boundary comments
  # Edge Cases

  Scenario: Components with code blocks

  Scenario: Handle components with hyphenated names
    Given I have registered a component named "progress-bar"
    When I render HTML containing '<bk-progress-bar value="50" />'
    Then the component should be properly expanded

  Scenario: Preserve whitespace in pre elements
    Given I have registered a code component
    When I render HTML containing:
      """
      <bk-code>
        function example() {
          return true;
        }
      </bk-code>
      """
    Then the output should preserve the indentation
    And the output should maintain line breaks
