package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/markbates/grift/grift"
)

func init() {
	// Register generator tasks
	registerGeneratorTasks()
}

func registerGeneratorTasks() {
	_ = grift.Namespace("buffkit:generate", func() {
		// Model generator
		_ = grift.Desc("model", "Generate a model with optional migration")
		_ = grift.Add("model", generateModel)

		// Action generator (controllers in Rails)
		_ = grift.Desc("action", "Generate Buffalo action handlers")
		_ = grift.Add("action", generateAction)

		// Resource generator (model + actions + views)
		_ = grift.Desc("resource", "Generate a complete resource (model, actions, views)")
		_ = grift.Add("resource", generateResource)

		// Migration generator with fields
		_ = grift.Desc("migration", "Generate a migration with fields")
		_ = grift.Add("migration", generateMigration)

		// Component generator
		_ = grift.Desc("component", "Generate a server-side component")
		_ = grift.Add("component", generateComponent)

		// Job generator
		_ = grift.Desc("job", "Generate a background job handler")
		_ = grift.Add("job", generateJob)

		// Mailer generator
		_ = grift.Desc("mailer", "Generate email templates and handler")
		_ = grift.Add("mailer", generateMailer)

		// SSE handler generator
		_ = grift.Desc("sse", "Generate a Server-Sent Events handler")
		_ = grift.Add("sse", generateSSE)
	})

	// Shorthand aliases
	_ = grift.Namespace("g", func() {
		_ = grift.Add("model", generateModel)
		_ = grift.Add("action", generateAction)
		_ = grift.Add("resource", generateResource)
		_ = grift.Add("migration", generateMigration)
		_ = grift.Add("component", generateComponent)
		_ = grift.Add("job", generateJob)
		_ = grift.Add("mailer", generateMailer)
		_ = grift.Add("sse", generateSSE)
	})
}

