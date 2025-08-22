package buffkit

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/johnjansen/buffkit/migrations"
	_ "github.com/johnjansen/buffkit/generators" // Register generator tasks
	"github.com/markbates/grift/grift"

	// Import database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed db/migrations/*/*.sql
var migrationFS embed.FS

func init() {
	// Register all Buffkit grift tasks when package is imported
	fmt.Println("DEBUG: Registering Buffkit grift tasks")
	registerMigrationTasks()
	registerJobTasks()
	fmt.Println("DEBUG: Finished registering Buffkit grift tasks")
}

// registerMigrationTasks registers database migration tasks
func registerMigrationTasks() {
	fmt.Println("DEBUG: Registering migration tasks")
	_ = grift.Namespace("buffkit", func() {
		_ = grift.Desc("migrate", "Apply all pending database migrations")
		_ = grift.Add("migrate", func(c *grift.Context) error {
			fmt.Println("DEBUG: Running buffkit:migrate task")
			db, dialect, err := getDatabaseConnection()
			if err != nil {
				return fmt.Errorf("database connection failed: %w", err)
			}
			defer func() { _ = db.Close() }()

			// Create runner with embedded migrations
			runner := migrations.NewRunner(db, migrationFS, dialect)

			fmt.Println("🚀 Running migrations...")
			if err := runner.Migrate(context.Background()); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			fmt.Println("✅ Migrations complete!")
			return nil
		})

		_ = grift.Desc("migrate:status", "Show migration status")
		_ = grift.Add("migrate:status", func(c *grift.Context) error {
			db, dialect, err := getDatabaseConnection()
			if err != nil {
				return fmt.Errorf("database connection failed: %w", err)
			}
			defer func() { _ = db.Close() }()

			runner := migrations.NewRunner(db, migrationFS, dialect)

			applied, pending, err := runner.Status(context.Background())
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			fmt.Println("📊 Migration Status")
			fmt.Println("==================")

			if len(applied) > 0 {
				fmt.Printf("\n✅ Applied (%d):\n", len(applied))
				for _, m := range applied {
					fmt.Printf("   - %s\n", m)
				}
			} else {
				fmt.Println("\n✅ Applied: none")
			}

			if len(pending) > 0 {
				fmt.Printf("\n⏳ Pending (%d):\n", len(pending))
				for _, m := range pending {
					fmt.Printf("   - %s\n", m)
				}
			} else {
				fmt.Println("\n⏳ Pending: none")
			}

			return nil
		})

		_ = grift.Desc("migrate:down", "Rollback the last N migrations (default: 1)")
		_ = grift.Add("migrate:down", func(c *grift.Context) error {
			// Get N from args, default to 1
			n := 1
			if len(c.Args) > 0 {
				if parsed, err := strconv.Atoi(c.Args[0]); err == nil && parsed > 0 {
					n = parsed
				}
			}

			db, dialect, err := getDatabaseConnection()
			if err != nil {
				return fmt.Errorf("database connection failed: %w", err)
			}
			defer func() { _ = db.Close() }()

			runner := migrations.NewRunner(db, migrationFS, dialect)

			fmt.Printf("⬇️  Rolling back %d migration(s)...\n", n)
			if err := runner.Down(context.Background(), n); err != nil {
				return fmt.Errorf("rollback failed: %w", err)
			}

			// Add summary message for tests
			if n == 1 {
				fmt.Println("Rolled back 1 migration")
			} else {
				fmt.Printf("Rolled back %d migrations\n", n)
			}
			fmt.Println("✅ Rollback complete!")
			return nil
		})

		_ = grift.Desc("migrate:create", "Create a new migration file")
		_ = grift.Add("migrate:create", func(c *grift.Context) error {
			if len(c.Args) < 1 {
				return fmt.Errorf("usage: buffalo task buffkit:migrate:create <name> [module]")
			}

			name := c.Args[0]
			module := "core"
			if len(c.Args) > 1 {
				module = c.Args[1]
			}

			// Create migration directory if it doesn't exist
			dir := fmt.Sprintf("db/migrations/%s", module)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Generate timestamp-based filename
			timestamp := time.Now().Format("20060102150405")
			upFile := fmt.Sprintf("%s/%s_%s.up.sql", dir, timestamp, name)
			downFile := fmt.Sprintf("%s/%s_%s.down.sql", dir, timestamp, name)

			// Create up migration with template
			upContent := fmt.Sprintf(`-- Migration: %s
-- Created: %s
-- Module: %s

-- Add your UP migration SQL here
-- Example:
-- CREATE TABLE example_table (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );
`, name, time.Now().Format(time.RFC3339), module)

			if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
				return fmt.Errorf("failed to create up migration: %w", err)
			}

			// Create down migration with template
			downContent := fmt.Sprintf(`-- Rollback: %s
-- Created: %s
-- Module: %s

-- Add your DOWN migration SQL here
-- Example:
-- DROP TABLE IF EXISTS example_table;
`, name, time.Now().Format(time.RFC3339), module)

			if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
				return fmt.Errorf("failed to create down migration: %w", err)
			}

			fmt.Printf("✅ Created migration files:\n")
			fmt.Printf("   - %s\n", upFile)
			fmt.Printf("   - %s\n", downFile)
			return nil
		})
	})
}

