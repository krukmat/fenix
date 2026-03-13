package scheduler

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFormatOptionalTime(t *testing.T) {
	t.Parallel()

	if got := formatOptionalTime(nil); got != nil {
		t.Fatalf("formatOptionalTime(nil) = %#v, want nil", got)
	}

	value := time.Date(2026, 3, 13, 10, 0, 0, 123, time.UTC)
	got, ok := formatOptionalTime(&value).(string)
	if !ok {
		t.Fatalf("formatOptionalTime(non-nil) type = %T, want string", formatOptionalTime(&value))
	}
	if got != formatTime(value) {
		t.Fatalf("formatOptionalTime(non-nil) = %q, want %q", got, formatTime(value))
	}
}

func TestParseTimeRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, err := parseTime("not-a-time"); err == nil {
		t.Fatal("parseTime(invalid) expected error")
	}
}

func TestNormalizeJSONUsesFallbackAndCopiesInput(t *testing.T) {
	t.Parallel()

	fallback := []byte(`{}`)
	got := normalizeJSON(nil, fallback)
	if string(got) != "{}" {
		t.Fatalf("normalizeJSON(nil) = %s, want {}", string(got))
	}

	raw := json.RawMessage(`{"ok":true}`)
	got = normalizeJSON(raw, fallback)
	raw[0] = '['
	if string(got) != `{"ok":true}` {
		t.Fatalf("normalizeJSON(raw) mutated copy = %s", string(got))
	}
}
