package features

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/markbates/grift/grift"

	// Import buffkit to register grift tasks
	_ "github.com/johnjansen/buffkit"

	// Import database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// GriftTestSuite provides a test context for grift task testing
type GriftTestSuite struct {
	output       string
	errorOutput  string
	lastError    error
	testDB       *sql.DB
	dbPath       string
	tempDir      string
	env          map[string]string
	cleanupFuncs []func()
	shared       *SharedContext // For accessing shared environment variables
}

// NewGriftTestSuite creates a new grift test suite
func NewGriftTestSuite() *GriftTestSuite {
	return &GriftTestSuite{
		env:          make(map[string]string),
		cleanupFuncs: make([]func(), 0),
	}
}

// Reset clears the test state
func (g *GriftTestSuite) Reset() {
	g.output = ""
	g.errorOutput = ""
	g.lastError = nil
	g.env = make(map[string]string)

	// Run cleanup functions
	for i := len(g.cleanupFuncs) - 1; i >= 0; i-- {
		if fn := g.cleanupFuncs[i]; fn != nil {
			fn()
		}
	}
	g.cleanupFuncs = nil

	// Close test database if open
	if g.testDB != nil {
		_ = g.testDB.Close()
		g.testDB = nil
	}

	// Clean up temp directory
	if g.tempDir != "" {
		_ = os.RemoveAll(g.tempDir)
		g.tempDir = ""
	}
}

// Step definitions

func (g *GriftTestSuite) setSharedContext(shared *SharedContext) {
	g.shared = shared
}

