package sse

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// Handler provides HTTP endpoints for SSE with reconnection support
type Handler struct {
	broker *Broker
	config SessionConfig
}

// NewHandler creates a new SSE handler with the given configuration
func NewHandler(config SessionConfig) *Handler {
	return &Handler{
		broker: NewBroker(config),
		config: config,
	}
}

// ServeHTTP handles SSE connections with reconnection support
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering for nginx

	// Register the client with the broker
	client, err := h.broker.RegisterClient(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set session cookie if we have a session ID
	if client.SessionID != "" && h.config.EnableReconnection {
		http.SetCookie(w, &http.Cookie{
			Name:     "sse-session-id",
			Value:    client.SessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})

		// Also send session ID in response header for non-cookie clients
		w.Header().Set("X-SSE-Session-ID", client.SessionID)
	}

	// Send initial connection event
	h.sendEvent(w, Event{
		Type: "connected",
		Data: fmt.Sprintf(`{"sessionId":"%s","reconnection":%v,"bufferSize":%d}`,
			client.SessionID, h.config.EnableReconnection, h.config.BufferSize),
	})

	// Flush the headers
	client.Flusher.Flush()

	// Keep the connection open and send events
	for {
		select {
		case event := <-client.EventChannel:
			// Send the event to the client
			if err := h.sendEvent(w, event); err != nil {
				log.Printf("Error sending event to client %s: %v", client.SessionID, err)
				h.broker.UnregisterClient(client.SessionID)
				return
			}
			client.Flusher.Flush()

		case <-client.Done:
			// Client disconnected
			log.Printf("Client %s disconnected", client.SessionID)
			return

		case <-r.Context().Done():
			// Request context cancelled (client disconnected)
			h.broker.UnregisterClient(client.SessionID)
			return
		}
	}
}

// sendEvent writes an SSE event to the response writer
func (h *Handler) sendEvent(w http.ResponseWriter, event Event) error {
	// Write event ID if present
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}

	// Write event type if present
	if event.Type != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
			return err
		}
	}

	// Write retry hint for reconnection
	if h.config.EnableReconnection {
		if _, err := fmt.Fprintf(w, "retry: 3000\n"); err != nil {
			return err
		}
	}

	// Write event data
	if event.Data != "" {
		if _, err := fmt.Fprintf(w, "data: %s\n", event.Data); err != nil {
			return err
		}
	}

	// Add metadata for replayed events
	if event.Replayed {
		if _, err := fmt.Fprintf(w, "data: {\"_replayed\":true,\"_originalTime\":%d}\n",
			event.Timestamp.Unix()); err != nil {
			return err
		}
	}

	// End of event
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return nil
}

// Broadcast sends an event to all connected clients
func (h *Handler) Broadcast(eventType, data string) {
	h.broker.Broadcast(eventType, data)
}

// SendToClient sends an event to a specific client
func (h *Handler) SendToClient(sessionID, eventType, data string) {
	h.broker.SendToClient(sessionID, eventType, data)
}

// SendToClients sends an event to specific clients
func (h *Handler) SendToClients(sessionIDs []string, eventType, data string) {
	h.broker.SendToClients(sessionIDs, eventType, data)
}

// GetStats returns statistics about the SSE connections
func (h *Handler) GetStats() map[string]interface{} {
	stats := h.broker.GetSessionStats()
	stats["config"] = map[string]interface{}{
		"buffer_size":          h.config.BufferSize,
		"buffer_ttl_seconds":   h.config.BufferTTL.Seconds(),
		"reconnection_enabled": h.config.EnableReconnection,
	}
	return stats
}

// Stop gracefully shuts down the SSE handler
func (h *Handler) Stop() {
	h.broker.Stop()
}

// TestEndpoints provides HTTP endpoints for testing SSE functionality
type TestEndpoints struct {
	handler *Handler
}

// NewTestEndpoints creates test endpoints for SSE
func NewTestEndpoints(handler *Handler) *TestEndpoints {
	return &TestEndpoints{handler: handler}
}

// RegisterRoutes registers test routes on an HTTP mux
func (te *TestEndpoints) RegisterRoutes(mux *http.ServeMux) {
	// SSE connection endpoint
	mux.HandleFunc("/events", te.handler.ServeHTTP)

	// Test broadcast endpoint
	mux.HandleFunc("/broadcast", te.handleBroadcast)

	// Test targeted send endpoint
	mux.HandleFunc("/send", te.handleSend)

	// Stats endpoint
	mux.HandleFunc("/stats", te.handleStats)

	// Simulate disconnect endpoint (for testing)
	mux.HandleFunc("/disconnect", te.handleDisconnect)
}

// handleBroadcast handles test broadcasts
func (te *TestEndpoints) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventType := r.URL.Query().Get("type")
	if eventType == "" {
		eventType = "message"
	}

	data := r.URL.Query().Get("data")
	if data == "" {
		data = fmt.Sprintf(`{"timestamp":%d,"test":true}`, time.Now().Unix())
	}

	te.handler.Broadcast(eventType, data)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Broadcast sent to %d clients\n", te.handler.broker.GetClientCount())
}

// handleSend handles targeted sends
func (te *TestEndpoints) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	eventType := r.URL.Query().Get("type")
	if eventType == "" {
		eventType = "message"
	}

	data := r.URL.Query().Get("data")
	if data == "" {
		data = fmt.Sprintf(`{"timestamp":%d,"targeted":true}`, time.Now().Unix())
	}

	te.handler.SendToClient(sessionID, eventType, data)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Event sent to session %s\n", sessionID)
}

// handleStats returns SSE statistics
func (te *TestEndpoints) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := te.handler.GetStats()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"active_clients": %v,
		"total_sessions": %v,
		"buffered_sessions": %v,
		"config": {
			"buffer_size": %v,
			"buffer_ttl_seconds": %v,
			"reconnection_enabled": %v
		}
	}`,
		stats["active_clients"],
		stats["total_sessions"],
		stats["buffered_sessions"],
		stats["config"].(map[string]interface{})["buffer_size"],
		stats["config"].(map[string]interface{})["buffer_ttl_seconds"],
		stats["config"].(map[string]interface{})["reconnection_enabled"],
	)
}

// handleDisconnect simulates a client disconnect (for testing)
func (te *TestEndpoints) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	te.handler.broker.UnregisterClient(sessionID)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Disconnected session %s\n", sessionID)
}
