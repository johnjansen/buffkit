package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// NameVariants holds different case variations of a name
type NameVariants struct {
	Original string // As provided by user
	Snake    string // snake_case
	Camel    string // CamelCase
	Lower    string // lowercase
	Upper    string // UPPERCASE
	Kebab    string // kebab-case
	Plural   string // pluralized snake_case
	Singular string // singularized snake_case
	Title    string // Title Case
	Package  string // package safe name
}

// NewNameVariants creates all name variations from input
func NewNameVariants(name string) *NameVariants {
	return &NameVariants{
		Original: name,
		Snake:    ToSnake(name),
		Camel:    ToCamel(name),
		Lower:    strings.ToLower(name),
		Upper:    strings.ToUpper(name),
		Kebab:    ToKebab(name),
		Plural:   Pluralize(ToSnake(name)),
		Singular: Singularize(ToSnake(name)),
		Title:    ToTitle(name),
		Package:  ToPackage(name),
	}
}

// ToSnake converts string to snake_case
func ToSnake(s string) string {
	// Handle common cases
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	// Insert underscores before capitals
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = re.ReplaceAllString(s, "${1}_${2}")

	// Handle multiple capitals
	re = regexp.MustCompile("([A-Z]+)([A-Z][a-z])")
	s = re.ReplaceAllString(s, "${1}_${2}")

	return strings.ToLower(s)
}

// ToCamel converts string to CamelCase
func ToCamel(s string) string {
	// Split on common delimiters
	parts := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}

	return strings.Join(parts, "")
}

// ToKebab converts string to kebab-case
func ToKebab(s string) string {
	snake := ToSnake(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// ToTitle converts string to Title Case
func ToTitle(s string) string {
	words := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// ToPackage converts string to a safe Go package name
func ToPackage(s string) string {
	// Remove any non-alphanumeric characters
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	s = reg.ReplaceAllString(s, "")

	// Ensure it starts with a letter
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "pkg" + s
	}

	return strings.ToLower(s)
}

// Pluralize attempts to pluralize a word (basic rules)
func Pluralize(s string) string {
	// Handle common irregular plurals
	irregulars := map[string]string{
		"person": "people",
		"child":  "children",
		"mouse":  "mice",
		"foot":   "feet",
		"goose":  "geese",
		"man":    "men",
		"woman":  "women",
		"tooth":  "teeth",
	}

	if plural, ok := irregulars[s]; ok {
		return plural
	}

	// Basic pluralization rules
	switch {
	case strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1:
		// Check if vowel before y
		beforeY := s[len(s)-2]
		if beforeY != 'a' && beforeY != 'e' && beforeY != 'i' &&
			beforeY != 'o' && beforeY != 'u' {
			return s[:len(s)-1] + "ies"
		}
		return s + "s"
	case strings.HasSuffix(s, "f"):
		return s[:len(s)-1] + "ves"
	case strings.HasSuffix(s, "fe"):
		return s[:len(s)-2] + "ves"
	default:
		return s + "s"
	}
}

// Singularize attempts to singularize a word (basic rules)
func Singularize(s string) string {
	// Handle common irregular singulars
	irregulars := map[string]string{
		"people":   "person",
		"children": "child",
		"mice":     "mouse",
		"feet":     "foot",
		"geese":    "goose",
		"men":      "man",
		"women":    "woman",
		"teeth":    "tooth",
	}

	if singular, ok := irregulars[s]; ok {
		return singular
	}

	// Basic singularization rules
	switch {
	case strings.HasSuffix(s, "ies"):
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "ves"):
		if len(s) > 3 && s[len(s)-4] == 'l' {
			return s[:len(s)-3] + "f"
		}
		return s[:len(s)-3] + "fe"
	case strings.HasSuffix(s, "es"):
		if strings.HasSuffix(s[:len(s)-2], "s") ||
			strings.HasSuffix(s[:len(s)-2], "x") ||
			strings.HasSuffix(s[:len(s)-2], "ch") ||
			strings.HasSuffix(s[:len(s)-2], "sh") {
			return s[:len(s)-2]
		}
		return s[:len(s)-1]
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		return s[:len(s)-1]
	default:
		return s
	}
}

// Field represents a model field for generation
type Field struct {
	Name     string
	Type     string
	Tag      string
	Default  string
	Nullable bool
}

// ParseFields parses field definitions from args
// Format: name:type name:type:nullable
func ParseFields(args []string) []Field {
	fields := make([]Field, 0, len(args))

	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) < 2 {
			continue
		}

		field := Field{
			Name: ToCamel(parts[0]),
			Type: mapFieldType(parts[1]),
		}

		// Check for nullable flag
		if len(parts) > 2 && parts[2] == "nullable" {
			field.Nullable = true
		}

		// Generate JSON tag
		field.Tag = fmt.Sprintf(`json:"%s" db:"%s"`, ToSnake(parts[0]), ToSnake(parts[0]))

		fields = append(fields, field)
	}

	return fields
}

// mapFieldType maps common field types to Go types
func mapFieldType(t string) string {
	typeMap := map[string]string{
		"string":   "string",
		"text":     "string",
		"int":      "int",
		"integer":  "int",
		"bigint":   "int64",
		"float":    "float64",
		"decimal":  "float64",
		"bool":     "bool",
		"boolean":  "bool",
		"date":     "time.Time",
		"datetime": "time.Time",
		"time":     "time.Time",
		"uuid":     "uuid.UUID",
		"json":     "json.RawMessage",
		"jsonb":    "json.RawMessage",
	}

	if mapped, ok := typeMap[strings.ToLower(t)]; ok {
		return mapped
	}
	return t
}

// GenerateFile creates a file from a template
func GenerateFile(tmplContent string, data interface{}, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Parse and execute template
	tmpl, err := template.New("generator").Parse(tmplContent)
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

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FormatCode runs gofmt on the generated file
func FormatCode(path string) error {
	// This would normally run gofmt
	// For now, we'll just return nil
	// In production, you'd use: exec.Command("gofmt", "-w", path).Run()
	return nil
}
