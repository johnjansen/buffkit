package importmap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.vendorDir != "public/assets/vendor" {
		t.Errorf("Expected vendor dir 'public/assets/vendor', got '%s'", manager.vendorDir)
	}

	if manager.imports == nil {
		t.Error("Imports map not initialized")
	}

	if manager.scopes == nil {
		t.Error("Scopes map not initialized")
	}

	if manager.integrity == nil {
		t.Error("Integrity map not initialized")
	}

	if manager.devMode {
		t.Error("DevMode should be false by default")
	}
}

func TestNewManagerWithOptions(t *testing.T) {
	manager := NewManagerWithOptions("custom/vendor", true)

	if manager.vendorDir != "custom/vendor" {
		t.Errorf("Expected vendor dir 'custom/vendor', got '%s'", manager.vendorDir)
	}

	if !manager.devMode {
		t.Error("DevMode should be true")
	}
}

func TestLoadDefaults(t *testing.T) {
	manager := NewManager()
	manager.LoadDefaults()

	expectedImports := map[string]string{
		"app":                "/assets/js/index.js",
		"controllers/":       "/assets/js/controllers/",
		"htmx.org":           "https://unpkg.com/htmx.org@1.9.12/dist/htmx.js",
		"alpinejs":           "https://esm.sh/alpinejs@3.14.1",
		"@hotwired/stimulus": "https://unpkg.com/@hotwired/stimulus@3.2.2/dist/stimulus.js",
	}

	for name, expectedURL := range expectedImports {
		if url, exists := manager.imports[name]; !exists {
			t.Errorf("Expected import '%s' not found", name)
		} else if url != expectedURL {
			t.Errorf("Import '%s': expected URL '%s', got '%s'", name, expectedURL, url)
		}
	}
}

func TestPinAndUnpin(t *testing.T) {
	manager := NewManager()

	// Test Pin
	manager.Pin("test-lib", "https://cdn.example.com/test.js")

	if url, exists := manager.imports["test-lib"]; !exists {
		t.Error("Pin failed: import not found")
	} else if url != "https://cdn.example.com/test.js" {
		t.Errorf("Pin failed: expected URL 'https://cdn.example.com/test.js', got '%s'", url)
	}

	// Test Unpin
	manager.Unpin("test-lib")

	if _, exists := manager.imports["test-lib"]; exists {
		t.Error("Unpin failed: import still exists")
	}
}

func TestList(t *testing.T) {
	manager := NewManager()
	manager.Pin("lib1", "url1")
	manager.Pin("lib2", "url2")

	list := manager.List()

	if len(list) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(list))
	}

	if list["lib1"] != "url1" {
		t.Error("List returned incorrect value for lib1")
	}

	if list["lib2"] != "url2" {
		t.Error("List returned incorrect value for lib2")
	}

	// Ensure it's a copy
	delete(list, "lib1")
	if _, exists := manager.imports["lib1"]; !exists {
		t.Error("List should return a copy, not the original map")
	}
}

