# SSE Reconnection with State Recovery

## Overview

This package implements a robust Server-Sent Events (SSE) system with automatic reconnection and state recovery capabilities. It solves the fundamental problem of maintaining real-time connections across network interruptions, browser refreshes, and server restarts.

## The Hard Problem We Solved

Traditional SSE implementations lose events when clients disconnect. Our solution provides:

- **Persistent Sessions**: Clients maintain identity across reconnections
- **Event Buffering**: Missed events are stored and replayed on reconnection
- **Automatic Recovery**: Seamless reconnection without data loss
- **Memory-Bounded**: Ring buffers prevent unbounded memory growth
- **Production-Ready**: Handles edge cases like rapid reconnects and buffer overflow

## Architecture

### Core Components

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Handler   │────▶│    Broker    │────▶│   Session   │
│  (HTTP/SSE) │     │  (Routing)   │     │  Manager    │
└─────────────┘     └──────────────┘     └─────────────┘
       │                    │                     │
       ▼                    ▼                     ▼
   [Clients]          [Active Conns]        [Buffers]
```

### Key Features

1. **Session Management**
   - Cryptographically secure session IDs
   - Configurable buffer size and TTL
   - Automatic cleanup of abandoned sessions

2. **Event Buffering**
   - Ring buffer for memory-bounded storage
   - Event sequence numbering
   - Selective replay based on Last-Event-ID

3. **Connection Handling**
   - Cookie and header-based session persistence
   - Graceful degradation when buffers disabled
   - Heartbeat system to maintain connections

## Usage

### Basic Setup

```go
import "github.com/johnjansen/buffkit/sse"

// Configure SSE with reconnection support
config := sse.SessionConfig{
    BufferSize:         1000,              // Events per session
    BufferTTL:          30 * time.Second,  // Keep disconnected sessions
    EnableReconnection: true,
    CleanupInterval:    10 * time.Second,
}

// Create handler
handler := sse.NewHandler(config)

// Register routes
http.HandleFunc("/events", handler.ServeHTTP)

// Broadcast events
handler.Broadcast("update", `{"message":"Hello World"}`)

// Target specific clients
handler.SendToClient(sessionID, "notification", `{"alert":"Important"}`)
```

### Client-Side Connection

```javascript
const eventSource = new EventSource('/events', {
    withCredentials: true  // Include cookies for session persistence
});

eventSource.addEventListener('connected', (e) => {
    const data = JSON.parse(e.data);
    console.log('Connected with session:', data.sessionId);
});

eventSource.addEventListener('message', (e) => {
    const data = JSON.parse(e.data);
    if (data._replayed) {
        console.log('Replayed event from', new Date(data._originalTime * 1000));
    }
});

eventSource.onerror = (e) => {
    // EventSource automatically reconnects
    console.log('Connection lost, reconnecting...');
};
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `BufferSize` | 1000 | Maximum events to buffer per session |
| `BufferTTL` | 30s | How long to keep disconnected sessions |
| `EnableReconnection` | true | Enable session persistence and replay |
| `CleanupInterval` | 10s | How often to clean expired sessions |

## Performance Characteristics

- **Memory Usage**: O(clients × buffer_size × event_size)
- **Event Broadcasting**: O(active_clients)
- **Reconnection**: O(buffer_size) for replay
- **Cleanup**: O(total_sessions) every cleanup interval

## Testing

### Unit Tests
```bash
go test ./sse/...
```

### BDD Scenarios
```bash
cd features
go test -v -run TestSSEReconnection
```

### Load Testing
```go
// See sse/benchmark_test.go for performance tests
go test -bench=. ./sse/...
```

## Production Considerations

### Scaling Horizontally

For multi-server deployments, implement Redis-based session storage:

```go
// TODO: Implement RedisSessionManager
sessionManager := sse.NewRedisSessionManager(redisClient, config)
```

### Security

1. **Session Validation**: Implement IP and user-agent checking
2. **Rate Limiting**: Add exponential backoff for reconnections
3. **Authentication**: Integrate with your auth system
4. **HTTPS Only**: Always use TLS in production

### Monitoring

Track these metrics:
- Active connections
- Buffer utilization
- Reconnection rate
- Memory usage per session
- Event throughput

## Limitations

1. **In-Memory Only**: Current implementation doesn't persist across server restarts
2. **No Compression**: Large events consume bandwidth
3. **No Binary Data**: Text-only events (SSE limitation)
4. **Single Server**: Requires Redis for multi-server setup

## Future Enhancements

- [ ] Redis session storage for horizontal scaling
- [ ] Event compression for bandwidth optimization
- [ ] Client SDKs for easier integration
- [ ] Prometheus metrics integration
- [ ] WebSocket fallback support
- [ ] Event replay from persistent storage

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Part of the Buffkit framework. See [LICENSE](../../LICENSE) for details.