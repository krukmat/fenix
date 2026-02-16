// Traces: FR-200
package copilot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

type evidenceStub struct {
	pack *knowledge.EvidencePack
	err  error
}

func (s *evidenceStub) BuildEvidencePack(_ context.Context, _ knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.pack, nil
}

type policyStub struct {
	filter policy.Filter
	err    error
}

func (s *policyStub) BuildPermissionFilter(_ context.Context, _ string) (policy.Filter, error) {
	if s.err != nil {
		return policy.Filter{}, s.err
	}
	return s.filter, nil
}

func (s *policyStub) RedactPII(_ context.Context, evidence []knowledge.Evidence) ([]knowledge.Evidence, error) {
	for i := range evidence {
		if evidence[i].Snippet != nil {
			v := strings.ReplaceAll(*evidence[i].Snippet, "john@acme.com", "[EMAIL_1]")
			evidence[i].Snippet = &v
			evidence[i].PiiRedacted = true
		}
	}
	return evidence, nil
}

type llmStub struct {
	resp string
	err  error
}

func (s *llmStub) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &llm.ChatResponse{Content: s.resp}, nil
}

func (s *llmStub) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{}, nil
}
func (s *llmStub) ModelInfo() llm.ModelMeta            { return llm.ModelMeta{ID: "stub", Provider: "stub"} }
func (s *llmStub) HealthCheck(_ context.Context) error { return nil }

type auditStub struct{ called int }

func (s *auditStub) LogWithDetails(_ context.Context, _, _ string, _ audit.ActorType, _ string, _, _ *string, _ *audit.EventDetails, _ audit.Outcome) error {
	s.called++
	return nil
}

func TestChat_StreamIncludesEvidenceTokenDone(t *testing.T) {
	sn := "customer email is john@acme.com"
	svc := NewChatService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{Snippet: &sn}}, Confidence: knowledge.ConfidenceHigh}},
		&llmStub{resp: "respuesta final"},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	ch, err := svc.Chat(context.Background(), ChatInput{WorkspaceID: "ws_1", UserID: "u_1", Query: "estado del caso"})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	chunks := make([]StreamChunk, 0)
	for c := range ch {
		chunks = append(chunks, c)
	}
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(chunks))
	}
	if chunks[0].Type != "evidence" {
		t.Fatalf("first chunk should be evidence, got %q", chunks[0].Type)
	}
	if chunks[len(chunks)-1].Type != "done" {
		t.Fatalf("last chunk should be done, got %q", chunks[len(chunks)-1].Type)
	}
}

func TestChat_PropagatesProviderError(t *testing.T) {
	svc := NewChatService(
		&evidenceStub{pack: &knowledge.EvidencePack{}},
		&llmStub{err: errors.New("llm down")},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	_, err := svc.Chat(context.Background(), ChatInput{WorkspaceID: "ws_1", UserID: "u_1", Query: "q"})
	if err == nil {
		t.Fatal("expected error")
	}
}
