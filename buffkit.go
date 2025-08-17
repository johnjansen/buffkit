// Package buffkit provides an opinionated SSR-first stack for Buffalo applications.
// It brings Rails-like batteries to server-rendered Go applications without the bloat.
//
// Buffkit is designed around several core principles:
//   - Server-side rendering first (HTML rendered on server, htmx for interactions)
//   - Zero JavaScript bundler (uses import maps instead)
//   - Batteries included (auth, jobs, mail, components out of the box)
//   - Database agnostic (uses database/sql, no ORM lock-in)
//   - Everything is shadowable (your app can override any template or asset)
//
// The main entry point is the Wire() function which installs all Buffkit
// packages into your Buffalo application with a single call.
package buffkit

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/johnjansen/buffkit/auth"
	"github.com/johnjansen/buffkit/components"
	"github.com/johnjansen/buffkit/importmap"
	"github.com/johnjansen/buffkit/jobs"
	"github.com/johnjansen/buffkit/mail"
	"github.com/johnjansen/buffkit/migrations"
	"github.com/johnjansen/buffkit/secure"
	"github.com/johnjansen/buffkit/ssr"
)

//go:embed public/*
var publicFS embed.FS

// Config holds all configuration for Buffkit packages.
// This is the main configuration struct that controls how Buffkit behaves.
// Each field maps to a specific subsystem's configuration needs.
type Config struct {
	// DevMode enables development features like mail preview at /__mail/preview
	// and relaxes certain security restrictions. Should be false in production.
	DevMode bool

	// AuthSecret is used for session encryption. This MUST be set to a secure
	// random value in production. The session cookies are encrypted with this key.
	// Required field - Wire() will error if not provided.
	AuthSecret []byte

	// RedisURL for background job processing via Asynq. If empty, job enqueuing
	// becomes a no-op (useful for development without Redis). Format:
	// "redis://username:password@localhost:6379/0"
	RedisURL string

	// SMTP configuration for mail sending. If SMTPAddr is empty, a development
	// mail sender is used that logs emails instead of sending them.
	SMTPAddr string // Host:port (e.g., "smtp.sendgrid.net:587")
	SMTPUser string // SMTP username for authentication
	SMTPPass string // SMTP password for authentication

	// Database dialect: "postgres" | "sqlite" | "mysql"
	// This is used for dialect-specific SQL in migrations and stores.
	Dialect string

	// Optional database connection. If not provided, Buffkit will attempt to
	// connect using the DATABASE_URL environment variable. This allows you to
	// either manage the connection yourself or let Buffkit handle it.
	DB *sql.DB
}

// Kit holds references to all Buffkit subsystems after wiring.
// This is returned from Wire() and provides access to all the initialized
// components. You can use these references to interact with Buffkit systems
// directly when needed (e.g., broadcasting SSE events, enqueuing jobs).
type Kit struct {
	// SSR broker for server-sent events. Use this to broadcast real-time
	// updates to connected clients: kit.Broker.Broadcast("event", htmlBytes)
	Broker *ssr.Broker

	// Jobs runtime for background processing. Access the Asynq client to
	// enqueue jobs: kit.Jobs.Client.Enqueue(task)
	Jobs *jobs.Runtime

	// Mail sender interface. Can be used directly to send emails:
	// kit.Mail.Send(ctx, message)
	Mail mail.Sender

	// Auth store for user management. Useful if you need to directly
	// query users: kit.AuthStore.ByEmail(ctx, email)
	AuthStore auth.UserStore

	// Import map manager for JavaScript dependencies. Can be used to
	// dynamically add pins: kit.ImportMap.Pin("name", "url")
	ImportMap *importmap.Manager

	// Component registry for server-side components. Register custom
	// components: kit.Components.Register("my-component", renderer)
	Components *components.Registry

	// Configuration that was used to initialize Buffkit. Useful for
	// checking settings at runtime.
	Config Config
}