// generateModel creates a model struct and optionally a migration
func generateModel(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:model <name> [field:type ...]")
	}

	name := c.Args[0]
	fields := ParseFields(c.Args[1:])
	names := NewNameVariants(name)

	// Generate model struct
	modelPath := fmt.Sprintf("models/%s.go", names.Snake)
	
	modelTemplate := `package models

import (
	"context"
	"database/sql"
	"time"
{{if .HasUUID}}	"github.com/gofrs/uuid"{{end}}
{{if .HasJSON}}	"encoding/json"{{end}}
)

// {{.Names.Camel}} represents a {{.Names.Snake}} in the database
type {{.Names.Camel}} struct {
	ID        int       ` + "`" + `json:"id" db:"id"` + "`" + `
{{range .Fields}}	{{.Name}} {{if .Nullable}}*{{end}}{{.Type}} ` + "`" + `{{.Tag}}` + "`" + `
{{end}}	CreatedAt time.Time ` + "`" + `json:"created_at" db:"created_at"` + "`" + `
	UpdatedAt time.Time ` + "`" + `json:"updated_at" db:"updated_at"` + "`" + `
}

// TableName returns the database table name
func ({{.Names.Lower}} *{{.Names.Camel}}) TableName() string {
	return "{{.Names.Plural}}"
}

// Create inserts the {{.Names.Snake}} into the database
func ({{.Names.Lower}} *{{.Names.Camel}}) Create(ctx context.Context, db *sql.DB) error {
	query := ` + "`" + `
		INSERT INTO {{.Names.Plural}} ({{.FieldNamesDB}}, created_at, updated_at)
		VALUES ({{.FieldPlaceholders}}, ?, ?)
		RETURNING id` + "`" + `
	
	now := time.Now()
	{{.Names.Lower}}.CreatedAt = now
	{{.Names.Lower}}.UpdatedAt = now
	
	err := db.QueryRowContext(ctx, query, {{.FieldValues}}, now, now).Scan(&{{.Names.Lower}}.ID)
	return err
}

// Update updates the {{.Names.Snake}} in the database
func ({{.Names.Lower}} *{{.Names.Camel}}) Update(ctx context.Context, db *sql.DB) error {
	query := ` + "`" + `
		UPDATE {{.Names.Plural}}
		SET {{.UpdateFields}}, updated_at = ?
		WHERE id = ?` + "`" + `
	
	{{.Names.Lower}}.UpdatedAt = time.Now()
	
	_, err := db.ExecContext(ctx, query, {{.FieldValues}}, {{.Names.Lower}}.UpdatedAt, {{.Names.Lower}}.ID)
	return err
}

// Delete removes the {{.Names.Snake}} from the database
func ({{.Names.Lower}} *{{.Names.Camel}}) Delete(ctx context.Context, db *sql.DB) error {
	query := ` + "`" + `DELETE FROM {{.Names.Plural}} WHERE id = ?` + "`" + `
	_, err := db.ExecContext(ctx, query, {{.Names.Lower}}.ID)
	return err
}

// Find{{.Names.Camel}} finds a {{.Names.Snake}} by ID
func Find{{.Names.Camel}}(ctx context.Context, db *sql.DB, id int) (*{{.Names.Camel}}, error) {
	{{.Names.Lower}} := &{{.Names.Camel}}{}
	query := ` + "`" + `SELECT * FROM {{.Names.Plural}} WHERE id = ?` + "`" + `
	
	err := db.QueryRowContext(ctx, query, id).Scan(
		&{{.Names.Lower}}.ID,
{{range .Fields}}		&{{$.Names.Lower}}.{{.Name}},
{{end}}		&{{.Names.Lower}}.CreatedAt,
		&{{.Names.Lower}}.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	return {{.Names.Lower}}, nil
}

// All{{.Names.Plural}} returns all {{.Names.Plural}} from the database
func All{{.Names.Plural}}(ctx context.Context, db *sql.DB) ([]*{{.Names.Camel}}, error) {
	query := ` + "`" + `SELECT * FROM {{.Names.Plural}} ORDER BY created_at DESC` + "`" + `
	
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var {{.Names.Plural}} []*{{.Names.Camel}}
	for rows.Next() {
		{{.Names.Lower}} := &{{.Names.Camel}}{}
		err := rows.Scan(
			&{{.Names.Lower}}.ID,
{{range .Fields}}			&{{$.Names.Lower}}.{{.Name}},
{{end}}			&{{.Names.Lower}}.CreatedAt,
			&{{.Names.Lower}}.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		{{.Names.Plural}} = append({{.Names.Plural}}, {{.Names.Lower}})
	}
	
	return {{.Names.Plural}}, rows.Err()
}
`

	// Prepare template data
	data := map[string]interface{}{
		"Names":  names,
		"Fields": fields,
		"HasUUID": hasFieldType(fields, "uuid.UUID"),
		"HasJSON": hasFieldType(fields, "json.RawMessage"),
		"FieldNamesDB": fieldNamesDB(fields),
		"FieldPlaceholders": fieldPlaceholders(fields),
		"FieldValues": fieldValues(fields, names.Lower),
		"UpdateFields": updateFields(fields),
	}

	if err := GenerateFile(modelTemplate, data, modelPath); err != nil {
		return fmt.Errorf("failed to generate model: %w", err)
	}

	fmt.Printf("✅ Generated model: %s\n", modelPath)

	// Optionally generate migration
	if len(fields) > 0 {
		if err := generateModelMigration(names, fields); err != nil {
			return fmt.Errorf("failed to generate migration: %w", err)
		}
	}

	return nil
}

