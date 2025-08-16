package sse

import (
	"testing"
	"time"
)

func TestSessionManager_CreateSession(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	metadata := SessionMeta{
		UserAgent:  "test-agent",
		RemoteAddr: "127.0.0.1",
	}

	session, err := sm.CreateSession(metadata)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.Metadata.UserAgent != "test-agent" {
		t.Errorf("Expected UserAgent 'test-agent', got '%s'", session.Metadata.UserAgent)
	}

	if !session.Active {
		t.Error("New session should be active")
	}

	if session.Reconnections != 0 {
		t.Error("New session should have 0 reconnections")
	}
}

func TestSessionManager_GetSession(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create a session
	metadata := SessionMeta{UserAgent: "test"}
	session, _ := sm.CreateSession(metadata)

	// Retrieve the session
	retrieved, exists := sm.GetSession(session.ID)
	if !exists {
		t.Fatal("Session should exist")
	}

	if retrieved.ID != session.ID {
		t.Error("Retrieved session ID doesn't match")
	}

	// Try to get non-existent session
	_, exists = sm.GetSession("non-existent")
	if exists {
		t.Error("Non-existent session should not be found")
	}
}

func TestSessionManager_ReconnectSession(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create and disconnect a session
	metadata := SessionMeta{UserAgent: "test"}
	session, _ := sm.CreateSession(metadata)
	sessionID := session.ID

	// Add some events to buffer
	events := []Event{
		{ID: "1", Type: "test", Data: "event1"},
		{ID: "2", Type: "test", Data: "event2"},
		{ID: "3", Type: "test", Data: "event3"},
	}

	// Buffer events while disconnected
	sm.DisconnectSession(sessionID)
	for _, event := range events {
		sm.BufferEvent(sessionID, event)
	}

	// Reconnect
	reconnected, replayed, err := sm.ReconnectSession(sessionID, "")
	if err != nil {
		t.Fatalf("Reconnection failed: %v", err)
	}

	if reconnected == nil {
		t.Fatal("Should have reconnected session")
	}

	if reconnected.Reconnections != 1 {
		t.Errorf("Expected 1 reconnection, got %d", reconnected.Reconnections)
	}

	if len(replayed) != len(events) {
		t.Errorf("Expected %d replayed events, got %d", len(events), len(replayed))
	}

	if !reconnected.Active {
		t.Error("Reconnected session should be active")
	}
}

func TestSessionManager_BufferEvent(t *testing.T) {
	config := SessionConfig{
		BufferSize:         3, // Small buffer for testing
		BufferTTL:          30 * time.Second,
		EnableReconnection: true,
		CleanupInterval:    10 * time.Second,
	}
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create and disconnect a session
	metadata := SessionMeta{UserAgent: "test"}
	session, _ := sm.CreateSession(metadata)
	sm.DisconnectSession(session.ID)

	// Buffer more events than buffer size
	for i := 0; i < 5; i++ {
		event := Event{
			ID:   string(rune('A' + i)),
			Type: "test",
			Data: "data",
		}
		sm.BufferEvent(session.ID, event)
	}

	// Reconnect and check we only get the last 3 events
	_, replayed, _ := sm.ReconnectSession(session.ID, "")

	if len(replayed) > 3 {
		t.Errorf("Expected max 3 events, got %d", len(replayed))
	}

	// The oldest events should have been dropped
	for _, event := range replayed {
		if event.ID == "A" || event.ID == "B" {
			t.Error("Old events should have been dropped from buffer")
		}
	}
}

func TestSessionManager_SessionCleanup(t *testing.T) {
	config := SessionConfig{
		BufferSize:         100,
		BufferTTL:          100 * time.Millisecond, // Short TTL for testing
		EnableReconnection: true,
		CleanupInterval:    50 * time.Millisecond, // Fast cleanup for testing
	}
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create and disconnect a session
	metadata := SessionMeta{UserAgent: "test"}
	session, _ := sm.CreateSession(metadata)
	sessionID := session.ID
	sm.DisconnectSession(sessionID)

	// Session should exist initially
	_, exists := sm.GetSession(sessionID)
	if !exists {
		t.Fatal("Session should exist initially")
	}

	// Wait for cleanup to trigger
	time.Sleep(200 * time.Millisecond)

	// Session should be cleaned up
	_, exists = sm.GetSession(sessionID)
	if exists {
		t.Error("Session should have been cleaned up after TTL")
	}
}

func TestSessionManager_ValidateSessionOwnership(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create an active session
	metadata := SessionMeta{
		UserAgent:  "test-agent",
		RemoteAddr: "127.0.0.1",
	}
	session, _ := sm.CreateSession(metadata)

	// Try to claim active session (should fail)
	newMetadata := SessionMeta{
		UserAgent:  "different-agent",
		RemoteAddr: "192.168.1.1",
	}
	valid := sm.ValidateSessionOwnership(session.ID, newMetadata)
	if valid {
		t.Error("Should not be able to claim active session")
	}

	// Disconnect the session
	sm.DisconnectSession(session.ID)

	// Now should be able to claim it
	valid = sm.ValidateSessionOwnership(session.ID, newMetadata)
	if !valid {
		t.Error("Should be able to claim disconnected session")
	}

	// Non-existent session should fail
	valid = sm.ValidateSessionOwnership("non-existent", newMetadata)
	if valid {
		t.Error("Should not validate non-existent session")
	}
}

