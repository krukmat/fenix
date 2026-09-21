package tool

import (
	"context"
	"errors"
	"strings"
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
