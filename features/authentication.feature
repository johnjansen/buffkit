Feature: Authentication System
  As a developer using Buffkit
  I want to implement user authentication
  So that I can secure my application and manage user sessions

  Background:
    Given I have a Buffalo application with Buffkit wired

  @skip
  Scenario: User registration
    Given I have an authentication system configured
    When a new user registers with email "user@example.com" and password "SecurePass123!"
    Then the user should be created in the database
    And the password should be securely hashed
    And a confirmation email should be sent

  @skip
  Scenario: User login with valid credentials
    Given I have a registered user with email "user@example.com"
    When the user logs in with correct credentials
    Then a session should be created
    And the user should be redirected to the dashboard
    And the session should contain the user ID

  @skip
  Scenario: User login with invalid credentials
    Given I have a registered user with email "user@example.com"
    When the user logs in with incorrect password
    Then the login should fail
    And an error message should be displayed
    And no session should be created

  @skip
  Scenario: Session management
    Given a user is logged in with an active session
    When the user accesses a protected route
    Then the session should be validated
    And the user should have access to the resource

  @skip
  Scenario: User logout
    Given a user is logged in
    When the user logs out
    Then the session should be destroyed
    And the user should be redirected to the login page
    And protected routes should no longer be accessible

  @skip
  Scenario: Password reset request
    Given I have a registered user
    When the user requests a password reset
    Then a reset token should be generated
    And a reset email should be sent with the token
    And the token should expire after a set time

  @skip
  Scenario: Password reset completion
    Given a user has a valid password reset token
    When the user submits a new password
    Then the password should be updated
    And the reset token should be invalidated
    And the user should be able to login with the new password

  @skip
  Scenario: Session timeout
    Given a user is logged in
    When the session timeout period expires
    Then the session should be invalidated
    And the user should be redirected to login on next request

  @skip
  Scenario: Remember me functionality
    Given a user logs in with "remember me" checked
    Then a persistent cookie should be created
    And the session should be restored on browser restart
    And the cookie should have appropriate security flags

  @skip
  Scenario: Concurrent session limiting
    Given a user is already logged in on one device
    When the same user logs in on another device
    Then the previous session should be invalidated
    And only the new session should be active
