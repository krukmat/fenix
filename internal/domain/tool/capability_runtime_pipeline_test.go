package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type runtimePlannerStub struct {
	decision CapabilityRuntimeDecision
	err      error
	calls    int
	facts    []CapabilityGovernanceFacts
}

func (s *runtimePlannerStub) PlanCapability(
	_ context.Context,
	_ CapabilityDescriptor,
	facts CapabilityGovernanceFacts,
) (CapabilityRuntimeDecision, error) {
	s.calls++
	s.facts = append(s.facts, facts)
	return s.decision, s.err
}

type runtimeEvidenceStub struct {
	result   CapabilityEvidenceResult
	err      error
	calls    int
	requests []CapabilityEvidenceRequest
}

func (s *runtimeEvidenceStub) RecordCapabilityEvidence(
	_ context.Context,
	request CapabilityEvidenceRequest,
) (CapabilityEvidenceResult, error) {
	s.calls++
	s.requests = append(s.requests, request)
	return s.result, s.err
}

func allowedRuntimeDecision(planned bool) CapabilityRuntimeDecision {
	requirement := RuntimeEvidenceNone
	if planned {
		requirement = RuntimeEvidenceOptional
	}
	return CapabilityRuntimeDecision{
		Allowed:             true,
		EvidenceRequirement: requirement,
		EvidencePlanned:     planned,
		EvidenceReason:      "runtime test",
		PolicyReference:     "policy:test",
	}
}

func newRuntimeTestRegistry(
	t *testing.T,
	descriptor CapabilityDescriptor,
	executor ToolExecutor,
	planner CapabilityGovernancePlanner,
	recorder CapabilityEvidenceRecorder,
) (*ToolRegistry, string, *toolAuditStub) {
	t.Helper()
	db := openToolTestDB(t)
	workspaceID := createWorkspace(t, db)
	auditStub := &toolAuditStub{}
	registry := NewToolRegistryWithRuntime(db, nil, auditStub)
	registry.SetCapabilityGovernancePlanner(planner)
	registry.SetCapabilityEvidenceRecorder(recorder)
	registerCapabilityForExecutionTest(t, registry, workspaceID, descriptor, executor)
	return registry, workspaceID, auditStub
}

func runtimeDescriptor(class SideEffectClass) CapabilityDescriptor {
	return CapabilityDescriptor{
		Name:            "runtime.capability",
		Version:         "1",
		Operation:       "execute",
		SideEffectClass: class,
	}
}

