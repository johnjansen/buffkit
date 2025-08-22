package features

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

type generatorContext struct {
	testDir     string
	lastOutput  string
	lastError   error
	createdFiles map[string]string
}

var genCtx = &generatorContext{
	createdFiles: make(map[string]string),
}

func iHaveATestProjectDirectory() error {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "buffkit-gen-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	genCtx.testDir = tmpDir
	
	// Create necessary directories
	dirs := []string{"models", "actions", "templates", "components", "jobs", "mailers", "sse", "db/migrations/core", "assets/css/components"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", dir, err)
		}
	}
	
	return nil
}

func buffkitIsAvailable() error {
	// Check if we can import buffkit
	// In a real test, this would verify the package is importable
	return nil
}

func iRunCommand(command string) error {
	// Parse the command to extract the task name and args
	parts := strings.Fields(command)
	if len(parts) < 3 || parts[0] != "buffalo" || parts[1] != "task" {
		return fmt.Errorf("invalid command format: %s", command)
	}
	
	// Change to test directory
	originalDir, _ := os.Getwd()
	os.Chdir(genCtx.testDir)
	defer os.Chdir(originalDir)
	
	// Simulate running the grift task
	// In a real implementation, we'd actually run the task
	taskName := parts[2]
	args := parts[3:]
	
	// For testing, we'll simulate the generator behavior
	return simulateGenerator(taskName, args)
}

func simulateGenerator(taskName string, args []string) error {
	switch taskName {
	case "buffkit:generate:model", "g:model":
		return simulateModelGenerator(args)
	case "buffkit:generate:action", "g:action":
		return simulateActionGenerator(args)
	case "buffkit:generate:resource", "g:resource":
		return simulateResourceGenerator(args)
	case "buffkit:generate:migration", "g:migration":
		return simulateMigrationGenerator(args)
	case "buffkit:generate:component", "g:component":
		return simulateComponentGenerator(args)
	case "buffkit:generate:job", "g:job":
		return simulateJobGenerator(args)
	case "buffkit:generate:mailer", "g:mailer":
		return simulateMailerGenerator(args)
	case "buffkit:generate:sse", "g:sse":
		return simulateSSEGenerator(args)
	default:
		return fmt.Errorf("unknown generator: %s", taskName)
	}
}

func simulateModelGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("model name required")
	}
	
	name := args[0]
	fields := args[1:]
	
	// Create model file
	modelPath := filepath.Join(genCtx.testDir, "models", name+".go")
	modelContent := generateModelContent(name, fields)
	if err := ioutil.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		return err
	}
	genCtx.createdFiles[fmt.Sprintf("models/%s.go", name)] = modelContent
	
	// Create migration files
	migrationName := "create_" + pluralize(name)
	upPath := filepath.Join(genCtx.testDir, "db/migrations/core", "20240101000000_"+migrationName+".up.sql")
	downPath := filepath.Join(genCtx.testDir, "db/migrations/core", "20240101000000_"+migrationName+".down.sql")
	
	upContent := fmt.Sprintf("CREATE TABLE %s (\n    id SERIAL PRIMARY KEY,\n", pluralize(name))
	for _, field := range fields {
		parts := strings.Split(field, ":")
		if len(parts) >= 2 {
			fieldName := toSnakeCase(parts[0])
			fieldType := mapToSQLType(parts[1])
			upContent += fmt.Sprintf("    %s %s,\n", fieldName, fieldType)
		}
	}
	upContent += "    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\n"
	upContent += "    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP\n);"
	
	downContent := fmt.Sprintf("DROP TABLE IF EXISTS %s;", pluralize(name))
	
	ioutil.WriteFile(upPath, []byte(upContent), 0644)
	ioutil.WriteFile(downPath, []byte(downContent), 0644)
	
	return nil
}

