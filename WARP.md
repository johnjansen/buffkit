# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Quick Start

Buffkit is an opinionated SSR-first plugin system for Buffalo (Go) applications that provides Rails-like functionality. The main entry point is the `Wire()` function that integrates all subsystems.

### Essential Commands
```bash
# Initial setup
make setup                    # Install deps + start Docker services (Redis, MailHog)

# Development
make examples                 # Run the example application
make dev                     # Start full development environment

# Testing (uses Godog BDD framework)
make test                    # Run all BDD feature tests
make test-focus              # Run tests tagged with @focus
make test-short              # Run non-integration tests only

# Code quality
make check                   # Format code and run vet
make lint                    # Run Go and YAML linters
make clean                   # Clean build artifacts
```

## Architecture Overview

Buffkit is built around a **composable plugin system** where `Wire(app, config)` integrates all subsystems into your Buffalo application in one call:

```go
kit, err := buffkit.Wire(app, buffkit.Config{
    DevMode:    true,
    AuthSecret: []byte("your-secret"),
    RedisURL:   "redis://localhost:6379/0",
    SMTPAddr:   "localhost:1025",
    Dialect:    "postgres", // or "sqlite", "mysql"
    DB:         db,
})
```

### Core Subsystems (accessed via Kit struct)
- **SSR Broker** (`kit.Broker`) - Server-Sent Events for real-time updates
- **Auth Store** (`kit.AuthStore`) - User management with session-based auth
- **Jobs Runtime** (`kit.Jobs`) - Background processing via Asynq/Redis
- **Mail Sender** (`kit.Mail`) - SMTP or development email sending
- **Import Map Manager** (`kit.ImportMap`) - JavaScript dependency management without bundlers
- **Component Registry** (`kit.Components`) - Server-side HTML component expansion

### Wire() Function Initialization Order
1. **SSE Broker** - Starts event broadcasting system at `/events`
2. **Authentication** - Mounts routes (`/login`, `/logout`, `/register`, etc.)
3. **Background Jobs** - Initializes Asynq workers (if Redis configured)
4. **Mail System** - SMTP or dev mode with preview at `/__mail/preview`
5. **Import Maps** - JavaScript module resolution system
6. **Security Middleware** - Headers, CSRF, content policy
7. **Component System** - Server-side HTML expansion middleware

## Development Commands

### Building and Dependencies
```bash
make deps                    # Download and tidy Go modules
make build                   # Build all packages
make install                 # Install buffkit globally
```

### Testing with BDD (Godog)
```bash
make test                    # Run all BDD scenarios with coverage
make test-short              # Skip @integration tagged scenarios
make test-focus              # Run only @focus tagged scenarios  
make test-verbose            # Pretty-formatted test output
make test-watch              # Watch for changes (manual restart)
```

**BDD Test Organization:**
- Feature files in `features/*.feature` (Gherkin syntax)
- Step definitions in `features/*_test.go`
- Tag scenarios with `@focus`, `@integration`, `@auth`, etc.
- Test philosophy: Behavior-driven, not implementation details

### Code Quality
```bash
make fmt                     # Format all Go code
make vet                     # Run go vet analysis  
make lint                    # golangci-lint + yamllint
make lint-all                # Run all linters
make check                   # fmt + vet combined
make coverage                # Generate and open HTML coverage report
```

### Docker Services
```bash
make docker-redis            # Start Redis (required for jobs)
make docker-mailhog          # Start MailHog (email testing UI at :8025)
make docker-services         # Start both Redis + MailHog
make docker-stop             # Stop all services
make docker-clean            # Remove containers
```

### Examples and Development
```bash
make examples                # Run example app at localhost:3000
make examples-build          # Build examples binary to bin/
make run                     # Alias for examples
make watch                   # Auto-reload with air (if installed)
make dev                     # Full environment: services + examples
```

## Core Concepts

