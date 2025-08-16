// Package ssr provides server-sent events (SSE) functionality for real-time updates.
// SSE allows the server to push updates to connected clients over a long-lived HTTP
// connection. This is simpler than WebSockets but perfect for server-to-client push.
//
// The main component is the Broker which manages all connected clients and handles
// broadcasting messages to them. Each client connection is kept alive with periodic
// heartbeats to prevent timeouts from proxies and load balancers.
//
// Usage in a handler:
//
//	broker := c.Value("broker").(*ssr.Broker)
//	broker.Broadcast("update", []byte(`<div>New content</div>`))
//
// Client-side JavaScript connects to /events and listens for messages.
package ssr

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"
)

// Event represents a server-sent event that will be sent to clients.
// Events have a name (event type) and data (typically HTML for live updates).
// The SSE protocol allows clients to listen for specific event types.
type Event struct {
	// Name is the event type. Clients can listen for specific event names.
	// Common names: "message", "update", "notification", "heartbeat"
	Name string

	// Data is the event payload, typically HTML fragments for DOM updates.
	// For Buffkit, this is usually rendered HTML that will replace elements
	// on the page via JavaScript.
	Data []byte
}

// Client represents a connected SSE client.
// Each browser connection gets its own Client instance that manages
// the connection lifecycle and message delivery.
type Client struct {
	// ID uniquely identifies this client connection.
	// Generated from timestamp to ensure uniqueness.
	ID string

	// Events channel receives events to be sent to this client.
	// Buffered to prevent slow clients from blocking the broker.
	// If the buffer fills, events are dropped for that client.
	Events chan Event

	// Closing channel signals when the connection should be closed.
	// Used for graceful shutdown of client connections.
	Closing chan bool

	// Response is the underlying HTTP response writer for this SSE connection.
	// We write SSE-formatted data directly to this writer.
	Response http.ResponseWriter
}

// Broker manages SSE connections and broadcasts.
// It's the central hub that coordinates all SSE clients, handling their
// lifecycle (connect/disconnect) and message distribution.
//
// The broker runs in a separate goroutine and uses channels for thread-safe
// communication. This allows multiple handlers to broadcast without locks.
type Broker struct {
	// broadcast channel receives events to send to all clients.
	// Handlers send events here, and the broker distributes them.
	// This is buffered to prevent slow distribution from blocking senders.
	broadcast chan Event

	// register channel receives new client connections.
	// When a client connects to /events, they're registered here.
	register chan *Client

	// unregister channel receives disconnected clients.
	// When a client disconnects (closes tab, network issue), they're removed.
	unregister chan *Client

	// clients map stores all active client connections.
	// Maps client ID to client instance for easy lookup and iteration.
	clients map[string]*Client

	// heartbeatInterval controls how often to send keepalive messages.
	// Default 25 seconds works well with most proxy timeouts (usually 30-60s).
	// These heartbeats prevent connections from being closed by intermediaries.
	heartbeatInterval time.Duration
}

// NewBroker creates a new SSE broker and starts its event loops.
// The broker immediately begins running in goroutines to handle:
//   - Client registration/unregistration
//   - Event broadcasting
//   - Heartbeat sending
//
// You typically create one broker per app and share it across handlers:
//
//	broker := ssr.NewBroker()
//	app.GET("/events", broker.ServeHTTP)
func NewBroker() *Broker {
	broker := &Broker{
		broadcast:         make(chan Event, 100),    // Buffer prevents blocking on broadcast
		register:          make(chan *Client),       // Unbuffered for immediate handling
		unregister:        make(chan *Client),       // Unbuffered for immediate cleanup
		clients:           make(map[string]*Client), // Active client registry
		heartbeatInterval: 25 * time.Second,         // Conservative heartbeat interval
	}

	// Start the broker's main event loop in a goroutine.
	// This handles all client lifecycle and message distribution.
	go broker.run()

	// Start heartbeat ticker in a separate goroutine.
	// This ensures connections stay alive through proxies.
	go broker.heartbeat()

	return broker
}

// run is the main event loop for the broker.
// This goroutine is the only one that modifies the clients map, ensuring
// thread safety without locks. All operations go through channels.
//
// The loop handles three types of operations:
//  1. Client registration - adds new SSE connections
//  2. Client unregistration - removes disconnected clients
//  3. Event broadcasting - distributes events to all clients
func (b *Broker) run() {
	for {
		select {
		case client := <-b.register:
			// New client connected - add to registry.
			// This happens when someone opens the page or reconnects.
			b.clients[client.ID] = client
			log.Printf("SSE: Client %s connected. Total clients: %d", client.ID, len(b.clients))

		case client := <-b.unregister:
			// Client disconnected - remove and cleanup.
			// This happens on tab close, navigation, or network issues.
			if _, ok := b.clients[client.ID]; ok {
				delete(b.clients, client.ID)
				close(client.Events)  // Stop sending events
				close(client.Closing) // Signal connection close
				log.Printf("SSE: Client %s disconnected. Total clients: %d", client.ID, len(b.clients))
			}

		case event := <-b.broadcast:
			// Broadcast event to all connected clients.
			// Each client gets the event in their personal channel.
			for _, client := range b.clients {
				select {
				case client.Events <- event:
					// Event successfully queued for this client
				default:
					// Client's event buffer is full - drop the event.
					// This prevents slow clients from blocking everyone.
					// In production, you might want to disconnect slow clients.
					log.Printf("SSE: Dropping event for slow client %s", client.ID)
				}
			}
		}
	}
}

