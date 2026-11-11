package api

import (
	"database/sql"
	"net/http"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

// NewDormantS2SAdmissionFromStore composes the real Fenix principal verifier,
// registry, capability-specific input validator and durable operation store.
// It creates a detached handler ONLY; nothing is attached to NewRouter.
//
// The registry is intentionally operator-provided and empty by default.
// A missing DB, registry or validator fails closed. No S2S token, capability,
// live route, handler registration, worker or workflow is provisioned.
func NewDormantS2SAdmissionFromStore(
	db *sql.DB, registry *integration.S2SCapabilityRegistry,
	inputSchema integration.S2SInputValidator,
) http.Handler {
	if db == nil || registry == nil || inputSchema == nil {
		return NewDormantS2SAdmissionHandler(nil)
	}
	return NewDormantS2SAdmissionHandler(&integration.S2SAdmission{
		Credentials: domainauth.NewServiceCredentialStore(db),
		Registry: registry,
		InputSchema: inputSchema,
		Operations: integration.NewServiceOperationStore(db),
	})
}

// NewDormantS2SAdmissionFromPolicies is the preferred B2.B4 composition.
// The policy store and input schema are both read through SQLite, rather
// than relying on an unrelated in-memory registry or a permissive validator.
// An empty policy table denies everything. The handler is never mounted.
func NewDormantS2SAdmissionFromPolicies(db *sql.DB) http.Handler {
	if db == nil {
		return NewDormantS2SAdmissionHandler(nil)
	}
	policies := integration.NewS2SCapabilityPolicyStore(db)
	return NewDormantS2SAdmissionFromStore(
		db,
		integration.NewS2SCapabilityRegistryFromPolicies(policies),
		integration.NewPersistedS2SInputValidator(policies),
	)
}
