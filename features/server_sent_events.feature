Feature: Server-Sent Events (SSE)
  As a developer using Buffkit
  I want to implement real-time server-to-client communication
  So that I can push updates to connected clients without polling

  Background:
    Given I have a Buffalo application with Buffkit wired

  @skip
  Scenario: Create SSE broker
    Given I have an SSE broker
    When a client connects to the SSE endpoint
    Then the client should be registered with the broker
    And the client should receive a connection event

  @skip
  Scenario: Broadcast event to all clients
    Given I have an SSE broker with multiple connected clients
    When I broadcast an event "update" with data "test message"
    Then all connected clients should receive the event
    And the event should have the correct type and data

  @skip
  Scenario: Client disconnection handling
    Given I have an SSE broker with a connected client
    When the client disconnects
    Then the client should be unregistered from the broker
    And no further events should be sent to that client

  @skip
  Scenario: Heartbeat mechanism
    Given I have an SSE broker with connected clients
    When the heartbeat interval passes
    Then all clients should receive a heartbeat event
    And the connection should remain active

  @skip
  Scenario: Selective event broadcasting
    Given I have an SSE broker with multiple clients in different groups
    When I broadcast an event to a specific group
    Then only clients in that group should receive the event

  @skip
  Scenario: Error handling
    Given I have an SSE broker with a connected client
    When an error occurs while sending an event
    Then the error should be logged
    And the client should be disconnected gracefully

  @skip
  Scenario: Reconnection handling
    Given I have an SSE broker
    And a client that was previously connected
    When the client reconnects with the same session ID
    Then the client should resume receiving events
    And any missed events should be replayed if configured
