package copilot

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

type EvidencePackBuilder interface {
	BuildEvidencePack(ctx context.Context, input knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error)
}

type PolicyEnforcer interface {
	BuildPermissionFilter(ctx context.Context, userID string) (policy.Filter, error)
	RedactPII(ctx context.Context, evidence []knowledge.Evidence) ([]knowledge.Evidence, error)
}

type AuditLogger interface {
	LogWithDetails(ctx context.Context, workspaceID, actorID string, actorType audit.ActorType, action string, entityType, entityID *string, details *audit.EventDetails, outcome audit.Outcome) error
}

type ChatService struct {
	evidence EvidencePackBuilder
	llm      llm.LLMProvider
	policy   PolicyEnforcer
	audit    AuditLogger
}

type ChatInput struct {
	WorkspaceID string
	UserID      string
	Query       string
	EntityType  *string
	EntityID    *string
}

type StreamChunk struct {
	Type    string               `json:"type"`
	Delta   string               `json:"delta,omitempty"`
	Sources []knowledge.Evidence `json:"sources,omitempty"`
	Meta    map[string]any       `json:"meta,omitempty"`
	Done    bool                 `json:"done,omitempty"`
	Error   string               `json:"error,omitempty"`
}

func NewChatService(e EvidencePackBuilder, l llm.LLMProvider, p PolicyEnforcer, a AuditLogger) *ChatService {
	return &ChatService{evidence: e, llm: l, policy: p, audit: a}
}

func (s *ChatService) Chat(ctx context.Context, in ChatInput) (<-chan StreamChunk, error) {
	filter, err := s.policy.BuildPermissionFilter(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	pack, err := s.evidence.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       in.Query,
		WorkspaceID: in.WorkspaceID,
		Limit:       10,
	})
	if err != nil {
		return nil, err
	}

	redacted, err := s.policy.RedactPII(ctx, pack.Sources)
	if err != nil {
		return nil, err
	}
	pack.Sources = redacted

	prompt := buildPrompt(in.Query, pack.Sources)
	resp, err := s.llm.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are FenixCRM Copilot. Always cite sources."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   512,
	})
	if err != nil {
		return nil, err
	}

	content := redactOutputPII(resp.Content)

	_ = s.audit.LogWithDetails(ctx, in.WorkspaceID, in.UserID, audit.ActorTypeUser, "copilot.chat", in.EntityType, in.EntityID, &audit.EventDetails{
		Metadata: map[string]any{
			"query":           in.Query,
			"permissionWhere": filter.Where,
			"filteredCount":   pack.FilteredCount,
			"confidence":      string(pack.Confidence),
		},
	}, audit.OutcomeSuccess)

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		ch <- StreamChunk{Type: "evidence", Sources: pack.Sources}
		for _, tk := range strings.Fields(content) {
			ch <- StreamChunk{Type: "token", Delta: tk + " "}
		}
		ch <- StreamChunk{Type: "done", Done: true, Meta: map[string]any{"at": time.Now().UTC().Format(time.RFC3339)}}
	}()

	return ch, nil
}

func buildPrompt(query string, sources []knowledge.Evidence) string {
	b := strings.Builder{}
	b.WriteString("User query: ")
	b.WriteString(query)
	b.WriteString("\nEvidence:\n")
	for i, s := range sources {
		if s.Snippet == nil {
			continue
		}
		b.WriteString("[")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("] ")
		b.WriteString(*s.Snippet)
		b.WriteString("\n")
	}
	return b.String()
}

var piiOutRe = regexp.MustCompile(`\b(?:[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}|\d{3}-\d{2}-\d{4})\b`)

func redactOutputPII(content string) string {
	return piiOutRe.ReplaceAllString(content, "[REDACTED]")
}
