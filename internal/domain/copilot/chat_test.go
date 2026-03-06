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
	call int
}

func (s *llmStub) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	s.call++
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
	sn := "estado del caso abierto para el cliente john@acme.com"
	llmSvc := &llmStub{resp: "respuesta final"}
	svc := NewChatService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{ID: "ev_1", Snippet: &sn}}, Confidence: knowledge.ConfidenceHigh}},
		llmSvc,
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	ch, err := svc.Chat(context.Background(), ChatInput{WorkspaceID: "ws_1", UserID: "u_1", Query: "estado del caso abierto"})
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
	if got := chunks[len(chunks)-1].Meta["answer_type"]; got != string(AnswerTypeGrounded) {
		t.Fatalf("expected grounded answer_type, got %#v", got)
	}
	if llmSvc.call != 1 {
		t.Fatalf("expected llm to be called once, got %d", llmSvc.call)
	}
}

func TestChat_PropagatesProviderError(t *testing.T) {
	sn := "pricing tiers for enterprise plan"
	svc := NewChatService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{ID: "ev_1", Snippet: &sn}}, Confidence: knowledge.ConfidenceHigh}},
		&llmStub{err: errors.New("llm down")},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	_, err := svc.Chat(context.Background(), ChatInput{WorkspaceID: "ws_1", UserID: "u_1", Query: "enterprise pricing tiers"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChat_AbstainsWhenEvidenceIsInsufficient(t *testing.T) {
	sn := "pricing information"
	llmSvc := &llmStub{resp: "should not be used"}
	svc := NewChatService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{ID: "ev_1", Snippet: &sn}}, Confidence: knowledge.ConfidenceLow}},
		llmSvc,
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	ch, err := svc.Chat(context.Background(), ChatInput{WorkspaceID: "ws_1", UserID: "u_1", Query: "pricing tiers"})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	chunks := collectChatChunks(ch)
	if got := chunks[len(chunks)-1].Meta["answer_type"]; got != string(AnswerTypeAbstention) {
		t.Fatalf("expected abstention answer_type, got %#v", got)
	}
	if got := chunks[len(chunks)-1].Meta["abstention_reason"]; got != string(AbstentionReasonInsufficientEvidence) {
		t.Fatalf("expected insufficient evidence reason, got %#v", got)
	}
	if llmSvc.call != 0 {
		t.Fatalf("expected llm not to be called, got %d", llmSvc.call)
	}
	if !streamContains(chunks, "No puedo responder de forma grounded") {
		t.Fatal("expected abstention content to be streamed")
	}
}

func TestChat_AbstainsWhenEvidenceIsIrrelevant(t *testing.T) {
	sn := "password reset instructions for support cases"
	llmSvc := &llmStub{resp: "should not be used"}
	svc := NewChatService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{ID: "ev_1", Snippet: &sn}}, Confidence: knowledge.ConfidenceHigh}},
		llmSvc,
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	ch, err := svc.Chat(context.Background(), ChatInput{WorkspaceID: "ws_1", UserID: "u_1", Query: "enterprise pricing tiers"})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	chunks := collectChatChunks(ch)
	if got := chunks[len(chunks)-1].Meta["abstention_reason"]; got != string(AbstentionReasonIrrelevantEvidence) {
		t.Fatalf("expected irrelevant evidence reason, got %#v", got)
	}
	if llmSvc.call != 0 {
		t.Fatalf("expected llm not to be called, got %d", llmSvc.call)
	}
}

func collectChatChunks(ch <-chan StreamChunk) []StreamChunk {
	chunks := make([]StreamChunk, 0)
	for c := range ch {
		chunks = append(chunks, c)
	}
	return chunks
}

func streamContains(chunks []StreamChunk, fragment string) bool {
	var b strings.Builder
	for _, chunk := range chunks {
		if chunk.Type == "token" {
			b.WriteString(chunk.Delta)
		}
	}
	return strings.Contains(b.String(), fragment)
}
