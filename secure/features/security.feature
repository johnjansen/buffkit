Feature: Security Middleware
  As a developer using Buffkit
  I want security headers and protections applied automatically
  So that my application is secure by default

  Background:
    Given I have a Buffalo application with Buffkit wired

  Scenario: Security headers are applied to all responses
    When I make a GET request to any endpoint
    Then the response should include header "X-Frame-Options" with value "DENY"
    And the response should include header "X-Content-Type-Options" with value "nosniff"
    And the response should include header "X-XSS-Protection" with value "1; mode=block"
    And the response should include header "Strict-Transport-Security"

  Scenario: CSRF protection on POST requests
    Given CSRF middleware is enabled
    When I POST to "/api/update" without a CSRF token
    Then the response status should be 403
    And the response should contain "CSRF token missing"

  Scenario: CSRF token validation success
    Given CSRF middleware is enabled
    And I have a valid CSRF token
    When I POST to "/api/update" with the CSRF token
    Then the request should be processed successfully
    And the response status should be 200

  Scenario: CSRF token generation
    Given CSRF middleware is enabled
    When I request a form page
    Then a CSRF token should be generated
    And the token should be available in the template context
    And the token should be cryptographically secure

  Scenario: CSRF exemption for safe methods
    Given CSRF middleware is enabled
    When I make a GET request
    Then CSRF validation should be skipped
    When I make a HEAD request
    Then CSRF validation should be skipped
    When I make an OPTIONS request
    Then CSRF validation should be skipped

  Scenario: Content Security Policy in production
    Given DevMode is false
    When I make a request to any endpoint
    Then the response should include a Content-Security-Policy header
    And the CSP should restrict inline scripts
    And the CSP should restrict inline styles

  Scenario: Relaxed security headers in development
    Given DevMode is true
    When I make a request to any endpoint
    Then security headers should be present
    But the Content-Security-Policy should allow localhost
    And the Strict-Transport-Security should be omitted

  Scenario: Rate limiting prevents abuse
    Given rate limiting is set to 60 requests per minute
    When I make 61 requests within 1 minute
    Then the 61st request should be rate limited
    And the response status should be 429
    And the response should include "Retry-After" header

  Scenario: Rate limiting by IP address
    Given rate limiting is enabled
    When requests come from different IP addresses
    Then each IP should have its own rate limit
    And limits should be tracked independently

  Scenario: X-Frame-Options prevents clickjacking
    When I request a page
    Then the response should include "X-Frame-Options: DENY"
    And the page should not be frameable

  Scenario: HSTS header in production
    Given DevMode is false
    When I make an HTTPS request
    Then the response should include Strict-Transport-Security
    And the max-age should be at least 31536000
    And includeSubDomains should be set

  Scenario: No HSTS in development
    Given DevMode is true
    When I make a request
    Then the Strict-Transport-Security header should be omitted

  Scenario: Custom security options
    Given I configure custom security options
    And I set X-Frame-Options to "SAMEORIGIN"
    When I make a request
    Then the response should include "X-Frame-Options: SAMEORIGIN"

  Scenario: Referrer Policy header
    When I make a request
    Then the response should include "Referrer-Policy" header
    And the value should be "strict-origin-when-cross-origin"

  Scenario: MIME type sniffing prevention
    When I serve any content
    Then the response should include "X-Content-Type-Options: nosniff"
    And browsers should not override the declared content-type

  Scenario: XSS protection header
    When I make a request
    Then the response should include "X-XSS-Protection: 1; mode=block"
    And the browser XSS filter should be enabled

  Scenario: CSRF token rotation
    Given CSRF middleware is enabled
    When I use a CSRF token successfully
    Then a new token should be generated for the next request
    And the old token should be invalidated

  Scenario: API endpoints with JSON content-type skip CSRF
    Given CSRF middleware is enabled
    When I POST JSON data with Content-Type "application/json"
    Then CSRF validation should be skipped
    And the request should be processed

  Scenario: File upload size limits
    Given security middleware is enabled
    When I upload a file larger than 10MB
    Then the request should be rejected
    And the response should indicate the size limit

  Scenario: SQL injection prevention through parameterized queries
    Given I have a SQL user store
    When I attempt to login with SQL injection in the email field
    Then the injection attempt should be safely handled
    And no SQL should be executed
    And the login should fail

  Scenario: Password hashing uses bcrypt
    When I hash a password
    Then bcrypt should be used
    And the cost factor should be at least 10
    And the hash should be salted

  Scenario: Session cookie security
    Given I am logged in
    When I inspect the session cookie
    Then the cookie should have HttpOnly flag
    And the cookie should have Secure flag in production
    And the cookie should have SameSite set to "Lax"

  Scenario: Clear security headers on static assets
    When I request a static CSS file
    Then basic security headers should be present
    But CSRF tokens should not be generated
