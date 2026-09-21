package tool

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrCapabilityContextMissing     = errors.New("external capability execution context is incomplete")
	ErrCapabilityGovernanceRequired = errors.New("external capability requires governance")
	ErrCapabilityGovernanceDenied   = errors.New("external capability denied by governance decision")
	ErrCapabilityEvidenceUnavailable = errors.New("planned capability evidence recorder is unavailable")
)

// SideEffectClass classifies the operational risk of a governed capability.
type SideEffectClass string

const (
	SideEffectRead         SideEffectClass = "read"
	SideEffectTransform    SideEffectClass = "transform"
	SideEffectVerify       SideEffectClass = "verify"
	SideEffectMutate       SideEffectClass = "mutate"
	SideEffectIrreversible SideEffectClass = "irreversible"
)

// CapabilityDescriptor describes the provider-neutral semantic contract for an
// externally-backed governed tool. Transport details intentionally stay out of
// this descriptor.
type CapabilityDescriptor struct {
	Name             string
	Version          string
	Operation        string
	SideEffectClass  SideEffectClass
	ApprovalRequired bool
	RetryPolicy      CapabilityRetryPolicy
	IdempotencyMode  CapabilityIdempotencyMode
}

// CapabilityGovernor owns additional execution gates that are not covered by
// normal tool permission/policy enforcement, such as approval requirements for
// mutating or irreversible external capabilities.
type CapabilityGovernor interface {
	CheckCapabilityExecution(ctx context.Context, descriptor CapabilityDescriptor) error
}

// CapabilityExecutionStatus is the observable final state of one logical capability execution.
type CapabilityExecutionStatus string

const (
	CapabilityStatusReady         CapabilityExecutionStatus = "ready"
	CapabilityStatusExecuting     CapabilityExecutionStatus = "executing"
	CapabilityStatusSucceeded     CapabilityExecutionStatus = "succeeded"
	CapabilityStatusDenied        CapabilityExecutionStatus = "denied"
	CapabilityStatusFailed        CapabilityExecutionStatus = "failed"
	CapabilityStatusIndeterminate CapabilityExecutionStatus = "indeterminate"
)

// CapabilityRetryPolicy controls bounded automatic retries for transient provider errors.
type CapabilityRetryPolicy struct {
	MaxAttempts int
}

// CapabilityIdempotencyMode states how a provider prevents duplicate logical mutations.
type CapabilityIdempotencyMode string

const (
	CapabilityIdempotencyNone        CapabilityIdempotencyMode = ""
	CapabilityIdempotencyExecutionID CapabilityIdempotencyMode = "execution_id"
)

// CapabilityRetryDisposition classifies provider failures without coupling Fenix to transport details.
type CapabilityRetryDisposition string

const (
	CapabilityRetryNone          CapabilityRetryDisposition = ""
	CapabilityRetryable          CapabilityRetryDisposition = "retryable"
	CapabilityRetryIndeterminate CapabilityRetryDisposition = "indeterminate"
)

// CapabilityExecutionError communicates retry semantics from a provider executor to the governed runtime.
type CapabilityExecutionError struct {
	Disposition CapabilityRetryDisposition
	Err         error
}

func (e *CapabilityExecutionError) Error() string {
	if e == nil || e.Err == nil {
		return "capability execution failed"
	}
	return e.Err.Error()
}