### 1. SSR-First with Progressive Enhancement
- **HTML rendered server-side** - No client-side routing or SPA complexity
- **htmx for interactions** - AJAX requests that swap HTML fragments  
- **Alpine.js for UI state** - Reactive behavior without frameworks
- **Server-Sent Events (SSE)** - Real-time updates pushed from server

### 2. Server-Side Components
Components are `<bk-*>` custom elements expanded server-side before sending to browser:

```html
<!-- In template -->
<bk-button variant="primary" href="/save">Save Changes</bk-button>

<!-- Becomes in browser -->
<button class="bk-button bk-button-primary" data-href="/save">Save Changes</button>
```

**Expansion Flow:**
1. Handler renders template with `<bk-*>` tags
2. Component middleware intercepts HTML response
3. Parses HTML and expands all `<bk-*>` elements
4. Client receives final HTML with no custom tags

### 3. Authentication System
Session-based auth with encrypted cookies:

```go
// Protect routes
protected := app.Group("/admin")
protected.Use(buffkit.RequireLogin)

// Or single routes
app.GET("/profile", buffkit.RequireLogin(ProfileHandler))

// Current user available in context
user := c.Value("current_user").(*auth.User)
```

**Auth Flow:**
- Passwords bcrypt-hashed (never stored plain)  
- Sessions encrypted with `AuthSecret`
- CSRF protection on forms
- Rate limiting on auth endpoints

### 4. Server-Sent Events (SSE)
Real-time updates without WebSocket complexity:

```go
// Broadcast to all connected clients
html, _ := buffkit.RenderPartial(c, "partials/item", data)
kit.Broker.Broadcast("item-update", html)

// Single source of truth: same HTML for HTTP response AND SSE
return c.Render(200, r.HTML(html))
```

**SSE Architecture:**
- Clients connect to `/events` endpoint
- Broker manages connections with heartbeats (25s intervals)  
- Thread-safe via goroutines and channels
- Automatic cleanup of dead connections

### 5. Import Maps (No Bundler)
JavaScript dependencies via native ES modules:

```html
<!-- Generated by <%= importmap() %> -->
<script type="importmap">
{
  "imports": {
    "htmx.org": "https://unpkg.com/htmx.org@1.9.12/dist/htmx.js",
    "alpinejs": "https://esm.sh/alpinejs@3.14.1"
  }
}
</script>

<!-- Use in your JavaScript -->
<script type="module">
import Alpine from "alpinejs"
import "htmx.org"
</script>
```

### 6. Background Jobs
Async processing with Asynq (requires Redis):

```go
// Enqueue job  
kit.Jobs.Enqueue("email:welcome", map[string]interface{}{
    "user_id": user.ID,
    "email": user.Email,
})

// Register handler
kit.Jobs.Mux.HandleFunc("email:welcome", func(ctx context.Context, t *asynq.Task) error {
    // Process job
    return kit.Mail.Send(ctx, message)
})
```

## Development Workflow

### Adding a New Feature
1. **Write BDD scenario first** in `features/your-feature.feature`
2. **Run test to see it fail** with `make test-focus`
3. **Implement step definitions** in `features/your-feature_test.go`
4. **Add minimal implementation** to make tests pass
5. **Refactor** while keeping tests green

### Creating Custom Components
```go
// Register component
kit.Components.Register("my-component", func(attrs map[string]string, slots map[string][]byte) ([]byte, error) {
    variant := attrs["variant"]
    content := slots["default"]
    
    html := fmt.Sprintf(`<div class="my-component my-component-%s">%s</div>`, 
        variant, content)
    return []byte(html), nil
})
```

### Broadcasting Real-time Updates
```go
func UpdateHandler(c buffalo.Context) error {
    // Update data
    item := updateItem(c.Param("id"))
    
    // Render once, use twice
    html, _ := buffkit.RenderPartial(c, "partials/item", map[string]interface{}{
        "item": item,
    })
    
    // HTTP response
    c.Render(200, r.HTML(html))
    
    // SSE broadcast  
    kit.Broker.Broadcast("item-update", html)
    
    return nil
}
```

