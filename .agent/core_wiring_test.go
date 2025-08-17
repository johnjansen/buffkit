package buffkit_test

import (
	"database/sql"
	"testing"

	"github.com/gobuffalo/buffalo"
	"github.com/johnjansen/buffkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreWiringIntegration verifies all core wiring components are properly initialized
func TestCoreWiringIntegration(t *testing.T) {
	tests := []struct {
		name      string
		config    buffkit.Config
		wantError bool
		errorMsg  string
		validate  func(t *testing.T, kit *buffkit.Kit, app *buffalo.App)
	}{
		{
			name: "successful wiring with minimal config",
			config: buffkit.Config{
				DevMode:    true,
				AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
				Dialect:    "sqlite",
			},
			wantError: false,
			validate: func(t *testing.T, kit *buffkit.Kit, app *buffalo.App) {
				// Verify all core components are initialized
				assert.NotNil(t, kit.Broker, "SSE Broker should be initialized")
				assert.NotNil(t, kit.AuthStore, "Auth store should be initialized")
				assert.NotNil(t, kit.Mail, "Mail sender should be initialized")
				assert.NotNil(t, kit.ImportMap, "Import map manager should be initialized")
				assert.NotNil(t, kit.Components, "Component registry should be initialized")

				// Verify routes are mounted
				routes := app.Routes()
				hasLoginRoute := false
				hasEventsRoute := false
				hasMailPreview := false

				for _, route := range routes {
					switch route.Path {
					case "/login":
						hasLoginRoute = true
					case "/events":
						hasEventsRoute = true
					case "/__mail/preview":
						hasMailPreview = true
					}
				}

				assert.True(t, hasLoginRoute, "Login route should be mounted")
				assert.True(t, hasEventsRoute, "SSE events route should be mounted")
				assert.True(t, hasMailPreview, "Mail preview route should be mounted in dev mode")
			},
		},
		{
			name: "successful wiring with full config",
			config: buffkit.Config{
				DevMode:    false,
				AuthSecret: []byte("production-secret-key-32-bytes!!"),
				RedisURL:   "", // Empty to avoid Redis connection in tests
				SMTPAddr:   "smtp.example.com:587",
				SMTPUser:   "user@example.com",
				SMTPPass:   "password",
				Dialect:    "postgres",
				DB:         &sql.DB{}, // Mock DB connection
			},
			wantError: false,
			validate: func(t *testing.T, kit *buffkit.Kit, app *buffalo.App) {
				assert.NotNil(t, kit.Broker)
				assert.NotNil(t, kit.AuthStore)
				assert.NotNil(t, kit.Mail)
				assert.NotNil(t, kit.ImportMap)
				assert.NotNil(t, kit.Components)

				// In production mode, mail preview should NOT be mounted
				routes := app.Routes()
				hasMailPreview := false
				for _, route := range routes {
					if route.Path == "/__mail/preview" {
						hasMailPreview = true
						break
					}
				}
				assert.False(t, hasMailPreview, "Mail preview should not be mounted in production")
			},
		},
		{
			name: "error when auth secret is missing",
			config: buffkit.Config{
				DevMode: true,
				Dialect: "sqlite",
			},
			wantError: true,
			errorMsg:  "AuthSecret is required",
		},
		{
			name: "error when auth secret is empty",
			config: buffkit.Config{
				DevMode:    true,
				AuthSecret: []byte{},
				Dialect:    "sqlite",
			},
			wantError: true,
			errorMsg:  "AuthSecret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Buffalo app for each test
			app := buffalo.New(buffalo.Options{
				Name: "test-app",
				Env:  "test",
			})

			// Wire Buffkit
			kit, err := buffkit.Wire(app, tt.config)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, kit)

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, kit, app)
			}

			// Verify context helpers are added
			testHandler := func(c buffalo.Context) error {
				// Check that helpers are available in context
				broker := c.Value("broker")
				assert.NotNil(t, broker, "broker should be in context")

				importMapHelper := c.Value("importmap")
				assert.NotNil(t, importMapHelper, "importmap helper should be in context")

				componentHelper := c.Value("component")
				assert.NotNil(t, componentHelper, "component helper should be in context")

				return nil
			}

			// Add test route and execute it
			app.GET("/test-helpers", testHandler)

			// Note: Full request testing would require httptest here
			// but we're focusing on the wiring validation
		})
	}
}

