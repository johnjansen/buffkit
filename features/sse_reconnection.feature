Feature: SSE Reconnection with State Recovery
  As a web application user
  I want my real-time connection to recover gracefully from disconnections
  So that I don't miss important updates even during network interruptions

  Background:
    Given I have a Buffalo application with Buffkit wired
    And SSE reconnection support is enabled with a 30 second buffer window

  @skip
  Scenario: Client receives a persistent session ID on first connection
    When I connect to the SSE endpoint for the first time
    Then I should receive a unique session ID in the response headers
    And the session ID should be stored as a secure cookie
    And the server should track my session in memory

  @skip
  Scenario: Graceful reconnection after brief network interruption
    Given I am connected to SSE with session ID "test-session-123"
    And I have received events with IDs "1", "2", "3"
    When my connection drops for 5 seconds
    And events "4", "5", "6" are broadcast during the disconnection
    And I reconnect with the same session ID
    Then I should receive the missed events "4", "5", "6" immediately
    And I should continue receiving new live events
    And the reconnection should be logged

  @skip
  Scenario: Reconnection with Last-Event-ID header
    Given I am connected to SSE with session ID "test-session-456"
    And I have received events up to ID "10"
    When I disconnect and events "11" through "15" are broadcast
    And I reconnect with Last-Event-ID header set to "10"
    Then I should receive events "11" through "15" in order
    And no events should be duplicated
    And the event sequence should be continuous

  @skip
  Scenario: Buffer overflow handling during extended disconnection
    Given I am connected to SSE with a buffer limit of 100 events
    When I disconnect for an extended period
    And 150 events are broadcast while disconnected
    And I reconnect with my session ID
    Then I should receive a special "buffer-overflow" event
    And I should receive the most recent 100 events
    And older events should be marked as dropped

  @skip
  Scenario: Session cleanup after abandonment timeout
    Given I am connected to SSE with session ID "test-session-789"
    When I disconnect and don't reconnect for 35 seconds
    Then my session should be cleaned up after 30 seconds
    And my event buffer should be freed
    And subsequent reconnection attempts should create a new session

  @skip
  Scenario: Rapid disconnect and reconnect cycles
    Given I am connected to SSE
    When I rapidly disconnect and reconnect 10 times within 2 seconds
    Then each reconnection should be handled gracefully
    And no events should be lost during the cycles
    And only one active connection should exist per session
    And connection thrashing should be logged

  @skip
  Scenario: Multiple clients with independent buffers
    Given client A is connected with session "session-A"
    And client B is connected with session "session-B"
    When client A disconnects
    And an event "shared-event" is broadcast to all clients
    And client B remains connected
    Then client B should receive "shared-event" immediately
    And client A should receive "shared-event" upon reconnection
    And the buffers should remain independent

  @skip
  Scenario: Reconnection after browser refresh
    Given I am connected to SSE with a session cookie
    When I refresh the browser page
    And the page reloads and re-establishes SSE connection
    Then I should reconnect with the same session ID from the cookie
    And I should receive any events missed during page reload
    And the transition should be seamless

  @skip
  Scenario: Event replay maintains correct order and timing
    Given I am connected and have received timestamped events
    When I disconnect and multiple events occur with timestamps
    And I reconnect requesting replay
    Then replayed events should maintain their original timestamps
    And replayed events should be marked with a "replayed" flag
    And the events should arrive in chronological order

  @skip
  Scenario: Memory usage remains bounded
    Given the server has 100 connected clients
    And each client has a buffer limit of 1000 events
    When 50 clients disconnect simultaneously
    And events continue to be broadcast
    Then memory usage should not exceed expected bounds
    And buffers should be cleaned up according to TTL
    And the server should remain responsive

  @skip
  Scenario: Client spoofing prevention
    Given a client is connected with session ID "legitimate-session"
    When another client attempts to connect with the same session ID
    Then the connection attempt should be rejected
    And a security event should be logged
    And the legitimate client should remain connected

  @skip
  Scenario: Graceful degradation when buffers are disabled
    Given SSE reconnection is configured with buffer size 0
    When a client disconnects and reconnects
    Then the client should receive only new events
    And no replay should occur
    And a "no-buffer" indicator should be sent

  @skip
  Scenario: Event deduplication during replay
    Given I am connected and tracking received event IDs
    When I disconnect after receiving events "1", "2", "3"
    And event "3" is still in the buffer when I reconnect
    And I reconnect with Last-Event-ID "2"
    Then I should receive event "3" only once
    And duplicate detection should prevent double delivery

  @skip
  Scenario: Connection state recovery with metadata
    Given I am connected with custom headers and query parameters
    When I disconnect and lose connection state
    And I reconnect with my session ID
    Then my connection metadata should be restored
    And subscription filters should be maintained
    And client preferences should persist

  @skip
  Scenario: Load balancer compatibility
    Given I am connected through a load balancer to server A
    When my connection drops and I reconnect through server B
    And both servers share session state via Redis
    Then I should successfully reconnect on server B
    And my buffered events should be available
    And the handoff should be transparent
