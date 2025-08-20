package migrations

import (
	"testing"
)

func TestGetBuffkitMigrations(t *testing.T) {
	fs := GetBuffkitMigrations()

	// Check that we can read the migrations
	entries, err := fs.ReadDir("buffkit")
	if err != nil {
		t.Fatalf("Failed to read buffkit migrations directory: %v", err)
	}

	// Should have at least the users migrations
	if len(entries) == 0 {
		t.Error("No migration files found in buffkit directory")
	}

	// Check for specific migration files
	expectedFiles := []string{
		"001_create_users.up.sql",
		"001_create_users.down.sql",
		"002_create_sessions.up.sql",
	}

	fileMap := make(map[string]bool)
	for _, entry := range entries {
		fileMap[entry.Name()] = true
	}

	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("Expected migration file %s not found", expected)
		}
	}

	// Verify we can read a migration file
	content, err := fs.ReadFile("buffkit/001_create_users.up.sql")
	if err != nil {
		t.Fatalf("Failed to read migration file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Migration file is empty")
	}

	// Check that it contains expected SQL
	contentStr := string(content)
	if !contains(contentStr, "CREATE TABLE") {
		t.Error("Migration doesn't contain CREATE TABLE statement")
	}
	if !contains(contentStr, "buffkit_users") {
		t.Error("Migration doesn't create buffkit_users table")
	}
}

func TestMigrationList(t *testing.T) {
	list := MigrationList()

	if len(list) == 0 {
		t.Error("Migration list is empty")
	}

	// Check first migration
	if list[0] != "001_create_users" {
		t.Errorf("Expected first migration to be 001_create_users, got %s", list[0])
	}

	// Check that migrations are in order
	for i := 1; i < len(list); i++ {
		if list[i] <= list[i-1] {
			t.Errorf("Migrations not in order: %s comes after %s", list[i], list[i-1])
		}
	}
}

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Error("Version should not be empty")
	}

	// Should follow semver format
	if !contains(v, ".") {
		t.Error("Version should follow semver format (e.g., 0.1.0)")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
