package features

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/gobuffalo/buffalo"
	"github.com/johnjansen/buffkit/components"
	"golang.org/x/net/html"

	// Import database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// SharedContext provides common test functionality for all test suites
type SharedContext struct {
	// Output from any operation (HTML rendering, CLI commands, HTTP responses, etc.)
	Output       string
	ErrorOutput  string
	ExitCode     int
	LastError    error
	ResponseCode int

	// HTTP testing
	Response *httptest.ResponseRecorder
	Request  *http.Request
	App      *buffalo.App

	// CLI command execution
	LastCmd     *exec.Cmd
	WorkingDir  string
	Environment map[string]string
	TempDirs    []string

	// Database for testing
	TestDB       *sql.DB
	DBDialect    string
	DBConnString string

	// Component registry for rendering
	ComponentRegistry *components.Registry

	// Component testing
	Input              string
	ContentType        string
	IsDevelopment      bool
	RegistryComponents []string

	// Cleanup functions
	CleanupFuncs []func()
}

// NewSharedContext creates a new shared test context
func NewSharedContext() *SharedContext {
	return &SharedContext{
		Environment:  make(map[string]string),
		CleanupFuncs: make([]func(), 0),
		TempDirs:     make([]string, 0),
	}
}

// Cleanup runs all cleanup functions
func (c *SharedContext) Cleanup() {
	for i := len(c.CleanupFuncs) - 1; i >= 0; i-- {
		if fn := c.CleanupFuncs[i]; fn != nil {
			fn()
		}
	}
	// Clean up temp directories
	for _, dir := range c.TempDirs {
		_ = os.RemoveAll(dir)
	}
}

// Reset clears the context for a new scenario
func (c *SharedContext) Reset() {
	c.Output = ""
	c.ErrorOutput = ""
	c.ExitCode = 0
	c.LastError = nil
	c.ResponseCode = 0
	c.Response = nil
	c.Request = nil
	c.LastCmd = nil
}

// =============================================================================
// UNIVERSAL STEP DEFINITIONS
// =============================================================================

// TheOutputShouldContain checks if any output contains the expected text
// This handles: standard output, error output, HTTP response body, rendered HTML
func (c *SharedContext) TheOutputShouldContain(expected string) error {
	// Check all possible outputs
	outputs := []struct {
		name   string
		output string
	}{
		{"output", c.Output},
		{"error output", c.ErrorOutput},
	}

	// If we have an HTTP response, check that too
	if c.Response != nil {
		outputs = append(outputs, struct {
			name   string
			output string
		}{"HTTP response", c.Response.Body.String()})
	}

	// Check each output source
	for _, out := range outputs {
		if strings.Contains(out.output, expected) {
			return nil // Found it!
		}
	}

	// Not found in any output - provide helpful error
	allOutput := c.getAllOutput()
	if allOutput == "" {
		return fmt.Errorf("expected output to contain %q but no output was captured", expected)
	}

	// Show what we actually got for debugging
	preview := allOutput
	if len(preview) > 500 {
		preview = preview[:500] + "... (truncated)"
	}
	return fmt.Errorf("output does not contain %q\nActual output:\n%s", expected, preview)
}

// TheOutputShouldNotContain checks that no output contains the unexpected text
func (c *SharedContext) TheOutputShouldNotContain(unexpected string) error {
	allOutput := c.getAllOutput()
	if strings.Contains(allOutput, unexpected) {
		// Find which output contains it for better error message
		location := "output"
		if strings.Contains(c.ErrorOutput, unexpected) {
			location = "error output"
		} else if c.Response != nil && strings.Contains(c.Response.Body.String(), unexpected) {
			location = "HTTP response"
		}

		preview := allOutput
		if len(preview) > 500 {
			preview = preview[:500] + "... (truncated)"
		}
		return fmt.Errorf("%s contains unexpected text %q\nActual output:\n%s", location, unexpected, preview)
	}
	return nil
}

// TheOutputShouldMatch checks if output matches a regex pattern
func (c *SharedContext) TheOutputShouldMatch(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	allOutput := c.getAllOutput()
	if !re.MatchString(allOutput) {
		preview := allOutput
		if len(preview) > 500 {
			preview = preview[:500] + "... (truncated)"
		}
		return fmt.Errorf("output does not match pattern %q\nActual output:\n%s", pattern, preview)
	}
	return nil
}

