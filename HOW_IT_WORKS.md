# How Buffkit Works: A Complete Architecture Guide

## Overview

Buffkit is a composable plugin system for Buffalo (Go) applications that provides Rails-like conveniences without the bloat. This document explains how all the pieces fit together to create a cohesive server-side rendering framework.

## The Big Picture

```
Your Buffalo App
       ↓
  Wire(app, config)
       ↓
┌─────────────────────────────────────────┐
│           Buffkit System                 │
├─────────────────────────────────────────┤
│  Middleware Layer (ordered)              │
│  1. Security Headers                     │
│  2. Component Expansion                  │
│  3. Context Helpers                      │
├─────────────────────────────────────────┤
│  Route Handlers                          │
│  - /events (SSE)                         │
│  - /login, /logout (Auth)                │
│  - /__mail/preview (Dev)                 │
├─────────────────────────────────────────┤
│  Subsystems                              │
│  - SSR/SSE Broker                        │
│  - Auth Store & Session                  │
│  - Job Queue (Asynq)                     │
│  - Mail Sender                           │
│  - Import Maps                           │
│  - Component Registry                    │
└─────────────────────────────────────────┘
```

## Core Concepts

### 1. The Wire() Function - Single Point of Integration

The `Wire()` function is the magic that connects everything. When you call:

```go
kit, err := buffkit.Wire(app, config)
```

Here's what happens in order:

1. **Configuration Validation**: Ensures required fields (like AuthSecret) are present
2. **SSE Broker Initialization**: Starts goroutines for real-time event handling
3. **Route Mounting**: Adds /events, /login, /logout endpoints
4. **Store Setup**: Configures auth and other data stores
5. **Middleware Chain**: Adds security, component expansion, and helper middleware
6. **Context Enrichment**: Adds helpers accessible in handlers and templates

### 2. Request Flow Through the System

When a request comes in, it flows through multiple layers:

```
HTTP Request
    ↓
[Security Middleware]
    - Adds headers (X-Frame-Options, CSP, etc.)
    - CSRF protection
    ↓
[Your Route Handler]
    - Can access kit.Broker, auth helpers, etc.
    - Renders HTML with <bk-*> components
    ↓
[Component Expansion Middleware]
    - Intercepts HTML responses
    - Expands <bk-*> tags to full HTML
    ↓
HTTP Response (with expanded HTML)
```

### 3. Authentication Flow

The authentication system is session-based with encrypted cookies:

```
User submits login form
    ↓
LoginHandler validates credentials
    ↓
Password check via bcrypt
    ↓
Session created with user ID
    ↓
Encrypted cookie sent to browser
    ↓
Future requests include cookie
    ↓
RequireLogin middleware checks session
    ↓
User object added to context
```

Key security features:
- Passwords are bcrypt hashed (never stored plain)
- Sessions are encrypted with AuthSecret
- CSRF tokens on all POST requests
- Timing-safe password comparison

### 4. Server-Sent Events (SSE) System

SSE provides real-time updates without WebSockets complexity:

```
Client connects to /events
    ↓
Broker creates Client instance
    ↓
Client registered in broker.clients map
    ↓
[Event Loop Running]
    │
    ├─→ Heartbeat every 25s (keeps connection alive)
    │
    ├─→ Handler calls broker.Broadcast()
    │       ↓
    │   Event sent to all clients
    │       ↓
    │   Client JavaScript updates DOM
    │
    └─→ Client disconnects
            ↓
        Cleanup and unregister
```

The broker runs in separate goroutines for thread safety:
- Main loop: Handles registration/unregistration/broadcast
- Heartbeat loop: Sends periodic keepalives

### 5. Component Expansion System

Server-side components work through HTML transformation:

```
Template contains: <bk-button variant="primary">Save</bk-button>
    ↓
Handler renders template to HTML
    ↓
Component Expansion Middleware intercepts
    ↓
Parses HTML into DOM tree
    ↓
Finds all <bk-*> elements
    ↓
For each component:
    - Extract attributes (variant="primary")
    - Extract slots (content between tags)
    - Call component renderer
    - Replace tag with rendered HTML
    ↓
Modified HTML sent to client
    ↓
Browser sees: <button class="bk-button bk-button-primary">Save</button>
```

### 6. Import Maps for JavaScript

Import maps eliminate the need for bundlers:

```
Manager stores pins (name → URL mappings)
    ↓
Template calls <%= importmap() %>
    ↓
Generates <script type="importmap">{...}</script>
    ↓
Browser can use: import Alpine from "alpinejs"
    ↓
Browser resolves to: https://esm.sh/alpinejs@3.14.1
```

### 7. Background Jobs with Asynq

Jobs are processed asynchronously via Redis:

```
Handler enqueues job
    ↓
kit.Jobs.Client.Enqueue(task)
    ↓
Task serialized to Redis queue
    ↓
Worker process (buffalo task jobs:worker)
    ↓
Pulls task from queue
    ↓
Routes to handler via Mux
    ↓
Handler processes job
```

## Data Flow Examples

### Example 1: Live Update via SSE

```go
// In your handler
func UpdateItemHandler(c buffalo.Context) error {
    // 1. Update database
    item := updateItem(c.Param("id"))
    
    // 2. Render HTML fragment
    html, _ := buffkit.RenderPartial(c, "items/item", map[string]interface{}{
        "item": item,
    })
    
    // 3. Broadcast to all connected clients
    broker := c.Value("broker").(*ssr.Broker)
    broker.Broadcast("item-update", html)
    
    // 4. Return response for the requesting client
    return c.Render(200, r.HTML(html))
}
```