func TestSessionManager_GetCounts(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Initially no sessions
	if sm.GetActiveSessionCount() != 0 {
		t.Error("Should have 0 active sessions initially")
	}

	if sm.GetTotalSessionCount() != 0 {
		t.Error("Should have 0 total sessions initially")
	}

	// Create some sessions
	metadata := SessionMeta{UserAgent: "test"}
	session1, _ := sm.CreateSession(metadata)
	session2, _ := sm.CreateSession(metadata)
	sm.CreateSession(metadata)

	if sm.GetActiveSessionCount() != 3 {
		t.Errorf("Expected 3 active sessions, got %d", sm.GetActiveSessionCount())
	}

	if sm.GetTotalSessionCount() != 3 {
		t.Errorf("Expected 3 total sessions, got %d", sm.GetTotalSessionCount())
	}

	// Disconnect some sessions
	sm.DisconnectSession(session1.ID)
	sm.DisconnectSession(session2.ID)

	if sm.GetActiveSessionCount() != 1 {
		t.Errorf("Expected 1 active session, got %d", sm.GetActiveSessionCount())
	}

	if sm.GetTotalSessionCount() != 3 {
		t.Errorf("Expected 3 total sessions, got %d", sm.GetTotalSessionCount())
	}
}

func TestSessionManager_ReconnectWithLastEventID(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create session
	metadata := SessionMeta{UserAgent: "test"}
	session, _ := sm.CreateSession(metadata)
	sessionID := session.ID

	// Disconnect and buffer events
	sm.DisconnectSession(sessionID)

	events := []Event{
		{ID: "100", Type: "test", Data: "event1"},
		{ID: "200", Type: "test", Data: "event2"},
		{ID: "300", Type: "test", Data: "event3"},
		{ID: "400", Type: "test", Data: "event4"},
	}

	for _, event := range events {
		sm.BufferEvent(sessionID, event)
	}

	// Reconnect with lastEventID = "200"
	_, replayed, err := sm.ReconnectSession(sessionID, "200")
	if err != nil {
		t.Fatalf("Reconnection failed: %v", err)
	}

	// Should only get events after "200"
	expectedCount := 2 // events "300" and "400"
	if len(replayed) != expectedCount {
		t.Errorf("Expected %d replayed events, got %d", expectedCount, len(replayed))
	}

	// Verify we got the correct events
	for _, event := range replayed {
		if event.ID == "100" || event.ID == "200" {
			t.Errorf("Should not replay event %s", event.ID)
		}
		if !event.Replayed {
			t.Error("Replayed events should be marked as replayed")
		}
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("Failed to generate session ID: %v", err)
		}

		if id == "" {
			t.Error("Generated ID should not be empty")
		}

		if len(id) != 32 { // 16 bytes hex encoded = 32 chars
			t.Errorf("Expected ID length 32, got %d", len(id))
		}

		if ids[id] {
			t.Error("Generated duplicate session ID")
		}
		ids[id] = true
	}
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Test concurrent session creation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			metadata := SessionMeta{
				UserAgent:  "test-agent",
				RemoteAddr: "127.0.0.1",
			}
			_, err := sm.CreateSession(metadata)
			if err != nil {
				t.Errorf("Failed to create session: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 sessions
	if sm.GetTotalSessionCount() != 10 {
		t.Errorf("Expected 10 sessions, got %d", sm.GetTotalSessionCount())
	}
}

func TestSessionMeta_Preservation(t *testing.T) {
	config := DefaultSessionConfig()
	sm := NewSessionManager(config)
	defer sm.Stop()

	// Create session with rich metadata
	metadata := SessionMeta{
		UserAgent:     "Mozilla/5.0",
		RemoteAddr:    "192.168.1.100",
		Headers:       map[string]string{"X-Custom": "value"},
		QueryParams:   map[string]string{"filter": "active"},
		Subscriptions: []string{"updates", "notifications"},
		ClientVersion: "1.2.3",
	}

	session, _ := sm.CreateSession(metadata)

	// Verify metadata is preserved
	if session.Metadata.UserAgent != metadata.UserAgent {
		t.Error("UserAgent not preserved")
	}

	if session.Metadata.RemoteAddr != metadata.RemoteAddr {
		t.Error("RemoteAddr not preserved")
	}

	if session.Metadata.Headers["X-Custom"] != "value" {
		t.Error("Headers not preserved")
	}

	if session.Metadata.QueryParams["filter"] != "active" {
		t.Error("QueryParams not preserved")
	}

	if len(session.Metadata.Subscriptions) != 2 {
		t.Error("Subscriptions not preserved")
	}

	if session.Metadata.ClientVersion != "1.2.3" {
		t.Error("ClientVersion not preserved")
	}
}
