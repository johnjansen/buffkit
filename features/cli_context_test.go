package features

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"

	// Import sqlite3 driver for testing
	_ "github.com/mattn/go-sqlite3"
)

// CLIContext provides a test context for CLI command testing
type CLIContext struct {
	// Command execution state
	lastCmd      *exec.Cmd
	stdout       *bytes.Buffer
	stderr       *bytes.Buffer
	exitCode     int
	executionErr error

	// Environment
	tempDir    string
	env        map[string]string
	workingDir string

	// Database for testing migrations
	testDB       *sql.DB
	dbDialect    string
	dbConnString string

	// Cleanup functions to run after test
	cleanupFuncs []func()
}

// NewCLIContext creates a new CLI testing context
func NewCLIContext() *CLIContext {
	return &CLIContext{
		env:          make(map[string]string),
		cleanupFuncs: make([]func(), 0),
	}
}

// Cleanup runs all cleanup functions
func (c *CLIContext) Cleanup() {
	for i := len(c.cleanupFuncs) - 1; i >= 0; i-- {
		if fn := c.cleanupFuncs[i]; fn != nil {
			fn()
		}
	}
}

// Step definitions

// iHaveACleanDatabase sets up a fresh test database
func (c *CLIContext) iHaveACleanDatabase() error {
	// Create a temporary SQLite database for testing
	tempDir, err := os.MkdirTemp("", "buffkit-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	c.tempDir = tempDir
	c.cleanupFuncs = append(c.cleanupFuncs, func() {
		os.RemoveAll(tempDir)
	})

	dbPath := filepath.Join(tempDir, "test.db")
	c.dbConnString = dbPath
	c.dbDialect = "sqlite3"

	// Set environment variable for the database
	c.env["DATABASE_URL"] = fmt.Sprintf("sqlite3://%s", dbPath)

	// Open connection to verify it works
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open test database: %w", err)
	}
	c.testDB = db
	c.cleanupFuncs = append(c.cleanupFuncs, func() {
		db.Close()
	})

	return nil
}

// iSetEnvironmentVariable sets an environment variable for the command
func (c *CLIContext) iSetEnvironmentVariable(key, value string) error {
	c.env[key] = value
	return nil
}

// iRunCommand executes a CLI command
func (c *CLIContext) iRunCommand(command string) error {
	return c.iRunCommandWithTimeout(command, 30)
}