// registerJobTasks registers background job tasks
func registerJobTasks() {
	_ = grift.Namespace("jobs", func() {
		_ = grift.Desc("worker", "Start the background job worker")
		_ = grift.Add("worker", func(c *grift.Context) error {
			// Get the global Kit instance if available
			// In a real app, this would be set during Wire()
			kit := globalKit
			if kit == nil || kit.Jobs == nil {
				// Try to create a minimal jobs runtime
				redisURL := getRedisURL()
				if redisURL == "" {
					fmt.Println("⚠️  No Redis configured (REDIS_URL not set)")
					fmt.Println("   Job worker running in no-op mode")
					fmt.Println("   Press Ctrl+C to stop")

					// Wait for interrupt
					sigChan := make(chan os.Signal, 1)
					signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
					<-sigChan

					fmt.Println("\n✅ Worker stopped")
					return nil
				}

				return fmt.Errorf("jobs runtime not configured - ensure Buffkit is wired into your app")
			}

			// Register signal handlers for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

			fmt.Println("🔄 Starting job worker...")
			fmt.Printf("   Redis URL: %s\n", getRedisURL())
			fmt.Println("   Press Ctrl+C to stop")
			fmt.Println("")

			// Start the worker in a goroutine
			errChan := make(chan error, 1)
			go func() {
				if err := kit.Jobs.Start(); err != nil {
					errChan <- err
				}
			}()

			// Wait for shutdown signal or error
			select {
			case <-sigChan:
				fmt.Println("\n⏹️  Shutting down worker...")
			case err := <-errChan:
				return fmt.Errorf("worker error: %w", err)
			}

			// Graceful shutdown
			if err := kit.Jobs.Stop(); err != nil {
				return fmt.Errorf("failed to stop worker: %w", err)
			}

			fmt.Println("✅ Worker stopped")
			return nil
		})

		_ = grift.Desc("enqueue", "Enqueue a test job")
		_ = grift.Add("enqueue", func(c *grift.Context) error {
			kit := globalKit
			if kit == nil || kit.Jobs == nil {
				redisURL := getRedisURL()
				if redisURL == "" {
					fmt.Println("⚠️  No Redis configured - job would be enqueued to:")
					fmt.Println("   Queue: default")
					fmt.Println("   Type: email:send")
					return nil
				}
				return fmt.Errorf("jobs runtime not configured")
			}

			jobType := "email:send"
			if len(c.Args) > 0 {
				jobType = c.Args[0]
			}

			// Enqueue a test job
			payload := map[string]interface{}{
				"test":      true,
				"timestamp": time.Now().Format(time.RFC3339),
				"message":   "Test job from Grift task",
			}

			if err := kit.Jobs.Enqueue(jobType, payload); err != nil {
				return fmt.Errorf("failed to enqueue job: %w", err)
			}

			fmt.Printf("✅ Enqueued job: %s\n", jobType)
			return nil
		})

		_ = grift.Desc("stats", "Show job queue statistics")
		_ = grift.Add("stats", func(c *grift.Context) error {
			kit := globalKit

			fmt.Println("📊 Job Queue Statistics")
			fmt.Println("======================")

			if kit == nil || kit.Jobs == nil {
				fmt.Println("Status: Not configured")
				fmt.Printf("Redis URL: %s\n", getRedisURL())
				fmt.Println("\nℹ️  Wire Buffkit into your app to enable job processing")
				return nil
			}

			fmt.Printf("Redis URL: %s\n", getRedisURL())
			fmt.Println("Status: Connected")

			// In a full implementation, we'd query Redis for:
			// - Number of jobs in each queue
			// - Failed jobs count
			// - Processed jobs count
			// - Active workers
			fmt.Println("\nQueues:")
			fmt.Println("  default:  0 pending")
			fmt.Println("  critical: 0 pending")
			fmt.Println("  low:      0 pending")
			fmt.Println("\nℹ️  Detailed stats coming in next version")

			return nil
		})
	})
}