// generateAction creates Buffalo action handlers
func generateAction(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:action <resource> [actions...]")
	}

	resource := c.Args[0]
	names := NewNameVariants(resource)
	
	// Default actions if none specified
	actions := c.Args[1:]
	if len(actions) == 0 {
		actions = []string{"index", "show", "new", "create", "edit", "update", "destroy"}
	}

	actionPath := fmt.Sprintf("actions/%s.go", names.Plural)
	
	actionTemplate := `package actions

import (
	"net/http"
	
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"your-app/models"
)
{{range .Actions}}
// {{$.Names.Plural}}{{. | title}} handles {{. | lower}} action for {{$.Names.Plural}}
func {{$.Names.Plural}}{{. | title}}(c buffalo.Context) error {
{{if eq . "index"}}	{{$.Names.Plural}}, err := models.All{{$.Names.Plural}}(c.Request().Context(), c.Value("db").(*sql.DB))
	if err != nil {
		return err
	}
	
	c.Set("{{$.Names.Plural}}", {{$.Names.Plural}})
	return c.Render(http.StatusOK, r.HTML("{{$.Names.Plural}}/index.plush.html"))
{{else if eq . "show"}}	{{$.Names.Lower}}, err := models.Find{{$.Names.Camel}}(c.Request().Context(), c.Value("db").(*sql.DB), c.Param("id"))
	if err != nil {
		return c.Error(http.StatusNotFound, err)
	}
	
	c.Set("{{$.Names.Lower}}", {{$.Names.Lower}})
	return c.Render(http.StatusOK, r.HTML("{{$.Names.Plural}}/show.plush.html"))
{{else if eq . "new"}}	{{$.Names.Lower}} := &models.{{$.Names.Camel}}{}
	c.Set("{{$.Names.Lower}}", {{$.Names.Lower}})
	return c.Render(http.StatusOK, r.HTML("{{$.Names.Plural}}/new.plush.html"))
{{else if eq . "create"}}	{{$.Names.Lower}} := &models.{{$.Names.Camel}}{}
	if err := c.Bind({{$.Names.Lower}}); err != nil {
		return err
	}
	
	if err := {{$.Names.Lower}}.Create(c.Request().Context(), c.Value("db").(*sql.DB)); err != nil {
		c.Set("{{$.Names.Lower}}", {{$.Names.Lower}})
		c.Set("errors", err)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("{{$.Names.Plural}}/new.plush.html"))
	}
	
	c.Flash().Add("success", "{{$.Names.Title}} was created successfully")
	return c.Redirect(http.StatusSeeOther, "/{{$.Names.Plural}}/%d", {{$.Names.Lower}}.ID)
{{else if eq . "edit"}}	{{$.Names.Lower}}, err := models.Find{{$.Names.Camel}}(c.Request().Context(), c.Value("db").(*sql.DB), c.Param("id"))
	if err != nil {
		return c.Error(http.StatusNotFound, err)
	}
	
	c.Set("{{$.Names.Lower}}", {{$.Names.Lower}})
	return c.Render(http.StatusOK, r.HTML("{{$.Names.Plural}}/edit.plush.html"))
{{else if eq . "update"}}	{{$.Names.Lower}}, err := models.Find{{$.Names.Camel}}(c.Request().Context(), c.Value("db").(*sql.DB), c.Param("id"))
	if err != nil {
		return c.Error(http.StatusNotFound, err)
	}
	
	if err := c.Bind({{$.Names.Lower}}); err != nil {
		return err
	}
	
	if err := {{$.Names.Lower}}.Update(c.Request().Context(), c.Value("db").(*sql.DB)); err != nil {
		c.Set("{{$.Names.Lower}}", {{$.Names.Lower}})
		c.Set("errors", err)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("{{$.Names.Plural}}/edit.plush.html"))
	}
	
	c.Flash().Add("success", "{{$.Names.Title}} was updated successfully")
	return c.Redirect(http.StatusSeeOther, "/{{$.Names.Plural}}/%d", {{$.Names.Lower}}.ID)
{{else if eq . "destroy"}}	{{$.Names.Lower}}, err := models.Find{{$.Names.Camel}}(c.Request().Context(), c.Value("db").(*sql.DB), c.Param("id"))
	if err != nil {
		return c.Error(http.StatusNotFound, err)
	}
	
	if err := {{$.Names.Lower}}.Delete(c.Request().Context(), c.Value("db").(*sql.DB)); err != nil {
		return err
	}
	
	c.Flash().Add("success", "{{$.Names.Title}} was deleted successfully")
	return c.Redirect(http.StatusSeeOther, "/{{$.Names.Plural}}")
{{else}}	// TODO: Implement {{.}} action
	return c.Render(http.StatusOK, r.HTML("{{$.Names.Plural}}/{{.}}.plush.html"))
{{end}}}
{{end}}
`

	// Create custom template functions
	funcMap := map[string]interface{}{
		"title": strings.Title,
		"lower": strings.ToLower,
	}

	// Prepare template data
	data := map[string]interface{}{
		"Names":   names,
		"Actions": actions,
	}

	if err := GenerateFileWithFuncs(actionTemplate, data, actionPath, funcMap); err != nil {
		return fmt.Errorf("failed to generate actions: %w", err)
	}

	fmt.Printf("✅ Generated actions: %s\n", actionPath)
	
	// Generate route registration helper
	fmt.Println("\n📝 Add these routes to your app:")
	fmt.Printf("app.Resource(\"/"+"%s\", buffalo.WrapHandlerFunc(actions.%s))\n", names.Plural, names.Plural+"Index")
	
	return nil
}