All connected clients receive the update instantly without polling.

### Example 2: Protected Route with Components

```go
// Route setup
protected := app.Group("/admin")
protected.Use(buffkit.RequireLogin)
protected.GET("/users", UsersHandler)

// Handler
func UsersHandler(c buffalo.Context) error {
    user := c.Value("current_user").(*auth.User) // Added by RequireLogin
    
    return c.Render(200, r.HTML("users/index", map[string]interface{}{
        "current_user": user,
        "users": getUsers(),
    }))
}
```

Template (users/index.html):
```html
<bk-card>
    <bk-slot name="header">
        <h1>Users</h1>
    </bk-slot>
    
    <% for (user) in users { %>
        <div><%= user.Email %></div>
    <% } %>
    
    <bk-slot name="footer">
        <bk-button variant="primary" href="/admin/users/new">
            Add User
        </bk-button>
    </bk-slot>
</bk-card>
```

## Middleware Ordering Matters

The order of middleware is critical:

1. **Security Headers** (first) - Must set headers before any response
2. **Auth Session** - Needs to load session before route handlers
3. **Your Routes** - Process requests
4. **Component Expansion** (near last) - Must capture complete HTML output
5. **Error Handling** (last) - Catch any errors from above

## Context Values Available

After Wire(), these are available in your handlers via `c.Value()`:

- `"broker"` - SSE broker for broadcasting
- `"importmap"` - Function to render import map HTML
- `"component"` - Function to render components programmatically
- `"current_user"` - Current logged-in user (after RequireLogin)
- `"buffkit.migrations"` - Migration runner for tasks

## Configuration Impact

Your Config struct affects behavior:

- **DevMode: true**
  - Enables /__mail/preview endpoint
  - Relaxes security headers
  - Uses development mail sender
  - Shows detailed error messages

- **RedisURL: empty**
  - Jobs become no-ops (logged but not queued)
  - Good for development without Redis

- **SMTPAddr: empty**
  - Emails are logged, not sent
  - Can preview at /__mail/preview in dev

## File Organization

Buffkit follows a clear package structure:

```
buffkit/
├── buffkit.go          # Main Wire() function and Kit struct
├── ssr/
│   └── broker.go       # SSE broker and client management
├── auth/
│   └── auth.go         # User model, stores, session handling
├── components/
│   └── registry.go     # Component registry and expansion
├── jobs/
│   └── runtime.go      # Asynq wrapper for background jobs
├── mail/
│   └── sender.go       # Email sending abstraction
├── importmap/
│   └── manager.go      # JavaScript dependency management
└── secure/
    └── middleware.go   # Security headers and CSRF
```

## Thread Safety

Buffkit is designed to be thread-safe:

- **SSE Broker**: Uses channels for all state changes (no locks needed)
- **Component Registry**: Read-only after initialization
- **Auth Store**: Database handles concurrency
- **Import Map Manager**: Synchronized at initialization

## Performance Considerations

1. **Component Expansion**: Happens on every HTML response
   - Consider caching expanded components in production
   - Only processes text/html responses

2. **SSE Connections**: Each client holds an open connection
   - Use heartbeats to detect dead connections
   - Consider connection limits for scaling

3. **Session Lookups**: Database query on every protected route
   - Consider caching current user in request context
   - Use connection pooling for database

## Extensibility Points

You can extend Buffkit by:

1. **Custom Components**: Register your own with `kit.Components.Register()`
2. **Custom User Store**: Implement `UserStore` interface
3. **Custom Mail Sender**: Implement `Sender` interface
4. **Job Handlers**: Add to `kit.Jobs.Mux`
5. **Template Overrides**: Shadow any Buffkit template

## Common Patterns

### Broadcasting Updates
```go
// Broadcast to all clients
kit.Broker.Broadcast("event-name", htmlBytes)

// Render once, use twice
html, _ := buffkit.RenderPartial(c, "partial", data)
c.Render(200, r.HTML(html))           // HTTP response
kit.Broker.Broadcast("update", html)  // SSE broadcast
```

### Protecting Routes
```go
// Single route
app.GET("/admin", buffkit.RequireLogin(AdminHandler))

// Route group
admin := app.Group("/admin")
admin.Use(buffkit.RequireLogin)
```

### Async Jobs
```go
// Enqueue
kit.Jobs.EnqueueEmail("user@example.com", "Welcome!", body)

// Process (in worker)
kit.Jobs.Mux.HandleFunc("email:welcome", handleWelcomeEmail)
```

## Debugging Tips

1. **SSE Not Working?**
   - Check browser console for EventSource errors
   - Verify /events endpoint is accessible
   - Look for proxy/nginx buffering issues

2. **Components Not Expanding?**
   - Ensure Content-Type is text/html
   - Check component is registered
   - Look for HTML parsing errors

3. **Auth Redirecting?**
   - Check session cookie is being set
   - Verify AuthSecret is consistent
   - Check for CSRF token in forms

4. **Jobs Not Processing?**
   - Ensure Redis is running
   - Check worker is started
   - Look for serialization errors

## Summary

Buffkit works by:
1. **Wiring** all subsystems into your Buffalo app with one function
2. **Intercepting** requests through middleware for security and features
3. **Expanding** components server-side for zero-JavaScript UI
4. **Broadcasting** updates via SSE for real-time features
5. **Managing** sessions securely for authentication
6. **Processing** jobs asynchronously for scalability

The key insight is that everything is **composable** - you can use what you need and ignore what you don't. The **single source of truth** principle (one HTML fragment for both HTTP and SSE) ensures consistency. And the **server-side first** approach keeps things simple while still providing modern UX.