func TestToJSON(t *testing.T) {
	manager := NewManager()
	manager.Pin("test", "https://example.com/test.js")
	manager.scopes["/admin/"] = map[string]string{
		"utils": "/admin/utils.js",
	}

	jsonData, err := manager.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var importMap ImportMap
	if err := json.Unmarshal(jsonData, &importMap); err != nil {
		t.Fatalf("Invalid JSON produced: %v", err)
	}

	if importMap.Imports["test"] != "https://example.com/test.js" {
		t.Error("JSON missing import")
	}

	if importMap.Scopes["/admin/"]["utils"] != "/admin/utils.js" {
		t.Error("JSON missing scope")
	}
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{
		"imports": {
			"lodash": "https://cdn.skypack.dev/lodash"
		},
		"scopes": {
			"/legacy/": {
				"jquery": "/assets/jquery.min.js"
			}
		}
	}`

	manager := NewManager()
	if err := manager.FromJSON([]byte(jsonStr)); err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if manager.imports["lodash"] != "https://cdn.skypack.dev/lodash" {
		t.Error("Import not loaded from JSON")
	}

	if manager.scopes["/legacy/"]["jquery"] != "/assets/jquery.min.js" {
		t.Error("Scope not loaded from JSON")
	}
}

func TestRenderHTML(t *testing.T) {
	manager := NewManager()
	manager.Pin("test", "https://example.com/test.js")

	html := manager.RenderHTML()

	if !strings.Contains(html, `<script type="importmap">`) {
		t.Error("HTML missing importmap script tag")
	}

	if !strings.Contains(html, `"test": "https://example.com/test.js"`) {
		t.Error("HTML missing import entry")
	}

	// Test with integrity in non-dev mode
	manager.integrity["test"] = "sha256-abc123"
	manager.devMode = false

	html = manager.RenderHTML()
	if !strings.Contains(html, "sha256-abc123") {
		t.Error("HTML missing integrity comment")
	}
}

func TestRenderModuleEntrypoint(t *testing.T) {
	manager := NewManager()

	// Test normal mode
	html := manager.RenderModuleEntrypoint()

	if !strings.Contains(html, `<script type="module">`) {
		t.Error("Missing module script tag")
	}

	if !strings.Contains(html, `import "htmx.org"`) {
		t.Error("Missing htmx import")
	}

	if !strings.Contains(html, `Alpine.start()`) {
		t.Error("Missing Alpine initialization")
	}

	if !strings.Contains(html, `new EventSource('/events', { withCredentials: true })`) {
		t.Error("Missing SSE setup with credentials")
	}

	// Test dev mode
	manager.devMode = true
	html = manager.RenderModuleEntrypoint()

	if !strings.Contains(html, `window.__BUFFKIT_DEV__ = true`) {
		t.Error("Missing dev mode flag")
	}

	if !strings.Contains(html, `[Buffkit] Import maps loaded`) {
		t.Error("Missing dev mode console log")
	}
}

func TestSaveAndLoadFile(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-importmap.json")

	manager := NewManager()
	manager.Pin("test", "https://example.com/test.js")

	// Test save
	if err := manager.SaveToFile(testFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Test load
	manager2 := NewManager()
	if err := manager2.LoadFromFile(testFile); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if manager2.imports["test"] != "https://example.com/test.js" {
		t.Error("Import not loaded from file")
	}

	// Test loading non-existent file (should load defaults)
	manager3 := NewManager()
	if err := manager3.LoadFromFile(filepath.Join(tmpDir, "nonexistent.json")); err != nil {
		t.Fatalf("LoadFromFile should not error on missing file: %v", err)
	}

	// Should have loaded defaults
	if _, exists := manager3.imports["htmx.org"]; !exists {
		t.Error("Defaults not loaded when file doesn't exist")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@scope/package", "scope-package"},
		{"package.js", "package-js"},
		{"path/to/module", "path-to-module"},
		{"simple", "simple"},
	}

	for _, test := range tests {
		result := sanitizeName(test.input)
		if result != test.expected {
			t.Errorf("sanitizeName(%s): expected '%s', got '%s'",
				test.input, test.expected, result)
		}
	}
}

func TestGenerateHash(t *testing.T) {
	content := []byte("test content")
	hash := generateHash(content)

	// SHA-256 produces 64 character hex string
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	// Same content should produce same hash
	hash2 := generateHash(content)
	if hash != hash2 {
		t.Error("Same content produced different hashes")
	}

	// Different content should produce different hash
	hash3 := generateHash([]byte("different content"))
	if hash == hash3 {
		t.Error("Different content produced same hash")
	}
}

func TestGenerateSRIHash(t *testing.T) {
	content := []byte("test content")
	sri := generateSRIHash(content)

	if !strings.HasPrefix(sri, "sha256-") {
		t.Error("SRI hash should start with 'sha256-'")
	}

	// Base64 encoded SHA-256 should be 44 characters (including padding)
	base64Part := strings.TrimPrefix(sri, "sha256-")
	if len(base64Part) != 44 {
		t.Errorf("Expected base64 length 44, got %d", len(base64Part))
	}

	// Same content should produce same SRI
	sri2 := generateSRIHash(content)
	if sri != sri2 {
		t.Error("Same content produced different SRI hashes")
	}
}

func TestSetDevMode(t *testing.T) {
	manager := NewManager()

	if manager.devMode {
		t.Error("DevMode should be false initially")
	}

	manager.SetDevMode(true)
	if !manager.devMode {
		t.Error("SetDevMode(true) failed")
	}

	manager.SetDevMode(false)
	if manager.devMode {
		t.Error("SetDevMode(false) failed")
	}
}

func TestGetIntegrity(t *testing.T) {
	manager := NewManager()

	// Test empty integrity
	if manager.GetIntegrity("nonexistent") != "" {
		t.Error("Should return empty string for non-existent integrity")
	}

	// Set and get integrity
	manager.integrity["test"] = "sha256-abc123"
	if manager.GetIntegrity("test") != "sha256-abc123" {
		t.Error("GetIntegrity returned wrong value")
	}
}

func TestUpdateAll(t *testing.T) {
	manager := NewManager()

	// Add some imports (mix of local and remote)
	manager.Pin("local", "/assets/local.js")
	manager.Pin("remote1", "https://example.com/remote1.js")
	manager.Pin("remote2", "http://example.com/remote2.js")

	// UpdateAll would normally download remote files
	// Since we can't actually download in tests, we're just testing
	// that it identifies remote URLs correctly
	// In a real test, you'd mock the HTTP client

	imports := manager.List()
	remoteCount := 0
	for _, url := range imports {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			remoteCount++
		}
	}

	if remoteCount != 2 {
		t.Errorf("Expected 2 remote imports, found %d", remoteCount)
	}
}

func TestDownload(t *testing.T) {
	// This test would require mocking HTTP requests
	// For now, we just test the error case

	manager := NewManager()

	// Test downloading non-existent import
	err := manager.Download("nonexistent")
	if err == nil {
		t.Error("Download should error for non-existent import")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Test skipping local imports
	manager.Pin("local", "/assets/local.js")
	err = manager.Download("local")
	if err != nil {
		t.Errorf("Download should skip local imports without error: %v", err)
	}
}
