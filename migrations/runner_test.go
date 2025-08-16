package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed testdata/*.sql
var testMigrations embed.FS

// setupTestDB creates a new in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

func TestNewRunner(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")

	if runner.DB != db {
		t.Error("DB not set correctly")
	}

	if runner.Dialect != "sqlite3" {
		t.Errorf("Expected dialect 'sqlite3', got '%s'", runner.Dialect)
	}

	if runner.Table != "buffkit_migrations" {
		t.Errorf("Expected table 'buffkit_migrations', got '%s'", runner.Table)
	}
}

func TestEnsureTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Ensure table doesn't exist initially
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='buffkit_migrations'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for table: %v", err)
	}
	if count != 0 {
		t.Fatal("Table should not exist initially")
	}

	// Create the table
	err = runner.ensureTable(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure table: %v", err)
	}

	// Check table exists
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='buffkit_migrations'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for table: %v", err)
	}
	if count != 1 {
		t.Fatal("Table should exist after ensureTable")
	}

	// Calling again should not error
	err = runner.ensureTable(ctx)
	if err != nil {
		t.Fatalf("ensureTable should be idempotent: %v", err)
	}
}

func TestLoadMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")

	migrations, err := runner.loadMigrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	// We should have 2 migrations
	if len(migrations) != 2 {
		t.Fatalf("Expected 2 migrations, got %d", len(migrations))
	}

	// Check first migration
	if migrations[0].Version != "20240101120000" {
		t.Errorf("Expected version '20240101120000', got '%s'", migrations[0].Version)
	}
	if migrations[0].Name != "create_users_table" {
		t.Errorf("Expected name 'create_users_table', got '%s'", migrations[0].Name)
	}
	if migrations[0].UpSQL == "" {
		t.Error("UpSQL should not be empty")
	}
	if migrations[0].DownSQL == "" {
		t.Error("DownSQL should not be empty")
	}

	// Check second migration
	if migrations[1].Version != "20240102093000" {
		t.Errorf("Expected version '20240102093000', got '%s'", migrations[1].Version)
	}
	if migrations[1].Name != "add_user_profile" {
		t.Errorf("Expected name 'add_user_profile', got '%s'", migrations[1].Name)
	}

	// Verify migrations are sorted
	if migrations[0].Version >= migrations[1].Version {
		t.Error("Migrations should be sorted by version")
	}
}

func TestMigrate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Run migrations
	err := runner.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Check that migrations were applied
	applied, pending, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(applied) != 2 {
		t.Errorf("Expected 2 applied migrations, got %d", len(applied))
	}
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending migrations, got %d", len(pending))
	}

	// Check that users table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for users table: %v", err)
	}
	if count != 1 {
		t.Fatal("Users table should exist after migration")
	}

	// Check that columns from second migration exist
	// SQLite doesn't have information_schema, so we use PRAGMA
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan column info: %v", err)
		}
		columns[name] = true
	}

	// Check for columns from first migration
	expectedColumns := []string{"id", "email", "username", "password_hash"}
	for _, col := range expectedColumns {
		if !columns[col] {
			t.Errorf("Column %s should exist", col)
		}
	}

	// Check for columns from second migration
	profileColumns := []string{"first_name", "last_name", "bio", "timezone"}
	for _, col := range profileColumns {
		if !columns[col] {
			t.Errorf("Profile column %s should exist", col)
		}
	}

	// Running migrate again should be idempotent
	err = runner.Migrate(ctx)
	if err != nil {
		t.Fatalf("Migrate should be idempotent: %v", err)
	}
}

func TestStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Before any migrations
	applied, pending, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("Expected 0 applied migrations initially, got %d", len(applied))
	}
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending migrations initially, got %d", len(pending))
	}

	// Apply migrations
	err = runner.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// After migrations
	applied, pending, err = runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status after migration: %v", err)
	}

	if len(applied) != 2 {
		t.Errorf("Expected 2 applied migrations, got %d", len(applied))
	}
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending migrations, got %d", len(pending))
	}

	// Check format of applied migrations
	expected := []string{
		"20240101120000_create_users_table",
		"20240102093000_add_user_profile",
	}
	for i, name := range expected {
		if applied[i] != name {
			t.Errorf("Expected migration %s, got %s", name, applied[i])
		}
	}
}

func TestDown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Apply migrations first
	err := runner.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Roll back one migration
	err = runner.Down(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Check status
	applied, pending, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(applied) != 1 {
		t.Errorf("Expected 1 applied migration after rollback, got %d", len(applied))
	}
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending migration after rollback, got %d", len(pending))
	}

	// The first migration should still be applied
	if applied[0] != "20240101120000_create_users_table" {
		t.Errorf("Wrong migration remained: %s", applied[0])
	}

	// The second migration should be pending
	if pending[0] != "20240102093000_add_user_profile" {
		t.Errorf("Wrong migration pending: %s", pending[0])
	}

	// Check that profile columns are gone
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan column info: %v", err)
		}
		columns[name] = true
	}

	// Profile columns should be gone
	profileColumns := []string{"first_name", "last_name", "bio", "timezone"}
	for _, col := range profileColumns {
		if columns[col] {
			t.Errorf("Profile column %s should not exist after rollback", col)
		}
	}

	// Original columns should still exist
	if !columns["email"] {
		t.Error("Email column should still exist")
	}

	// Roll back the remaining migration
	err = runner.Down(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to rollback remaining migration: %v", err)
	}

	// Users table should be gone
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for users table: %v", err)
	}
	if count != 0 {
		t.Fatal("Users table should not exist after full rollback")
	}
}

func TestReset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Apply migrations
	err := runner.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Reset
	err = runner.Reset(ctx)
	if err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	// Check that migrations were reapplied
	applied, pending, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status after reset: %v", err)
	}

	if len(applied) != 2 {
		t.Errorf("Expected 2 applied migrations after reset, got %d", len(applied))
	}
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending migrations after reset, got %d", len(pending))
	}

	// Users table should exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for users table: %v", err)
	}
	if count != 1 {
		t.Fatal("Users table should exist after reset")
	}
}

func TestGetAppliedMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Ensure table exists
	err := runner.ensureTable(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure table: %v", err)
	}

	// Initially no migrations
	applied, err := runner.getAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("Expected 0 applied migrations, got %d", len(applied))
	}

	// Insert a test migration record
	now := time.Now()
	_, err = db.Exec(
		fmt.Sprintf("INSERT INTO %s (version, name, applied_at) VALUES (?, ?, ?)", runner.Table),
		"20240101120000", "test_migration", now,
	)
	if err != nil {
		t.Fatalf("Failed to insert test migration: %v", err)
	}

	// Should now have one migration
	applied, err = runner.getAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}
	if len(applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(applied))
	}

	migration, exists := applied["20240101120000"]
	if !exists {
		t.Fatal("Migration should exist in map")
	}
	if migration.Name != "test_migration" {
		t.Errorf("Expected name 'test_migration', got '%s'", migration.Name)
	}
}

func TestApplyMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Ensure table exists
	err := runner.ensureTable(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure table: %v", err)
	}

	// Create a simple migration
	migration := Migration{
		Version: "20240103100000",
		Name:    "test_table",
		UpSQL:   "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)",
	}

	// Apply the migration
	err = runner.applyMigration(ctx, migration)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Check that table was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for test_table: %v", err)
	}
	if count != 1 {
		t.Fatal("test_table should exist after migration")
	}

	// Check that migration was recorded
	var recordedVersion string
	err = db.QueryRow(
		fmt.Sprintf("SELECT version FROM %s WHERE version = ?", runner.Table),
		"20240103100000",
	).Scan(&recordedVersion)
	if err != nil {
		t.Fatalf("Failed to check migration record: %v", err)
	}
	if recordedVersion != "20240103100000" {
		t.Error("Migration should be recorded in tracking table")
	}
}

func TestRollbackMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	// Setup: ensure table and apply a migration
	err := runner.ensureTable(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure table: %v", err)
	}

	migration := Migration{
		Version: "20240103100000",
		Name:    "test_table",
		UpSQL:   "CREATE TABLE test_rollback (id INTEGER PRIMARY KEY)",
		DownSQL: "DROP TABLE test_rollback",
	}

	// Apply the migration first
	err = runner.applyMigration(ctx, migration)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Now rollback
	err = runner.rollbackMigration(ctx, migration)
	if err != nil {
		t.Fatalf("Failed to rollback migration: %v", err)
	}

	// Check that table was dropped
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_rollback'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for test_rollback table: %v", err)
	}
	if count != 0 {
		t.Fatal("test_rollback table should not exist after rollback")
	}

	// Check that migration record was removed
	err = db.QueryRow(
		fmt.Sprintf("SELECT version FROM %s WHERE version = ?", runner.Table),
		"20240103100000",
	).Scan(&count)
	if err != sql.ErrNoRows {
		t.Error("Migration record should be removed from tracking table")
	}
}

func TestDialectSpecificSQL(t *testing.T) {
	testCases := []struct {
		dialect  string
		expected string
	}{
		{"postgres", "VARCHAR(14)"},
		{"mysql", "VARCHAR(14)"},
		{"sqlite3", "TEXT"},
		{"sqlite", "TEXT"},
	}

	for _, tc := range testCases {
		t.Run(tc.dialect, func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			runner := NewRunner(db, testMigrations, tc.dialect)

			// Just verify the runner accepts the dialect
			if runner.Dialect != tc.dialect {
				t.Errorf("Expected dialect %s, got %s", tc.dialect, runner.Dialect)
			}
		})
	}
}

func TestInvalidDialect(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "invalid")
	ctx := context.Background()

	err := runner.ensureTable(ctx)
	if err == nil {
		t.Fatal("Should error with invalid dialect")
	}
	if err.Error() != "unsupported dialect: invalid" {
		t.Errorf("Expected unsupported dialect error, got: %v", err)
	}
}

func TestDownWithInvalidN(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, testMigrations, "sqlite3")
	ctx := context.Background()

	err := runner.Down(ctx, 0)
	if err == nil {
		t.Fatal("Should error with n=0")
	}

	err = runner.Down(ctx, -1)
	if err == nil {
		t.Fatal("Should error with negative n")
	}
}