func (g *GriftTestSuite) iHaveACleanTestDatabase() error {
	// Create a temporary directory for test database
	tempDir, err := os.MkdirTemp("", "buffkit-grift-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	g.tempDir = tempDir
	g.cleanupFuncs = append(g.cleanupFuncs, func() {
		_ = os.RemoveAll(tempDir)
	})

	// Create test database path
	g.dbPath = fmt.Sprintf("%s/test.db", tempDir)

	// Set environment variable for database
	_ = os.Setenv("DATABASE_URL", g.dbPath)
	fmt.Printf("DEBUG: Set DATABASE_URL to %s in iHaveACleanTestDatabase\n", g.dbPath)
	g.cleanupFuncs = append(g.cleanupFuncs, func() {
		_ = os.Unsetenv("DATABASE_URL")
	})

	// Open database to verify it works
	db, err := sql.Open("sqlite3", g.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open test database: %w", err)
	}
	g.testDB = db

	return nil
}

func (g *GriftTestSuite) iRunGriftTask(taskName string) error {
	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes to capture output
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Ensure we restore stdout/stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Channel to collect output
	outChan := make(chan string)
	errChan := make(chan string)

	// Read from pipes in goroutines
	go func() {
		buf := make([]byte, 1024)
		var output strings.Builder
		for {
			n, err := rOut.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		outChan <- output.String()
	}()

	go func() {
		buf := make([]byte, 1024)
		var output strings.Builder
		for {
			n, err := rErr.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		errChan <- output.String()
	}()

	// Apply environment variables from shared context if available
	if g.shared != nil {
		for k, v := range g.shared.Environment {
			oldVal := os.Getenv(k)
			_ = os.Setenv(k, v)
			defer func(key, val string) { _ = os.Setenv(key, val) }(k, oldVal)
			fmt.Printf("DEBUG: Setting env %s=%s (was %s)\n", k, v, oldVal)
		}
	}

	// Debug: Check current DATABASE_URL
	fmt.Printf("DEBUG: DATABASE_URL before task: %s\n", os.Getenv("DATABASE_URL"))
	fmt.Printf("DEBUG: Running grift task: %s\n", taskName)

	// List available tasks for debugging
	fmt.Printf("DEBUG: Available grift tasks:\n")
	for _, name := range grift.List() {
		fmt.Printf("DEBUG:   - %s\n", name)
	}

	// Create grift context and run task
	ctx := grift.NewContext(taskName)
	g.lastError = grift.Run(taskName, ctx)

	fmt.Printf("DEBUG: Task error: %v\n", g.lastError)

	// Close write ends to signal EOF
	_ = wOut.Close()
	_ = wErr.Close()

	// Collect output
	g.output = <-outChan
	g.errorOutput = <-errChan

	// If task had an error but no error output was captured, use the error message
	if g.lastError != nil && g.errorOutput == "" {
		g.errorOutput = g.lastError.Error()
	}

	// Debug output to see what we're getting
	if g.output != "" {
		fmt.Printf("DEBUG: Grift task output: %q\n", g.output)
	}
	if g.errorOutput != "" {
		fmt.Printf("DEBUG: Grift task error: %q\n", g.errorOutput)
	}

	// Sync output with shared context if available
	if g.shared != nil {
		g.shared.Output = g.output
		g.shared.ErrorOutput = g.errorOutput
		if g.lastError != nil {
			g.shared.ExitCode = 1
		} else {
			g.shared.ExitCode = 0
		}
	}

	return nil
}

func (g *GriftTestSuite) iRunGriftTaskWithArgs(taskName string, args string) error {
	// Parse arguments
	argList := strings.Fields(args)

	// Capture stdout and stderr (same as above)
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	outChan := make(chan string)
	errChan := make(chan string)

	go func() {
		buf := make([]byte, 1024)
		var output strings.Builder
		for {
			n, err := rOut.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		outChan <- output.String()
	}()

	go func() {
		buf := make([]byte, 1024)
		var output strings.Builder
		for {
			n, err := rErr.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		errChan <- output.String()
	}()

	// Create grift context with arguments
	ctx := grift.NewContext(taskName)
	ctx.Args = argList
	g.lastError = grift.Run(taskName, ctx)

	_ = wOut.Close()
	_ = wErr.Close()

	// Collect output
	g.output = <-outChan
	g.errorOutput = <-errChan

	// Sync output with shared context if available
	if g.shared != nil {
		g.shared.Output = g.output
		g.shared.ErrorOutput = g.errorOutput
		if g.lastError != nil {
			g.shared.ExitCode = 1
		} else {
			g.shared.ExitCode = 0
		}
	}

	return nil
}

func (g *GriftTestSuite) theTaskShouldSucceed() error {
	if g.lastError != nil {
		return fmt.Errorf("expected task to succeed but got error: %v\nOutput: %s\nError: %s",
			g.lastError, g.output, g.errorOutput)
	}
	return nil
}

func (g *GriftTestSuite) theTaskShouldFail() error {
	if g.lastError == nil && g.errorOutput == "" {
		return fmt.Errorf("expected task to fail but it succeeded\nOutput: %s", g.output)
	}
	return nil
}

func (g *GriftTestSuite) theMigrationsTableShouldExist() error {
	// Re-open the database to check the table
	// The migration task has its own connection, so we need a fresh one
	db, err := sql.Open("sqlite3", g.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database for verification: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			// Log error but don't fail the test
			fmt.Printf("Failed to close database: %v\n", err)
		}
	}()

	var tableName string
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name='buffkit_migrations'`
	err = db.QueryRow(query).Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("migrations table does not exist")
		}
		return fmt.Errorf("failed to query for migrations table: %w", err)
	}

	if tableName != "buffkit_migrations" {
		return fmt.Errorf("expected table 'buffkit_migrations', got '%s'", tableName)
	}

	return nil
}

// InitializeGriftScenario registers grift task testing steps
func InitializeGriftScenario(ctx *godog.ScenarioContext, bridge ...*SharedBridge) {
	suite := NewGriftTestSuite()

	// If bridge is provided, use its shared context for environment variables
	if len(bridge) > 0 && bridge[0] != nil {
		suite.setSharedContext(bridge[0].shared)
	}

	// Before each scenario
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		suite.Reset()
		// Re-set shared context after reset
		if len(bridge) > 0 && bridge[0] != nil {
			suite.setSharedContext(bridge[0].shared)
		}
		return ctx, nil
	})

	// After each scenario
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		suite.Reset()
		return ctx, nil
	})

	// Given steps
	ctx.Step(`^I have a clean test database$`, suite.iHaveACleanTestDatabase)
	// When steps
	ctx.Step(`^I run grift task "([^"]*)"$`, suite.iRunGriftTask)
	ctx.Step(`^I run grift task "([^"]*)" with args "([^"]*)"$`, suite.iRunGriftTaskWithArgs)

	// Then steps
	ctx.Step(`^the task should succeed$`, suite.theTaskShouldSucceed)
	ctx.Step(`^the task should fail$`, suite.theTaskShouldFail)
	ctx.Step(`^the migrations table should exist$`, suite.theMigrationsTableShouldExist)
}

// TestGriftTasks runs grift task tests
func TestGriftTasks(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			// Create a shared bridge for environment variables
			bridge := NewSharedBridge()
			bridge.RegisterBridgedSteps(ctx)

			// Initialize grift scenario with bridge
			InitializeGriftScenario(ctx, bridge)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"grift_tasks.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run grift task tests")
	}
}