func TestRuntimeGovernance_ProviderCanExecuteWithoutEvidence(t *testing.T) {
	executor := &sequenceCapabilityExecutor{}
	planner := &runtimePlannerStub{decision: allowedRuntimeDecision(false)}
	recorder := &runtimeEvidenceStub{}
	registry, workspaceID, auditStub := newRuntimeTestRegistry(
		t, runtimeDescriptor(SideEffectVerify), executor, planner, recorder,
	)

	_, err := registry.Execute(
		capabilityTestContext(), workspaceID, "runtime.capability",
		json.RawMessage(`{"value":"x"}`),
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if planner.calls != 1 || executor.calls != 1 || recorder.calls != 0 {
		t.Fatalf("calls planner=%d executor=%d recorder=%d", planner.calls, executor.calls, recorder.calls)
	}
	evidenceMeta := auditStub.details[0]["evidence"].(map[string]any)
	if evidenceMeta["state"] != evidenceStateNotPlanned {
		t.Fatalf("unexpected evidence metadata: %#v", evidenceMeta)
	}
}

func TestRuntimeGovernance_ProviderAndEvidenceExecuteIndependently(t *testing.T) {
	executor := &sequenceCapabilityExecutor{}
	planner := &runtimePlannerStub{decision: allowedRuntimeDecision(true)}
	recorder := &runtimeEvidenceStub{
		result: CapabilityEvidenceResult{
			State: "verified",
			AuditMetadata: map[string]any{
				"event_id": "event-1",
			},
		},
	}
	registry, workspaceID, auditStub := newRuntimeTestRegistry(
		t, runtimeDescriptor(SideEffectTransform), executor, planner, recorder,
	)

	_, err := registry.Execute(
		capabilityTestContext(), workspaceID, "runtime.capability",
		json.RawMessage(`{"value":"x"}`),
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if executor.calls != 1 || recorder.calls != 1 {
		t.Fatalf("calls executor=%d recorder=%d", executor.calls, recorder.calls)
	}
	if recorder.requests[0].Status != CapabilityStatusSucceeded {
		t.Fatalf("evidence status = %q", recorder.requests[0].Status)
	}
	evidenceMeta := auditStub.details[0]["evidence"].(map[string]any)
	if evidenceMeta["state"] != "verified" || evidenceMeta["event_id"] != "event-1" {
		t.Fatalf("unexpected evidence metadata: %#v", evidenceMeta)
	}
}

func TestRuntimeGovernance_DenialCanBeEvidencedWithoutProviderInvocation(t *testing.T) {
	executor := &sequenceCapabilityExecutor{}
	planner := &runtimePlannerStub{
		decision: CapabilityRuntimeDecision{
			Allowed:             false,
			DenialReason:        "policy_denied",
			EvidenceRequirement: RuntimeEvidenceRequired,
			EvidencePlanned:     true,
			EvidenceReason:      "record governance denial",
			PolicyReference:     "policy:test",
		},
	}
	recorder := &runtimeEvidenceStub{
		result: CapabilityEvidenceResult{State: "recorded"},
	}
	registry, workspaceID, _ := newRuntimeTestRegistry(
		t, runtimeDescriptor(SideEffectVerify), executor, planner, recorder,
	)

	_, err := registry.Execute(
		capabilityTestContext(), workspaceID, "runtime.capability",
		json.RawMessage(`{"value":"x"}`),
	)
	if !IsToolExecutionErrorCode(err, ToolErrorGovernanceDenied) {
		t.Fatalf("expected governance denial, got %v", err)
	}
	if executor.calls != 0 || recorder.calls != 1 {
		t.Fatalf("calls executor=%d recorder=%d", executor.calls, recorder.calls)
	}
	if recorder.requests[0].Status != CapabilityStatusDenied {
		t.Fatalf("evidence status = %q, want denied", recorder.requests[0].Status)
	}
}

func TestRuntimeGovernance_ApprovalFailureCannotBeOverriddenByPlanner(t *testing.T) {
	executor := &sequenceCapabilityExecutor{}
	decision := allowedRuntimeDecision(false)
	decision.ApprovalRequired = true
	planner := &runtimePlannerStub{decision: decision}
	registry, workspaceID, _ := newRuntimeTestRegistry(
		t, CapabilityDescriptor{
			Name:            "runtime.capability",
			Version:         "1",
			Operation:       "execute",
			SideEffectClass: SideEffectIrreversible,
		},
		executor,
		planner,
		nil,
	)

	_, err := registry.Execute(
		capabilityTestContext(), workspaceID, "runtime.capability",
		json.RawMessage(`{"value":"x"}`),
	)
	if !IsToolExecutionErrorCode(err, ToolErrorGovernanceDenied) {
		t.Fatalf("expected governance denial, got %v", err)
	}
	if executor.calls != 0 {
		t.Fatalf("provider invoked %d times", executor.calls)
	}
	if len(planner.facts) != 1 || planner.facts[0].GovernorPassed {
		t.Fatalf("unexpected governance facts: %#v", planner.facts)
	}
}

func TestRuntimeGovernance_EvidenceFailureDoesNotRepeatProvider(t *testing.T) {
	executor := &sequenceCapabilityExecutor{}
	planner := &runtimePlannerStub{decision: allowedRuntimeDecision(true)}
	recorder := &runtimeEvidenceStub{err: errors.New("evidence timeout")}
	registry, workspaceID, _ := newRuntimeTestRegistry(
		t, runtimeDescriptor(SideEffectTransform), executor, planner, recorder,
	)

	output, err := registry.Execute(
		capabilityTestContext(), workspaceID, "runtime.capability",
		json.RawMessage(`{"value":"x"}`),
	)
	if !IsToolExecutionErrorCode(err, ToolErrorEvidenceIndeterminate) {
		t.Fatalf("expected evidence indeterminate, got %v", err)
	}
	if len(output) == 0 {
		t.Fatal("provider output must be preserved after evidence failure")
	}
	if executor.calls != 1 || recorder.calls != 1 {
		t.Fatalf("calls executor=%d recorder=%d", executor.calls, recorder.calls)
	}
}

func TestRuntimeGovernance_VerificationFailureIsExplicit(t *testing.T) {
	executor := &sequenceCapabilityExecutor{}
	planner := &runtimePlannerStub{decision: allowedRuntimeDecision(true)}
	recorder := &runtimeEvidenceStub{
		result: CapabilityEvidenceResult{State: "verification_failed"},
	}
	registry, workspaceID, _ := newRuntimeTestRegistry(
		t, runtimeDescriptor(SideEffectVerify), executor, planner, recorder,
	)

	output, err := registry.Execute(
		capabilityTestContext(), workspaceID, "runtime.capability",
		json.RawMessage(`{"value":"x"}`),
	)
	if !IsToolExecutionErrorCode(err, ToolErrorEvidenceVerificationFailed) {
		t.Fatalf("expected verification failure, got %v", err)
	}
	if len(output) == 0 || executor.calls != 1 {
		t.Fatalf("provider result lost or repeated: output=%q calls=%d", output, executor.calls)
	}
}
