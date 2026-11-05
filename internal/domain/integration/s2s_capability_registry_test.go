package integration

import (
	"errors"
	"sync"
	"testing"
)

func enabledBinding() S2SCapabilityBinding {
	return S2SCapabilityBinding{PrincipalID: "principal-1", WorkspaceID: "workspace-1", CapabilityKey: "integration.diagnostics", InputSchemaVersion: "1", HandlerID: "diagnostic-handler", Enabled: true}
}
func TestS2SRegistryDefaultDenyAndScopedLookup(t *testing.T) {
	r := NewS2SCapabilityRegistry()
	b := enabledBinding()
	for _, pair := range []struct{ p, w, k, v string }{
		{b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "1"},
		{"other", b.WorkspaceID, b.CapabilityKey, "1"},
	} {
		if _, err := r.Resolve(pair.p, pair.w, pair.k, pair.v); !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatal("empty registry allowed access")
		}
	}
	b.Enabled = false
	if err := r.Register(b); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "1"); !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatal("disabled cap allowed")
	}
	b.Enabled = true
	if err := r.Register(b); err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct{ p, w, k, v string }{
		{"other", b.WorkspaceID, b.CapabilityKey, "1"},
		{b.PrincipalID, "other", b.CapabilityKey, "1"},
		{b.PrincipalID, b.WorkspaceID, "other.capability", "1"},
		{b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "2"},
	} {
		if _, err := r.Resolve(pair.p, pair.w, pair.k, pair.v); !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("scope escaped: %+v", pair)
		}
	}
	got, err := r.Resolve(b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "1")
	if err != nil || got.HandlerID != b.HandlerID {
		t.Fatalf("expected exact authorized entry: %+v %v", got, err)
	}
	got.HandlerID = "tampered"
	fresh, err := r.Resolve(b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "1")
	if err != nil || fresh.HandlerID != b.HandlerID {
		t.Fatal("returned binding alias mutated registry")
	}
	if err := r.Disable(b.PrincipalID, b.WorkspaceID, b.CapabilityKey); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "1"); !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatal("disabled entry still readable")
	}
}
func TestS2SRegistryRejectInvalidRegistrations(t *testing.T) {
	r := NewS2SCapabilityRegistry()
	base := enabledBinding()
	cases := []func(*S2SCapabilityBinding){
		func(b *S2SCapabilityBinding) { b.PrincipalID = "" },
		func(b *S2SCapabilityBinding) { b.WorkspaceID = "" },
		func(b *S2SCapabilityBinding) { b.WorkspaceID = "a\x00b" },
		func(b *S2SCapabilityBinding) { b.CapabilityKey = "bad key" },
		func(b *S2SCapabilityBinding) { b.CapabilityKey = "a" },
		func(b *S2SCapabilityBinding) { b.InputSchemaVersion = "0" },
		func(b *S2SCapabilityBinding) { b.InputSchemaVersion = "1.0" },
		func(b *S2SCapabilityBinding) { b.HandlerID = "" },
	}
	for i, mutate := range cases {
		b := base
		mutate(&b)
		if err := r.Register(b); !errors.Is(err, ErrCapabilityConfiguration) {
			t.Fatalf("invalid config %d accepted: %v", i, err)
		}
	}
	if err := r.Disable("nobody", "none", "bad.capability"); !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatal("nonexistent accepted")
	}
	var nilRegistry *S2SCapabilityRegistry
	if _, err := nilRegistry.Resolve(base.PrincipalID, base.WorkspaceID, base.CapabilityKey, "1"); !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatal("nil registry allowed")
	}
}
func TestS2SRegistryConcurrentReadsAndDisable(t *testing.T) {
	r := NewS2SCapabilityRegistry()
	b := enabledBinding()
	if err := r.Register(b); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, err := r.Resolve(b.PrincipalID, b.WorkspaceID, b.CapabilityKey, "1")
				if err != nil && !errors.Is(err, ErrCapabilityUnavailable) {
					t.Error(err)
				}
			}
		}()
	}
	if err := r.Disable(b.PrincipalID, b.WorkspaceID, b.CapabilityKey); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
