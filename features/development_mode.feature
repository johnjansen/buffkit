Feature: Development Mode Features
  As a developer working on a Buffkit application
  I want development-specific features like mail preview
  So that I can debug and test email functionality without sending real emails

  Background:
    Given I have a Buffalo application
    And Buffkit is configured with development mode enabled

  Scenario: Mail preview endpoint is available in dev mode
    Given the application is wired with DevMode set to true
    When I visit "/__mail/preview"
    Then I should see the mail preview interface
    And the response status should be 200

  Scenario: Mail preview endpoint is not available in production
    Given the application is wired with DevMode set to false
    When I visit "/__mail/preview"
    Then the response status should be 404
    And the endpoint should not exist

  Scenario: Development mail sender logs emails instead of sending
    Given I have a development mail sender
    When I send an email with subject "Test Email"
    Then the email should be logged instead of sent
    And I should be able to view it in the mail preview

  Scenario: Development mail sender stores multiple emails
    Given I have a development mail sender
    When I send an email with subject "First Email"
    And I send an email with subject "Second Email"
    Then both emails should be stored
    And I should see both in the mail preview interface

  Scenario: Mail preview shows HTML and text versions
    Given I have a development mail sender
    When I send an HTML email with content "<h1>Welcome</h1>"
    Then the email should be stored with HTML content
    And I should be able to preview the rendered HTML
    And the preview should show both HTML and text versions

  Scenario: Development vs production mail behavior
    Given I have the same Buffkit configuration
    When DevMode is true and I send an email
    Then the email should be captured for preview
    When DevMode is false and I send an email
    Then the email should be sent via SMTP
    And no preview should be generated

  Scenario: Security headers are present but relaxed in dev mode
    Given the application is running in development mode
    When I make a request to any endpoint
    Then security headers should be present
    But they should be configured for development convenience
