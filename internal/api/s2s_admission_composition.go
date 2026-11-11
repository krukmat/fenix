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