### Template Shadowing
Override any Buffkit template by placing it at the same path in your app:

```
your-app/
├── templates/
│   ├── auth/
│   │   └── login.plush.html    # Override Buffkit login
│   └── components/
│       └── button.plush.html   # Override bk-button component
```

## Grift Tasks (CLI Commands)

Buffkit provides several `buffalo task` commands:

### Database Migrations
```bash
buffalo task buffkit:migrate                    # Apply pending migrations
buffalo task buffkit:migrate:status             # Show migration status
buffalo task buffkit:migrate:down 2             # Rollback 2 migrations  
buffalo task buffkit:migrate:create add_users   # Create new migration
```

### Background Jobs
```bash
buffalo task jobs:worker                         # Start job worker
buffalo task jobs:enqueue email:welcome         # Enqueue test job
buffalo task jobs:stats                          # Show queue statistics
```

## Context Values Available

After `Wire()`, these are accessible via `c.Value()` in handlers:

- `"broker"` - `*ssr.Broker` for SSE broadcasting
- `"mail_sender"` - `mail.Sender` for sending emails
- `"buffkit"` - `*buffkit.Kit` for full access to all subsystems
- `"importmap"` - `func() string` to render import map HTML
- `"component"` - `func(string, map[string]string) string` for programmatic rendering
- `"current_user"` - `*auth.User` (available after `RequireLogin` middleware)

## Common Patterns

### Protected Route Groups
```go
admin := app.Group("/admin")  
admin.Use(buffkit.RequireLogin)
admin.GET("/dashboard", DashboardHandler)
admin.POST("/users", CreateUserHandler)
```

### Live Updates with SSE
```html
<!-- Template -->
<div id="notifications" hx-sse="connect:/events" hx-sse-swap="notification"></div>

<!-- Handler broadcasts -->
kit.Broker.Broadcast("notification", []byte("<div>New message!</div>"))
```

### Component with Slots
```html
<bk-card>
    <bk-slot name="header">
        <h2>Card Title</h2>  
    </bk-slot>
    
    <p>Card body content</p>
    
    <bk-slot name="footer">
        <button>Action</button>
    </bk-slot>
</bk-card>
```

## Troubleshooting

### SSE Not Working
- Check `/events` endpoint is accessible (no 404)
- Verify Redis is running for job-related SSE
- Look for proxy buffering issues in production
- Check browser console for EventSource errors

### Components Not Expanding  
- Ensure response `Content-Type` is `text/html`
- Verify component is registered in registry
- Check for HTML parsing errors in logs
- Confirm middleware order (component expansion runs late)

### Authentication Issues
- Verify `AuthSecret` is consistent across restarts
- Check session cookie is being set (`Set-Cookie` header)  
- Ensure CSRF tokens are present in forms
- Look for timing attacks (use constant-time comparison)

### Job Processing Problems
- Confirm Redis is running and accessible
- Start worker with `buffalo task jobs:worker`  
- Check job is registered with mux
- Look for serialization/deserialization errors

## Performance Notes

- **Component expansion** happens on every HTML response - consider caching in production
- **SSE connections** hold server resources - implement connection limits for scale  
- **Session lookups** query database on protected routes - use connection pooling
- **Import maps** can be preloaded/vendored for offline development

## File Structure

Key files for understanding the codebase:
- `buffkit.go` - Main `Wire()` function and `Kit` struct  
- `grifts.go` - CLI tasks for migrations and jobs
- `HOW_IT_WORKS.md` - Detailed architectural explanation
- `features/*.feature` - BDD test scenarios in Gherkin
- `examples/main.go` - Complete working example
- `auth/` - Authentication system (sessions, users, middleware)
- `ssr/` - Server-Sent Events broker and client management
- `components/` - Server-side component registry and expansion
- `jobs/` - Background job processing with Asynq
- `templates/` - Default templates (can be shadowed by apps)
