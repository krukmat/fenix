package integration

import (
	"context"
	"crypto/sha256"
	"errors"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrCapabilityUnavailable   = errors.New("capability unavailable")
	ErrCapabilityConfiguration = errors.New("invalid capability registration")
	capabilityKeyPattern       = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+){1,6}$`)
	schemaVersionPattern       = regexp.MustCompile(`^[1-9][0-9]{0,2}$`)
)

// S2SCapabilityBinding is a logical, operator-provisioned capability, not a
// workflow or ToolRegistry executor. No handler is called by this package.
// Runtime tool permissions must be evaluated separately by Fenix when B6 is used.
type S2SCapabilityBinding struct {
	PrincipalID        string
	WorkspaceID        string
	CapabilityKey      string
	InputSchemaVersion string
	HandlerID          string
	SchemaSHA256       [sha256.Size]byte // nonzero only for operator-persisted schema
	Enabled            bool
}

// S2SCapabilityRegistry starts empty and fail-closed on every query. It is an
// isolated in-memory policy precondition; B2.B4 must decide durable, audited
// operator registration before Fenix activates any service endpoint.
type S2SCapabilityRegistry struct {
	mu       sync.RWMutex
	bindings map[string]S2SCapabilityBinding
	policies  *S2SCapabilityPolicyStore // source of managed registrations
	durable   bool // remains fail-closed even when the store is nil
}

func NewS2SCapabilityRegistry() *S2SCapabilityRegistry {
	return &S2SCapabilityRegistry{bindings: make(map[string]S2SCapabilityBinding)}
}

// NewS2SCapabilityRegistryFromPolicies resolves current enabled policies
// from SQLite on every admission/reservation, not from a stale cached copy.
// It neither registers a capability nor exposes an operator HTTP API.
func NewS2SCapabilityRegistryFromPolicies(store *S2SCapabilityPolicyStore) *S2SCapabilityRegistry {
	return &S2SCapabilityRegistry{policies: store, durable: true}
}

func capabilityTuple(principal, workspace, capability string) string {
	return principal + "\x00" + workspace + "\x00" + capability
}

func validScopePart(s string) bool {
	return s != "" && len(s) <= 128 && !strings.ContainsAny(s, "\x00\r\n\t ")
}

func validCapabilityBinding(b S2SCapabilityBinding) bool {
	return validScopePart(b.PrincipalID) && validScopePart(b.WorkspaceID) &&
		len(b.CapabilityKey) <= 96 && capabilityKeyPattern.MatchString(b.CapabilityKey) &&
		schemaVersionPattern.MatchString(b.InputSchemaVersion) && validCapabilityHandler(b)
}

func validCapabilityHandler(b S2SCapabilityBinding) bool {
	return (!b.Enabled || validScopePart(b.HandlerID)) &&
		(b.HandlerID == "" || validScopePart(b.HandlerID))
}

func availableCapabilityBinding(b S2SCapabilityBinding, schemaVersion string) bool {
	return b.Enabled && b.HandlerID != "" && b.InputSchemaVersion == schemaVersion
}

// Register is ONLY a future operator API, not an HTTP self-registration path.
// It never permits an enabled capability without a nonempty approved handler.
func (r *S2SCapabilityRegistry) Register(b S2SCapabilityBinding) error {
	if r == nil || r.durable || !validCapabilityBinding(b) {
		return ErrCapabilityConfiguration
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bindings == nil {
		r.bindings = make(map[string]S2SCapabilityBinding)
	}
	r.bindings[capabilityTuple(b.PrincipalID, b.WorkspaceID, b.CapabilityKey)] = b
	return nil
}

// Resolve checks exact principal/workspace/key/version with a single generic
// error for missing, disabled and mismatched entries to prevent enumeration.
func (r *S2SCapabilityRegistry) Resolve(principal, workspace, key, schemaVersion string) (S2SCapabilityBinding, error) {
	return r.ResolveContext(context.Background(), principal, workspace, key, schemaVersion)
}

// ResolveContext is the policy admission boundary used by B2.E and C2.
// Durable registries re-read enabled policy AND the active service principal.
func (r *S2SCapabilityRegistry) ResolveContext(
	ctx context.Context, principal, workspace, key, schemaVersion string,
) (S2SCapabilityBinding, error) {
	if r == nil || !validScopePart(principal) || !validScopePart(workspace) {
		return S2SCapabilityBinding{}, ErrCapabilityUnavailable
	}
	if r.durable {
		if r.policies == nil {
			return S2SCapabilityBinding{}, ErrCapabilityUnavailable
		}
		binding, _, err := r.policies.ResolveEnabled(ctx, principal, workspace, key, schemaVersion)
		if err != nil {
			return S2SCapabilityBinding{}, ErrCapabilityUnavailable
		}
		return binding, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.bindings[capabilityTuple(principal, workspace, key)]
	if !ok || !availableCapabilityBinding(b, schemaVersion) {
		return S2SCapabilityBinding{}, ErrCapabilityUnavailable
	}
	return b, nil
}

// Disable is a protective operator action and never starts an agent.
func (r *S2SCapabilityRegistry) Disable(principal, workspace, key string) error {
	if r == nil || r.durable {
		return ErrCapabilityUnavailable
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	tuple := capabilityTuple(principal, workspace, key)
	b, ok := r.bindings[tuple]
	if !ok {
		return ErrCapabilityUnavailable
	}
	b.Enabled = false
	r.bindings[tuple] = b
	return nil
}
