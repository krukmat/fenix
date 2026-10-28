package dubbridge

import (
	"errors"
	"testing"
)

func TestConfigEnforcesTargetAndTLS(t *testing.T) {
	const org = "550e8400-e29b-41d4-a716-446655440000"
	const proj = "550e8400-e29b-41d4-a716-446655440001"
	for _, base := range []string{"https://dubbridge.example", "http://localhost:7000", "http://127.0.0.1:7000/api"} {
		c, err := NewClient(Config{BaseURL: base, BearerToken: "opaque-test-secret", AllowedOrgID: org, AllowedProjectID: proj})
		if err != nil || c == nil {
			t.Fatalf("expected legal config for %s: %v", base, err)
		}
	}
	for _, base := range []string{"http://dubbridge.example", "http://example.com:8000", "https://example.com@evil.example", "https://example.com/foo", "file:///etc/passwd"} {
		if _, err := NewClient(Config{BaseURL: base, BearerToken: "opaque-test-secret", AllowedOrgID: org, AllowedProjectID: proj}); !errors.Is(err, ErrConfiguration) {
			t.Fatalf("expected rejection for %s", base)
		}
	}
}
func TestUUIDRejectsPathManipulation(t *testing.T) {
	for _, v := range []string{"../../orgs", "550e8400-e29b-41d4-a716-446655440000/../", "", "550e8400e29b41d4a716446655440000"} {
		if ValidUUID(v) {
			t.Errorf("accepted %q", v)
		}
	}
	if !ValidUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("rejected valid ID")
	}
}

func TestTokenInjectionFailsClosed(t *testing.T) {
	const org = "550e8400-e29b-41d4-a716-446655440000"
	const project = "550e8400-e29b-41d4-a716-446655440001"
	for _, token := range []string{"", "abc def", "abc\nadmin", " leading", "trailing ", "abc\x00def"} {
		if _, err := NewClient(Config{BaseURL: "http://localhost:8080", BearerToken: token, AllowedOrgID: org, AllowedProjectID: project}); !errors.Is(err, ErrConfiguration) {
			t.Errorf("unsafe bearer accepted")
		}
	}
}
