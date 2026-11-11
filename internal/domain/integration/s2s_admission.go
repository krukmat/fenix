package integration

import (
	"context"
	"errors"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

// S2SCredentialVerifier is implemented by Fenix's persisted credential store.
// The caller must never inject an identity assembled from DubBridge JSON.
type S2SCredentialVerifier interface {
	AuthenticateServiceToken(context.Context, string, time.Time) (*domainauth.ServiceIdentity, error)
}

// S2SInputValidator checks a versioned capability-specific input schema.
// It must be explicitly supplied by the trusted operator composition layer;
// nil validators deny admission, even for an otherwise enabled capability.
// No validator is installed or registered by this implementation.
type S2SInputValidator interface {
	ValidateS2SInput(context.Context, S2SCapabilityBinding, map[string]any) error
}

// S2SAdmission is an inert in-process preflight and durable reservation
// adapter. It does not mount a route, return agent status, or run a handler.
type S2SAdmission struct {
	Credentials S2SCredentialVerifier
	Registry    *S2SCapabilityRegistry
	InputSchema S2SInputValidator
	Operations  *ServiceOperationStore
}

// S2SAdmissionReceipt deliberately excludes principal, scope, hashes,
// run IDs, internal worker leases, result data and operation status.
type S2SAdmissionReceipt struct {
	OperationID string
	TraceID     string
	Replayed    bool
}

// Admit validates a Fenix-issued bearer first, then the v1 envelope, then
// an enabled policy binding and its input schema, and only then reserves a
// durable operation. Replays are inert and do not dispatch a provider.
func (a *S2SAdmission) Admit(
	ctx context.Context, bearer string, raw []byte, now time.Time,
) (S2SAdmissionReceipt, error) {
	return a.admit(ctx, bearer, raw, "", false, now)
}

// AdmitWithIdempotencyKey is the HTTP-facing variant of Admit. The header key
// must match the validated B1 operation_id BEFORE any SQLite reservation.
// The existing in-process Admit method remains available for non-HTTP callers.
func (a *S2SAdmission) AdmitWithIdempotencyKey(
	ctx context.Context, bearer string, raw []byte, operationID string, now time.Time,
) (S2SAdmissionReceipt, error) {
	return a.admit(ctx, bearer, raw, operationID, true, now)
}

func (a *S2SAdmission) admit(
	ctx context.Context, bearer string, raw []byte, requiredOperationID string,
	requireOperationID bool, now time.Time,
) (S2SAdmissionReceipt, error) {
	if !a.configured() {
		return S2SAdmissionReceipt{}, ErrS2SAdmissionDenied
	}
	if now.Unix() <= 0 {
		return S2SAdmissionReceipt{}, ErrS2SAdmissionInvalid
	}
	principal, err := a.Credentials.AuthenticateServiceToken(ctx, bearer, now)
	if err != nil {
		return S2SAdmissionReceipt{}, classifyS2SIdentityError(err)
	}
	if requireOperationID && !validS2SOperationID(requiredOperationID) {
		return S2SAdmissionReceipt{}, ErrS2SAdmissionInvalid
	}
	return a.admitAuthenticated(ctx, principal, raw, requiredOperationID, now)
}

func (a *S2SAdmission) configured() bool {
	return a != nil && a.Credentials != nil && a.Registry != nil &&
		a.InputSchema != nil && a.Operations != nil
}

func (a *S2SAdmission) admitAuthenticated(
	ctx context.Context, principal *domainauth.ServiceIdentity, raw []byte,
	requiredOperationID string, now time.Time,
) (S2SAdmissionReceipt, error) {
	envelope, err := parseVerifiedS2SEnvelope(raw, principal)
	if err != nil {
		return S2SAdmissionReceipt{}, err
	}
	if requiredOperationID != "" && envelope.request.OperationID != requiredOperationID {
		return S2SAdmissionReceipt{}, ErrS2SAdmissionInvalid
	}
	binding, err := a.Registry.ResolveContext(ctx, envelope.request.Scope.PrincipalID,
		envelope.request.Scope.WorkspaceID, envelope.request.CapabilityKey,
		envelope.request.SchemaVersion)
	if err != nil {
		return S2SAdmissionReceipt{}, ErrS2SAdmissionDenied
	}
	if err := a.InputSchema.ValidateS2SInput(ctx, binding, envelope.input); err != nil {
		return S2SAdmissionReceipt{}, ErrS2SAdmissionDenied
	}
	// A subsequent operator disable/version change must not silently swap
	// policy between input validation and C2 atomic reservation.
	if binding.SchemaSHA256 != [32]byte{} {
		envelope.request.ExpectedConfigSHA256 = serviceCapabilityFingerprint(binding)
		envelope.request.ManagedSchemaSHA256 = binding.SchemaSHA256
		envelope.request.ManagedHandlerID = binding.HandlerID
	}
	return a.reserve(ctx, envelope.request, now)
}

func classifyS2SIdentityError(err error) error {
	if errors.Is(err, domainauth.ErrServiceIdentityUnavailable) {
		return ErrS2SAdmissionUnavailable
	}
	return ErrS2SAdmissionDenied
}

func (a *S2SAdmission) reserve(
	ctx context.Context, request ServiceOperationRequest, now time.Time,
) (S2SAdmissionReceipt, error) {
	record, replayed, err := a.Operations.Reserve(ctx, request, a.Registry, now)
	if err != nil {
		return S2SAdmissionReceipt{}, classifyS2SReservationError(err)
	}
	return S2SAdmissionReceipt{
		OperationID: record.OperationID,
		TraceID:     record.TraceID,
		Replayed:    replayed,
	}, nil
}

func classifyS2SReservationError(err error) error {
	switch {
	case errors.Is(err, ErrServiceOperationInvalid):
		return ErrS2SAdmissionInvalid
	case errors.Is(err, ErrServiceOperationConflict):
		return ErrS2SAdmissionConflict
	case errors.Is(err, ErrServiceOperationDenied):
		return ErrS2SAdmissionDenied
	default:
		return ErrS2SAdmissionUnavailable
	}
}