// generateResource generates a complete resource (model + actions + views)
func generateResource(c *grift.Context) error {
	// First generate model
	if err := generateModel(c); err != nil {
		return err
	}
	
	// Then generate actions
	if err := generateAction(c); err != nil {
		return err
	}
	
	// Generate basic view templates
	name := c.Args[0]
	names := NewNameVariants(name)
	
	viewsDir := fmt.Sprintf("templates/%s", names.Plural)
	views := []string{"index", "show", "new", "edit", "_form"}
	
	for _, view := range views {
		viewPath := filepath.Join(viewsDir, view+".plush.html")
		if err := generateView(names, view, viewPath); err != nil {
			return fmt.Errorf("failed to generate view %s: %w", view, err)
		}
		fmt.Printf("✅ Generated view: %s\n", viewPath)
	}
	
	return nil
}

// generateMigration creates an enhanced migration with field definitions
func generateMigration(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:migration <name> [field:type ...]")
	}

	name := c.Args[0]
	fields := ParseFields(c.Args[1:])
	
	// Detect migration type from name
	var migrationType string
	if strings.HasPrefix(name, "create_") {
		migrationType = "create"
	} else if strings.HasPrefix(name, "add_") {
		migrationType = "add"
	} else if strings.HasPrefix(name, "remove_") {
		migrationType = "remove"
	} else {
		migrationType = "change"
	}

	// Generate timestamp-based filename
	timestamp := time.Now().Format("20060102150405")
	dir := "db/migrations/core"
	upFile := fmt.Sprintf("%s/%s_%s.up.sql", dir, timestamp, ToSnake(name))
	downFile := fmt.Sprintf("%s/%s_%s.down.sql", dir, timestamp, ToSnake(name))

	// Create migration directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate UP migration
	var upContent string
	var downContent string

	switch migrationType {
	case "create":
		tableName := strings.TrimPrefix(name, "create_")
		upContent = generateCreateTableSQL(tableName, fields)
		downContent = fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)
	case "add":
		parts := strings.Split(name, "_to_")
		if len(parts) == 2 {
			tableName := parts[1]
			upContent = generateAddColumnsSQL(tableName, fields)
			downContent = generateDropColumnsSQL(tableName, fields)
		}
	default:
		upContent = "-- Add your migration SQL here\n"
		downContent = "-- Add your rollback SQL here\n"
	}

	// Write migration files
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to create up migration: %w", err)
	}

	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to create down migration: %w", err)
	}

	fmt.Printf("✅ Created migration files:\n")
	fmt.Printf("   - %s\n", upFile)
	fmt.Printf("   - %s\n", downFile)
	
	return nil
}