// Wire installs all Buffkit packages into a Buffalo application.
// This is the main integration point - call this once in your app.go:
//
//	app := buffalo.New(buffalo.Options{...})
//	kit, err := buffkit.Wire(app, buffkit.Config{
//	    DevMode:    ENV == "development",
//	    AuthSecret: []byte(envy.Get("SESSION_SECRET", "change-me")),
//	    RedisURL:   envy.Get("REDIS_URL", ""),
//	})
//
// Wire performs the following setup:
//  1. Validates configuration (ensures required fields are set)
//  2. Initializes SSR broker and mounts /events endpoint
//  3. Sets up authentication with login/logout routes
//  4. Configures background job processing (if Redis available)
//  5. Initializes mail sending (SMTP or dev mode)
//  6. Sets up import maps with default JavaScript libraries
//  7. Adds security middleware to the request chain
//  8. Registers default server-side components
//  9. Adds helper functions to Buffalo context
//
// The order of initialization matters as some systems depend on others.
// Wire handles this ordering correctly.
func Wire(app *buffalo.App, cfg Config) (*Kit, error) {
	// Validate required configuration.
	// AuthSecret is critical for security - without it, sessions can't be encrypted.
	if len(cfg.AuthSecret) == 0 {
		return nil, fmt.Errorf("buffkit: AuthSecret is required")
	}

	// Initialize the Kit that will hold all our subsystem references
	kit := &Kit{
		Config: cfg,
	}

	// Initialize SSR broker for server-sent events.
	// The broker manages all connected SSE clients and handles broadcasting.
	// It runs in a separate goroutine and includes automatic heartbeats
	// to keep connections alive through proxies and load balancers.
	broker := ssr.NewBroker()
	kit.Broker = broker

	// Mount SSE endpoint at /events.
	// Clients connect here to receive real-time updates. The endpoint
	// handles connection management, heartbeats, and message delivery.
	app.GET("/events", broker.ServeHTTP)

	// Initialize authentication system.
	// Creates a SQL-based user store (or in-memory for development).
	// The store handles user CRUD operations and password verification.
	authStore := auth.NewSQLStore(cfg.DB, cfg.Dialect)
	if authStore != nil {
		kit.AuthStore = authStore
		auth.UseStore(authStore) // Set as global auth store for package-level functions
	} else {
		// Use memory store when no database is configured
		memStore := auth.NewMemoryStore()
		kit.AuthStore = memStore
		auth.UseStore(memStore)
	}

	// Mount authentication routes.
	// These provide the standard login/logout flow:
	// GET /login - shows login form
	// POST /login - processes login (checks credentials, sets session)
	// POST /logout - clears session
	app.GET("/login", auth.LoginFormHandler)
	app.POST("/login", auth.LoginHandler)
	app.POST("/logout", auth.LogoutHandler)

	// Registration routes
	app.GET("/register", auth.RegistrationFormHandler)
	app.POST("/register", auth.RegistrationHandler)

	// Add rate limiting to auth endpoints
	if authStore != nil {
		if extStore, ok := kit.AuthStore.(auth.ExtendedUserStore); ok {
			// Use database-backed rate limiting if extended store is available
			app.Use(auth.DBRateLimitMiddleware(extStore))
		} else {
			// Use in-memory rate limiting as fallback
			app.Use(auth.RateLimitMiddleware(auth.NewRateLimiter()))
		}
	}

	// Email verification
	app.GET("/verify-email", auth.EmailVerificationHandler)

	// Password reset routes
	app.GET("/forgot-password", auth.ForgotPasswordFormHandler)
	app.POST("/forgot-password", auth.ForgotPasswordHandler)
	app.GET("/reset-password", auth.ResetPasswordFormHandler)
	app.POST("/reset-password", auth.ResetPasswordHandler)

	// Profile routes (protected)
	profileGroup := app.Group("/profile")
	profileGroup.Use(auth.RequireLogin)
	profileGroup.GET("/", auth.ProfileHandler)
	profileGroup.POST("/", auth.ProfileUpdateHandler)

	// Session management (protected)
	app.GET("/sessions", auth.RequireLogin(auth.SessionsHandler))
	app.POST("/sessions/{session_id}/revoke", auth.RequireLogin(auth.RevokeSessionHandler))

	// Initialize background job processing if Redis is configured.
	// Jobs use Asynq which requires Redis for queue management.
	// If Redis isn't available, job enqueuing becomes a no-op.
	if cfg.RedisURL != "" {
		runtime, err := jobs.NewRuntime(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("buffkit: failed to initialize jobs: %w", err)
		}
		kit.Jobs = runtime

		// Register default job handlers (email sending, cleanup tasks, etc.)
		runtime.RegisterDefaults()

		// Register authentication background jobs
		if kit.AuthStore != nil {
			if extStore, ok := kit.AuthStore.(auth.ExtendedUserStore); ok {
				auth.RegisterAuthJobs(runtime.Mux, extStore)
			}
		}
	}

	// Initialize mail sending.
	// Uses SMTP if configured, otherwise falls back to development mode
	// which logs emails instead of sending them.
	if cfg.SMTPAddr != "" {
		kit.Mail = mail.NewSMTPSender(mail.SMTPConfig{
			Addr:     cfg.SMTPAddr,
			User:     cfg.SMTPUser,
			Password: cfg.SMTPPass,
		})
	} else {
		// Development sender logs emails and stores them for preview
		kit.Mail = mail.NewDevSender()
	}

	// Set the global mail sender so mail.Send() works
	mail.UseSender(kit.Mail)

	// Mount mail preview endpoint in development mode.
	// This allows developers to see sent emails at /__mail/preview
	// without actually sending them through SMTP.
	if cfg.DevMode {
		app.GET("/__mail/preview", mail.PreviewHandler)
	}

	// Initialize import map manager for JavaScript dependencies.
	// Import maps let us use ES modules without a bundler.
	// The manager handles pins (name->URL mappings) and generates
	// the appropriate <script type="importmap"> tags.
	manager := importmap.NewManager()
	kit.ImportMap = manager

	// Load default pins for common libraries.
	// This includes htmx, Alpine.js, and other essentials.
	// Apps can override these or add their own pins.
	manager.LoadDefaults()

	// Add security middleware to the request chain.
	// This adds headers like X-Frame-Options, X-Content-Type-Options,
	// Content-Security-Policy, etc. DevMode relaxes some restrictions
	// for easier development.
	app.Use(secure.Middleware(secure.Options{
		DevMode: cfg.DevMode,
	}))

	// Initialize the component registry for server-side components.
	// Components are custom HTML elements like <bk-button> that get
	// expanded server-side into full HTML before sending to the client.
	registry := components.NewRegistry()
	kit.Components = registry

	// Register default components (button, card, dropdown, etc.)
	// These provide a base component library that apps can use immediately.
	registry.RegisterDefaults()

	// Add component expansion middleware.
	// This middleware intercepts HTML responses and expands any <bk-*>
	// tags into their full HTML representation. It only processes
	// text/html responses to avoid affecting API responses.
	app.Use(components.ExpanderMiddleware(registry))

	// Add helper functions to Buffalo context.
	// These helpers are available in handlers and templates, making it
	// easy to access Buffkit functionality without passing references around.
	app.Use(func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Add broker for SSE broadcasts.
			// Handlers can access this via c.Value("broker").(*ssr.Broker)
			// to send real-time updates to connected clients.
			c.Set("broker", kit.Broker)

			// Add buffkit reference for auth email sending
			c.Set("buffkit", kit)

			// Add mail sender for direct access
			c.Set("mail_sender", kit.Mail)

			// Add import map helper for templates.
			// Templates can call <%= importmap() %> to render the
			// import map script tag with all configured pins.
			c.Set("importmap", func() string {
				return kit.ImportMap.RenderHTML()
			})

			// Add component render helper for programmatic rendering.
			// Useful for rendering components from handlers:
			// c.Value("component").(func(string, map[string]string) string)("bk-button", attrs)
			c.Set("component", func(name string, attrs map[string]string) string {
				html, _ := kit.Components.Render(name, attrs, nil)
				return string(html)
			})

			return next(c)
		}
	})

	// Set up template override system.
	// Set up template override system.
	// Buffkit templates are embedded and added first, then app templates
	// override them. This allows apps to shadow any Buffkit template.
	// Note: Template override system requires Buffalo app configuration
	// that may not be directly accessible. This is a placeholder for
	// future implementation when Buffalo provides better access to internals.

	// Set up static asset override system.
	// Similar to templates, Buffkit's assets are served first,
	// then app's assets can override them.
	publicRoot, err := fs.Sub(publicFS, "public")
	if err == nil {
		// Mount Buffkit's embedded assets
		// Convert fs.FS to http.FileSystem
		app.ServeFiles("/", http.FS(publicRoot))
	}

	// Initialize database migrations if database is configured.
	// The migration runner handles applying SQL migrations in order.
	// It tracks applied migrations in a buffkit_migrations table.
	if cfg.DB != nil {
		migrationRunner := &MigrationRunner{
			DB:      cfg.DB,
			Dialect: cfg.Dialect,
		}
		// Store runner in context for access by grift tasks.
		// Tasks like buffkit:migrate can retrieve this to run migrations.
		app.Use(func(next buffalo.Handler) buffalo.Handler {
			return func(c buffalo.Context) error {
				c.Set("buffkit.migrations", migrationRunner)
				return next(c)
			}
		})
	}

	return kit, nil
}

