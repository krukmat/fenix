package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

const (
	testCapabilityName      = "external.retry"
	testCapabilityOperation = "execute"
)

type sequenceCapabilityExecutor struct {
	errs         []error
	calls        int
	executionIDs []string
}

func (e *sequenceCapabilityExecutor) Execute(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
	e.calls++
	e.executionIDs = append(e.executionIDs, contextValue(ctx, ctxkeys.ExecutionID))
	if e.calls <= len(e.errs) && e.errs[e.calls-1] != nil {
		return nil, e.errs[e.calls-1]
	}
	return json.RawMessage(`{"ok":true}`), nil
}

func registerCapabilityForExecutionTest(
	t *testing.T,
	r *ToolRegistry,
	workspaceID string,
	descriptor CapabilityDescriptor,
	executor ToolExecutor,
) {
	t.Helper()
	if err := r.RegisterCapability(descriptor, executor); err != nil {
		t.Fatalf("RegisterCapability returned error: %v", err)
	}
	_, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID: workspaceID,
		Name:        descriptor.Name,
		InputSchema: json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"additionalProperties":false}`),
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}
}

func capabilityTestContext() context.Context {
	return context.WithValue(context.Background(), ctxkeys.TraceID, testCapabilityTraceID)
}

func TestCapabilityExecution_SafeRetryReusesExecutionID(t *testing.T) {
	db := openToolTestDB(t)
	workspaceID := createWorkspace(t, db)
	r := NewToolRegistry(db)
	executor := &sequenceCapabilityExecutor{
		errs: []error{NewRetryableCapabilityError(errors.New("temporary"))},
	}
	descriptor := CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       testCapabilityOperation,
		SideEffectClass: SideEffectRead,
		RetryPolicy:     CapabilityRetryPolicy{MaxAttempts: 3},
	}
	registerCapabilityForExecutionTest(t, r, workspaceID, descriptor, executor)

	if _, err := r.Execute(capabilityTestContext(), workspaceID, descriptor.Name, json.RawMessage(`{"value":"x"}`)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if executor.calls != 2 {
		t.Fatalf("calls = %d, want 2", executor.calls)
	}
	if executor.executionIDs[0] == "" || executor.executionIDs[0] != executor.executionIDs[1] {
		t.Fatalf("execution ids changed across retry: %#v", executor.executionIDs)
	}
}

func TestCapabilityExecution_MutationWithoutIdempotencyDoesNotRetry(t *testing.T) {
	db := openToolTestDB(t)
	workspaceID := createWorkspace(t, db)
	r := NewToolRegistry(db)
	r.SetCapabilityGovernor(&capabilityGovernorStub{})
	executor := &sequenceCapabilityExecutor{
		errs: []error{NewRetryableCapabilityError(errors.New("temporary"))},
	}
	descriptor := CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       testCapabilityOperation,
		SideEffectClass: SideEffectMutate,
		RetryPolicy:     CapabilityRetryPolicy{MaxAttempts: 3},
	}
	registerCapabilityForExecutionTest(t, r, workspaceID, descriptor, executor)

	_, err := r.Execute(capabilityTestContext(), workspaceID, descriptor.Name, json.RawMessage(`{"value":"x"}`))
	if !IsToolExecutionErrorCode(err, ToolErrorCapabilityFailed) {
		t.Fatalf("expected ToolErrorCapabilityFailed, got %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("calls = %d, want 1", executor.calls)
	}
}

func TestCapabilityExecution_IdempotentMutationCanRetry(t *testing.T) {
	db := openToolTestDB(t)
	workspaceID := createWorkspace(t, db)
	r := NewToolRegistry(db)
	r.SetCapabilityGovernor(&capabilityGovernorStub{})
	executor := &sequenceCapabilityExecutor{
		errs: []error{NewRetryableCapabilityError(errors.New("temporary"))},
	}
	descriptor := CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       testCapabilityOperation,
		SideEffectClass: SideEffectMutate,
		RetryPolicy:     CapabilityRetryPolicy{MaxAttempts: 3},
		IdempotencyMode: CapabilityIdempotencyExecutionID,
	}
	registerCapabilityForExecutionTest(t, r, workspaceID, descriptor, executor)

	if _, err := r.Execute(capabilityTestContext(), workspaceID, descriptor.Name, json.RawMessage(`{"value":"x"}`)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if executor.calls != 2 {
		t.Fatalf("calls = %d, want 2", executor.calls)
	}
	if executor.executionIDs[0] == "" || executor.executionIDs[0] != executor.executionIDs[1] {
		t.Fatalf("execution ids changed across retry: %#v", executor.executionIDs)
	}
}

func TestCapabilityExecution_IndeterminateNeverRetries(t *testing.T) {
	db := openToolTestDB(t)
	workspaceID := createWorkspace(t, db)
	r := NewToolRegistry(db)
	executor := &sequenceCapabilityExecutor{
		errs: []error{NewIndeterminateCapabilityError(errors.New("unknown provider outcome"))},
	}
	descriptor := CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       testCapabilityOperation,
		SideEffectClass: SideEffectRead,
		RetryPolicy:     CapabilityRetryPolicy{MaxAttempts: 3},
	}
	registerCapabilityForExecutionTest(t, r, workspaceID, descriptor, executor)

	_, err := r.Execute(capabilityTestContext(), workspaceID, descriptor.Name, json.RawMessage(`{"value":"x"}`))
	if !IsToolExecutionErrorCode(err, ToolErrorCapabilityIndeterminate) {
		t.Fatalf("expected ToolErrorCapabilityIndeterminate, got %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("calls = %d, want 1", executor.calls)
	}
}

func TestCapabilityExecution_AuditIncludesFinalStatusAndAttempts(t *testing.T) {
	db := openToolTestDB(t)
	workspaceID := createWorkspace(t, db)
	auditStub := &toolAuditStub{}
	r := NewToolRegistryWithRuntime(db, nil, auditStub)
	executor := &sequenceCapabilityExecutor{
		errs: []error{NewRetryableCapabilityError(errors.New("temporary"))},
	}
	descriptor := CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       testCapabilityOperation,
		SideEffectClass: SideEffectVerify,
		RetryPolicy:     CapabilityRetryPolicy{MaxAttempts: 2},
	}
	registerCapabilityForExecutionTest(t, r, workspaceID, descriptor, executor)

	if _, err := r.Execute(capabilityTestContext(), workspaceID, descriptor.Name, json.RawMessage(`{"value":"x"}`)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(auditStub.details) != 1 {
		t.Fatalf("audit details = %d, want 1", len(auditStub.details))
	}
	if auditStub.details[0]["capability_execution_status"] != string(CapabilityStatusSucceeded) {
		t.Fatalf("unexpected final status: %#v", auditStub.details[0])
	}
	if auditStub.details[0]["attempt_count"] != 2 {
		t.Fatalf("unexpected attempt count: %#v", auditStub.details[0])
	}
}
