package relationship

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

type fakeRedactor struct {
	output []knowledge.Evidence
	err    error
	calls  int
}

func (f *fakeRedactor) RedactPII(_ context.Context, evidence []knowledge.Evidence) ([]knowledge.Evidence, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if f.output != nil {
		return f.output, nil
	}
	return evidence, nil
}

type fakeLifecycleRepo struct {
	staleMemories  []Memory
	staleSignals   []InteractionSignal
	updatedMems    []struct{ id, summary string }
	updatedSignals []struct{ id, summary string }
	erased         []struct {
		workspaceID string
		entityType  EntityType
		entityID    string
	}
	listMemErr        error
	listSignalErr     error
	updateMemoryErr   error
	updateSignalErr   error
	eraseArtifactsErr error
}

func (f *fakeLifecycleRepo) ListStaleMemories(_ context.Context, _ string, _ time.Time) ([]Memory, error) {
	return f.staleMemories, f.listMemErr
}

func (f *fakeLifecycleRepo) ListStaleSignals(_ context.Context, _ string, _ time.Time) ([]InteractionSignal, error) {
	return f.staleSignals, f.listSignalErr
}

func (f *fakeLifecycleRepo) UpdateMemorySummary(_ context.Context, memoryID, summary string) error {
	f.updatedMems = append(f.updatedMems, struct{ id, summary string }{id: memoryID, summary: summary})
	return f.updateMemoryErr
}

func (f *fakeLifecycleRepo) UpdateSignalSummary(_ context.Context, signalID, summary string) error {
	f.updatedSignals = append(f.updatedSignals, struct{ id, summary string }{id: signalID, summary: summary})
	return f.updateSignalErr
}

func (f *fakeLifecycleRepo) EraseEntityArtifacts(_ context.Context, workspaceID string, entityType EntityType, entityID string) error {
	f.erased = append(f.erased, struct {
		workspaceID string
		entityType  EntityType
		entityID    string
	}{workspaceID: workspaceID, entityType: entityType, entityID: entityID})
	return f.eraseArtifactsErr
}

func TestLifecycleService_DecayWorkspace_RedactsStaleSummaries(t *testing.T) {
	repo := &fakeLifecycleRepo{
		staleMemories: []Memory{{ID: "mem-1", Summary: "email john@example.com"}},
		staleSignals:  []InteractionSignal{{ID: "sig-1", Summary: "call +34 555 111 222"}},
	}
	memSummary := "[EMAIL_1]"
	signalSummary := "[PHONE_1]"
	redactor := &fakeRedactor{
		output: []knowledge.Evidence{{Snippet: &memSummary, PiiRedacted: true}},
	}
	svc := NewLifecycleService(repo, redactor)

	if err := svc.DecayWorkspace(context.Background(), "ws-1", time.Now().UTC()); err != nil {
		t.Fatalf("DecayWorkspace error: %v", err)
	}

	if len(repo.updatedMems) != 1 {
		t.Fatalf("expected 1 updated memory, got %d", len(repo.updatedMems))
	}
	if repo.updatedMems[0].summary != memSummary {
		t.Errorf("memory summary: want %q, got %q", memSummary, repo.updatedMems[0].summary)
	}

	redactor.output = []knowledge.Evidence{{Snippet: &signalSummary, PiiRedacted: true}}
	repo.updatedSignals = nil
	if err := svc.DecayWorkspace(context.Background(), "ws-1", time.Now().UTC()); err != nil {
		t.Fatalf("DecayWorkspace second run error: %v", err)
	}
	if len(repo.updatedSignals) != 1 {
		t.Fatalf("expected 1 updated signal, got %d", len(repo.updatedSignals))
	}
	if repo.updatedSignals[0].summary != signalSummary {
		t.Errorf("signal summary: want %q, got %q", signalSummary, repo.updatedSignals[0].summary)
	}
}

func TestLifecycleService_DecayWorkspace_SkipsRecentRecords(t *testing.T) {
	repo := &fakeLifecycleRepo{}
	redactor := &fakeRedactor{}
	svc := NewLifecycleService(repo, redactor)

	if err := svc.DecayWorkspace(context.Background(), "ws-1", time.Now().UTC()); err != nil {
		t.Fatalf("DecayWorkspace error: %v", err)
	}
	if len(repo.updatedMems) != 0 || len(repo.updatedSignals) != 0 {
		t.Fatalf("expected no updates, got memories=%d signals=%d", len(repo.updatedMems), len(repo.updatedSignals))
	}
}

func TestLifecycleService_EraseEntityMemory_RemovesDerivedArtifacts(t *testing.T) {
	repo := &fakeLifecycleRepo{}
	svc := NewLifecycleService(repo, &fakeRedactor{})

	if err := svc.EraseEntityMemory(context.Background(), "ws-1", EntityTypeContact, "con-1"); err != nil {
		t.Fatalf("EraseEntityMemory error: %v", err)
	}
	if len(repo.erased) != 1 {
		t.Fatalf("expected 1 erase call, got %d", len(repo.erased))
	}
	if repo.erased[0].entityType != EntityTypeContact || repo.erased[0].entityID != "con-1" {
		t.Fatalf("unexpected erase args: %#v", repo.erased[0])
	}
}

func TestLifecycleService_RepoErrorPropagates(t *testing.T) {
	repo := &fakeLifecycleRepo{eraseArtifactsErr: errors.New("db down")}
	svc := NewLifecycleService(repo, &fakeRedactor{})

	err := svc.EraseEntityMemory(context.Background(), "ws-1", EntityTypeContact, "con-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLifecycleService_RedactSignalSummary_UsesPolicyEngine(t *testing.T) {
	redacted := "[EMAIL_1]"
	redactor := &fakeRedactor{
		output: []knowledge.Evidence{{Snippet: &redacted, PiiRedacted: true}},
	}
	svc := NewLifecycleService(&fakeLifecycleRepo{}, redactor)

	summary, changed, err := svc.RedactSignalSummary(context.Background(), "contact jane@example.com")
	if err != nil {
		t.Fatalf("RedactSignalSummary error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if summary != redacted {
		t.Fatalf("summary: want %q, got %q", redacted, summary)
	}
	if redactor.calls != 1 {
		t.Fatalf("expected 1 redaction call, got %d", redactor.calls)
	}
}
