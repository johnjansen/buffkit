# Cleanup Items

## Completed ✅
- [x] Database Migration Runner - Full implementation with up/down migrations
- [x] Migration transaction support for Postgres, SQLite, MySQL
- [x] Migration status tracking and rollback capabilities
- [x] Comprehensive migration tests with SQLite

## SSE Reconnection Implementation

### Immediate Cleanup Needed
- [ ] Add proper error handling for ring buffer operations
- [ ] Implement proper session ID validation/signing to prevent spoofing
- [ ] Add metrics/monitoring hooks for production use
- [ ] Implement Redis-based session storage for multi-server deployments
- [ ] Add rate limiting for reconnection attempts to prevent abuse

### Code Quality Improvements
- [ ] Extract magic numbers into named constants (e.g., timeouts, buffer sizes)
- [ ] Add comprehensive logging with levels (debug, info, warn, error)
- [ ] Implement proper context cancellation throughout the SSE pipeline
- [ ] Add connection pooling for better resource management
- [ ] Create integration tests for the full reconnection flow

### Performance Optimizations
- [ ] Implement lazy loading of buffered events
- [ ] Add compression for large event payloads
- [ ] Optimize ring buffer allocation strategy
- [ ] Implement event batching for high-throughput scenarios
- [ ] Add configurable backpressure mechanisms

### Security Enhancements
- [ ] Implement CSRF protection for SSE endpoints
- [ ] Add session token rotation on reconnection
- [ ] Implement IP-based validation for session ownership
- [ ] Add configurable max connections per session
- [ ] Implement event encryption for sensitive data

### Documentation Needs
- [ ] Add inline documentation for all public APIs
- [ ] Create usage examples for common scenarios
- [ ] Document configuration options and defaults
- [ ] Add troubleshooting guide for common issues
- [ ] Create performance tuning guide

### Testing Gaps
- [ ] Add unit tests for session manager
- [ ] Add unit tests for broker event handling
- [ ] Test buffer overflow scenarios
- [ ] Test rapid connect/disconnect cycles
- [ ] Add load testing for memory usage validation
- [ ] Test cross-server session handoff (Redis scenario)

### Integration Work
- [ ] Wire SSE handler into main Buffkit framework
- [ ] Add configuration options to Buffkit Config struct
- [ ] Create middleware for SSE authentication
- [ ] Integrate with existing auth system
- [ ] Add htmx-specific event formatting support

### Monitoring & Observability
- [ ] Add Prometheus metrics for:
  - Active connections
  - Buffer usage
  - Reconnection rate
  - Event throughput
  - Memory usage per session
- [ ] Add structured logging with correlation IDs
- [ ] Create health check endpoint for SSE subsystem
- [ ] Add distributed tracing support

### Technical Debt
- [ ] Refactor getAllSessionIDs() to properly iterate through SessionManager
- [ ] Improve event parsing in test client (currently simplified)
- [ ] Add proper cleanup for abandoned sessions
- [ ] Implement exponential backoff for reconnection attempts
- [ ] Add circuit breaker for failing clients

## Migration System Remaining Work

### Integration Tasks
- [ ] Create grift tasks for CLI migration commands
- [ ] Add buffalo generate migration command support
- [ ] Create more example migrations for common patterns
- [ ] Add migration validation before execution
- [ ] Support for seed data migrations

### Database-Specific Improvements
- [ ] Test with real PostgreSQL instance
- [ ] Test with real MySQL instance
- [ ] Add support for migration dependencies
- [ ] Implement dry-run mode for migrations
- [ ] Add migration performance monitoring

### Future Enhancements
- [ ] Add support for event replay from persistent storage
- [ ] Implement event compression for bandwidth optimization
- [ ] Add support for binary event data
- [ ] Create client SDK for easier integration
- [ ] Add WebSocket fallback for older browsers