// Task 1.1: Project Setup - Tests for version package
package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	// Test that String() returns formatted version info
	result := String()

	// Should contain "fenix version"
	if !strings.Contains(result, "fenix version") {
		t.Errorf("String() = %q, should contain 'fenix version'", result)
	}

	// Should contain version number
	if !strings.Contains(result, Version) {
		t.Errorf("String() = %q, should contain version %q", result, Version)
	}

	// Should contain build time
	if !strings.Contains(result, "built") {
		t.Errorf("String() = %q, should contain 'built'", result)
	}
}

func TestDefaultValues(t *testing.T) {
	// Default version should be "dev"
	if Version != "dev" {
		t.Errorf("Version = %q, want 'dev'", Version)
	}

	// Default build time should be "unknown"
	if BuildTime != "unknown" {
		t.Errorf("BuildTime = %q, want 'unknown'", BuildTime)
	}
}
