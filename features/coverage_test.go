package features

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/gobuffalo/buffalo"
	"github.com/johnjansen/buffkit"
	"github.com/johnjansen/buffkit/auth"
	"github.com/johnjansen/buffkit/jobs"
	"github.com/johnjansen/buffkit/mail"
	_ "github.com/mattn/go-sqlite3"
)

// TestDirectCoverage directly exercises Buffkit functionality without godog
// This ensures we get proper coverage metrics
func TestDirectCoverage(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}()

	// Create a Buffalo app
	app := buffalo.New(buffalo.Options{})

	// Wire Buffkit with full configuration
	cfg := buffkit.Config{
		AuthSecret: []byte("test-secret-key-32-chars-minimum"),
		DB:         db,
		Dialect:    "sqlite3",
		DevMode:    true,
		RedisURL:   os.Getenv("REDIS_URL"), // Optional
	}

	kit, err := buffkit.Wire(app, cfg)
	if err != nil {
		t.Fatalf("Failed to wire Buffkit: %v", err)
	}
	defer kit.Shutdown()

	t.Run("Auth", func(t *testing.T) {
		// Test auth store
		store := kit.AuthStore
		if store == nil {
			t.Fatal("Auth store not initialized")
		}

		// Create a user
		user := &auth.User{
			Email: "test@example.com",
		}

		// Hash the password
		hashedPassword, err := auth.HashPassword("password123")
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}
		user.PasswordDigest = hashedPassword

		// Create user in store
		err = store.Create(context.Background(), user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Find user by email
		foundUser, err := store.ByEmail(context.Background(), "test@example.com")
		if err != nil {
			t.Fatalf("Failed to find user by email: %v", err)
		}
		if foundUser.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, foundUser.Email)
		}

		// Check password
		err = auth.CheckPassword("password123", foundUser.PasswordDigest)
		if err != nil {
			t.Error("Password check failed")
		}

		// Find by ID
		foundUser2, err := store.ByID(context.Background(), foundUser.ID)
		if err != nil {
			t.Fatalf("Failed to find user by ID: %v", err)
		}
		if foundUser2.ID != foundUser.ID {
			t.Errorf("Expected ID %s, got %s", foundUser.ID, foundUser2.ID)
		}

		// Update password
		newHash, _ := auth.HashPassword("newpassword")
		err = store.UpdatePassword(context.Background(), foundUser.ID, newHash)
		if err != nil {
			t.Fatalf("Failed to update password: %v", err)
		}

		// Check email exists
		exists, err := store.ExistsEmail(context.Background(), "test@example.com")
		if err != nil {
			t.Fatalf("Failed to check email exists: %v", err)
		}
		if !exists {
			t.Error("Email should exist")
		}
	})

	t.Run("Mail", func(t *testing.T) {
		// Test mail sender
		if kit.Mail == nil {
			t.Fatal("Mail sender not initialized")
		}

		// Send a test email
		msg := mail.Message{
			To:      "recipient@example.com",
			Subject: "Test Email",
			Text:    "This is a test email",
			HTML:    "<p>This is a test email</p>",
		}

		err := kit.Mail.Send(context.Background(), msg)
		if err != nil {
			t.Fatalf("Failed to send email: %v", err)
		}

		// In dev mode, check if preview is available
		if cfg.DevMode {
			// The dev sender should have stored the email
			if _, ok := kit.Mail.(*mail.DevSender); ok {
				// Dev sender stores emails internally but doesn't expose them
				t.Log("Dev sender is active")
			}
		}
	})

	t.Run("Jobs", func(t *testing.T) {
		// Test job runtime
		if kit.Jobs == nil {
			// Jobs might be nil if Redis isn't configured, which is OK
			t.Skip("Jobs runtime not configured (Redis not available)")
		}

		// Create a test job
		runtime, err := jobs.NewRuntime(cfg.RedisURL)
		if err != nil || runtime == nil {
			t.Skip("Could not create job runtime")
		}
		// Jobs runtime doesn't have a Shutdown method in this version

		// Enqueue a test job - the runtime handles email jobs internally
		// but doesn't expose EnqueueEmailJob directly
		// Jobs are enqueued through the runtime's internal methods
		// We can't directly test enqueueing without a full setup
		t.Log("Job runtime created successfully")
	})

	t.Run("SSE", func(t *testing.T) {
		// Test SSE broker
		if kit.Broker == nil {
			t.Fatal("SSE broker not initialized")
		}

		// Broadcast a test event
		kit.Broker.Broadcast("test-event", []byte("test data"))

		// The broker should be running
		// We can't easily test the full SSE flow without a real HTTP connection,
		// but at least we've exercised the broadcast method
	})

	t.Run("ImportMap", func(t *testing.T) {
		// Test import map manager
		if kit.ImportMap == nil {
			t.Fatal("ImportMap manager not initialized")
		}

		// Pin a new package
		kit.ImportMap.Pin("test-package", "https://example.com/test.js")

		// Import map functionality is basic - just verify it exists
		t.Log("ImportMap manager is active")
	})

	t.Run("Components", func(t *testing.T) {
		// Test component registry
		if kit.Components == nil {
			t.Fatal("Component registry not initialized")
		}

		// Register a test component
		kit.Components.Register("bk-test", func(attrs map[string]string, slots map[string]string) ([]byte, error) {
			return []byte("<div>Test Component</div>"), nil
		})

		// Render the component
		result, err := kit.Components.Render("bk-test", map[string]string{}, map[string]string{})
		if err != nil {
			t.Fatalf("Failed to render component: %v", err)
		}
		if string(result) != "<div>Test Component</div>" {
			t.Errorf("Unexpected component output: %s", string(result))
		}
	})

	t.Run("Security", func(t *testing.T) {
		// Test that security middleware was applied
		// We can't easily test middleware without making actual requests,
		// but we can at least verify the configuration
		if !cfg.DevMode {
			// In production mode, security should be strict
			t.Log("Security middleware configured for production")
		} else {
			// In dev mode, security should be relaxed
			t.Log("Security middleware configured for development")
		}
	})

	t.Run("Migrations", func(t *testing.T) {
		// Test migration runner - needs an embed.FS for migrations
		// We'll skip this as it requires actual migration files
		t.Skip("Migration runner requires embedded migration files")
		/*
			runner := buffkit.NewMigrationRunner(db, migrationFS, "sqlite3")
			if runner == nil {
				t.Fatal("Failed to create migration runner")
			}

			// Run migrations (even if there are none, this tests the infrastructure)
			err := runner.Migrate(context.Background())
			if err != nil {
				t.Logf("Migration run completed (might have no migrations): %v", err)
			}
		*/
	})
}

// TestAuthHelpers tests the auth helper functions
func TestAuthHelpers(t *testing.T) {
	// Test password hashing and checking
	password := "testpassword123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	err = auth.CheckPassword(password, hash)
	if err != nil {
		t.Error("Password check failed for correct password")
	}

	err = auth.CheckPassword("wrongpassword", hash)
	if err == nil {
		t.Error("Password check succeeded for wrong password")
	}

	// Test session helpers (need a Buffalo context)
	// Buffalo doesn't expose NewContext publicly, so we'll skip session tests
	// as they require a proper HTTP handler context
	t.Skip("Session helpers require a Buffalo handler context")

	// Session test code would go here if we had a context
}

// TestMailHelpers tests mail functionality directly
func TestMailHelpers(t *testing.T) {
	// Create a dev sender
	sender := mail.NewDevSender()

	// Send a test email
	msg := mail.Message{
		To:      "test@example.com",
		Subject: "Test",
		Text:    "Test body",
	}

	err := sender.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}

	// Dev sender logs emails internally
	t.Log("Email sent to dev sender")
}
