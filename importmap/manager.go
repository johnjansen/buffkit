package importmap

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ImportMap represents the import map structure
type ImportMap struct {
	Imports map[string]string            `json:"imports"`
	Scopes  map[string]map[string]string `json:"scopes,omitempty"`
}

// Manager handles import map operations
type Manager struct {
	imports   map[string]string
	scopes    map[string]map[string]string
	vendorDir string
	integrity map[string]string // SRI hashes for vendored files
	devMode   bool              // Development mode flag
}

// NewManager creates a new import map manager
func NewManager() *Manager {
	return &Manager{
		imports:   make(map[string]string),
		scopes:    make(map[string]map[string]string),
		vendorDir: "public/assets/vendor",
		integrity: make(map[string]string),
		devMode:   false,
	}
}

// NewManagerWithOptions creates a new import map manager with options
func NewManagerWithOptions(vendorDir string, devMode bool) *Manager {
	return &Manager{
		imports:   make(map[string]string),
		scopes:    make(map[string]map[string]string),
		vendorDir: vendorDir,
		integrity: make(map[string]string),
		devMode:   devMode,
	}
}

// LoadDefaults loads the default import map pins
func (m *Manager) LoadDefaults() {
	// Default imports for a Buffkit app
	m.imports["app"] = "/assets/js/index.js"
	m.imports["controllers/"] = "/assets/js/controllers/"
	m.imports["htmx.org"] = "https://unpkg.com/htmx.org@1.9.12/dist/htmx.js"
	m.imports["alpinejs"] = "https://esm.sh/alpinejs@3.14.1"
	m.imports["@hotwired/stimulus"] = "https://unpkg.com/@hotwired/stimulus@3.2.2/dist/stimulus.js"
}

// Pin adds or updates an import mapping
func (m *Manager) Pin(name, url string) {
	m.imports[name] = url
}

// Unpin removes an import mapping
func (m *Manager) Unpin(name string) {
	delete(m.imports, name)
}

// Download downloads a pinned URL to the vendor directory
func (m *Manager) Download(name string) error {
	url, exists := m.imports[name]
	if !exists {
		return fmt.Errorf("import '%s' not found", name)
	}

	// Skip if already local
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") {
		return nil
	}

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Generate filename with content hash
	hash := generateHash(content)
	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".js"
	}
	filename := fmt.Sprintf("%s-%s%s", sanitizeName(name), hash[:8], ext)

	// Generate SRI hash for integrity
	sriHash := generateSRIHash(content)
	m.integrity[name] = sriHash

	// Ensure vendor directory exists
	vendorPath := filepath.Join(m.vendorDir, filename)
	if err := os.MkdirAll(m.vendorDir, 0755); err != nil {
		return fmt.Errorf("failed to create vendor directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(vendorPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write vendor file: %w", err)
	}

	// Update import to use local path
	m.imports[name] = "/assets/vendor/" + filename

	return nil
}

// ToJSON returns the import map as JSON
func (m *Manager) ToJSON() ([]byte, error) {
	im := ImportMap{
		Imports: m.imports,
		Scopes:  m.scopes,
	}
	return json.MarshalIndent(im, "", "  ")
}

// FromJSON loads import map from JSON
func (m *Manager) FromJSON(data []byte) error {
	var im ImportMap
	if err := json.Unmarshal(data, &im); err != nil {
		return err
	}
	m.imports = im.Imports
	m.scopes = im.Scopes
	return nil
}

// RenderHTML returns the import map as an HTML script tag
func (m *Manager) RenderHTML() string {
	jsonData, err := m.ToJSON()
	if err != nil {
		return fmt.Sprintf("<!-- Error generating import map: %v -->", err)
	}

	// Add integrity attributes for vendored files if not in dev mode
	integrityAttrs := ""
	if !m.devMode && len(m.integrity) > 0 {
		for name, sri := range m.integrity {
			integrityAttrs += fmt.Sprintf("\n  <!-- %s integrity: %s -->", name, sri)
		}
	}

	return fmt.Sprintf(`<script type="importmap">
%s
</script>%s`, jsonData, integrityAttrs)
}

// RenderModuleEntrypoint returns the module entry script tag
func (m *Manager) RenderModuleEntrypoint() string {
	// In development mode, add helpful debugging
	debugCode := ""
	if m.devMode {
		debugCode = `
  // Development mode helpers
  window.__BUFFKIT_DEV__ = true;
  console.log('[Buffkit] Import maps loaded in development mode');
`
	}

	return fmt.Sprintf(`<script type="module">%s
  // Import core libraries
  import "htmx.org";
  import Alpine from "alpinejs";

  // Initialize Alpine
  window.Alpine = Alpine;
  Alpine.start();

  // Import app entry point
  import "app";

  // Setup SSE connection with reconnection support
  if (typeof EventSource !== 'undefined') {
    const source = new EventSource('/events', { withCredentials: true });

    source.addEventListener('message', function(e) {
      console.log('SSE message:', e.data);
    });

    source.addEventListener('fragment', function(e) {
      // Handle fragment updates
      try {
        const data = JSON.parse(e.data);
        if (data.target && data.html) {
          const target = document.querySelector(data.target);
          if (target) {
            target.outerHTML = data.html;
          }
        }
      } catch (err) {
        console.error('SSE fragment error:', err);
      }
    });

    source.addEventListener('heartbeat', function(e) {
      console.debug('SSE heartbeat:', e.data);
    });

    source.onerror = function(e) {
      console.error('SSE error:', e);
      // EventSource will automatically reconnect
    };
  }
</script>`, debugCode)
}

// List returns all current imports
func (m *Manager) List() map[string]string {
	result := make(map[string]string)
	for k, v := range m.imports {
		result[k] = v
	}
	return result
}

// SaveToFile saves the import map to a JSON file
func (m *Manager) SaveToFile(path string) error {
	data, err := m.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFromFile loads the import map from a JSON file
func (m *Manager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			m.LoadDefaults()
			return nil
		}
		return err
	}
	return m.FromJSON(data)
}

// UpdateAll downloads and vendors all remote imports
func (m *Manager) UpdateAll() error {
	for name, url := range m.imports {
		// Only download remote URLs
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			if err := m.Download(name); err != nil {
				return fmt.Errorf("failed to download %s: %w", name, err)
			}
		}
	}
	return nil
}

// GetIntegrity returns the SRI hash for a vendored import
func (m *Manager) GetIntegrity(name string) string {
	return m.integrity[name]
}

// SetDevMode sets the development mode flag
func (m *Manager) SetDevMode(devMode bool) {
	m.devMode = devMode
}

// Helper functions

func sanitizeName(name string) string {
	// Remove special characters from name for filename
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "@", "")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

func generateHash(content []byte) string {
	// Use SHA-256 for secure content hashing
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// generateSRIHash generates a Subresource Integrity hash
func generateSRIHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("sha256-%s", base64.StdEncoding.EncodeToString(hash[:]))
}