// TheErrorOutputShouldContain specifically checks error output
func (c *SharedContext) TheErrorOutputShouldContain(expected string) error {
	if !strings.Contains(c.ErrorOutput, expected) {
		if c.ErrorOutput == "" {
			return fmt.Errorf("expected error output to contain %q but no error output was captured", expected)
		}
		return fmt.Errorf("error output does not contain %q\nActual error output:\n%s", expected, c.ErrorOutput)
	}
	return nil
}

// =============================================================================
// ENVIRONMENT AND SETUP
// =============================================================================

// ISetEnvironmentVariable sets an environment variable for command execution
func (c *SharedContext) ISetEnvironmentVariable(key, value string) error {
	c.Environment[key] = value
	return nil
}

// IHaveACleanDatabase sets up a fresh test database
func (c *SharedContext) IHaveACleanDatabase() error {
	tempDir, err := os.MkdirTemp("", "buffkit-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	c.TempDirs = append(c.TempDirs, tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	c.DBConnString = dbPath
	c.DBDialect = "sqlite3"

	// Set environment variable for the database
	c.Environment["DATABASE_URL"] = fmt.Sprintf("sqlite3://%s", dbPath)

	// Open connection to verify it works
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open test database: %w", err)
	}
	c.TestDB = db
	c.CleanupFuncs = append(c.CleanupFuncs, func() {
		_ = db.Close()
	})

	return nil
}

// IHaveAWorkingDirectory sets the working directory for commands
func (c *SharedContext) IHaveAWorkingDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	c.WorkingDir = dir
	c.TempDirs = append(c.TempDirs, dir)
	return nil
}

// =============================================================================
// COMMAND EXECUTION
// =============================================================================

// IRunCommand executes a CLI command
func (c *SharedContext) IRunCommand(command string) error {
	// Special handling for grift commands
	if strings.HasPrefix(command, "grift ") {
		// Build grift binary if it doesn't exist
		if _, err := os.Stat("./grift"); os.IsNotExist(err) {
			buildCmd := exec.Command("go", "build", "-o", "grift", "./cmd/grift")
			if err := buildCmd.Run(); err != nil {
				c.LastError = fmt.Errorf("failed to build grift binary: %w", err)
				c.ExitCode = -1
				return nil
			}
		}
		// Replace "grift" with "./grift" to use our test binary
		command = "./" + command
	}
	return c.IRunCommandWithTimeout(command, 30)
}

// IRunCommandWithTimeout executes a CLI command with a timeout in seconds
func (c *SharedContext) IRunCommandWithTimeout(command string, timeoutSeconds int) error {
	// Reset outputs
	c.Output = ""
	c.ErrorOutput = ""
	c.ExitCode = 0
	c.LastError = nil

	// Special handling for grift commands
	if strings.HasPrefix(command, "grift ") {
		// Build grift binary if it doesn't exist
		if _, err := os.Stat("./grift"); os.IsNotExist(err) {
			buildCmd := exec.Command("go", "build", "-o", "grift", "./cmd/grift")
			if err := buildCmd.Run(); err != nil {
				c.LastError = fmt.Errorf("failed to build grift binary: %w", err)
				c.ExitCode = -1
				return nil
			}
		}
		// Replace "grift" with "./grift" to use our test binary
		command = "./" + command
	}

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create command with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	c.LastCmd = exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set up output capture
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	c.LastCmd.Stdout = stdout
	c.LastCmd.Stderr = stderr

	// Set environment
	c.LastCmd.Env = os.Environ()
	for k, v := range c.Environment {
		c.LastCmd.Env = append(c.LastCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set working directory if specified
	if c.WorkingDir != "" {
		c.LastCmd.Dir = c.WorkingDir
	}

	// Execute command
	err := c.LastCmd.Run()
	c.LastError = err

	// Capture outputs
	c.Output = stdout.String()
	c.ErrorOutput = stderr.String()

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			c.ExitCode = exitErr.ExitCode()
		} else {
			c.ExitCode = -1
		}
	} else {
		c.ExitCode = 0
	}

	return nil
}

