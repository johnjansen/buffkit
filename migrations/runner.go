package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a single database migration
type Migration struct {
	Version   string    // Version identifier (e.g., "20240101120000")
	Name      string    // Human-readable name (e.g., "create_users_table")
	UpSQL     string    // SQL to apply the migration
	DownSQL   string    // SQL to rollback the migration (optional)
	AppliedAt time.Time // When the migration was applied
}

// Runner handles database migrations for Buffkit applications
type Runner struct {
	DB      *sql.DB  // Database connection
	FS      embed.FS // Embedded filesystem containing migration files
	Dialect string   // Database dialect ("postgres", "sqlite", "mysql")
	Table   string   // Table name for tracking migrations
}

// NewRunner creates a new migration runner with default settings
func NewRunner(db *sql.DB, migrationFS embed.FS, dialect string) *Runner {
	return &Runner{
		DB:      db,
		FS:      migrationFS,
		Dialect: dialect,
		Table:   "buffkit_migrations",
	}
}

// ensureTable creates the migrations tracking table if it doesn't exist
func (r *Runner) ensureTable(ctx context.Context) error {
	var query string

	switch r.Dialect {
	case "postgres":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version VARCHAR(14) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`, r.Table)

	case "mysql":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version VARCHAR(14) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`, r.Table)

	case "sqlite", "sqlite3":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`, r.Table)

	default:
		return fmt.Errorf("unsupported dialect: %s", r.Dialect)
	}

	_, err := r.DB.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns a list of already applied migration versions
func (r *Runner) getAppliedMigrations(ctx context.Context) (map[string]Migration, error) {
	query := fmt.Sprintf("SELECT version, name, applied_at FROM %s ORDER BY version", r.Table)

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]Migration)
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.Name, &m.AppliedAt); err != nil {
			return nil, err
		}
		applied[m.Version] = m
	}

	return applied, rows.Err()
}

// loadMigrations reads all migration files from the embedded filesystem
func (r *Runner) loadMigrations() ([]Migration, error) {
	var migrations []Migration

	// Walk through the migrations directory
	err := fs.WalkDir(r.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(path, ".sql") {
			return nil
		}

		// Parse filename: {version}_{name}.{up|down}.sql
		base := filepath.Base(path)
		parts := strings.Split(base, "_")
		if len(parts) < 2 {
			return nil // Skip malformed filenames
		}

		version := parts[0]

		// Extract name and direction
		remaining := strings.Join(parts[1:], "_")
		var name, direction string

		if strings.HasSuffix(remaining, ".up.sql") {
			name = strings.TrimSuffix(remaining, ".up.sql")
			direction = "up"
		} else if strings.HasSuffix(remaining, ".down.sql") {
			name = strings.TrimSuffix(remaining, ".down.sql")
			direction = "down"
		} else {
			return nil // Skip non-migration files
		}

		// Read file content
		content, err := fs.ReadFile(r.FS, path)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", path, err)
		}

		// Find or create migration entry
		var migration *Migration
		for i := range migrations {
			if migrations[i].Version == version {
				migration = &migrations[i]
				break
			}
		}

		if migration == nil {
			migrations = append(migrations, Migration{
				Version: version,
				Name:    name,
			})
			migration = &migrations[len(migrations)-1]
		}

		// Store SQL content
		if direction == "up" {
			migration.UpSQL = string(content)
		} else {
			migration.DownSQL = string(content)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Migrate applies all pending migrations in order
func (r *Runner) Migrate(ctx context.Context) error {
	// Ensure migrations table exists
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	// Load all migrations
	migrations, err := r.loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		// Skip if already applied
		if _, exists := applied[migration.Version]; exists {
			continue
		}

		// Skip if no up migration
		if migration.UpSQL == "" {
			continue
		}

		// Apply migration
		if err := r.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("applying migration %s_%s: %w",
				migration.Version, migration.Name, err)
		}

		fmt.Printf("Applied migration: %s_%s\n", migration.Version, migration.Name)
	}

	return nil
}

// applyMigration applies a single migration with transaction support where available
func (r *Runner) applyMigration(ctx context.Context, migration Migration) error {
	// MySQL doesn't support transactional DDL well, so we handle it differently
	useTransaction := r.Dialect != "mysql"

	var tx *sql.Tx
	var err error

	if useTransaction {
		tx, err = r.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()
	}

	// Execute the migration SQL
	if useTransaction {
		_, err = tx.ExecContext(ctx, migration.UpSQL)
	} else {
		_, err = r.DB.ExecContext(ctx, migration.UpSQL)
	}

	if err != nil {
		return fmt.Errorf("executing migration SQL: %w", err)
	}

	// Record the migration
	recordQuery := fmt.Sprintf(
		"INSERT INTO %s (version, name, applied_at) VALUES ($1, $2, $3)",
		r.Table,
	)

	// Handle parameter placeholders for different dialects
	if r.Dialect == "mysql" {
		recordQuery = strings.ReplaceAll(recordQuery, "$1", "?")
		recordQuery = strings.ReplaceAll(recordQuery, "$2", "?")
		recordQuery = strings.ReplaceAll(recordQuery, "$3", "?")
	}

	now := time.Now()
	if useTransaction {
		_, err = tx.ExecContext(ctx, recordQuery, migration.Version, migration.Name, now)
	} else {
		_, err = r.DB.ExecContext(ctx, recordQuery, migration.Version, migration.Name, now)
	}

	if err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	// Commit transaction if used
	if useTransaction {
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("committing transaction: %w", err)
		}
	}

	return nil
}

// Status returns the list of applied and pending migrations
func (r *Runner) Status(ctx context.Context) (applied, pending []string, err error) {
	// Ensure migrations table exists
	if err := r.ensureTable(ctx); err != nil {
		return nil, nil, fmt.Errorf("creating migrations table: %w", err)
	}

	// Get applied migrations
	appliedMap, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("getting applied migrations: %w", err)
	}

	// Load all migrations
	migrations, err := r.loadMigrations()
	if err != nil {
		return nil, nil, fmt.Errorf("loading migrations: %w", err)
	}

	// Build lists
	for _, migration := range migrations {
		name := fmt.Sprintf("%s_%s", migration.Version, migration.Name)

		if _, exists := appliedMap[migration.Version]; exists {
			applied = append(applied, name)
		} else if migration.UpSQL != "" {
			pending = append(pending, name)
		}
	}

	return applied, pending, nil
}

// Down rolls back the last N migrations that have down files
func (r *Runner) Down(ctx context.Context, n int) error {
	if n <= 0 {
		return fmt.Errorf("n must be positive")
	}

	// Ensure migrations table exists
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Get applied migrations in reverse order
	query := fmt.Sprintf(
		"SELECT version, name FROM %s ORDER BY version DESC LIMIT %d",
		r.Table, n,
	)

	// MySQL uses LIMIT syntax differently
	if r.Dialect == "postgres" {
		query = fmt.Sprintf(
			"SELECT version, name FROM %s ORDER BY version DESC LIMIT $1",
			r.Table,
		)
	}

	var rows *sql.Rows
	var err error

	if r.Dialect == "postgres" {
		rows, err = r.DB.QueryContext(ctx, query, n)
	} else {
		rows, err = r.DB.QueryContext(ctx, query)
	}

	if err != nil {
		return fmt.Errorf("querying migrations to rollback: %w", err)
	}
	defer rows.Close()

	// Collect migrations to rollback
	var toRollback []Migration
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.Name); err != nil {
			return err
		}
		toRollback = append(toRollback, m)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Load migration files to get down SQL
	allMigrations, err := r.loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	// Create map for quick lookup
	migrationMap := make(map[string]Migration)
	for _, m := range allMigrations {
		migrationMap[m.Version] = m
	}

	// Rollback each migration
	for _, migration := range toRollback {
		// Get the full migration with down SQL
		fullMigration, exists := migrationMap[migration.Version]
		if !exists {
			return fmt.Errorf("migration file not found for version %s", migration.Version)
		}

		if fullMigration.DownSQL == "" {
			return fmt.Errorf("no down migration for %s_%s", migration.Version, migration.Name)
		}

		// Apply rollback
		if err := r.rollbackMigration(ctx, fullMigration); err != nil {
			return fmt.Errorf("rolling back migration %s_%s: %w",
				migration.Version, migration.Name, err)
		}

		fmt.Printf("Rolled back migration: %s_%s\n", migration.Version, migration.Name)
	}

	return nil
}

// rollbackMigration rolls back a single migration
func (r *Runner) rollbackMigration(ctx context.Context, migration Migration) error {
	// MySQL doesn't support transactional DDL well
	useTransaction := r.Dialect != "mysql"

	var tx *sql.Tx
	var err error

	if useTransaction {
		tx, err = r.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()
	}

	// Execute the down migration SQL
	if useTransaction {
		_, err = tx.ExecContext(ctx, migration.DownSQL)
	} else {
		_, err = r.DB.ExecContext(ctx, migration.DownSQL)
	}

	if err != nil {
		return fmt.Errorf("executing down migration SQL: %w", err)
	}

	// Remove the migration record
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE version = $1", r.Table)

	// Handle parameter placeholders for different dialects
	if r.Dialect == "mysql" {
		deleteQuery = strings.ReplaceAll(deleteQuery, "$1", "?")
	}

	if useTransaction {
		_, err = tx.ExecContext(ctx, deleteQuery, migration.Version)
	} else {
		_, err = r.DB.ExecContext(ctx, deleteQuery, migration.Version)
	}

	if err != nil {
		return fmt.Errorf("removing migration record: %w", err)
	}

	// Commit transaction if used
	if useTransaction {
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("committing transaction: %w", err)
		}
	}

	return nil
}

// Reset drops all tables and reruns all migrations (DANGEROUS!)
// This is useful for testing but should never be used in production
func (r *Runner) Reset(ctx context.Context) error {
	// First, get all applied migrations to roll them back
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		// Table might not exist, which is fine
		if err := r.ensureTable(ctx); err != nil {
			return err
		}
	} else if len(applied) > 0 {
		// Roll back all migrations
		if err := r.Down(ctx, len(applied)); err != nil {
			return fmt.Errorf("rolling back migrations: %w", err)
		}
	}

	// Drop the migrations table
	dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", r.Table)
	if _, err := r.DB.ExecContext(ctx, dropQuery); err != nil {
		return fmt.Errorf("dropping migrations table: %w", err)
	}

	// Now run all migrations fresh
	return r.Migrate(ctx)
}
