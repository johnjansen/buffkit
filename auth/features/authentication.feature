Feature: Authentication System
  As a web application user
  I want to authenticate with the system
  So that I can access protected resources

  Background:
    Given I have a Buffalo application with Buffkit wired

  Scenario: Accessing login form
    When I visit "/login"
    Then I should see the login form
    And the response status should be 200

  Scenario: Login form accepts POST requests
    When I submit a POST request to "/login"
    Then the route should exist
    And the response should not be 404

  Scenario: Logout accepts POST requests
    When I submit a POST request to "/logout"
    Then the route should exist
    And the response should not be 404

  Scenario: Protected routes require authentication
    Given I have a handler that requires login
    When I access the protected route without authentication
    Then I should be redirected to login

  Scenario: RequireLogin middleware exists
    When I apply the RequireLogin middleware to a handler
    Then the middleware should be callable
    And it should return a handler function

  Scenario: Authenticated users can access protected routes
    Given I am logged in as a valid user
    When I access a protected route
    Then I should see the protected content
    And I should not be redirected

  Scenario: User context is available in protected routes
    Given I am logged in as a valid user
    When I access a protected route
    Then the current user should be available in the context
    And I can access user information
