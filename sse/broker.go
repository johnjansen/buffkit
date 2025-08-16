package sse

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Client represents a connected SSE client with session support
type Client struct {
	SessionID    string        // Persistent session identifier
	EventChannel chan Event    // Channel for sending events to client
	Done         chan struct{} // Channel to signal client disconnection
	Request      *http.Request // Original HTTP request
	Writer       http.ResponseWriter
	Flusher      http.Flusher
}

// Message represents a broadcast message with targeting capabilities
type Message struct {
	Event     Event    // The event to broadcast
	TargetIDs []string // Specific session IDs to target (empty = broadcast to all)
}

// Broker manages SSE connections with reconnection support
type Broker struct {
	clients        map[string]*Client // Active clients by session ID
	register       chan *Client       // Channel for registering new clients
	unregister     chan string        // Channel for unregistering clients (by session ID)
	broadcast      chan Message       // Channel for broadcasting messages
	sessionManager *SessionManager    // Manages persistent sessions
	mu             sync.RWMutex       // Protects clients map
	stopCh         chan struct{}      // Signal to stop the broker
	wg             sync.WaitGroup     // Wait group for graceful shutdown
}

// NewBroker creates a new SSE broker with reconnection support
func NewBroker(config SessionConfig) *Broker {
	broker := &Broker{
		clients:        make(map[string]*Client),
		register:       make(chan *Client, 100),
		unregister:     make(chan string, 100),
		broadcast:      make(chan Message, 1000),
		sessionManager: NewSessionManager(config),
		stopCh:         make(chan struct{}),
	}

	// Start the main event loop
	broker.wg.Add(1)
	go broker.run()

	// Start heartbeat sender
	broker.wg.Add(1)
	go broker.heartbeat()

	return broker
}

// run is the main event loop for the broker
func (broker *Broker) run() {
	defer broker.wg.Done()

	for {
		select {
		case client := <-broker.register:
			broker.handleRegister(client)

		case sessionID := <-broker.unregister:
			broker.handleUnregister(sessionID)

		case message := <-broker.broadcast:
			broker.handleBroadcast(message)

		case <-broker.stopCh:
			broker.handleShutdown()
			return
		}
	}
}

// handleRegister processes new client registrations
func (broker *Broker) handleRegister(client *Client) {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	// Check if this is a reconnection
	session, replayEvents, err := broker.sessionManager.ReconnectSession(
		client.SessionID,
		client.Request.Header.Get("Last-Event-ID"),
	)

	if err != nil {
		log.Printf("Error reconnecting session %s: %v", client.SessionID, err)
		return
	}

	// If no existing session, create a new one
	if session == nil {
		metadata := SessionMeta{
			UserAgent:   client.Request.UserAgent(),
			RemoteAddr:  client.Request.RemoteAddr,
			Headers:     extractHeaders(client.Request),
			QueryParams: extractQueryParams(client.Request),
		}

		session, err = broker.sessionManager.CreateSession(metadata)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			return
		}

		client.SessionID = session.ID
	}

	// Store the active client
	broker.clients[client.SessionID] = client

	// Send session ID to client
	broker.sendSessionInfo(client, session)

	// Replay buffered events if this is a reconnection
	if len(replayEvents) > 0 {
		broker.replayEvents(client, replayEvents)
	}

	log.Printf("Client registered: session=%s, reconnections=%d, replayed=%d events",
		session.ID, session.Reconnections, len(replayEvents))
}

// handleUnregister processes client disconnections
func (broker *Broker) handleUnregister(sessionID string) {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	client, exists := broker.clients[sessionID]
	if !exists {
		return
	}

	// Mark session as disconnected (keeps buffer alive)
	broker.sessionManager.DisconnectSession(sessionID)

	// Clean up client resources
	close(client.Done)
	delete(broker.clients, sessionID)

	log.Printf("Client unregistered: session=%s", sessionID)
}

