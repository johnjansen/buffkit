Feature: Enhanced Authentication System
  As a web application user
  I want advanced authentication features
  So that I can securely manage my account

  Background:
    Given I have a Buffalo application with Buffkit wired
    And I have an extended user store configured
  # Registration Flow

  Scenario: User registration form is accessible
    When I visit "/register"
    Then I should see the registration form
    And the response status should be 200

  Scenario: User can register with valid credentials
    When I submit a registration with email "newuser@example.com" and password "SecurePass123!"
    Then a new user account should be created
    And I should receive a verification email
    And I should be redirected to a success page

  Scenario: Registration fails with weak password
    When I submit a registration with email "user@example.com" and password "weak"
    Then I should see an error message about password strength
    And no user account should be created

  Scenario: Registration fails with duplicate email
    Given a user exists with email "existing@example.com"
    When I submit a registration with email "existing@example.com"
    Then I should see an error message about email already taken
    And only one user should exist with that email
  # Email Verification

  Scenario: User can verify email with valid token
    Given I have registered but not verified my email
    And I have a valid verification token
    When I visit the verification link
    Then my account should be marked as verified
    And I should see a success message

  Scenario: Email verification fails with invalid token
    When I visit "/verify-email?token=invalid"
    Then I should see an error message
    And no accounts should be verified

  Scenario: Email verification fails with expired token
    Given I have a verification token older than 24 hours
    When I visit the verification link
    Then I should see an expiration error message
    And my account should remain unverified
  # Password Reset Flow

  Scenario: Forgot password form is accessible
    When I visit "/forgot-password"
    Then I should see the forgot password form
    And the response status should be 200

  Scenario: User can request password reset
    Given a user exists with email "user@example.com"
    When I submit a password reset request for "user@example.com"
    Then a password reset email should be sent
    And a reset token should be created
    And I should see a success message

  Scenario: Password reset silently succeeds for non-existent email
    When I submit a password reset request for "nonexistent@example.com"
    Then I should see a success message
    And no email should be sent
    And no reset token should be created

  Scenario: User can reset password with valid token
    Given I have a valid password reset token
    When I visit the reset password link
    And I submit a new password "NewSecurePass123!"
    Then my password should be updated
    And the reset token should be invalidated
    And I should be redirected to login

  Scenario: Password reset fails with mismatched passwords
    Given I have a valid password reset token
    When I visit the reset password link
    And I submit mismatched passwords
    Then I should see an error about passwords not matching
    And my password should not be changed

  Scenario: Password reset fails with expired token
    Given I have a password reset token older than 1 hour
    When I visit the reset password link
    Then I should see an expiration error
    And I should be redirected to forgot password
  # Profile Management

  Scenario: User can view their profile
    Given I am logged in as a valid user
    When I visit "/profile"
    Then I should see my profile information
    And the response status should be 200

  Scenario: User can update their profile
    Given I am logged in as a valid user
    When I visit "/profile"
    And I update my name to "New Name"
    And I submit the profile form
    Then my profile should be updated
    And I should see a success message

  Scenario: Profile page requires authentication
    Given I am not logged in
    When I visit "/profile"
    Then I should be redirected to login
  # Session Management

  Scenario: User can view active sessions
    Given I am logged in as a valid user
    When I visit "/sessions"
    Then I should see my active sessions
    And I should see session details like IP and user agent

  Scenario: User can revoke a session
    Given I am logged in with multiple sessions
    When I revoke a specific session
    Then that session should be invalidated
    And I should see a success message

  Scenario: Sessions page requires authentication
    Given I am not logged in
    When I visit "/sessions"
    Then I should be redirected to login
  # Rate Limiting

  Scenario: Login attempts are rate limited
    When I make 5 failed login attempts within 1 minute
    Then subsequent login attempts should be blocked
    And I should see a rate limit error message

  Scenario: Registration attempts are rate limited
    When I make 3 registration attempts within 1 minute
    Then subsequent registration attempts should be blocked
    And I should see a rate limit error message
  # Account Security

  Scenario: Account is locked after too many failed attempts
    Given a user exists with email "user@example.com"
    When I make 5 failed login attempts for that user
    Then the account should be locked
    And valid credentials should also fail
    And I should see an account locked message

  Scenario: Locked account is unlocked after timeout
    Given an account is locked until 5 minutes ago
    When I attempt to login with valid credentials
    Then the login should succeed
    And the account should be unlocked
  # Audit Logging

  Scenario: Login attempts are audit logged
    When I attempt to login
    Then an audit log entry should be created
    And it should include timestamp, IP, and result

  Scenario: Password changes are audit logged
    Given I am logged in as a valid user
    When I change my password
    Then an audit log entry should be created
    And it should record the password change event
  # Background Jobs

  Scenario: Session cleanup job removes expired sessions
    Given there are expired sessions older than 24 hours
    When the session cleanup job runs
    Then expired sessions should be deleted
    And active sessions should remain

  Scenario: Account unlock job unlocks expired locks
    Given there are accounts with expired lock times
    When the account unlock job runs
    Then those accounts should be unlocked
    And recently locked accounts should remain locked
  # Email Integration

  Scenario: Verification emails are sent through mail system
    When I register a new account
    Then the mail system should receive a send request
    And the email should contain a verification link

  Scenario: Password reset emails are sent through mail system
    When I request a password reset
    Then the mail system should receive a send request
    And the email should contain a reset link
  # Remember Me

  Scenario: Remember me cookie extends session
    When I login with remember me checked
    Then a persistent cookie should be set
    And my session should persist across browser restarts

  Scenario: Regular login creates session cookie only
    When I login without remember me checked
    Then only a session cookie should be set
    And my session should end when browser closes
  # Security Headers

  Scenario: Authentication pages have security headers
    When I visit "/login"
    Then the response should include security headers
    And CSP should prevent inline scripts
    And X-Frame-Options should prevent clickjacking
  # Password Strength

  Scenario: Password strength is validated on registration
    When I try to register with password "password"
    Then I should see password strength requirements
    And the registration should fail

  Scenario: Password strength is validated on reset
    Given I have a valid password reset token
    When I try to set password "12345678"
    Then I should see password strength requirements
    And the password should not be changed
  # Multi-device Support

  Scenario: User can be logged in on multiple devices
    Given I am logged in on device A
    When I login on device B
    Then both sessions should be active
    And I should see both devices in sessions list

  Scenario: Revoking session logs out specific device
    Given I am logged in on multiple devices
    When I revoke the session for device B
    Then device B should be logged out
    And device A should remain logged in
