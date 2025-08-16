Feature: Development Mode Features
  As a developer working on a Buffkit application
  I want special development-mode features
  So that I can debug and develop more efficiently

  Background:
    Given I have a Buffalo application
    And Buffkit is configured with development mode enabled

  Scenario: Mail preview endpoint is available in dev mode
    Given the application is wired with DevMode set to true
    When I visit "/__mail/preview"
    Then I should see the mail preview interface
    And the response status should be 200
    And I should see a list of sent emails

  Scenario: Mail preview endpoint is not available in production
    Given the application is wired with DevMode set to false
    When I visit "/__mail/preview"
    Then the response status should be 404
    And the endpoint should not exist

  Scenario: Development mail sender logs emails
    Given I have a development mail sender
    When I send an email with subject "Test Email"
    And I send an email with subject "Another Test"
    Then the emails should be logged instead of sent
    And I should be able to view them in the mail preview
    And the preview should show both email subjects

  Scenario: Development mail sender stores email content
    Given I have a development mail sender
    When I send an HTML email with content "<h1>Welcome</h1>"
    Then the email should be stored with HTML content
    And I should be able to preview the rendered HTML
    And the email should include both HTML and text versions

  Scenario: Security headers are relaxed in dev mode
    Given the application is running in development mode
    When I make a request to any endpoint
    Then the security headers should be present but relaxed
    And the Content-Security-Policy should allow development tools
    And debugging should be easier

  Scenario: Error messages are verbose in dev mode
    Given the application is running in development mode
    When an error occurs during request processing
    Then I should see detailed error messages
    And stack traces should be included
    And debugging information should be available

  Scenario: Hot reloading compatibility
    Given the application is running in development mode
    When I make changes to templates or assets
    Then the changes should be reflected without restart
    And the import maps should support development workflows
    And asset serving should prioritize development speed

  Scenario: Development vs production mail behavior
    Given I have the same Buffkit configuration
    When DevMode is true and I send an email
    Then the email should be captured for preview
    When DevMode is false and I send an email
    Then the email should be sent via SMTP
    And no preview should be generated

  Scenario: Development diagnostics
    Given the application is running in development mode
    When I access diagnostic endpoints
    Then I should see information about:
      | Component | Status |
      | SSE Broker | Active connections count |
      | Auth Store | User count |
      | Job Queue | Pending jobs |
      | Import Maps | Loaded dependencies |
      | Components | Registered components |

  Scenario: Development-only middleware
    Given the application is wired with development mode
    When I inspect the middleware stack
    Then development-specific middleware should be present
    And production optimizations should be disabled
    And debugging tools should be available
