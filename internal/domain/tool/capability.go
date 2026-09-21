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
)

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
	Name            string
	Version         string
	Operation       string
	SideEffectClass SideEffectClass
}

// CapabilityGovernor owns additional execution gates that are not covered by
// normal tool permission/policy enforcement, such as approval requirements for
// mutating or irreversible external capabilities.
type CapabilityGovernor interface {
	CheckCapabilityExecution(ctx context.Context, descriptor CapabilityDescriptor) error
}

type IntegrationActor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type IntegrationCapabilityRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

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
	if strings.TrimSpace(d.Name) == "" ||
		strings.TrimSpace(d.Version) == "" ||
		strings.TrimSpace(d.Operation) == "" ||
		!isValidSideEffectClass(d.SideEffectClass) {
		return ErrToolDefinitionInvalid
	}
	return nil
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