// generateComponent creates a server-side component
func generateComponent(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:component <name>")
	}

	name := c.Args[0]
	names := NewNameVariants(name)
	
	// Generate component file
	componentPath := fmt.Sprintf("components/%s.go", names.Snake)
	
	componentTemplate := `package components

import (
	"bytes"
	"fmt"
	"html/template"
)

// {{.Names.Camel}}Component renders a {{.Names.Kebab}} component
func {{.Names.Camel}}Component(attrs map[string]string, slots map[string][]byte) ([]byte, error) {
	// Extract attributes
	variant := attrs["variant"]
	if variant == "" {
		variant = "default"
	}
	
	class := attrs["class"]
	id := attrs["id"]
	
	// Get content from default slot
	content := slots["default"]
	
	// Build component HTML
	tmpl := ` + "`" + `<div 
		{{if .ID}}id="{{.ID}}"{{end}}
		class="bk-{{.Names.Kebab}} bk-{{.Names.Kebab}}-{{.Variant}}{{if .Class}} {{.Class}}{{end}}"
		data-component="{{.Names.Kebab}}"
	>
		{{if .Header}}
		<div class="bk-{{.Names.Kebab}}-header">
			{{.Header}}
		</div>
		{{end}}
		
		<div class="bk-{{.Names.Kebab}}-content">
			{{.Content}}
		</div>
		
		{{if .Footer}}
		<div class="bk-{{.Names.Kebab}}-footer">
			{{.Footer}}
		</div>
		{{end}}
	</div>` + "`" + `
	
	// Parse and execute template
	t, err := template.New("{{.Names.Snake}}").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse {{.Names.Snake}} template: %w", err)
	}
	
	data := map[string]interface{}{
		"Names":   map[string]string{"Kebab": "{{.Names.Kebab}}"},
		"ID":      id,
		"Class":   class,
		"Variant": variant,
		"Content": template.HTML(content),
		"Header":  template.HTML(slots["header"]),
		"Footer":  template.HTML(slots["footer"]),
	}
	
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute {{.Names.Snake}} template: %w", err)
	}
	
	return buf.Bytes(), nil
}

// Register registers the {{.Names.Snake}} component
func Register{{.Names.Camel}}(registry *Registry) {
	registry.Register("{{.Names.Kebab}}", {{.Names.Camel}}Component)
}
`

	data := map[string]interface{}{
		"Names": names,
	}

	if err := GenerateFile(componentTemplate, data, componentPath); err != nil {
		return fmt.Errorf("failed to generate component: %w", err)
	}

	fmt.Printf("✅ Generated component: %s\n", componentPath)
	fmt.Printf("\n📝 Register your component in your app setup:\n")
	fmt.Printf("kit.Components.Register(\"%s\", components.%sComponent)\n", names.Kebab, names.Camel)
	
	// Generate CSS file
	cssPath := fmt.Sprintf("assets/css/components/%s.css", names.Kebab)
	cssTemplate := `/* {{.Names.Title}} Component Styles */

.bk-{{.Names.Kebab}} {
	display: block;
	padding: 1rem;
	border: 1px solid #e5e7eb;
	border-radius: 0.375rem;
	background: white;
}

.bk-{{.Names.Kebab}}-header {
	font-weight: 600;
	margin-bottom: 0.75rem;
	padding-bottom: 0.75rem;
	border-bottom: 1px solid #e5e7eb;
}

.bk-{{.Names.Kebab}}-content {
	padding: 0.5rem 0;
}

.bk-{{.Names.Kebab}}-footer {
	margin-top: 0.75rem;
	padding-top: 0.75rem;
	border-top: 1px solid #e5e7eb;
}

/* Variants */
.bk-{{.Names.Kebab}}-primary {
	border-color: #3b82f6;
	background: #eff6ff;
}

.bk-{{.Names.Kebab}}-success {
	border-color: #10b981;
	background: #f0fdf4;
}

.bk-{{.Names.Kebab}}-warning {
	border-color: #f59e0b;
	background: #fffbeb;
}

.bk-{{.Names.Kebab}}-danger {
	border-color: #ef4444;
	background: #fef2f2;
}
`

	if err := GenerateFile(cssTemplate, data, cssPath); err != nil {
		fmt.Printf("⚠️  Could not generate CSS file: %v\n", err)
	} else {
		fmt.Printf("✅ Generated CSS: %s\n", cssPath)
	}
	
	return nil
}

// generateJob creates a background job handler
func generateJob(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:job <name>")
	}

	name := c.Args[0]
	names := NewNameVariants(name)
	
	jobPath := fmt.Sprintf("jobs/%s.go", names.Snake)
	
	jobTemplate := `package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/hibiken/asynq"
)

// {{.Names.Camel}}Job represents the payload for {{.Names.Snake}} job
type {{.Names.Camel}}Job struct {
	ID        string    ` + "`" + `json:"id"` + "`" + `
	Data      string    ` + "`" + `json:"data"` + "`" + `
	Timestamp time.Time ` + "`" + `json:"timestamp"` + "`" + `
}

// {{.Names.Camel}}Handler processes {{.Names.Snake}} jobs
func {{.Names.Camel}}Handler(ctx context.Context, t *asynq.Task) error {
	// Parse job payload
	var job {{.Names.Camel}}Job
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal {{.Names.Snake}} job: %w", err)
	}
	
	// Log job start
	fmt.Printf("Processing {{.Names.Snake}} job %s at %v\n", job.ID, job.Timestamp)
	
	// TODO: Implement your job logic here
	// Example:
	// - Send emails
	// - Process data
	// - Call external APIs
	// - Update database records
	
	// Simulate work
	select {
	case <-time.After(2 * time.Second):
		// Job completed successfully
		fmt.Printf("Completed {{.Names.Snake}} job %s\n", job.ID)
		return nil
		
	case <-ctx.Done():
		// Job was cancelled
		return fmt.Errorf("{{.Names.Snake}} job %s was cancelled", job.ID)
	}
}

// Enqueue{{.Names.Camel}} enqueues a new {{.Names.Snake}} job
func Enqueue{{.Names.Camel}}(client *asynq.Client, data string) error {
	job := {{.Names.Camel}}Job{
		ID:        generateJobID(),
		Data:      data,
		Timestamp: time.Now(),
	}
	
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal {{.Names.Snake}} job: %w", err)
	}
	
	task := asynq.NewTask("{{.Names.Snake}}", payload)
	
	// Enqueue with options
	info, err := client.Enqueue(task,
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	
	if err != nil {
		return fmt.Errorf("failed to enqueue {{.Names.Snake}} job: %w", err)
	}
	
	fmt.Printf("Enqueued {{.Names.Snake}} job %s (task ID: %s)\n", job.ID, info.ID)
	return nil
}

// Register{{.Names.Camel}}Handler registers the job handler with the mux
func Register{{.Names.Camel}}Handler(mux *asynq.ServeMux) {
	mux.HandleFunc("{{.Names.Snake}}", {{.Names.Camel}}Handler)
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("{{.Names.Snake}}_%d", time.Now().UnixNano())
}
`

	data := map[string]interface{}{
		"Names": names,
	}

	if err := GenerateFile(jobTemplate, data, jobPath); err != nil {
		return fmt.Errorf("failed to generate job: %w", err)
	}

	fmt.Printf("✅ Generated job handler: %s\n", jobPath)
	fmt.Printf("\n📝 Register your job handler in your app setup:\n")
	fmt.Printf("jobs.Register%sHandler(kit.Jobs.Mux)\n", names.Camel)
	
	return nil
}