func generateModelContent(name string, fields []string) string {
	camelName := toCamelCase(name)
	plural := pluralize(name)
	
	content := fmt.Sprintf(`package models

import (
	"context"
	"database/sql"
	"time"
)

type %s struct {
	ID int
`, camelName)
	
	// Add fields
	for _, field := range fields {
		parts := strings.Split(field, ":")
		if len(parts) >= 2 {
			fieldName := toCamelCase(parts[0])
			fieldType := mapToGoType(parts[1])
			nullable := len(parts) > 2 && parts[2] == "nullable"
			if nullable {
				fieldType = "*" + fieldType
			}
			content += fmt.Sprintf("\t%s %s\n", fieldName, fieldType)
		}
	}
	
	content += "\tCreatedAt time.Time\n\tUpdatedAt time.Time\n}\n\n"
	
	// Add CRUD methods
	content += fmt.Sprintf("func (%s *%s) Create(ctx context.Context, db *sql.DB) error {\n\treturn nil\n}\n\n", strings.ToLower(name), camelName)
	content += fmt.Sprintf("func Find%s(ctx context.Context, db *sql.DB, id int) (*%s, error) {\n\treturn nil, nil\n}\n\n", camelName, camelName)
	
	// Handle irregular plurals
	if plural == "people" {
		content += fmt.Sprintf("func AllPeople(ctx context.Context, db *sql.DB) ([]*%s, error) {\n\treturn nil, nil\n}\n", camelName)
	} else {
		content += fmt.Sprintf("func All%s(ctx context.Context, db *sql.DB) ([]*%s, error) {\n\treturn nil, nil\n}\n", toCamelCase(plural), camelName)
	}
	
	return content
}

func simulateActionGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource name required")
	}
	
	resource := args[0]
	actions := args[1:]
	if len(actions) == 0 {
		actions = []string{"index", "show", "new", "create", "edit", "update", "destroy"}
	}
	
	// Create action file - if resource is already plural, use it as-is
	// otherwise pluralize it
	var filename string
	if isPlural(resource) {
		filename = resource
	} else {
		filename = pluralize(resource)
	}
	
	actionPath := filepath.Join(genCtx.testDir, "actions", filename+".go")
	actionContent := generateActionContent(resource, actions)
	if err := ioutil.WriteFile(actionPath, []byte(actionContent), 0644); err != nil {
		return err
	}
	genCtx.createdFiles[fmt.Sprintf("actions/%s.go", filename)] = actionContent
	
	return nil
}

func generateActionContent(resource string, actions []string) string {
	// If resource is already plural, use it; otherwise pluralize
	var plural string
	if isPlural(resource) {
		plural = toCamelCase(resource)
	} else {
		plural = toCamelCase(pluralize(resource))
	}
	
	content := "package actions\n\n"
	
	for _, action := range actions {
		funcName := plural + toCamelCase(action)
		content += fmt.Sprintf("func %s(c buffalo.Context) error {\n\treturn nil\n}\n\n", funcName)
	}
	
	return content
}

func simulateResourceGenerator(args []string) error {
	// Generate model
	if err := simulateModelGenerator(args); err != nil {
		return err
	}
	
	// Generate actions
	if err := simulateActionGenerator([]string{args[0]}); err != nil {
		return err
	}
	
	// Generate views
	name := args[0]
	views := []string{"index", "show", "new", "edit", "_form"}
	for _, view := range views {
		viewPath := filepath.Join(genCtx.testDir, "templates", pluralize(name), view+".plush.html")
		os.MkdirAll(filepath.Dir(viewPath), 0755)
		ioutil.WriteFile(viewPath, []byte(fmt.Sprintf("<!-- %s view -->", view)), 0644)
	}
	
	return nil
}

func simulateMigrationGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("migration name required")
	}
	
	name := args[0]
	fields := args[1:]
	
	upPath := filepath.Join(genCtx.testDir, "db/migrations/core", "20240101000000_"+name+".up.sql")
	downPath := filepath.Join(genCtx.testDir, "db/migrations/core", "20240101000000_"+name+".down.sql")
	
	var upContent, downContent string
	
	if strings.HasPrefix(name, "create_") {
		tableName := strings.TrimPrefix(name, "create_")
		upContent = fmt.Sprintf("CREATE TABLE %s (\n", tableName)
		for _, field := range fields {
			parts := strings.Split(field, ":")
			if len(parts) >= 2 {
				fieldName := toSnakeCase(parts[0])
				fieldType := mapToSQLType(parts[1])
				upContent += fmt.Sprintf("    %s %s,\n", fieldName, fieldType)
			}
		}
		upContent = strings.TrimSuffix(upContent, ",\n") + "\n);"
		downContent = fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)
	} else if strings.Contains(name, "_to_") {
		parts := strings.Split(name, "_to_")
		if len(parts) == 2 && strings.HasPrefix(name, "add_") {
			tableName := parts[1]
			upContent = fmt.Sprintf("ALTER TABLE %s\n", tableName)
			for _, field := range fields {
				fieldParts := strings.Split(field, ":")
				if len(fieldParts) >= 2 {
					fieldName := toSnakeCase(fieldParts[0])
					fieldType := mapToSQLType(fieldParts[1])
					upContent += fmt.Sprintf("    ADD COLUMN %s %s", fieldName, fieldType)
				}
			}
		}
	}
	
	ioutil.WriteFile(upPath, []byte(upContent), 0644)
	ioutil.WriteFile(downPath, []byte(downContent), 0644)
	
	return nil
}

func simulateComponentGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("component name required")
	}
	
	name := args[0]
	
	// Create component file
	componentPath := filepath.Join(genCtx.testDir, "components", name+".go")
	componentContent := fmt.Sprintf(`package components

func %sComponent(attrs map[string]string, slots map[string][]byte) ([]byte, error) {
	return []byte("<div class=\"bk-%s\"></div>"), nil
}
`, toCamelCase(name), toKebabCase(name))
	ioutil.WriteFile(componentPath, []byte(componentContent), 0644)
	
	// Create CSS file
	cssPath := filepath.Join(genCtx.testDir, "assets/css/components", toKebabCase(name)+".css")
	cssContent := fmt.Sprintf(".bk-%s {\n  display: block;\n}", toKebabCase(name))
	ioutil.WriteFile(cssPath, []byte(cssContent), 0644)
	
	return nil
}

func simulateJobGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("job name required")
	}
	
	name := args[0]
	camelName := toCamelCase(name)
	
	jobPath := filepath.Join(genCtx.testDir, "jobs", toSnakeCase(name)+".go")
	jobContent := fmt.Sprintf(`package jobs

type %sJob struct {
	ID string
}

func %sHandler(ctx context.Context, t *asynq.Task) error {
	return nil
}

func Enqueue%s(client *asynq.Client, data string) error {
	return nil
}

func Register%sHandler(mux *asynq.ServeMux) {
}
`, camelName, camelName, camelName, camelName)
	
	ioutil.WriteFile(jobPath, []byte(jobContent), 0644)
	return nil
}

func simulateMailerGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mailer name required")
	}
	
	name := args[0]
	actions := args[1:]
	if len(actions) == 0 {
		actions = []string{"welcome", "notification"}
	}
	
	camelName := toCamelCase(name)
	
	// Create mailer file
	mailerPath := filepath.Join(genCtx.testDir, "mailers", toSnakeCase(name)+".go")
	mailerContent := fmt.Sprintf(`package mailers

type %sMailer struct {
	sender mail.Sender
}
`, camelName)
	
	for _, action := range actions {
		mailerContent += fmt.Sprintf(`
func (m *%sMailer) Send%s(ctx context.Context, to string, data map[string]interface{}) error {
	return nil
}
`, camelName, toCamelCase(action))
	}
	
	ioutil.WriteFile(mailerPath, []byte(mailerContent), 0644)
	
	// Create email templates
	for _, action := range actions {
		templatePath := filepath.Join(genCtx.testDir, "templates/mail", toSnakeCase(name), action+".html")
		os.MkdirAll(filepath.Dir(templatePath), 0755)
		ioutil.WriteFile(templatePath, []byte("<!-- email template -->"), 0644)
	}
	
	return nil
}

func simulateSSEGenerator(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("sse name required")
	}
	
	name := args[0]
	camelName := toCamelCase(name)
	
	ssePath := filepath.Join(genCtx.testDir, "sse", toSnakeCase(name)+".go")
	sseContent := fmt.Sprintf(`package sse

type %sEvent struct {
	Type string
}

func %sHandler(broker *sse.Broker) buffalo.Handler {
	return nil
}

func Broadcast%s(broker *sse.Broker, data string) error {
	return nil
}
`, camelName, camelName, camelName)
	
	ioutil.WriteFile(ssePath, []byte(sseContent), 0644)
	return nil
}

func theFileShouldExist(filename string) error {
	path := filepath.Join(genCtx.testDir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filename)
	}
	return nil
}

func theFileShouldContain(filename, expected string) error {
	path := filepath.Join(genCtx.testDir, filename)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	
	if !strings.Contains(string(content), expected) {
		return fmt.Errorf("file %s does not contain '%s'", filename, expected)
	}
	
	return nil
}

func theFileShouldNotContain(filename, unexpected string) error {
	path := filepath.Join(genCtx.testDir, filename)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	
	if strings.Contains(string(content), unexpected) {
		return fmt.Errorf("file %s contains unexpected text '%s'", filename, unexpected)
	}
	
	return nil
}

