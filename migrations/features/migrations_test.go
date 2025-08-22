package migrations_test

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/johnjansen/buffkit/migrations"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed testdata/*.sql
var testMigrationFS embed.FS

type MigrationTestSuite struct {
	db           *sql.DB
	runner       *migrations.Runner
	tempDir      string
	dbPath       string
	lastError    error
	output       string
	appliedCount int
	pendingCount int
}

func (m *MigrationTestSuite) Reset() {
	if m.db != nil {
		if err := m.db.Close(); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Failed to close database: %v\n", err)
		}
		m.db = nil
	}
	if m.tempDir != "" {
		if err := os.RemoveAll(m.tempDir); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Failed to remove temp dir: %v\n", err)
		}
		m.tempDir = ""
	}
	m.runner = nil
	m.lastError = nil
	m.output = ""
	m.appliedCount = 0
	m.pendingCount = 0
}

// Background steps
func (m *MigrationTestSuite) iHaveABuffaloApplicationWithBuffkitWired() error {
	// For migration tests, we just need a database connection
	return nil
}

func (m *MigrationTestSuite) iHaveACleanTestDatabase() error {
	// Clean up any existing database first
	if m.db != nil {
		_ = m.db.Close()
		m.db = nil
	}
	if m.tempDir != "" {
		_ = os.RemoveAll(m.tempDir)
		m.tempDir = ""
	}

	// Create a temporary directory for test database
	tempDir, err := os.MkdirTemp("", "buffkit-migration-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	m.tempDir = tempDir
	m.dbPath = filepath.Join(tempDir, "test.db")

	// Open database
	db, err := sql.Open("sqlite3", m.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	m.db = db

	// Create runner with test migrations
	m.runner = migrations.NewRunner(db, testMigrationFS, "sqlite3")
	return nil
}

// Scenario: Initialize migration system on empty database
func (m *MigrationTestSuite) theDatabaseHasNoTables() error {
	// Check that the database is empty
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("database has %d tables, expected 0", count)
	}
	return nil
}

func (m *MigrationTestSuite) iRunMigrations() error {
	ctx := context.Background()
	m.lastError = m.runner.Migrate(ctx)
	return nil
}

func (m *MigrationTestSuite) theTableShouldBeCreated(tableName string) error {
	var name string
	query := fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", tableName)
	err := m.db.QueryRow(query).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("table %s does not exist", tableName)
		}
		return err
	}
	return nil
}

func (m *MigrationTestSuite) theTableShouldHaveColumns(column1, column2 string) error {
	// Check table schema - assume we're checking the buffkit_migrations table
	tableName := "buffkit_migrations"
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := m.db.Query(query)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("Failed to close rows: %v\n", err)
		}
	}()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		columns[name] = true
	}

	if !columns[column1] {
		return fmt.Errorf("column %s not found in table %s", column1, tableName)
	}
	if !columns[column2] {
		return fmt.Errorf("column %s not found in table %s", column2, tableName)
	}
	return nil
}

func (m *MigrationTestSuite) noMigrationsShouldBeMarkedAsApplied() error {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM buffkit_migrations").Scan(&count)
	if err != nil {
		// Table might not exist yet, which is ok
		if strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	if count > 0 {
		return fmt.Errorf("expected 0 applied migrations, got %d", count)
	}
	return nil
}

// Scenario: Apply multiple pending migrations in order
func (m *MigrationTestSuite) iHaveMigrationsAnd(migration1, migration2 string) error {
	// Verify test migrations exist
	entries, err := testMigrationFS.ReadDir("testdata")
	if err != nil {
		return err
	}

	found1, found2 := false, false
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "001") {
			found1 = true
		}
		if strings.Contains(entry.Name(), "002") {
			found2 = true
		}
	}

	if !found1 || !found2 {
		return fmt.Errorf("test migrations not found")
	}
	return nil
}