// generateMailer creates email templates and handler
func generateMailer(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:mailer <name> [actions...]")
	}

	name := c.Args[0]
	names := NewNameVariants(name)
	
	// Default mail actions if none specified
	actions := c.Args[1:]
	if len(actions) == 0 {
		actions = []string{"welcome", "notification"}
	}
	
	mailerPath := fmt.Sprintf("mailers/%s.go", names.Snake)
	
	mailerTemplate := `package mailers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	
	"github.com/johnjansen/buffkit/mail"
)

// {{.Names.Camel}}Mailer handles {{.Names.Snake}} emails
type {{.Names.Camel}}Mailer struct {
	sender mail.Sender
}

// New{{.Names.Camel}}Mailer creates a new {{.Names.Snake}} mailer
func New{{.Names.Camel}}Mailer(sender mail.Sender) *{{.Names.Camel}}Mailer {
	return &{{.Names.Camel}}Mailer{
		sender: sender,
	}
}
{{range .Actions}}
// Send{{. | title}} sends a {{.}} email
func (m *{{$.Names.Camel}}Mailer) Send{{. | title}}(ctx context.Context, to string, data map[string]interface{}) error {
	// Load template
	tmpl, err := template.ParseFiles("templates/mail/{{$.Names.Snake}}/{{.}}.html")
	if err != nil {
		return fmt.Errorf("failed to parse {{.}} template: %w", err)
	}
	
	// Execute template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute {{.}} template: %w", err)
	}
	
	// Create message
	msg := mail.Message{
		To:      []string{to},
		Subject: "{{. | title}} from {{$.Names.Title}}",
		HTML:    body.String(),
	}
	
	// Send email
	return m.sender.Send(ctx, msg)
}
{{end}}
`

	// Create custom template functions
	funcMap := map[string]interface{}{
		"title": strings.Title,
	}

	data := map[string]interface{}{
		"Names":   names,
		"Actions": actions,
	}

	if err := GenerateFileWithFuncs(mailerTemplate, data, mailerPath, funcMap); err != nil {
		return fmt.Errorf("failed to generate mailer: %w", err)
	}

	fmt.Printf("✅ Generated mailer: %s\n", mailerPath)
	
	// Generate email templates
	for _, action := range actions {
		templatePath := fmt.Sprintf("templates/mail/%s/%s.html", names.Snake, action)
		emailTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #3b82f6; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9fafb; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            
            <!-- Add your email content here -->
            <p>This is a ` + action + ` email from ` + names.Title + `.</p>
            
            <p>Best regards,<br>The Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 Your Company. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
		
		if err := GenerateFile(emailTemplate, nil, templatePath); err != nil {
			fmt.Printf("⚠️  Could not generate email template %s: %v\n", action, err)
		} else {
			fmt.Printf("✅ Generated email template: %s\n", templatePath)
		}
	}
	
	return nil
}