// handleBroadcast sends events to targeted or all clients
func (broker *Broker) handleBroadcast(message Message) {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	// Generate event ID if not set
	if message.Event.ID == "" {
		message.Event.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Set timestamp if not set
	if message.Event.Timestamp.IsZero() {
		message.Event.Timestamp = time.Now()
	}

	// Determine target clients
	var targetClients []*Client
	if len(message.TargetIDs) > 0 {
		// Send to specific clients
		for _, targetID := range message.TargetIDs {
			if client, exists := broker.clients[targetID]; exists {
				targetClients = append(targetClients, client)
			} else {
				// Client not connected, buffer the event
				broker.sessionManager.BufferEvent(targetID, message.Event)
			}
		}
	} else {
		// Broadcast to all connected clients
		for _, client := range broker.clients {
			targetClients = append(targetClients, client)
		}

		// Buffer for all disconnected sessions
		broker.bufferForDisconnectedSessions(message.Event)
	}

	// Send event to all target clients
	for _, client := range targetClients {
		select {
		case client.EventChannel <- message.Event:
			// Event sent successfully
		case <-time.After(5 * time.Second):
			// Client is not receiving events, disconnect them
			log.Printf("Client %s is not responding, disconnecting", client.SessionID)
			go func(sessionID string) {
				broker.unregister <- sessionID
			}(client.SessionID)
		}
	}
}

// bufferForDisconnectedSessions buffers an event for all disconnected sessions
func (broker *Broker) bufferForDisconnectedSessions(event Event) {
	// Get all sessions from the session manager
	totalSessions := broker.sessionManager.GetTotalSessionCount()
	activeSessions := len(broker.clients)

	// Only buffer if there are disconnected sessions
	if totalSessions > activeSessions {
		// This is a simplified approach - in production you'd want to
		// iterate through sessions more efficiently
		for sessionID := range broker.getAllSessionIDs() {
			if _, isActive := broker.clients[sessionID]; !isActive {
				broker.sessionManager.BufferEvent(sessionID, event)
			}
		}
	}
}

// getAllSessionIDs returns all known session IDs
// This is a simplified implementation - in production you'd want
// the SessionManager to expose this functionality
func (broker *Broker) getAllSessionIDs() map[string]bool {
	// For now, just return active clients
	// In a real implementation, SessionManager would track all sessions
	ids := make(map[string]bool)
	for id := range broker.clients {
		ids[id] = true
	}
	return ids
}

// sendSessionInfo sends session information to the client
func (broker *Broker) sendSessionInfo(client *Client, session *ClientSession) {
	// Send session ID as a special event
	sessionEvent := Event{
		ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		Type:      "session",
		Data:      fmt.Sprintf(`{"sessionId":"%s","reconnections":%d}`, session.ID, session.Reconnections),
		Timestamp: time.Now(),
	}

	select {
	case client.EventChannel <- sessionEvent:
		// Sent successfully
	case <-time.After(1 * time.Second):
		log.Printf("Failed to send session info to client %s", session.ID)
	}
}

// replayEvents sends buffered events to a reconnecting client
func (broker *Broker) replayEvents(client *Client, events []Event) {
	// Send a replay start marker
	replayStart := Event{
		ID:        fmt.Sprintf("replay-start-%d", time.Now().UnixNano()),
		Type:      "replay-start",
		Data:      fmt.Sprintf(`{"count":%d}`, len(events)),
		Timestamp: time.Now(),
	}

	select {
	case client.EventChannel <- replayStart:
	case <-time.After(1 * time.Second):
		return
	}

	// Send each buffered event
	for _, event := range events {
		select {
		case client.EventChannel <- event:
			// Small delay to avoid overwhelming the client
			time.Sleep(10 * time.Millisecond)
		case <-time.After(1 * time.Second):
			log.Printf("Failed to replay event to client %s", client.SessionID)
			return
		}
	}

	// Send a replay end marker
	replayEnd := Event{
		ID:        fmt.Sprintf("replay-end-%d", time.Now().UnixNano()),
		Type:      "replay-end",
		Data:      "{}",
		Timestamp: time.Now(),
	}

	select {
	case client.EventChannel <- replayEnd:
	case <-time.After(1 * time.Second):
		// Continue anyway
	}
}

// heartbeat sends periodic heartbeat events to keep connections alive
func (broker *Broker) heartbeat() {
	defer broker.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			heartbeatEvent := Event{
				ID:        fmt.Sprintf("heartbeat-%d", time.Now().UnixNano()),
				Type:      "heartbeat",
				Data:      fmt.Sprintf(`{"timestamp":%d}`, time.Now().Unix()),
				Timestamp: time.Now(),
			}

			broker.broadcast <- Message{
				Event:     heartbeatEvent,
				TargetIDs: nil, // Broadcast to all
			}

		case <-broker.stopCh:
			return
		}
	}
}