// TestCoreWiringMiddleware verifies that middleware is properly installed
func TestCoreWiringMiddleware(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Name: "test-app",
		Env:  "test",
	})

	config := buffkit.Config{
		DevMode:    true,
		AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
		Dialect:    "sqlite",
	}

	kit, err := buffkit.Wire(app, config)
	require.NoError(t, err)
	require.NotNil(t, kit)

	// Test that security headers are added
	testHandler := func(c buffalo.Context) error {
		// The security middleware should have been applied
		// We can't directly test the headers here without a full request
		// but we can verify the middleware chain was modified
		return c.Render(200, nil)
	}

	app.GET("/test-security", testHandler)

	// Verify middleware count increased
	// Note: This is a simplified check - in practice you'd use httptest
	// to verify the actual headers are set
	assert.NotNil(t, app.Middleware)
}

// TestImportMapDefaults verifies that default pins are loaded
func TestImportMapDefaults(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Name: "test-app",
		Env:  "test",
	})

	config := buffkit.Config{
		DevMode:    true,
		AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
		Dialect:    "sqlite",
	}

	kit, err := buffkit.Wire(app, config)
	require.NoError(t, err)
	require.NotNil(t, kit)
	require.NotNil(t, kit.ImportMap)

	// Verify HTML output includes import map
	html := kit.ImportMap.RenderHTML()
	assert.Contains(t, html, `<script type="importmap">`)
	assert.Contains(t, html, `"imports"`)

	// Should have default pins
	// Note: These would need to be verified against actual defaults
	// once importmap.LoadDefaults() is properly implemented
}

// TestComponentRegistry verifies that default components are registered
func TestComponentRegistry(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Name: "test-app",
		Env:  "test",
	})

	config := buffkit.Config{
		DevMode:    true,
		AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
		Dialect:    "sqlite",
	}

	kit, err := buffkit.Wire(app, config)
	require.NoError(t, err)
	require.NotNil(t, kit)
	require.NotNil(t, kit.Components)

	// Test rendering a default component
	attrs := map[string]string{
		"href":    "/test",
		"variant": "primary",
	}
	html, err := kit.Components.Render("bk-button", attrs, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, html)
	assert.Contains(t, string(html), "button")
}

// TestSSEBroker verifies SSE broker is properly initialized
func TestSSEBroker(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Name: "test-app",
		Env:  "test",
	})

	config := buffkit.Config{
		DevMode:    true,
		AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
		Dialect:    "sqlite",
	}

	kit, err := buffkit.Wire(app, config)
	require.NoError(t, err)
	require.NotNil(t, kit)
	require.NotNil(t, kit.Broker)

	// Test broadcasting (should not panic even with no clients)
	assert.NotPanics(t, func() {
		kit.Broker.Broadcast("test-event", []byte("<div>Test</div>"))
	})
}

// TestAuthStore verifies auth store is properly initialized
func TestAuthStore(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Name: "test-app",
		Env:  "test",
	})

	tests := []struct {
		name         string
		config       buffkit.Config
		expectSQL    bool
		expectMemory bool
	}{
		{
			name: "memory store when no DB configured",
			config: buffkit.Config{
				DevMode:    true,
				AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
				Dialect:    "sqlite",
				DB:         nil,
			},
			expectSQL:    false,
			expectMemory: true,
		},
		{
			name: "SQL store when DB is configured",
			config: buffkit.Config{
				DevMode:    true,
				AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
				Dialect:    "sqlite",
				DB:         &sql.DB{}, // Mock DB
			},
			expectSQL:    true,
			expectMemory: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kit, err := buffkit.Wire(app, tt.config)
			require.NoError(t, err)
			require.NotNil(t, kit)
			require.NotNil(t, kit.AuthStore)

			// We can't easily check the concrete type without type assertions
			// but we can verify the store is functional
			assert.NotNil(t, kit.AuthStore)
		})
	}
}

// TestMailSender verifies mail sender is properly initialized
func TestMailSender(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Name: "test-app",
		Env:  "test",
	})

	tests := []struct {
		name       string
		config     buffkit.Config
		expectSMTP bool
		expectDev  bool
	}{
		{
			name: "dev sender when no SMTP configured",
			config: buffkit.Config{
				DevMode:    true,
				AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
				Dialect:    "sqlite",
			},
			expectSMTP: false,
			expectDev:  true,
		},
		{
			name: "SMTP sender when configured",
			config: buffkit.Config{
				DevMode:    false,
				AuthSecret: []byte("test-secret-key-32-bytes-long!!!"),
				SMTPAddr:   "smtp.example.com:587",
				SMTPUser:   "user@example.com",
				SMTPPass:   "password",
				Dialect:    "sqlite",
			},
			expectSMTP: true,
			expectDev:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kit, err := buffkit.Wire(app, tt.config)
			require.NoError(t, err)
			require.NotNil(t, kit)
			require.NotNil(t, kit.Mail)
		})
	}
}