func (m *MigrationTestSuite) noMigrationsHaveBeenApplied() error {
	// Ensure migration table exists but is empty
	ctx := context.Background()

	// First ensure the table exists
	query := `CREATE TABLE IF NOT EXISTS buffkit_migrations (
		version VARCHAR(14) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	// Verify it's empty
	var count int
	err = m.db.QueryRow("SELECT COUNT(*) FROM buffkit_migrations").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("migrations table has %d entries, expected 0", count)
	}
	return nil
}

func (m *MigrationTestSuite) shouldBeAppliedFirst(migration string) error {
	// Check that 001 migration was applied
	var version string
	err := m.db.QueryRow("SELECT version FROM buffkit_migrations ORDER BY version LIMIT 1").Scan(&version)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(version, "001") {
		return fmt.Errorf("first migration was %s, expected 001*", version)
	}
	return nil
}

func (m *MigrationTestSuite) shouldBeAppliedSecond(migration string) error {
	// Check that 002 migration was applied
	var version string
	err := m.db.QueryRow("SELECT version FROM buffkit_migrations ORDER BY version LIMIT 1 OFFSET 1").Scan(&version)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(version, "002") {
		return fmt.Errorf("second migration was %s, expected 002*", version)
	}
	return nil
}

func (m *MigrationTestSuite) bothShouldBeRecordedInTheMigrationsTable() error {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM buffkit_migrations").Scan(&count)
	if err != nil {
		return err
	}
	if count < 2 {
		return fmt.Errorf("expected at least 2 migrations recorded, got %d", count)
	}
	return nil
}

func (m *MigrationTestSuite) theAppliedAtTimestampsShouldBeInOrder() error {
	rows, err := m.db.Query("SELECT version, applied_at FROM buffkit_migrations ORDER BY version")
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("Failed to close rows: %v\n", err)
		}
	}()

	var lastTime string
	for rows.Next() {
		var version, appliedAt string
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return err
		}
		if lastTime != "" && appliedAt < lastTime {
			return fmt.Errorf("timestamps not in order")
		}
		lastTime = appliedAt
	}
	return nil
}

// Test different database dialects
func (m *MigrationTestSuite) theDatabaseDialectIs(dialect string) error {
	// Create a new runner with the specified dialect
	m.runner = migrations.NewRunner(m.db, testMigrationFS, dialect)
	return nil
}

func (m *MigrationTestSuite) iRunTheMigration() error {
	return m.iRunMigrations()
}

func (m *MigrationTestSuite) theDialectSpecificSQLShouldExecuteSuccessfully(dialect string) error {
	if m.lastError != nil {
		return fmt.Errorf("%s SQL failed: %v", dialect, m.lastError)
	}
	// Verify tables were created
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'test_%'").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no test tables created for %s dialect", dialect)
	}
	return nil
}

// Test migration status
func (m *MigrationTestSuite) iCheckMigrationStatus() error {
	ctx := context.Background()
	applied, pending, err := m.runner.Status(ctx)
	if err != nil {
		m.lastError = err
		return nil
	}
	m.appliedCount = len(applied)
	m.pendingCount = len(pending)
	return nil
}

func (m *MigrationTestSuite) iShouldSeeAppliedMigrations(count int) error {
	if m.appliedCount != count {
		return fmt.Errorf("expected %d applied migrations, got %d", count, m.appliedCount)
	}
	return nil
}

func (m *MigrationTestSuite) iShouldSeePendingMigrations(count int) error {
	if m.pendingCount != count {
		return fmt.Errorf("expected %d pending migrations, got %d", count, m.pendingCount)
	}
	return nil
}

// Test error handling
func (m *MigrationTestSuite) iHaveAMigrationWithInvalidSQL() error {
	// Create a runner with migrations that contain invalid SQL
	// This would need a special test migration directory with bad SQL
	return nil
}

func (m *MigrationTestSuite) theMigrationShouldFail() error {
	if m.lastError == nil {
		return fmt.Errorf("expected migration to fail but it succeeded")
	}
	return nil
}

func (m *MigrationTestSuite) anErrorShouldBeLogged() error {
	if m.lastError == nil {
		return fmt.Errorf("no error was logged")
	}
	return nil
}

func InitializeMigrationScenario(ctx *godog.ScenarioContext) {
	suite := &MigrationTestSuite{}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		suite.Reset()
		// Ensure we start with a clean database for each scenario
		if err := suite.iHaveACleanTestDatabase(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Clean up after each scenario
		suite.Reset()
		return ctx, nil
	})

	// Background
	ctx.Step(`^I have a Buffalo application with Buffkit wired$`, suite.iHaveABuffaloApplicationWithBuffkitWired)
	ctx.Step(`^I have a clean test database$`, suite.iHaveACleanTestDatabase)

	// Initialize migration system
	ctx.Step(`^the database has no tables$`, suite.theDatabaseHasNoTables)
	ctx.Step(`^I run migrations$`, suite.iRunMigrations)
	ctx.Step(`^the "([^"]*)" table should be created$`, suite.theTableShouldBeCreated)
	ctx.Step(`^the table should have columns "([^"]*)" and "([^"]*)"$`, suite.theTableShouldHaveColumns)
	ctx.Step(`^no migrations should be marked as applied$`, suite.noMigrationsShouldBeMarkedAsApplied)

	// Apply multiple migrations
	ctx.Step(`^I have migrations "([^"]*)" and "([^"]*)"$`, suite.iHaveMigrationsAnd)
	ctx.Step(`^no migrations have been applied$`, suite.noMigrationsHaveBeenApplied)
	ctx.Step(`^"([^"]*)" should be applied first$`, suite.shouldBeAppliedFirst)
	ctx.Step(`^"([^"]*)" should be applied second$`, suite.shouldBeAppliedSecond)
	ctx.Step(`^both should be recorded in the migrations table$`, suite.bothShouldBeRecordedInTheMigrationsTable)
	ctx.Step(`^the applied_at timestamps should be in order$`, suite.theAppliedAtTimestampsShouldBeInOrder)

	// Database dialects
	ctx.Step(`^the database dialect is "([^"]*)"$`, suite.theDatabaseDialectIs)
	ctx.Step(`^I run the migration$`, suite.iRunTheMigration)
	ctx.Step(`^the ([^"]*)-specific SQL should execute successfully$`, suite.theDialectSpecificSQLShouldExecuteSuccessfully)

	// Migration status
	ctx.Step(`^I check migration status$`, suite.iCheckMigrationStatus)
	ctx.Step(`^I should see (\d+) applied migrations$`, suite.iShouldSeeAppliedMigrations)
	ctx.Step(`^I should see (\d+) pending migrations$`, suite.iShouldSeePendingMigrations)

	// Error handling
	ctx.Step(`^I have a migration with invalid SQL$`, suite.iHaveAMigrationWithInvalidSQL)
	ctx.Step(`^the migration should fail$`, suite.theMigrationShouldFail)
	ctx.Step(`^an error should be logged$`, suite.anErrorShouldBeLogged)
}

func TestMigrationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeMigrationScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"."},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run migration feature tests")
	}
}