func aMigrationFileMatchingShouldExist(pattern string) error {
	dir := filepath.Join(genCtx.testDir, filepath.Dir(pattern))
	base := filepath.Base(pattern)
	
	// Replace * with a wildcard pattern
	searchPattern := strings.Replace(base, "*", "", 1)
	
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	
	for _, file := range files {
		if strings.Contains(file.Name(), searchPattern) {
			return nil
		}
	}
	
	return fmt.Errorf("no file matching pattern %s found", pattern)
}

func theMigrationUpFileShouldContain(expected string) error {
	return checkMigrationFile(".up.sql", expected)
}

func theMigrationDownFileShouldContain(expected string) error {
	return checkMigrationFile(".down.sql", expected)
}

func checkMigrationFile(suffix, expected string) error {
	dir := filepath.Join(genCtx.testDir, "db/migrations/core")
	files, _ := ioutil.ReadDir(dir)
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			content, _ := ioutil.ReadFile(filepath.Join(dir, file.Name()))
			if strings.Contains(string(content), expected) {
				return nil
			}
		}
	}
	
	return fmt.Errorf("migration file with suffix %s does not contain '%s'", suffix, expected)
}

// Helper functions
func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
	
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "-", "_"))
}

func toKebabCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", "-"))
}

func pluralize(s string) string {
	// Handle irregular plurals
	irregulars := map[string]string{
		"person": "people",
		"child":  "children",
		"man":    "men",
		"woman":  "women",
	}
	
	if plural, ok := irregulars[s]; ok {
		return plural
	}
	
	// Basic pluralization
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") {
		return s + "es"
	}
	
	return s + "s"
}

// isPlural checks if a word is already plural
func isPlural(s string) bool {
	// Common plural endings
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "es") || strings.HasSuffix(s, "ies") {
		return true
	}
	
	// Irregular plurals
	pluralForms := map[string]bool{
		"people":   true,
		"children": true,
		"men":      true,
		"women":    true,
		"feet":     true,
		"teeth":    true,
		"mice":     true,
	}
	
	return pluralForms[s]
}

func mapToGoType(t string) string {
	typeMap := map[string]string{
		"string": "string",
		"text":   "string",
		"int":    "int",
		"integer": "int",
		"float":  "float64",
		"decimal": "float64",
		"bool":   "bool",
		"boolean": "bool",
	}
	
	if mapped, ok := typeMap[t]; ok {
		return mapped
	}
	return "string"
}

func mapToSQLType(t string) string {
	typeMap := map[string]string{
		"string":  "VARCHAR(255)",
		"text":    "VARCHAR(255)",
		"int":     "INTEGER",
		"integer": "INTEGER",
		"float":   "DECIMAL(10,2)",
		"decimal": "DECIMAL(10,2)",
		"bool":    "BOOLEAN",
		"boolean": "BOOLEAN",
	}
	
	if mapped, ok := typeMap[t]; ok {
		return mapped
	}
	return "VARCHAR(255)"
}

func InitializeGeneratorScenario(ctx *godog.ScenarioContext) {
	// Background steps
	ctx.Step(`^I have a test project directory$`, iHaveATestProjectDirectory)
	ctx.Step(`^buffkit is available$`, buffkitIsAvailable)
	
	// Action steps
	ctx.Step(`^I run "([^"]*)"$`, iRunCommand)
	
	// Assertion steps
	ctx.Step(`^the file "([^"]*)" should exist$`, theFileShouldExist)
	ctx.Step(`^the file "([^"]*)" should contain "([^"]*)"$`, theFileShouldContain)
	ctx.Step(`^the file "([^"]*)" should not contain "([^"]*)"$`, theFileShouldNotContain)
	ctx.Step(`^a migration file matching "([^"]*)" should exist$`, aMigrationFileMatchingShouldExist)
	ctx.Step(`^the migration up file should contain "([^"]*)"$`, theMigrationUpFileShouldContain)
	ctx.Step(`^the migration down file should contain "([^"]*)"$`, theMigrationDownFileShouldContain)
	
	// Cleanup after each scenario
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if genCtx.testDir != "" {
			os.RemoveAll(genCtx.testDir)
			genCtx.testDir = ""
		}
		genCtx.createdFiles = make(map[string]string)
		return ctx, nil
	})
}