// handleShutdown gracefully shuts down the broker
func (broker *Broker) handleShutdown() {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	// Close all client connections
	for sessionID, client := range broker.clients {
		close(client.Done)
		delete(broker.clients, sessionID)

		// Mark sessions as disconnected so they can reconnect later
		broker.sessionManager.DisconnectSession(sessionID)
	}

	// Stop the session manager
	broker.sessionManager.Stop()
}

// Stop gracefully shuts down the broker
func (broker *Broker) Stop() {
	close(broker.stopCh)
	broker.wg.Wait()
}

// RegisterClient registers a new SSE client
func (broker *Broker) RegisterClient(w http.ResponseWriter, r *http.Request) (*Client, error) {
	// Ensure we can flush to the client
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	// Check for existing session ID from cookie or header
	sessionID := broker.extractSessionID(r)

	// Validate session ownership if reconnecting
	if sessionID != "" {
		metadata := SessionMeta{
			UserAgent:  r.UserAgent(),
			RemoteAddr: r.RemoteAddr,
		}

		if !broker.sessionManager.ValidateSessionOwnership(sessionID, metadata) {
			// Session hijacking attempt or session in use
			sessionID = "" // Force new session
		}
	}

	// Create client
	client := &Client{
		SessionID:    sessionID,
		EventChannel: make(chan Event, 100),
		Done:         make(chan struct{}),
		Request:      r,
		Writer:       w,
		Flusher:      flusher,
	}

	// Register with broker
	broker.register <- client

	return client, nil
}

// UnregisterClient unregisters an SSE client
func (broker *Broker) UnregisterClient(sessionID string) {
	broker.unregister <- sessionID
}

// Broadcast sends an event to all connected clients
func (broker *Broker) Broadcast(eventType, data string) {
	event := Event{
		Type: eventType,
		Data: data,
	}

	broker.broadcast <- Message{
		Event:     event,
		TargetIDs: nil,
	}
}

// SendToClient sends an event to a specific client
func (broker *Broker) SendToClient(sessionID, eventType, data string) {
	event := Event{
		Type: eventType,
		Data: data,
	}

	broker.broadcast <- Message{
		Event:     event,
		TargetIDs: []string{sessionID},
	}
}

// SendToClients sends an event to specific clients
func (broker *Broker) SendToClients(sessionIDs []string, eventType, data string) {
	event := Event{
		Type: eventType,
		Data: data,
	}

	broker.broadcast <- Message{
		Event:     event,
		TargetIDs: sessionIDs,
	}
}

// GetClientCount returns the number of connected clients
func (broker *Broker) GetClientCount() int {
	broker.mu.RLock()
	defer broker.mu.RUnlock()
	return len(broker.clients)
}

// GetSessionStats returns statistics about sessions
func (broker *Broker) GetSessionStats() map[string]interface{} {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	return map[string]interface{}{
		"active_clients":    len(broker.clients),
		"total_sessions":    broker.sessionManager.GetTotalSessionCount(),
		"buffered_sessions": broker.sessionManager.GetTotalSessionCount() - len(broker.clients),
	}
}

// extractSessionID extracts session ID from cookie or header
func (broker *Broker) extractSessionID(r *http.Request) string {
	// Check cookie first
	if cookie, err := r.Cookie("sse-session-id"); err == nil {
		return cookie.Value
	}

	// Check header as fallback
	if sessionID := r.Header.Get("X-SSE-Session-ID"); sessionID != "" {
		return sessionID
	}

	return ""
}

// Helper functions

func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for key := range r.Header {
		headers[key] = r.Header.Get(key)
	}
	return headers
}

func extractQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}
