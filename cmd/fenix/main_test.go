package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_Default_PrintsVersion(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code := run([]string{}, &out)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "fenix version") {
		t.Fatalf("expected version output, got %q", out.String())
	}
}

func TestRun_Help_PrintsUsage(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code := run([]string{"--help"}, &out)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected help output, got %q", out.String())
	}
}

func TestRun_InvalidFlag_Returns2(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code := run([]string{"--unknown-flag"}, &out)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
