# Regrets

## SSE Reconnection Implementation

### Things I Regret Doing

1. **Over-engineering the initial design**
   - Created too many abstractions before proving the concept works
   - Should have started with a simpler prototype and evolved it
   - The separation between Broker, SessionManager, and Handler added complexity

2. **Not starting with integration tests**
   - Built components in isolation without testing them together first
   - Should have written end-to-end tests before unit tests
   - Would have caught interface mismatches earlier

3. **Using ring buffer for event storage**
   - While memory-efficient, it's harder to debug than a simple slice
   - Makes it difficult to implement selective event replay
   - Should have started with a simple slice and optimized later

4. **Hardcoding timeout values**
   - 5-second timeout for event delivery is arbitrary
   - Should have made all timeouts configurable from the start
   - Now requires refactoring to make production-ready

5. **Not implementing interfaces first**
   - Concrete types make testing harder
   - Should have defined SessionStore and EventBuffer interfaces
   - Would have made Redis implementation easier to add later

### Things I Regret Not Doing

1. **Not implementing Redis support from the start**
   - The in-memory implementation doesn't scale horizontally
   - Adding Redis now requires significant refactoring
   - Should have abstracted storage from day one

2. **Not adding comprehensive logging**
   - Debug information is minimal
   - Hard to troubleshoot issues in production
   - Should have used structured logging throughout

3. **Not implementing rate limiting**
   - Clients can reconnect as often as they want
   - No protection against reconnection storms
   - Should have added exponential backoff

4. **Not creating a client SDK**
   - Clients have to implement SSE parsing themselves
   - No standard way to handle reconnection on client side
   - Should have provided JavaScript/TypeScript client library

5. **Not adding metrics collection**
   - No way to monitor system health in production
   - Can't track buffer usage or reconnection rates
   - Should have added Prometheus metrics from the start

6. **Not implementing event compression**
   - Large events consume significant bandwidth
   - No optimization for repeated data patterns
   - Should have added gzip compression for event data

7. **Not handling authentication properly**
   - Session IDs are not signed or encrypted
   - No integration with existing auth systems
   - Should have used JWT or similar for session tokens

8. **Not testing edge cases thoroughly**
   - What happens with 10,000 concurrent connections?
   - How does it behave under memory pressure?
   - Should have added load testing and chaos engineering tests

### Architectural Regrets

1. **Tight coupling to HTTP**
   - SSE is tied to HTTP transport
   - Hard to add WebSocket fallback
   - Should have abstracted the transport layer

2. **No event sourcing**
   - Events are ephemeral, lost after TTL
   - Can't replay historical events
   - Should have considered event store pattern

3. **No multi-tenancy support**
   - All sessions share the same namespace
   - No way to isolate different applications
   - Should have added tenant/namespace concept

4. **Missing circuit breaker pattern**
   - Failed clients can consume resources indefinitely
   - No automatic recovery from cascading failures
   - Should have implemented circuit breakers

### Process Regrets

1. **Not doing TDD properly**
   - Wrote implementation before tests
   - Tests are retrofitted, not driving design
   - Should have written failing tests first

2. **Not getting user feedback early**
   - Built features based on assumptions
   - No validation of actual use cases
   - Should have created MVP and tested with users

3. **Not documenting decisions**
   - Why ring buffer over slice?
   - Why 30-second TTL?
   - Should have created ADRs (Architecture Decision Records)

### Performance Regrets

1. **Not benchmarking early**
   - Don't know actual performance characteristics
   - No baseline for optimization
   - Should have added benchmarks with initial implementation

2. **Not optimizing the hot path**
   - Event broadcasting iterates through all clients
   - No optimization for large numbers of clients
   - Should have profiled and optimized critical paths

### To Address in Next Iteration

1. **Add Redis support** - Critical for production use
2. **Implement rate limiting** - Prevent abuse
3. **Add comprehensive logging** - Improve debuggability
4. **Create client SDKs** - Ease adoption
5. **Add metrics** - Enable monitoring
6. **Implement authentication** - Security requirement
7. **Add compression** - Reduce bandwidth
8. **Create load tests** - Validate scalability
9. **Abstract transport** - Future flexibility
10. **Document architecture** - Help future maintainers