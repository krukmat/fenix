package uuid

import (
	"regexp"
	"testing"
)

func TestNewV7_SetsVersionAndVariant(t *testing.T) {
	t.Parallel()

	u := NewV7()

	// Version nibble in byte 6 must be 0b0111 (v7)
	if (u[6]>>4)&0x0f != 0x07 {
		t.Fatalf("expected version 7 nibble, got %x", (u[6]>>4)&0x0f)
	}

	// Variant in byte 7 must be RFC4122 (10xxxxxx)
	if (u[7] & 0xc0) != 0x80 {
		t.Fatalf("expected RFC4122 variant bits 10xxxxxx, got %08b", u[7])
	}
}

func TestUUID_String_Format(t *testing.T) {
	t.Parallel()

	u := NewV7()
	s := u.String()

	if len(s) != 36 {
		t.Fatalf("expected UUID string len=36, got %d (%q)", len(s), s)
	}

	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !re.MatchString(s) {
		t.Fatalf("expected canonical uuid format, got %q", s)
	}
}