// generateSSE creates a Server-Sent Events handler
func generateSSE(c *grift.Context) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("usage: buffalo task buffkit:generate:sse <name>")
	}

	name := c.Args[0]
	names := NewNameVariants(name)
	
	ssePath := fmt.Sprintf("sse/%s.go", names.Snake)
	
	sseTemplate := `package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/gobuffalo/buffalo"
	"github.com/johnjansen/buffkit/sse"
)

// {{.Names.Camel}}Event represents a {{.Names.Snake}} SSE event
type {{.Names.Camel}}Event struct {
	Type      string    ` + "`" + `json:"type"` + "`" + `
	Data      string    ` + "`" + `json:"data"` + "`" + `
	Timestamp time.Time ` + "`" + `json:"timestamp"` + "`" + `
}

// {{.Names.Camel}}Handler handles {{.Names.Snake}} SSE connections
func {{.Names.Camel}}Handler(broker *sse.Broker) buffalo.Handler {
	return func(c buffalo.Context) error {
		// Get event type from query params
		eventType := c.Param("type")
		if eventType == "" {
			eventType = "{{.Names.Snake}}"
		}
		
		// Subscribe to broker
		client := broker.Subscribe(eventType)
		defer broker.Unsubscribe(client)
		
		// Set SSE headers
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		
		// Send events
		for {
			select {
			case event := <-client.Events:
				// Format and send event
				if _, err := fmt.Fprintf(c.Response(), "data: %s\n\n", event); err != nil {
					return err
				}
				c.Response().(http.Flusher).Flush()
				
			case <-c.Request().Context().Done():
				// Client disconnected
				return nil
			}
		}
	}
}

// Broadcast{{.Names.Camel}} broadcasts a {{.Names.Snake}} event
func Broadcast{{.Names.Camel}}(broker *sse.Broker, data string) error {
	event := {{.Names.Camel}}Event{
		Type:      "{{.Names.Snake}}",
		Data:      data,
		Timestamp: time.Now(),
	}
	
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal {{.Names.Snake}} event: %w", err)
	}
	
	broker.Broadcast("{{.Names.Snake}}", payload)
	return nil
}

// Setup{{.Names.Camel}}Routes sets up SSE routes for {{.Names.Snake}}
func Setup{{.Names.Camel}}Routes(app *buffalo.App, broker *sse.Broker) {
	app.GET("/events/{{.Names.Kebab}}", {{.Names.Camel}}Handler(broker))
}
`

	data := map[string]interface{}{
		"Names": names,
	}

	if err := GenerateFile(sseTemplate, data, ssePath); err != nil {
		return fmt.Errorf("failed to generate SSE handler: %w", err)
	}

	fmt.Printf("✅ Generated SSE handler: %s\n", ssePath)
	fmt.Printf("\n📝 Set up your SSE routes in your app:\n")
	fmt.Printf("sse.Setup%sRoutes(app, kit.Broker)\n", names.Camel)
	
	return nil
}

// Helper functions

func generateModelMigration(names *NameVariants, fields []Field) error {
	timestamp := time.Now().Format("20060102150405")
	dir := "db/migrations/core"
	upFile := fmt.Sprintf("%s/%s_create_%s.up.sql", dir, timestamp, names.Plural)
	downFile := fmt.Sprintf("%s/%s_create_%s.down.sql", dir, timestamp, names.Plural)

	upContent := generateCreateTableSQL(names.Plural, fields)
	downContent := fmt.Sprintf("DROP TABLE IF EXISTS %s;", names.Plural)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return err
	}

	fmt.Printf("✅ Generated migration: %s\n", upFile)
	return nil
}

func generateCreateTableSQL(tableName string, fields []Field) string {
	sql := fmt.Sprintf("CREATE TABLE %s (\n", tableName)
	sql += "    id SERIAL PRIMARY KEY,\n"
	
	for _, field := range fields {
		sqlType := mapToSQLType(field.Type)
		nullable := ""
		if !field.Nullable {
			nullable = " NOT NULL"
		}
		sql += fmt.Sprintf("    %s %s%s,\n", ToSnake(field.Name), sqlType, nullable)
	}
	
	sql += "    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\n"
	sql += "    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP\n"
	sql += ");"
	
	return sql
}

func generateAddColumnsSQL(tableName string, fields []Field) string {
	sql := fmt.Sprintf("ALTER TABLE %s\n", tableName)
	for i, field := range fields {
		sqlType := mapToSQLType(field.Type)
		nullable := ""
		if !field.Nullable {
			nullable = " NOT NULL"
		}
		sql += fmt.Sprintf("    ADD COLUMN %s %s%s", ToSnake(field.Name), sqlType, nullable)
		if i < len(fields)-1 {
			sql += ","
		}
		sql += "\n"
	}
	sql += ";"
	return sql
}