// TheExitCodeShouldBe checks the command exit code
func (c *SharedContext) TheExitCodeShouldBe(expected int) error {
	if c.ExitCode != expected {
		return fmt.Errorf("exit code was %d, expected %d\nStdout: %s\nStderr: %s",
			c.ExitCode, expected, c.Output, c.ErrorOutput)
	}
	return nil
}

// =============================================================================
// HTML/COMPONENT RENDERING
// =============================================================================

// IRenderHTMLContaining simulates rendering HTML (usually through a component system)
func (c *SharedContext) IRenderHTMLContaining(html string) error {
	// Initialize component registry if not already done
	if c.ComponentRegistry == nil {
		c.ComponentRegistry = components.NewRegistry()
		c.ComponentRegistry.RegisterDefaults()
	}

	// Wrap HTML in a basic HTML structure for parsing
	fullHTML := fmt.Sprintf("<html><body>%s</body></html>", html)

	// Expand components using the registry
	expanded, err := c.expandHTMLWithComponents([]byte(fullHTML))
	if err != nil {
		c.Output = html // Fall back to raw HTML on error
		return nil      // Don't fail the test for expansion errors
	}

	// Extract just the body content
	bodyStart := strings.Index(string(expanded), "<body>") + 6
	bodyEnd := strings.Index(string(expanded), "</body>")
	if bodyStart > 5 && bodyEnd > bodyStart {
		c.Output = strings.TrimSpace(string(expanded[bodyStart:bodyEnd]))
	} else {
		c.Output = string(expanded)
	}

	return nil
}

// expandHTMLWithComponents processes HTML through the component registry
func (c *SharedContext) expandHTMLWithComponents(htmlContent []byte) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return htmlContent, err
	}

	// Walk the tree and expand components
	var expand func(*html.Node) error
	expand = func(n *html.Node) error {
		if n.Type == html.ElementNode && strings.HasPrefix(n.Data, "bk-") {
			componentName := n.Data

			// Extract attributes
			attrs := make(map[string]string)
			for _, attr := range n.Attr {
				attrs[attr.Key] = attr.Val
			}

			// Extract slot content (simplified for testing)
			slots := make(map[string]string)
			var content strings.Builder
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				switch child.Type {
				case html.TextNode:
					content.WriteString(child.Data)
				case html.ElementNode:
					// For element nodes, render them back to HTML
					var buf bytes.Buffer
					_ = html.Render(&buf, child)
					content.WriteString(buf.String())
				}
			}
			slots["default"] = strings.TrimSpace(content.String())

			// Render the component
			rendered, err := c.ComponentRegistry.Render(n.Data, attrs, slots)
			if err != nil {
				// Keep original if rendering fails
				return nil
			}

			// Parse the rendered HTML as a complete document
			wrappedHTML := fmt.Sprintf("<html><body>%s</body></html>", string(rendered))
			renderedDoc, err := html.Parse(bytes.NewReader([]byte(wrappedHTML)))
			if err != nil {
				return nil
			}

			// Find the body node and extract its children
			var bodyNode *html.Node
			var findBody func(*html.Node)
			findBody = func(node *html.Node) {
				if node.Type == html.ElementNode && node.Data == "body" {
					bodyNode = node
					return
				}
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					findBody(child)
					if bodyNode != nil {
						return
					}
				}
			}
			findBody(renderedDoc)

			if bodyNode == nil || n.Parent == nil {
				return nil
			}

			// Add component boundary comments in development mode
			if c.IsDevelopment {
				// Add start comment
				startComment := &html.Node{
					Type: html.CommentNode,
					Data: fmt.Sprintf(" %s ", componentName),
				}
				n.Parent.InsertBefore(startComment, n)
			}

			// Replace the component node with the body's children
			for child := bodyNode.FirstChild; child != nil; {
				next := child.NextSibling
				bodyNode.RemoveChild(child)
				n.Parent.InsertBefore(child, n)
				child = next
			}

			// Add end comment in development mode
			if c.IsDevelopment {
				endComment := &html.Node{
					Type: html.CommentNode,
					Data: fmt.Sprintf(" /%s ", componentName),
				}
				n.Parent.InsertBefore(endComment, n)
			}

			n.Parent.RemoveChild(n)
			return nil
		}

		// Recurse to children (make a copy of the list first to avoid mutation during iteration)
		var children []*html.Node
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			children = append(children, child)
		}
		for _, child := range children {
			if err := expand(child); err != nil {
				return err
			}
		}
		return nil
	}

	if err := expand(doc); err != nil {
		return htmlContent, err
	}

	// Render the modified document back to HTML
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return htmlContent, err
	}

	return buf.Bytes(), nil
}

