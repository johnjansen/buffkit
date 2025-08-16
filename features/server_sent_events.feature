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

  Scenario: Direct broker testing - client registration
    Given I have an SSE broker
    When I register a mock client
    Then the broker should track the client
    And the client count should increase

  Scenario: Direct broker testing - client unregistration
    Given I have an SSE broker with a connected client
    When I unregister the client
    Then the broker should remove the client
    And the client count should decrease

  Scenario: Direct broker testing - event broadcasting
    Given I have an SSE broker with multiple clients
    When I broadcast an event directly to the broker
    Then all clients should receive the event in their channels
    And the event should contain the correct data

  Scenario: Direct broker testing - heartbeat system
    Given I have an SSE broker with connected clients
    When the heartbeat timer triggers
    Then all clients should receive a heartbeat event
    And connections should remain alive
