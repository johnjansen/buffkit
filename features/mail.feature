Feature: Mail System
  As a developer using Buffkit
  I want to send emails through a unified interface
  So that I can handle mail in both development and production

  Background:
    Given I have a Buffalo application with Buffkit wired

  Scenario: Send email via SMTP in production
    Given Buffkit is configured with SMTP settings
    And DevMode is false
    When I send an email with subject "Welcome" to "user@example.com"
    Then the email should be sent via SMTP
    And the SMTP sender should be used

  Scenario: Send email with HTML and text content
    Given I have a configured mail sender
    When I send an email with HTML content "<h1>Welcome</h1>" and text content "Welcome"
    Then the email should contain both HTML and text parts
    And the content should be properly formatted

  Scenario: Log email in development mode
    Given DevMode is true
    And no SMTP configuration is provided
    When I send an email with subject "Test Email"
    Then the email should not be sent via SMTP
    And the email should be logged to the development sender
    And the email should be stored for preview

  Scenario: Mail preview endpoint available in development
    Given DevMode is true
    And I have sent emails with subjects:
      | subject      |
      | First Email  |
      | Second Email |
      | Third Email  |
    When I visit "/__mail/preview"
    Then the response status should be 200
    And I should see "First Email"
    And I should see "Second Email"
    And I should see "Third Email"

  Scenario: Mail preview endpoint blocked in production
    Given DevMode is false
    When I visit "/__mail/preview"
    Then the response status should be 404

  Scenario: Mail preview shows email details
    Given DevMode is true
    And I have sent an email with:
      | field   | value               |
      | to      | user@example.com    |
      | subject | Test Subject        |
      | html    | <p>HTML Content</p> |
      | text    | Text Content        |
    When I visit "/__mail/preview"
    Then I should see "user@example.com"
    And I should see "Test Subject"
    And I should see the HTML preview
    And I should see the text preview

  Scenario: Send email with missing fields
    Given I have a configured mail sender
    When I send an email without a "to" address
    Then an error should be returned
    And the error should mention "recipient required"

  Scenario: SMTP configuration validation
    Given I configure SMTP with invalid settings
    When I try to send an email
    Then the send should fail
    And an appropriate error should be returned

  Scenario: Development sender stores multiple emails
    Given DevMode is true
    When I send 10 emails
    Then all 10 emails should be stored
    And I should be able to preview all of them

  Scenario: Use custom mail sender
    Given I register a custom mail sender
    When I send an email
    Then my custom sender should be used
    And the default sender should not be called

  Scenario: Mail sender context handling
    Given I have a configured mail sender
    When I send an email with a context that has a timeout
    Then the send operation should respect the context
    And timeout errors should be properly handled

  Scenario: Email with attachments placeholder
    Given I have a configured mail sender
    When I send an email with attachment metadata
    Then the attachment information should be included
    # Note: Actual attachment handling is not in v0.1 spec

  Scenario: Bulk email sending
    Given I have a configured mail sender
    When I send 100 emails in a loop
    Then all emails should be queued or sent
    And no emails should be lost

  Scenario: Mail sender graceful degradation
    Given no mail configuration is provided
    When I send an email in production mode
    Then the email should be logged to stdout
    And a warning should be logged about missing configuration