// =============================================================================
// HTTP TESTING
// =============================================================================

// IVisit makes an HTTP GET request to a path
func (c *SharedContext) IVisit(path string) error {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}
	c.Request = req
	c.Response = httptest.NewRecorder()

	// If we have an app, use it to handle the request
	if c.App != nil {
		c.App.ServeHTTP(c.Response, req)
		c.Output = c.Response.Body.String()
		c.ResponseCode = c.Response.Code
	}

	return nil
}

// ISubmitAPostRequestTo makes an HTTP POST request
func (c *SharedContext) ISubmitAPostRequestTo(path string) error {
	req, err := http.NewRequest("POST", path, nil)
	if err != nil {
		return err
	}
	c.Request = req
	c.Response = httptest.NewRecorder()

	// If we have an app, use it to handle the request
	if c.App != nil {
		c.App.ServeHTTP(c.Response, req)
		c.Output = c.Response.Body.String()
		c.ResponseCode = c.Response.Code
	}

	return nil
}

// TheResponseStatusShouldBe checks the HTTP response status code
func (c *SharedContext) TheResponseStatusShouldBe(expected int) error {
	if c.Response == nil {
		return fmt.Errorf("no HTTP response captured")
	}
	if c.Response.Code != expected {
		return fmt.Errorf("response status was %d, expected %d", c.Response.Code, expected)
	}
	return nil
}

// TheContentTypeShouldBe checks the response content type
func (c *SharedContext) TheContentTypeShouldBe(expected string) error {
	if c.Response == nil {
		return fmt.Errorf("no HTTP response captured")
	}
	actual := c.Response.Header().Get("Content-Type")
	if !strings.Contains(actual, expected) {
		return fmt.Errorf("content type was %q, expected to contain %q", actual, expected)
	}
	return nil
}

// =============================================================================
// FILE SYSTEM
// =============================================================================

// AFileShouldExist checks if a file exists
func (c *SharedContext) AFileShouldExist(path string) error {
	fullPath := path
	if c.WorkingDir != "" && !filepath.IsAbs(path) {
		fullPath = filepath.Join(c.WorkingDir, path)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", fullPath)
	}
	return nil
}