// RequireLogin is middleware that ensures user is authenticated.
// Add this to routes or groups that need protection:
//
//	app.GET("/admin", buffkit.RequireLogin(AdminHandler))
//	// or
//	protected := app.Group("/admin")
//	protected.Use(buffkit.RequireLogin)
//
// If the user is not logged in, they are redirected to /login.
// If authenticated, the user object is added to context as "current_user".
func RequireLogin(next buffalo.Handler) buffalo.Handler {
	return auth.RequireLogin(next)
}

// RenderPartial renders a partial template with data.
// This is a helper for rendering fragments that can be used for both
// htmx responses AND SSE broadcasts - ensuring single source of truth
// for HTML fragments:
//
//	html, err := buffkit.RenderPartial(c, "partials/item", map[string]interface{}{
//	    "item": item,
//	})
//	// Send as htmx response
//	c.Render(200, r.HTML(html))
//	// AND broadcast via SSE
//	kit.Broker.Broadcast("item-update", html)
func RenderPartial(c buffalo.Context, name string, data map[string]interface{}) ([]byte, error) {
	return ssr.RenderPartial(c, name, data)
}

// MigrationRunner handles database migrations for Buffkit.
// It manages the buffkit_migrations table that tracks which migrations
// have been applied. Migrations are simple SQL files that are run in
// lexical order.
type MigrationRunner struct {
	// Database connection to run migrations against
	DB *sql.DB

	// Dialect for database-specific SQL ("postgres", "sqlite", "mysql")
	Dialect string

	// Table name for tracking migrations (defaults to "buffkit_migrations")
	Table string
}

// NewMigrationRunner creates a new migration runner.
// It uses the new migrations package implementation.
func NewMigrationRunner(db *sql.DB, migrationFS embed.FS, dialect string) *migrations.Runner {
	return migrations.NewRunner(db, migrationFS, dialect)
}

// Version returns the current Buffkit version.
// This is useful for debugging and ensuring compatibility.
func Version() string {
	return "0.1.0-alpha"
}
