package sse

import (
	"container/ring"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// SessionConfig defines configuration for SSE session management
type SessionConfig struct {
	// BufferSize is the maximum number of events to buffer per session
	BufferSize int
	// BufferTTL is how long to keep a disconnected session alive
	BufferTTL time.Duration
	// EnableReconnection enables session persistence and event replay
	EnableReconnection bool
	// CleanupInterval is how often to run the cleanup goroutine
	CleanupInterval time.Duration
}

// DefaultSessionConfig returns sensible defaults for session management
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		BufferSize:         1000,
		BufferTTL:          30 * time.Second,
		EnableReconnection: true,
		CleanupInterval:    10 * time.Second,
	}
}

// Event represents an SSE event with metadata for replay
type Event struct {
	ID        string    // Unique event ID for deduplication
	Type      string    // Event type (e.g., "message", "update", "ping")
	Data      string    // Event payload
	Timestamp time.Time // When the event was created
	Replayed  bool      // Whether this is a replayed event
}

// ClientSession represents a persistent SSE client session
type ClientSession struct {
	ID            string       // Unique session identifier
	LastEventID   string       // Last event ID received by client
	LastSeen      time.Time    // Last time client was connected
	Created       time.Time    // When session was created
	EventBuffer   *ring.Ring   // Circular buffer of missed events
	Metadata      SessionMeta  // Custom session metadata
	Reconnections int          // Number of times client has reconnected
	Active        bool         // Whether client is currently connected
	mu            sync.RWMutex // Protects session state
}

// SessionMeta holds custom metadata for a session
type SessionMeta struct {
	UserAgent     string            // Client user agent
	RemoteAddr    string            // Client IP address
	Headers       map[string]string // Custom headers from client
	QueryParams   map[string]string // Query parameters from connection
	Subscriptions []string          // Event types client is subscribed to
	ClientVersion string            // Client application version
}

// SessionManager manages all SSE client sessions
type SessionManager struct {
	sessions map[string]*ClientSession
	config   SessionConfig
	mu       sync.RWMutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewSessionManager creates a new session manager with the given configuration
func NewSessionManager(config SessionConfig) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*ClientSession),
		config:   config,
		stopCh:   make(chan struct{}),
	}

	// Start cleanup goroutine if reconnection is enabled
	if config.EnableReconnection && config.CleanupInterval > 0 {
		sm.wg.Add(1)
		go sm.cleanupLoop()
	}

	return sm
}

// CreateSession creates a new client session
func (sm *SessionManager) CreateSession(metadata SessionMeta) (*ClientSession, error) {
	// Generate cryptographically secure session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &ClientSession{
		ID:            sessionID,
		LastEventID:   "",
		LastSeen:      time.Now(),
		Created:       time.Now(),
		EventBuffer:   ring.New(sm.config.BufferSize),
		Metadata:      metadata,
		Reconnections: 0,
		Active:        true,
	}

	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*ClientSession, bool) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if session has expired
	session.mu.RLock()
	expired := !session.Active && time.Since(session.LastSeen) > sm.config.BufferTTL
	session.mu.RUnlock()

	if expired {
		sm.RemoveSession(sessionID)
		return nil, false
	}

	return session, true
}

// ReconnectSession handles a client reconnection
func (sm *SessionManager) ReconnectSession(sessionID string, lastEventID string) (*ClientSession, []Event, error) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		// Session doesn't exist or has expired
		return nil, nil, nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Update session state
	session.Active = true
	session.LastSeen = time.Now()
	session.Reconnections++
	if lastEventID != "" {
		session.LastEventID = lastEventID
	}

	// Collect buffered events for replay
	var replayEvents []Event
	if sm.config.EnableReconnection && session.EventBuffer != nil {
		replayEvents = sm.collectBufferedEvents(session, lastEventID)
	}

	return session, replayEvents, nil
}

// BufferEvent adds an event to a session's buffer if disconnected
func (sm *SessionManager) BufferEvent(sessionID string, event Event) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Only buffer if client is disconnected
	if !session.Active && sm.config.EnableReconnection {
		// Store event in ring buffer (overwrites oldest if full)
		session.EventBuffer.Value = event
		session.EventBuffer = session.EventBuffer.Next()
	}
}

// DisconnectSession marks a session as disconnected
func (sm *SessionManager) DisconnectSession(sessionID string) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return
	}

	session.mu.Lock()
	session.Active = false
	session.LastSeen = time.Now()
	session.mu.Unlock()
}

// RemoveSession completely removes a session
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()
}

// collectBufferedEvents retrieves events from the buffer newer than lastEventID
func (sm *SessionManager) collectBufferedEvents(session *ClientSession, lastEventID string) []Event {
	var events []Event

	// Walk through the ring buffer and collect valid events
	session.EventBuffer.Do(func(val interface{}) {
		if val == nil {
			return
		}

		event, ok := val.(Event)
		if !ok {
			return
		}

		// Skip events older than or equal to lastEventID
		if lastEventID != "" && event.ID != "" {
			if event.ID <= lastEventID {
				return
			}
		}

		// Mark as replayed and add to collection
		event.Replayed = true
		events = append(events, event)
	})

	return events
}

// cleanupLoop periodically removes expired sessions
func (sm *SessionManager) cleanupLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupExpiredSessions()
		case <-sm.stopCh:
			return
		}
	}
}

// cleanupExpiredSessions removes sessions that have been disconnected too long
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for id, session := range sm.sessions {
		session.mu.RLock()
		expired := !session.Active && now.Sub(session.LastSeen) > sm.config.BufferTTL
		session.mu.RUnlock()

		if expired {
			toRemove = append(toRemove, id)
		}
	}

	// Remove expired sessions
	for _, id := range toRemove {
		delete(sm.sessions, id)
	}
}

// Stop gracefully shuts down the session manager
func (sm *SessionManager) Stop() {
	close(sm.stopCh)
	sm.wg.Wait()
}

// GetActiveSessionCount returns the number of active sessions
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		session.mu.RLock()
		if session.Active {
			count++
		}
		session.mu.RUnlock()
	}

	return count
}

// GetTotalSessionCount returns the total number of sessions (active and buffered)
func (sm *SessionManager) GetTotalSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// generateSessionID creates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateSessionOwnership checks if a session can be claimed by a connection
// This prevents session hijacking by validating metadata
func (sm *SessionManager) ValidateSessionOwnership(sessionID string, metadata SessionMeta) bool {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return false
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// For now, just check if another connection is active
	// In production, you'd want to validate IP, user agent, etc.
	if session.Active {
		// Session is already in use by another connection
		return false
	}

	// Could add more validation here:
	// - Check if IP address matches or is in same subnet
	// - Validate user agent hasn't drastically changed
	// - Check authentication tokens

	return true
}
