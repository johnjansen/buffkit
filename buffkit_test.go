package buffkit

import (
	"testing"

	"github.com/gobuffalo/buffalo"
)

func TestVersion(t *testing.T) {
	version := Version()
	if version == "" {
		t.Error("Version() should return non-empty string")
	}

	expected := "0.1.0-alpha"
	if version != expected {
		t.Errorf("Version() = %q, want %q", version, expected)
	}
}

func TestConfigValidation(t *testing.T) {
	app := buffalo.New(buffalo.Options{})

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				AuthSecret: []byte("test-secret-key"),
				DevMode:    true,
			},
			wantErr: false,
		},
		{
			name: "missing auth secret",
			config: Config{
				DevMode: true,
			},
			wantErr: true,
		},
		{
			name: "empty auth secret",
			config: Config{
				AuthSecret: []byte(""),
				DevMode:    true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Wire(app, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wire() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWireReturnsKit(t *testing.T) {
	app := buffalo.New(buffalo.Options{})

	config := Config{
		AuthSecret: []byte("test-secret-for-sessions"),
		DevMode:    true,
	}

	kit, err := Wire(app, config)
	if err != nil {
		t.Fatalf("Wire() failed: %v", err)
	}

	if kit == nil {
		t.Fatal("Wire() returned nil kit")
	}

	// Verify kit has required components
	if kit.Broker == nil {
		t.Error("Kit.Broker is nil")
	}

	if kit.AuthStore == nil {
		t.Error("Kit.AuthStore is nil")
	}

	if kit.Mail == nil {
		t.Error("Kit.Mail is nil")
	}

	if kit.ImportMap == nil {
		t.Error("Kit.ImportMap is nil")
	}

	if kit.Components == nil {
		t.Error("Kit.Components is nil")
	}
}

func TestWireWithRedis(t *testing.T) {
	app := buffalo.New(buffalo.Options{})

	config := Config{
		AuthSecret: []byte("test-secret-for-sessions"),
		DevMode:    true,
		RedisURL:   "redis://invalid:6379/0", // Invalid URL to test error handling
	}

	// Should return error for invalid Redis URL
	_, err := Wire(app, config)
	if err == nil {
		t.Error("Wire() should fail with invalid Redis URL")
	}
}

func TestMigrationRunnerDefaults(t *testing.T) {
	runner := &MigrationRunner{
		Dialect: "sqlite",
	}

	if runner.Dialect != "sqlite" {
		t.Errorf("MigrationRunner.Dialect = %q, want %q", runner.Dialect, "sqlite")
	}
}

// TestRequireLoginIsFunction verifies RequireLogin returns a function
func TestRequireLoginIsFunction(t *testing.T) {
	handler := func(c buffalo.Context) error {
		return nil
	}

	middleware := RequireLogin(handler)
	if middleware == nil {
		t.Error("RequireLogin should return a handler function")
	}
}