// iRunCommandWithTimeout executes a CLI command with a timeout in seconds
func (c *CLIContext) iRunCommandWithTimeout(command string, timeoutSeconds int) error {
	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create command with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	c.lastCmd = exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set up output capture
	c.stdout = &bytes.Buffer{}
	c.stderr = &bytes.Buffer{}
	c.lastCmd.Stdout = c.stdout
	c.lastCmd.Stderr = c.stderr

	// Set environment
	c.lastCmd.Env = os.Environ()
	for k, v := range c.env {
		c.lastCmd.Env = append(c.lastCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set working directory if specified
	if c.workingDir != "" {
		c.lastCmd.Dir = c.workingDir
	}

	// Execute command
	c.executionErr = c.lastCmd.Run()

	// Get exit code
	if c.executionErr != nil {
		if exitErr, ok := c.executionErr.(*exec.ExitError); ok {
			c.exitCode = exitErr.ExitCode()
		} else {
			c.exitCode = -1
		}
	} else {
		c.exitCode = 0
	}

	return nil
}

// theOutputShouldContain checks if stdout contains expected text
func (c *CLIContext) theOutputShouldContain(expected string) error {
	output := c.stdout.String()
	if !strings.Contains(output, expected) {
		return fmt.Errorf("output does not contain %q\nGot: %s", expected, output)
	}
	return nil
}

// theOutputShouldNotContain checks that stdout doesn't contain text
func (c *CLIContext) theOutputShouldNotContain(unexpected string) error {
	output := c.stdout.String()
	if strings.Contains(output, unexpected) {
		return fmt.Errorf("output contains unexpected %q\nGot: %s", unexpected, output)
	}
	return nil
}

// theErrorOutputShouldContain checks if stderr contains expected text
func (c *CLIContext) theErrorOutputShouldContain(expected string) error {
	output := c.stderr.String()
	if !strings.Contains(output, expected) {
		return fmt.Errorf("error output does not contain %q\nGot: %s", expected, output)
	}
	return nil
}

// theExitCodeShouldBe checks the command exit code
func (c *CLIContext) theExitCodeShouldBe(expected int) error {
	if c.exitCode != expected {
		return fmt.Errorf("exit code was %d, expected %d\nStdout: %s\nStderr: %s",
			c.exitCode, expected, c.stdout.String(), c.stderr.String())
	}
	return nil
}

// theMigrationsTableShouldExist checks if migrations table was created
func (c *CLIContext) theMigrationsTableShouldExist() error {
	if c.testDB == nil {
		return fmt.Errorf("no test database connection")
	}

	var tableName string
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name='buffkit_migrations'`
	err := c.testDB.QueryRow(query).Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("migrations table does not exist")
		}
		return fmt.Errorf("failed to check migrations table: %w", err)
	}

	if tableName != "buffkit_migrations" {
		return fmt.Errorf("expected table 'buffkit_migrations', got '%s'", tableName)
	}

	return nil
}

// iHaveAWorkingDirectory sets the working directory for commands
func (c *CLIContext) iHaveAWorkingDirectory(dir string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	c.workingDir = dir
	return nil
}

// aFileShouldExist checks if a file exists
func (c *CLIContext) aFileShouldExist(path string) error {
	fullPath := path
	if c.workingDir != "" && !filepath.IsAbs(path) {
		fullPath = filepath.Join(c.workingDir, path)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", fullPath)
	}
	return nil
}

// theFileShouldContain checks if a file contains expected text
func (c *CLIContext) theFileShouldContain(path, expected string) error {
	fullPath := path
	if c.workingDir != "" && !filepath.IsAbs(path) {
		fullPath = filepath.Join(c.workingDir, path)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	if !strings.Contains(string(content), expected) {
		return fmt.Errorf("file %s does not contain %q", fullPath, expected)
	}
	return nil
}

// InitializeCLIScenario registers CLI testing steps with godog
func InitializeCLIScenario(ctx *godog.ScenarioContext) {
	cliCtx := NewCLIContext()

	// Setup and environment steps
	ctx.Step(`^I have a clean database$`, cliCtx.iHaveACleanDatabase)
	ctx.Step(`^I set environment variable "([^"]*)" to "([^"]*)"$`, cliCtx.iSetEnvironmentVariable)
	ctx.Step(`^I set environment variable '([^']*)' to '([^']*)'$`, cliCtx.iSetEnvironmentVariable)
	ctx.Step(`^I have a working directory "([^"]*)"$`, cliCtx.iHaveAWorkingDirectory)

	// Command execution steps
	ctx.Step(`^I run "([^"]*)"$`, cliCtx.iRunCommand)
	ctx.Step(`^I run "([^"]*)" with timeout (\d+) seconds$`, cliCtx.iRunCommandWithTimeout)

	// Output assertion steps - both quoted and unquoted versions
	ctx.Step(`^the output should contain "([^"]*)"$`, cliCtx.theOutputShouldContain)
	ctx.Step(`^the output should contain '([^']*)'$`, cliCtx.theOutputShouldContain)
	ctx.Step(`^the output should not contain "([^"]*)"$`, cliCtx.theOutputShouldNotContain)
	ctx.Step(`^the output should not contain '([^']*)'$`, cliCtx.theOutputShouldNotContain)
	ctx.Step(`^the error output should contain "([^"]*)"$`, cliCtx.theErrorOutputShouldContain)
	ctx.Step(`^the error output should contain '([^']*)'$`, cliCtx.theErrorOutputShouldContain)
	ctx.Step(`^the exit code should be (\d+)$`, cliCtx.theExitCodeShouldBe)

	// Database assertion steps
	ctx.Step(`^the migrations table should exist$`, cliCtx.theMigrationsTableShouldExist)

	// File system assertion steps
	ctx.Step(`^a file "([^"]*)" should exist$`, cliCtx.aFileShouldExist)
	ctx.Step(`^the file "([^"]*)" should contain "([^"]*)"$`, cliCtx.theFileShouldContain)

	// Cleanup after each scenario
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		cliCtx.Cleanup()
		return ctx, nil
	})
}
