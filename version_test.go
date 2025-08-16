package buffkit

import (
	"testing"
)

func TestVersion(t *testing.T) {
	v := Version()

	if v == "" {
		t.Error("Version() returned empty string")
	}

	if v != "0.1.0-alpha" {
		t.Errorf("Version() = %s; want 0.1.0-alpha", v)
	}
}
