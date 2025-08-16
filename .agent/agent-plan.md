# Agent Plan: SSE Reconnection with State Recovery

## Problem Statement
Implement a robust Server-Sent Events (SSE) reconnection mechanism with state recovery that handles:
- Client disconnections (network issues, browser refresh, etc.)
- Missed events during disconnection
- Client identity persistence across reconnections
- Event replay/catch-up on reconnection
- Race conditions between disconnect and reconnect

## Why This Is Hard
1. **State Management**: Need to track client state across disconnections
2. **Event Buffering**: Must buffer events for disconnected clients without memory leaks
3. **Identity**: Clients need persistent identity that survives reconnections
4. **Timing**: Handle rapid disconnect/reconnect cycles gracefully
5. **Cleanup**: Distinguish between temporary disconnects and permanent abandonment

## Implementation Strategy

### Phase 1: Client Identity System
- [ ] Implement persistent client IDs (using cookies/headers)
- [ ] Create client session tracking
- [ ] Build reconnection detection logic

### Phase 2: Event Buffering
- [ ] Create per-client event buffer with size limits
- [ ] Implement event sequence numbering
- [ ] Add TTL (time-to-live) for buffered events

### Phase 3: Reconnection Protocol
- [ ] Detect reconnection attempts
- [ ] Replay missed events from buffer
- [ ] Handle "last-event-id" header for catch-up

### Phase 4: Resource Management
- [ ] Implement buffer cleanup strategies
- [ ] Add configurable buffer sizes and TTLs
- [ ] Create monitoring/metrics for buffer usage

### Phase 5: BDD Testing
- [ ] Write comprehensive feature scenarios
- [ ] Test edge cases (rapid reconnects, buffer overflow)
- [ ] Validate memory usage and cleanup

## Technical Approach

### Client Identity
```go
type ClientSession struct {
    ID            string
    LastEventID   string
    LastSeen      time.Time
    EventBuffer   *ring.Buffer
    Reconnections int
}
```

### Event Buffering
- Use ring buffer with configurable size (e.g., 1000 events)
- Store events with timestamps and sequence numbers
- Clean up buffers for clients gone > 5 minutes

### Reconnection Flow
1. Client connects with session cookie/header
2. Server checks for existing session
3. If reconnecting, replay buffered events
4. Continue with live events

## Success Criteria
- Clients can reconnect after network interruption
- No events lost during brief disconnections (< 30s)
- Memory usage remains bounded
- Clean separation between temporary and permanent disconnects
- Full BDD test coverage

## Risks and Mitigations
- **Memory leak**: Use ring buffers and TTLs
- **Race conditions**: Use proper locking/channels
- **Client spoofing**: Sign session IDs cryptographically
- **Buffer overflow**: Drop oldest events, notify client

## Current Status
Starting implementation - no code written yet