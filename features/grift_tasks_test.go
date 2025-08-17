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
		g.testDB.Close()
		g.testDB = nil
	}

	// Clean up temp directory
	if g.tempDir != "" {
		os.RemoveAll(g.tempDir)
		g.tempDir = ""
	}
}

// Step definitions

func (g *GriftTestSuite) iHaveACleanTestDatabase() error {
	// Create a temporary directory for test database
	tempDir, err := os.MkdirTemp("", "buffkit-grift-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	g.tempDir = tempDir
	g.cleanupFuncs = append(g.cleanupFuncs, func() {
		os.RemoveAll(tempDir)
	})

	// Create test database path
	g.dbPath = fmt.Sprintf("%s/test.db", tempDir)

	// Set environment variable for database
	os.Setenv("DATABASE_URL", g.dbPath)
	g.cleanupFuncs = append(g.cleanupFuncs, func() {
		os.Unsetenv("DATABASE_URL")
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

	// Create grift context and run task
	ctx := grift.NewContext(taskName)
	g.lastError = grift.Run(taskName, ctx)

	// Close write ends to signal EOF
	wOut.Close()
	wErr.Close()

	// Collect output
	g.output = <-outChan
	g.errorOutput = <-errChan

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

	wOut.Close()
	wErr.Close()

	g.output = <-outChan
	g.errorOutput = <-errChan

	return nil
}

func (g *GriftTestSuite) theOutputShouldContain(expected string) error {
	if !strings.Contains(g.output, expected) {
		return fmt.Errorf("output does not contain %q\nGot: %s", expected, g.output)
	}
	return nil
}

func (g *GriftTestSuite) theErrorOutputShouldContain(expected string) error {
	if !strings.Contains(g.errorOutput, expected) {
		return fmt.Errorf("error output does not contain %q\nGot: %s", expected, g.errorOutput)
	}
	return nil
}

func (g *GriftTestSuite) theTaskShouldSucceed() error {
	if g.lastError != nil {
		return fmt.Errorf("task failed with error: %v\nOutput: %s\nError: %s",
			g.lastError, g.output, g.errorOutput)
	}
	return nil
}

func (g *GriftTestSuite) theTaskShouldFail() error {
	if g.lastError == nil {
		return fmt.Errorf("expected task to fail but it succeeded")
	}
	return nil
}

func (g *GriftTestSuite) theMigrationsTableShouldExist() error {
	if g.testDB == nil {
		return fmt.Errorf("no test database connection")
	}

	var tableName string
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name='buffkit_migrations'`
	err := g.testDB.QueryRow(query).Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("migrations table does not exist")
		}
		return fmt.Errorf("failed to check migrations table: %w", err)
	}

	return nil
}

func (g *GriftTestSuite) iSetEnvironmentVariable(key, value string) error {
	g.env[key] = value
	os.Setenv(key, value)
	g.cleanupFuncs = append(g.cleanupFuncs, func() {
		os.Unsetenv(key)
	})
	return nil
}

// InitializeGriftScenario registers grift task testing steps
func InitializeGriftScenario(ctx *godog.ScenarioContext) {
	suite := NewGriftTestSuite()

	// Before each scenario
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		suite.Reset()
		return ctx, nil
	})

	// After each scenario
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		suite.Reset()
		return ctx, nil
	})

	// Given steps
	ctx.Step(`^I have a clean test database$`, suite.iHaveACleanTestDatabase)
	ctx.Step(`^I set environment variable "([^"]*)" to "([^"]*)"$`, suite.iSetEnvironmentVariable)

	// When steps
	ctx.Step(`^I run grift task "([^"]*)"$`, suite.iRunGriftTask)
	ctx.Step(`^I run grift task "([^"]*)" with args "([^"]*)"$`, suite.iRunGriftTaskWithArgs)

	// Then steps
	ctx.Step(`^the output should contain "([^"]*)"$`, suite.theOutputShouldContain)
	ctx.Step(`^the error output should contain "([^"]*)"$`, suite.theErrorOutputShouldContain)
	ctx.Step(`^the task should succeed$`, suite.theTaskShouldSucceed)
	ctx.Step(`^the task should fail$`, suite.theTaskShouldFail)
	ctx.Step(`^the migrations table should exist$`, suite.theMigrationsTableShouldExist)
}

// TestGriftTasks runs grift task tests
func TestGriftTasks(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeGriftScenario,
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