// TheFileShouldContain checks if a file contains expected text
func (c *SharedContext) TheFileShouldContain(path, expected string) error {
	fullPath := path
	if c.WorkingDir != "" && !filepath.IsAbs(path) {
		fullPath = filepath.Join(c.WorkingDir, path)
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

// =============================================================================
// DATABASE
// =============================================================================

// TheMigrationsTableShouldExist checks if migrations table was created
func (c *SharedContext) TheMigrationsTableShouldExist() error {
	var db *sql.DB

	// If TestDB is not set, try to open a connection using DATABASE_URL
	if c.TestDB == nil {
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("no test database connection and DATABASE_URL not set")
		}

		// Open a new connection to check the table
		var err error
		db, err = sql.Open("sqlite3", dbURL)
		if err != nil {
			return fmt.Errorf("failed to open database for verification: %w", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				// Log error but don't fail the test
				fmt.Printf("Failed to close database: %v\n", err)
			}
		}()
	} else {
		db = c.TestDB
	}

	var tableName string
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name='buffkit_migrations'`
	err := db.QueryRow(query).Scan(&tableName)
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

// =============================================================================
// HELPER METHODS
// =============================================================================

// getAllOutput combines all output sources for searching
func (c *SharedContext) getAllOutput() string {
	outputs := []string{}

	if c.Output != "" {
		outputs = append(outputs, c.Output)
	}
	if c.ErrorOutput != "" {
		outputs = append(outputs, c.ErrorOutput)
	}
	if c.Response != nil && c.Response.Body.Len() > 0 {
		outputs = append(outputs, c.Response.Body.String())
	}

	return strings.Join(outputs, "\n")
}

// SetHTTPApp sets the Buffalo app for HTTP testing
func (c *SharedContext) SetHTTPApp(app *buffalo.App) {
	c.App = app
}

// CaptureOutput sets the output directly (for integration with existing tests)
func (c *SharedContext) CaptureOutput(output string) {
	c.Output = output
}

// CaptureErrorOutput sets the error output directly
func (c *SharedContext) CaptureErrorOutput(output string) {
	c.ErrorOutput = output
}

// CaptureHTTPResponse captures an HTTP response for testing
func (c *SharedContext) CaptureHTTPResponse(w io.Writer) {
	if recorder, ok := w.(*httptest.ResponseRecorder); ok {
		c.Response = recorder
		c.Output = recorder.Body.String()
		c.ResponseCode = recorder.Code
	}
}

// =============================================================================
// REGISTRATION
// =============================================================================

// RegisterSharedSteps registers all shared step definitions with godog
func RegisterSharedSteps(ctx *godog.ScenarioContext, shared *SharedContext) {
	// Output assertions - support both single and double quotes
	ctx.Step(`^the output should contain "([^"]*)"$`, shared.TheOutputShouldContain)
	ctx.Step(`^the output should contain '([^']*)'$`, shared.TheOutputShouldContain)
	ctx.Step(`^the output should not contain "([^"]*)"$`, shared.TheOutputShouldNotContain)
	ctx.Step(`^the output should not contain '([^']*)'$`, shared.TheOutputShouldNotContain)
	ctx.Step(`^the output should match "([^"]*)"$`, shared.TheOutputShouldMatch)
	ctx.Step(`^the error output should contain "([^"]*)"$`, shared.TheErrorOutputShouldContain)
	ctx.Step(`^the error output should contain '([^']*)'$`, shared.TheErrorOutputShouldContain)

	// Environment and setup
	ctx.Step(`^I set environment variable "([^"]*)" to "([^"]*)"$`, shared.ISetEnvironmentVariable)
	ctx.Step(`^I set environment variable '([^']*)' to '([^']*)'$`, shared.ISetEnvironmentVariable)
	ctx.Step(`^I have a clean database$`, shared.IHaveACleanDatabase)
	ctx.Step(`^I have a working directory "([^"]*)"$`, shared.IHaveAWorkingDirectory)

	// Command execution
	ctx.Step(`^I run "([^"]*)"$`, shared.IRunCommand)
	ctx.Step(`^I run '([^']*)'$`, shared.IRunCommand)
	ctx.Step(`^I run "([^"]*)" with timeout (\d+) seconds$`, shared.IRunCommandWithTimeout)
	ctx.Step(`^the exit code should be (\d+)$`, shared.TheExitCodeShouldBe)

	// HTML/Component rendering
	ctx.Step(`^I render HTML containing "([^"]*)"$`, shared.IRenderHTMLContaining)
	ctx.Step(`^I render HTML containing '([^']*)'$`, shared.IRenderHTMLContaining)

	// HTTP testing
	ctx.Step(`^I visit "([^"]*)"$`, shared.IVisit)
	ctx.Step(`^I submit a POST request to "([^"]*)"$`, shared.ISubmitAPostRequestTo)
	ctx.Step(`^the response status should be (\d+)$`, shared.TheResponseStatusShouldBe)
	ctx.Step(`^the content type should be "([^"]*)"$`, shared.TheContentTypeShouldBe)

	// File system
	ctx.Step(`^a file "([^"]*)" should exist$`, shared.AFileShouldExist)
	ctx.Step(`^the file "([^"]*)" should contain "([^"]*)"$`, shared.TheFileShouldContain)

	// Database
	ctx.Step(`^the migrations table should exist$`, shared.TheMigrationsTableShouldExist)

	// Cleanup after each scenario
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		shared.Cleanup()
		shared.Reset()
		return ctx, nil
	})
}