func (e *CapabilityExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewRetryableCapabilityError marks a provider failure as safe to retry subject to the capability retry contract.
func NewRetryableCapabilityError(err error) error {
	return &CapabilityExecutionError{Disposition: CapabilityRetryable, Err: err}
}

// NewIndeterminateCapabilityError marks a provider result as uncertain and therefore never automatically retried.
func NewIndeterminateCapabilityError(err error) error {
	return &CapabilityExecutionError{Disposition: CapabilityRetryIndeterminate, Err: err}
}

// CapabilityApprovalResourceType binds approval records to one logical capability execution.
const CapabilityApprovalResourceType = "capability_execution"

// CapabilityApprovalAction returns the canonical approval action key for a capability operation.
func CapabilityApprovalAction(descriptor CapabilityDescriptor) string {
	return "capability:" + descriptor.Name + ":" + descriptor.Operation
}

// RequiresApproval reports whether the capability must present an approved request before execution.
func (d CapabilityDescriptor) RequiresApproval() bool {
	return d.SideEffectClass == SideEffectIrreversible || d.ApprovalRequired
}

func (d CapabilityDescriptor) maxAttempts() int {
	if d.RetryPolicy.MaxAttempts <= 0 {
		return 1
	}
	return d.RetryPolicy.MaxAttempts
}

func (d CapabilityDescriptor) canRetryAutomatically() bool {
	switch d.SideEffectClass {
	case SideEffectRead, SideEffectTransform, SideEffectVerify:
		return true
	case SideEffectMutate, SideEffectIrreversible:
		return d.IdempotencyMode == CapabilityIdempotencyExecutionID
	default:
		return false
	}
}

func capabilityRetryDisposition(err error) CapabilityRetryDisposition {
	var capabilityErr *CapabilityExecutionError
	if errors.As(err, &capabilityErr) {
		return capabilityErr.Disposition
	}
	return CapabilityRetryNone
}

// IntegrationActor identifies the principal behind a governed execution.
type IntegrationActor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// IntegrationCapabilityRef identifies the versioned semantic capability contract.
type IntegrationCapabilityRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// IntegrationExecutionContext is the provider-neutral execution identity propagated across external boundaries.
type IntegrationExecutionContext struct {
	SchemaVersion string                   `json:"schema_version"`
	WorkspaceID   string                   `json:"workspace_id"`
	TraceID       string                   `json:"trace_id"`
	ExecutionID   string                   `json:"execution_id"`
	RunID         string                   `json:"run_id,omitempty"`
	WorkflowID    string                   `json:"workflow_id,omitempty"`
	Actor         IntegrationActor         `json:"actor"`
	ApprovalID    string                   `json:"approval_id,omitempty"`
	Capability    IntegrationCapabilityRef `json:"capability"`
	StartedAt     time.Time                `json:"started_at"`
}

func prepareCapabilityExecutionContext(ctx context.Context, workspaceID string) context.Context {
	if strings.TrimSpace(contextValue(ctx, ctxkeys.WorkspaceID)) == "" {
		ctx = ctxkeys.WithValue(ctx, ctxkeys.WorkspaceID, workspaceID)
	}
	if strings.TrimSpace(contextValue(ctx, ctxkeys.ExecutionID)) == "" {
		ctx = ctxkeys.WithValue(ctx, ctxkeys.ExecutionID, uuid.NewV7().String())
	}
	return ctx
}

func contextValue(ctx context.Context, key ctxkeys.Key) string {
	value, _ := ctx.Value(key).(string)
	return strings.TrimSpace(value)
}

func (d CapabilityDescriptor) validate() error {
	if !d.hasValidIdentity() ||
		!isValidRetryPolicy(d.RetryPolicy) ||
		!isValidIdempotencyMode(d.IdempotencyMode) {
		return ErrToolDefinitionInvalid
	}
	return nil
}

func (d CapabilityDescriptor) hasValidIdentity() bool {
	return strings.TrimSpace(d.Name) != "" &&
		strings.TrimSpace(d.Version) != "" &&
		strings.TrimSpace(d.Operation) != "" &&
		isValidSideEffectClass(d.SideEffectClass)
}

func isValidRetryPolicy(policy CapabilityRetryPolicy) bool {
	return policy.MaxAttempts >= 0 && policy.MaxAttempts <= 5
}

func isValidIdempotencyMode(mode CapabilityIdempotencyMode) bool {
	return mode == CapabilityIdempotencyNone || mode == CapabilityIdempotencyExecutionID
}

func isValidSideEffectClass(class SideEffectClass) bool {
	switch class {
	case SideEffectRead, SideEffectTransform, SideEffectVerify, SideEffectMutate, SideEffectIrreversible:
		return true
	default:
		return false
	}
}

func requiresCapabilityGovernor(class SideEffectClass) bool {
	return class == SideEffectMutate || class == SideEffectIrreversible
}
