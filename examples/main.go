package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/envy"
	"github.com/johnjansen/buffkit"
	"github.com/johnjansen/buffkit/auth"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed templates/*
var templates embed.FS

// App creates and configures the example Buffalo application demonstrating Buffkit features
func App() *buffalo.App {
	// Load environment
	envy.Load()

	// Create Buffalo app
	app := buffalo.New(buffalo.Options{
		Env:         envy.Get("GO_ENV", "development"),
		SessionName: "_buffkit_example_session",
		Host:        envy.Get("HOST", "http://127.0.0.1:3000"),
	})

	// Setup renderer
	app.Use(func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			c.Set("render", render.New(render.Options{
				HTMLLayout:  "application.plush.html",
				TemplatesFS: templates,
			}))
			return next(c)
		}
	})

	// Setup a simple SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_digest TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Failed to create users table:", err)
	}

	// Create a test user
	hashedPassword, _ := auth.HashPassword("password")
	_, err = db.Exec(`
		INSERT INTO users (id, email, password_digest)
		VALUES ('test-user-1', 'test@example.com', ?)
	`, hashedPassword)
	if err != nil {
		log.Printf("Failed to create test user: %v", err)
	}

	// Wire in Buffkit
	kit, err := buffkit.Wire(app, buffkit.Config{
		DevMode:    true,
		AuthSecret: []byte("test-secret-key-change-in-production"),
		RedisURL:   envy.Get("REDIS_URL", ""), // Optional
		SMTPAddr:   envy.Get("SMTP_ADDR", "localhost:1025"),
		Dialect:    "sqlite",
		DB:         db,
	})
	if err != nil {
		log.Fatal("Failed to wire Buffkit:", err)
	}

	// Public routes
	app.GET("/", HomeHandler)
	app.GET("/about", AboutHandler)
	app.GET("/components", ComponentsHandler)
	app.GET("/sse-demo", SSEDemoHandler)
	app.POST("/broadcast", BroadcastHandler(kit))

	// Protected routes
	protected := app.Group("/protected")
	protected.Use(buffkit.RequireLogin)
	protected.GET("/dashboard", DashboardHandler)
	protected.GET("/profile", ProfileHandler)

	// API routes for testing
	api := app.Group("/api")
	api.GET("/status", StatusHandler(kit))

	return app
}

// Handlers

func HomeHandler(c buffalo.Context) error {
	data := map[string]interface{}{
		"title":   "Buffkit Examples",
		"message": "Welcome to the Buffkit example application!",
		"features": []string{
			"Session Authentication",
			"Server-Sent Events (SSE)",
			"Server-Side Components",
			"Import Maps",
			"Security Headers",
			"Background Jobs (with Redis)",
			"Email Sending",
		},
	}
	return c.Render(http.StatusOK, r{}.HTML("home.plush.html", data))
}

func AboutHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r{}.HTML("about.plush.html"))
}

func ComponentsHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r{}.HTML("components.plush.html"))
}

func SSEDemoHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r{}.HTML("sse_demo.plush.html"))
}

func BroadcastHandler(kit *buffkit.Kit) buffalo.Handler {
	return func(c buffalo.Context) error {
		message := c.Param("message")
		if message == "" {
			message = "Broadcast from server at " + time.Now().Format(time.RFC3339)
		}

		// Create HTML fragment
		html := fmt.Sprintf(`
			<div class="alert alert-info" role="alert">
				<strong>Broadcast:</strong> %s
			</div>
		`, message)

		// Broadcast to all SSE clients
		kit.Broker.Broadcast("message", []byte(html))

		// Return response for htmx
		return c.Render(http.StatusOK, r{}.HTML(html))
	}
}

func DashboardHandler(c buffalo.Context) error {
	user := c.Value("current_user")
	return c.Render(http.StatusOK, r{}.HTML("dashboard.plush.html", map[string]interface{}{
		"user": user,
	}))
}

func ProfileHandler(c buffalo.Context) error {
	user := c.Value("current_user")
	return c.Render(http.StatusOK, r{}.HTML("profile.plush.html", map[string]interface{}{
		"user": user,
	}))
}

func StatusHandler(kit *buffkit.Kit) buffalo.Handler {
	return func(c buffalo.Context) error {
		status := map[string]interface{}{
			"buffkit_version":  buffkit.Version(),
			"environment":      c.Value("env"),
			"dev_mode":         kit.Config.DevMode,
			"redis_configured": kit.Config.RedisURL != "",
			"smtp_configured":  kit.Config.SMTPAddr != "",
			"timestamp":        time.Now().Format(time.RFC3339),
		}
		return c.Render(http.StatusOK, r{}.JSON(status))
	}
}

// Simple renderer helper
type r struct{}

type htmlRenderer struct {
	name string
	data map[string]interface{}
}

func (htmlRenderer) ContentType() string {
	return "text/html; charset=utf-8"
}

func (h htmlRenderer) Render(w io.Writer, ctx render.Data) error {
	// Merge data
	if h.data != nil {
		for k, v := range h.data {
			ctx[k] = v
		}
	}

	// Render based on template name
	switch h.name {
	case "home.plush.html":
		return renderHome(w, ctx)
	case "about.plush.html":
		return renderAbout(w, ctx)
	case "components.plush.html":
		return renderComponents(w, ctx)
	case "sse_demo.plush.html":
		return renderSSEDemo(w, ctx)
	case "dashboard.plush.html":
		return renderDashboard(w, ctx)
	case "profile.plush.html":
		return renderProfile(w, ctx)
	default:
		// For simple HTML strings
		_, err := w.Write([]byte(h.name))
		return err
	}
}

func (r) HTML(name string, data ...map[string]interface{}) render.Renderer {
	var d map[string]interface{}
	if len(data) > 0 {
		d = data[0]
	}
	return htmlRenderer{name: name, data: d}
}

func (r) JSON(data interface{}) render.Renderer {
	return render.JSON(data)
}

// Template rendering functions

func renderLayout(w io.Writer, title, content string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Buffkit Examples</title>
    <style>
        body { font-family: system-ui, sans-serif; line-height: 1.6; margin: 0; padding: 0; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        nav { background: #333; color: white; padding: 1rem; }
        nav a { color: white; text-decoration: none; margin-right: 1rem; }
        nav a:hover { text-decoration: underline; }
        .alert { padding: 1rem; margin: 1rem 0; border-radius: 4px; }
        .alert-info { background: #d1ecf1; border: 1px solid #bee5eb; color: #0c5460; }
        .alert-success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .button { display: inline-block; padding: 0.5rem 1rem; background: #007bff; color: white;
                  text-decoration: none; border-radius: 4px; border: none; cursor: pointer; }
        .button:hover { background: #0056b3; }
        .card { border: 1px solid #ddd; border-radius: 4px; padding: 1rem; margin: 1rem 0; }
        .form-group { margin-bottom: 1rem; }
        .form-control { width: 100%%; padding: 0.5rem; border: 1px solid #ddd; border-radius: 4px; }
        #sse-messages { border: 1px solid #ddd; padding: 1rem; min-height: 200px; margin: 1rem 0; }
    </style>
</head>
<body>
    <nav>
        <div class="container">
            <a href="/">Home</a>
            <a href="/about">About</a>
            <a href="/components">Components</a>
            <a href="/sse-demo">SSE Demo</a>
            <a href="/protected/dashboard">Dashboard (Protected)</a>
            <a href="/login">Login</a>
            <a href="/__mail/preview">Mail Preview (Dev)</a>
        </div>
    </nav>
    <div class="container">
        %s
    </div>
</body>
</html>`, title, content)

	_, err := w.Write([]byte(html))
	return err
}

func renderHome(w io.Writer, data render.Data) error {
	content := `
        <h1>Welcome to Buffkit Examples</h1>
        <p>This is an example application demonstrating the Buffkit plugin system.</p>

        <h2>Features</h2>
        <ul>
            <li>✅ Session-based Authentication</li>
            <li>✅ Server-Sent Events (SSE)</li>
            <li>✅ Server-Side Components</li>
            <li>✅ Import Maps for JavaScript</li>
            <li>✅ Security Headers</li>
            <li>✅ Background Jobs (Redis optional)</li>
            <li>✅ Email Sending</li>
        </ul>

        <h2>Test Credentials</h2>
        <p>Email: test@example.com<br>Password: password</p>

        <h2>Quick Links</h2>
        <p>
            <a href="/api/status" class="button">API Status</a>
            <a href="/login" class="button">Login</a>
            <a href="/components" class="button">Component Demo</a>
        </p>
    `
	return renderLayout(w, "Home", content)
}

func renderAbout(w io.Writer, data render.Data) error {
	content := `
        <h1>About Buffkit</h1>
        <p>Buffkit is an opinionated SSR-first stack for Buffalo that brings Rails-like batteries to server-rendered applications.</p>

        <div class="card">
            <h3>Philosophy</h3>
            <ul>
                <li>Server-side rendering first</li>
                <li>Zero JavaScript bundler</li>
                <li>Batteries included</li>
                <li>Database agnostic</li>
                <li>Everything is shadowable</li>
            </ul>
        </div>

        <div class="card">
            <h3>Version</h3>
            <p>Buffkit v0.1.0-alpha</p>
        </div>
    `
	return renderLayout(w, "About", content)
}

func renderComponents(w io.Writer, data render.Data) error {
	content := `
        <h1>Server-Side Components Demo</h1>
        <p>These components are rendered server-side using the <code>&lt;bk-*&gt;</code> tag system.</p>

        <h2>Button Component</h2>
        <bk-button variant="primary">Primary Button</bk-button>
        <bk-button variant="danger">Danger Button</bk-button>
        <bk-button href="/about">Link Button</bk-button>

        <h2>Card Component</h2>
        <bk-card>
            <bk-slot name="header">
                <h3>Card Header</h3>
            </bk-slot>
            <p>This is the card body content.</p>
            <bk-slot name="footer">
                Card footer with actions
            </bk-slot>
        </bk-card>

        <h2>Alert Component</h2>
        <bk-alert variant="info">This is an info alert</bk-alert>
        <bk-alert variant="success">This is a success alert</bk-alert>
        <bk-alert variant="warning" dismissible="true">This is a dismissible warning</bk-alert>

        <h2>Form Components</h2>
        <bk-form action="/test" method="POST">
            <bk-input name="email" type="email" label="Email Address" placeholder="you@example.com" required="true"></bk-input>
            <bk-input name="password" type="password" label="Password" required="true"></bk-input>
            <bk-button variant="primary">Submit</bk-button>
        </bk-form>
    `
	return renderLayout(w, "Components", content)
}

func renderSSEDemo(w io.Writer, data render.Data) error {
	content := `
        <h1>Server-Sent Events Demo</h1>
        <p>This page demonstrates real-time updates using SSE.</p>

        <div class="card">
            <h3>Send Broadcast</h3>
            <form action="/broadcast" method="POST">
                <div class="form-group">
                    <input type="text" name="message" class="form-control" placeholder="Enter message to broadcast">
                </div>
                <button type="submit" class="button">Broadcast Message</button>
            </form>
        </div>

        <h3>Messages</h3>
        <div id="sse-messages">
            <p>Waiting for messages...</p>
        </div>

        <script>
            if (typeof EventSource !== 'undefined') {
                const source = new EventSource('/events');
                const messagesDiv = document.getElementById('sse-messages');

                source.addEventListener('connected', function(e) {
                    console.log('SSE Connected:', e.data);
                    messagesDiv.innerHTML = '<p>Connected to SSE stream</p>';
                });

                source.addEventListener('message', function(e) {
                    console.log('SSE Message:', e.data);
                    const messageEl = document.createElement('div');
                    messageEl.innerHTML = e.data;
                    messagesDiv.appendChild(messageEl);
                });

                source.addEventListener('heartbeat', function(e) {
                    console.log('SSE Heartbeat:', e.data);
                });

                source.onerror = function(e) {
                    console.error('SSE Error:', e);
                    messagesDiv.innerHTML += '<p style="color: red;">Connection error</p>';
                };
            } else {
                document.getElementById('sse-messages').innerHTML = '<p>Your browser does not support SSE</p>';
            }
        </script>
    `
	return renderLayout(w, "SSE Demo", content)
}

func renderDashboard(w io.Writer, data render.Data) error {
	user := data["user"]
	content := fmt.Sprintf(`
        <h1>Protected Dashboard</h1>
        <div class="alert alert-success">
            You are logged in! User: %v
        </div>

        <div class="card">
            <h3>Dashboard Content</h3>
            <p>This is a protected page that requires authentication.</p>
            <p>Only logged-in users can see this content.</p>
        </div>

        <form action="/logout" method="POST">
            <button type="submit" class="button">Logout</button>
        </form>
    `, user)
	return renderLayout(w, "Dashboard", content)
}

func renderProfile(w io.Writer, data render.Data) error {
	user := data["user"]
	content := fmt.Sprintf(`
        <h1>User Profile</h1>
        <div class="card">
            <h3>Profile Information</h3>
            <p>User details: %v</p>
        </div>
    `, user)
	return renderLayout(w, "Profile", content)
}

func main() {
	// Get the app
	app := App()
	if app == nil {
		log.Fatal("Failed to create app")
	}

	// Print startup info
	fmt.Println("🚀 Buffkit Example App Starting...")
	fmt.Println("📍 URL: http://localhost:3000")
	fmt.Println("👤 Test Login: test@example.com / password")
	fmt.Println("📧 Mail Preview: http://localhost:3000/__mail/preview")
	fmt.Println("🔌 SSE Demo: http://localhost:3000/sse-demo")
	fmt.Println("\nPress Ctrl+C to stop")

	// Start the server
	port := envy.Get("PORT", "3000")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), app))
}