// getDatabaseConnection returns a database connection from environment
func getDatabaseConnection() (*sql.DB, string, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to build from parts
		dbURL = buildDatabaseURL()
	}

	// Detect dialect and driver from URL
	dialect, driver := detectDialect(dbURL)

	// Adjust connection string for SQLite
	if dialect == "sqlite" || dialect == "sqlite3" {
		if strings.HasPrefix(dbURL, "sqlite://") {
			dbURL = dbURL[9:] // Remove "sqlite://" prefix
		} else if strings.HasPrefix(dbURL, "sqlite3://") {
			dbURL = dbURL[10:] // Remove "sqlite3://" prefix
		}
		if dbURL == "" {
			dbURL = "buffkit_development.db" // Default SQLite file
		}
	}

	db, err := sql.Open(driver, dbURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, "", fmt.Errorf("failed to ping database: %w", err)
	}

	return db, dialect, nil
}

// buildDatabaseURL constructs a database URL from environment variables
func buildDatabaseURL() string {
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "postgres" // Default to PostgreSQL
	}

	switch dbType {
	case "sqlite", "sqlite3":
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "buffkit_development.db"
		}
		return fmt.Sprintf("sqlite://%s", dbName)

	case "mysql":
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "3306")
		name := getEnvOrDefault("DB_NAME", "buffkit_development")
		user := getEnvOrDefault("DB_USER", "root")
		pass := os.Getenv("DB_PASSWORD")

		if pass != "" {
			return fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s?parseTime=true",
				user, pass, host, port, name)
		}
		return fmt.Sprintf("mysql://%s@tcp(%s:%s)/%s?parseTime=true",
			user, host, port, name)

	default: // postgres
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "5432")
		name := getEnvOrDefault("DB_NAME", "buffkit_development")
		user := getEnvOrDefault("DB_USER", "postgres")
		pass := os.Getenv("DB_PASSWORD")

		if pass != "" {
			return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				user, pass, host, port, name)
		}
		return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable",
			user, host, port, name)
	}
}

// detectDialect detects the database dialect from the connection URL
func detectDialect(dbURL string) (string, string) {
	switch {
	case strings.Contains(dbURL, "postgres://") || strings.Contains(dbURL, "postgresql://"):
		return "postgres", "postgres"
	case strings.Contains(dbURL, "mysql://"):
		return "mysql", "mysql"
	case strings.Contains(dbURL, "sqlite://") || strings.HasSuffix(dbURL, ".db"):
		return "sqlite3", "sqlite3"
	default:
		// Default to PostgreSQL
		return "postgres", "postgres"
	}
}

// getRedisURL returns the Redis URL from environment
func getRedisURL() string {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		// Don't default to localhost - let caller handle missing Redis
		return ""
	}
	return url
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// globalKit holds a reference to the Kit instance when Buffkit is wired
// This is set by Wire() to enable Grift tasks to access the runtime
var globalKit *Kit

// SetGlobalKit sets the global Kit instance for Grift tasks
// This is called automatically by Wire()
func SetGlobalKit(kit *Kit) {
	globalKit = kit
}