// heartbeat sends periodic keepalive messages to prevent connection timeout.
// Many proxies, load balancers, and CDNs will close idle connections after
// 30-60 seconds. By sending a heartbeat every 25 seconds, we ensure the
// connection stays active.
//
// Heartbeats are sent as regular SSE events that clients can ignore or use
// to verify the connection is still alive.
func (b *Broker) heartbeat() {
	ticker := time.NewTicker(b.heartbeatInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		// Send heartbeat event with current timestamp.
		// Clients can use this to detect connection health.
		b.broadcast <- Event{
			Name: "heartbeat",
			Data: []byte(time.Now().Format(time.RFC3339)),
		}
	}
}

// Broadcast sends an event to all connected clients.
// This is the main API for sending real-time updates:
//
//	broker.Broadcast("notification", []byte(`<div class="alert">New message!</div>`))
//
// The eventName allows clients to listen for specific types of updates.
// The html should be a rendered HTML fragment that the client-side JavaScript
// will insert into the DOM.
//
// Broadcasting is non-blocking - if the broadcast channel is full, the event
// is dropped with a warning log. This prevents a backup of events from blocking
// the application.
func (b *Broker) Broadcast(eventName string, html []byte) {
	event := Event{
		Name: eventName,
		Data: html,
	}

	// Non-blocking send to prevent deadlocks.
	// If the broadcast buffer is full, we drop the event rather than block.
	select {
	case b.broadcast <- event:
		// Event successfully queued for broadcast
	default:
		// Broadcast channel is full - this indicates a serious problem
		// (either too many events or the broker goroutine is stuck)
		log.Printf("SSE: Broadcast channel full, dropping event %s", eventName)
	}
}

// ServeHTTP handles SSE connections from clients.
// This is a Buffalo handler that should be mounted on a GET route:
//
//	app.GET("/events", broker.ServeHTTP)
//
// When a client connects, this handler:
//  1. Sets appropriate SSE headers
//  2. Creates a Client instance
//  3. Registers the client with the broker
//  4. Sends an initial connection event
//  5. Enters a loop sending events until disconnect
//
// The connection is kept open until the client disconnects or an error occurs.
func (b *Broker) ServeHTTP(c buffalo.Context) error {
	// Get the underlying ResponseWriter for direct HTTP access
	w := c.Response()
	r := c.Request()

	// Set SSE-specific headers.
	// These tell the browser this is an event stream, not a regular response.
	w.Header().Set("Content-Type", "text/event-stream") // SSE MIME type
	w.Header().Set("Cache-Control", "no-cache")         // Prevent caching of events
	w.Header().Set("Connection", "keep-alive")          // Keep connection open
	w.Header().Set("X-Accel-Buffering", "no")           // Disable Nginx buffering

	// Create new client instance for this connection.
	// Each connection gets a unique ID and its own event channel.
	client := &Client{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()), // Simple unique ID
		Events:   make(chan Event, 10),                     // Buffered to prevent blocking
		Closing:  make(chan bool, 1),                       // Signal channel for shutdown
		Response: w,                                        // Store response writer
	}

	// Register client with broker.
	// This adds the client to the active clients map.
	b.register <- client

	// Ensure cleanup when this function exits.
	// This handles both normal disconnects and errors.
	defer func() {
		b.unregister <- client
	}()

	// Get flusher for immediate writes.
	// SSE requires flushing after each event to ensure immediate delivery.
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// Send initial connection event.
	// This confirms to the client that SSE is working and provides the client ID.
	// Format: "event: connected\ndata: {json}\n\n"
	fmt.Fprintf(w, "event: connected\ndata: {\"id\":\"%s\"}\n\n", client.ID)
	flusher.Flush()

	// Listen for client disconnect via request context.
	// When the HTTP connection closes, the context is cancelled.
	notify := r.Context().Done()

	// Main event loop for this client.
	// This loop runs until the client disconnects or the server closes the connection.
	for {
		select {
		case event := <-client.Events:
			// Send event to client in SSE format.
			// Format: "event: <name>\ndata: <data>\n\n"
			// The double newline signals end of event.
			if event.Name != "" {
				fmt.Fprintf(w, "event: %s\n", event.Name)
			}
			fmt.Fprintf(w, "data: %s\n\n", event.Data)
			flusher.Flush() // Immediately send to client

		case <-notify:
			// Client disconnected (tab closed, navigated away, network issue).
			// Exit gracefully - cleanup handled by defer.
			return nil

		case <-client.Closing:
			// Server closing this connection (shutdown, max clients, etc).
			// Exit gracefully - cleanup handled by defer.
			return nil
		}
	}
}

// RenderPartial renders a partial template with data.
// This helper ensures the same HTML is used for both regular HTTP responses
// and SSE broadcasts, maintaining a single source of truth for fragments.
//
// Usage:
//
//	html, err := ssr.RenderPartial(c, "partials/notification", map[string]interface{}{
//	    "message": "Hello, world!",
//	})
//	// Use for htmx response
//	c.Render(200, r.HTML(html))
//	// AND/OR broadcast via SSE
//	broker.Broadcast("notification", html)
//
// WHY: This prevents divergence between what's rendered for direct requests
// versus what's pushed via SSE, ensuring consistency.
func RenderPartial(c buffalo.Context, name string, data map[string]interface{}) ([]byte, error) {
	// TODO: In production, this would use Buffalo's template renderer
	// to actually render the named partial with the provided data.
	// For now, we return a stub HTML fragment.

	// Create a temporary rendering context
	// In a real implementation, this would:
	// 1. Load the template from templates/partials/{name}.html
	// 2. Execute the template with the provided data
	// 3. Return the rendered HTML

	// Stub implementation for demonstration
	html := fmt.Sprintf(`<div data-partial="%s"><!-- Rendered partial: %s --></div>`, name, name)

	return []byte(html), nil
}