func generateDropColumnsSQL(tableName string, fields []Field) string {
	sql := fmt.Sprintf("ALTER TABLE %s\n", tableName)
	for i, field := range fields {
		sql += fmt.Sprintf("    DROP COLUMN IF EXISTS %s", ToSnake(field.Name))
		if i < len(fields)-1 {
			sql += ","
		}
		sql += "\n"
	}
	sql += ";"
	return sql
}

func mapToSQLType(goType string) string {
	typeMap := map[string]string{
		"string":          "VARCHAR(255)",
		"int":             "INTEGER",
		"int64":           "BIGINT",
		"float64":         "DECIMAL(10,2)",
		"bool":            "BOOLEAN",
		"time.Time":       "TIMESTAMP",
		"uuid.UUID":       "UUID",
		"json.RawMessage": "JSONB",
	}
	
	if sqlType, ok := typeMap[goType]; ok {
		return sqlType
	}
	return "VARCHAR(255)"
}

func hasFieldType(fields []Field, fieldType string) bool {
	for _, field := range fields {
		if field.Type == fieldType {
			return true
		}
	}
	return false
}

func fieldNamesDB(fields []Field) string {
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = ToSnake(field.Name)
	}
	return strings.Join(names, ", ")
}

func fieldPlaceholders(fields []Field) string {
	placeholders := make([]string, len(fields))
	for i := range fields {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ", ")
}

func fieldValues(fields []Field, varName string) string {
	values := make([]string, len(fields))
	for i, field := range fields {
		values[i] = fmt.Sprintf("%s.%s", varName, field.Name)
	}
	return strings.Join(values, ", ")
}

func updateFields(fields []Field) string {
	updates := make([]string, len(fields))
	for i, field := range fields {
		updates[i] = fmt.Sprintf("%s = ?", ToSnake(field.Name))
	}
	return strings.Join(updates, ", ")
}

func generateView(names *NameVariants, view, path string) error {
	// Simple view templates
	templates := map[string]string{
		"index": `<h1>{{.Names.Title}} List</h1>
<%= for ({{.Names.Lower}}) in {{.Names.Plural}} { %>
  <div>
    <%= {{.Names.Lower}}.ID %> - 
    <a href="/{{.Names.Plural}}/<%= {{.Names.Lower}}.ID %>">View</a>
  </div>
<% } %>
<a href="/{{.Names.Plural}}/new">New {{.Names.Title}}</a>`,
		
		"show": `<h1>{{.Names.Title}} Details</h1>
<p>ID: <%= {{.Names.Lower}}.ID %></p>
<a href="/{{.Names.Plural}}/<%= {{.Names.Lower}}.ID %>/edit">Edit</a>
<a href="/{{.Names.Plural}}">Back to List</a>`,
		
		"new": `<h1>New {{.Names.Title}}</h1>
<%= form_for({{.Names.Lower}}, {action: "/{{.Names.Plural}}", method: "POST"}) { %>
  <%= partial("{{.Names.Plural}}/form.html") %>
  <button type="submit">Create</button>
<% } %>`,
		
		"edit": `<h1>Edit {{.Names.Title}}</h1>
<%= form_for({{.Names.Lower}}, {action: "/{{.Names.Plural}}/" + {{.Names.Lower}}.ID, method: "PUT"}) { %>
  <%= partial("{{.Names.Plural}}/form.html") %>
  <button type="submit">Update</button>
<% } %>`,
		
		"_form": `<!-- Add your form fields here -->
<div>
  <label>Field Name</label>
  <input type="text" name="field_name" value="<%= {{.Names.Lower}}.FieldName %>" />
</div>`,
	}
	
	tmpl, ok := templates[view]
	if !ok {
		tmpl = fmt.Sprintf("<!-- %s view for %s -->", view, names.Title)
	}
	
	data := map[string]interface{}{
		"Names": names,
	}
	
	return GenerateFile(tmpl, data, path)
}

// GenerateFileWithFuncs creates a file from a template with custom functions
func GenerateFileWithFuncs(tmplContent string, data interface{}, outputPath string, funcs template.FuncMap) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Parse and execute template with functions
	tmpl, err := template.New("generator").Funcs(funcs).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()
	
	// Execute template to file
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	
	return nil
}
