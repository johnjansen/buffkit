Feature: Server-Sent Events (SSE)
  As a web application developer
  I want to send real-time updates to connected clients
  So that I can provide live, interactive experiences

  Background:
    Given I have a Buffalo application with Buffkit wired

  Scenario: SSE endpoint is available
    When I connect to "/events" with SSE headers
    Then I should receive an SSE connection
    And the content type should be "text/event-stream"
    And the response status should be 200

  Scenario: Broadcasting events to all clients
    Given I have multiple clients connected to SSE
    When I broadcast an event "user-update" with data "Hello World"
    Then all connected clients should receive the event
    And the event type should be "user-update"
    And the event data should be "Hello World"

  Scenario: Client connection management
    Given I connect to the SSE endpoint
    When the connection is established
    Then I should receive heartbeat events
    And my connection should be tracked by the broker

  Scenario: Connection cleanup on disconnect
    Given I have a client connected to SSE
    When the client disconnects
    Then the broker should remove the connection
    And resources should be cleaned up

  Scenario: Broadcasting HTML fragments
    Given I have clients connected to SSE
    When I render a partial template and broadcast it
    Then clients should receive the rendered HTML
    And the HTML should be properly formatted

  Scenario: Event filtering and targeting
    Given I have multiple clients with different interests
    When I broadcast an event to specific clients
    Then only targeted clients should receive the event
    And other clients should not be affected

  Scenario: SSE with htmx integration
    Given I have an htmx-enabled page connected to SSE
    When I broadcast an update event
    Then the page content should update automatically
    And no page refresh should be required

  Scenario: Error handling in SSE connections
    Given I have a client connected to SSE
    When a broadcast error occurs
    Then the client connection should remain stable
    And the error should be logged appropriately

  Scenario: SSE broker lifecycle
    When I initialize a new SSE broker
    Then it should start the message handling goroutine
    And it should initialize the client tracking systems
    And it should be ready to accept connections
