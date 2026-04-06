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
	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
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

type UsageRecorder interface {
	RecordEvent(ctx context.Context, input usage.RecordEventInput) (*usage.Event, error)
}

type ChatService struct {
	evidence EvidencePackBuilder
	llm      llm.LLMProvider
	policy   PolicyEnforcer
	audit    AuditLogger
	usage    UsageRecorder
}

type AnswerType string

const (
	AnswerTypeGrounded   AnswerType = "grounded_answer"
	AnswerTypeAbstention AnswerType = "abstention"
)

type AbstentionReason string

const (
	AbstentionReasonInsufficientEvidence AbstentionReason = "insufficient_evidence"
	AbstentionReasonIrrelevantEvidence   AbstentionReason = "irrelevant_evidence"
)

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

type ChatResult struct {
	AnswerType       AnswerType
	Content          string
	Sources          []knowledge.Evidence
	AbstentionReason *AbstentionReason
}

func NewChatService(e EvidencePackBuilder, l llm.LLMProvider, p PolicyEnforcer, a AuditLogger) *ChatService {
	return NewChatServiceWithUsage(e, l, p, a, nil)
}

func NewChatServiceWithUsage(e EvidencePackBuilder, l llm.LLMProvider, p PolicyEnforcer, a AuditLogger, u UsageRecorder) *ChatService {
	return &ChatService{evidence: e, llm: l, policy: p, audit: a, usage: u}
}

func (s *ChatService) Chat(ctx context.Context, in ChatInput) (<-chan StreamChunk, error) {
	startedAt := time.Now()
	filter, pack, err := s.prepareChatContext(ctx, in)
	if err != nil {
		return nil, err
	}

	result, record, err := s.buildChatResult(ctx, in.Query, pack)
	if err != nil {
		return nil, err
	}

	s.auditChat(ctx, in, filter, pack, result)
	s.recordUsage(ctx, in, record, time.Since(startedAt))
	return streamChatResult(result), nil
}

func (s *ChatService) prepareChatContext(ctx context.Context, in ChatInput) (policy.Filter, *knowledge.EvidencePack, error) {
	filter, err := s.policy.BuildPermissionFilter(ctx, in.UserID)
	if err != nil {
		return policy.Filter{}, nil, err
	}

	pack, err := s.evidence.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       in.Query,
		WorkspaceID: in.WorkspaceID,
		Limit:       10,
	})
	if err != nil {
		return policy.Filter{}, nil, err
	}

	redacted, err := s.policy.RedactPII(ctx, pack.Sources)
	if err != nil {
		return policy.Filter{}, nil, err
	}
	pack.Sources = redacted

	return filter, pack, nil
}

type chatUsageRecord struct {
	modelName   *string
	inputUnits  int64
	outputUnits int64
	cost        float64
}

func (s *ChatService) buildChatResult(ctx context.Context, query string, pack *knowledge.EvidencePack) (ChatResult, chatUsageRecord, error) {
	if reason := evaluateAbstention(pack, query); reason != nil {
		return newAbstentionResult(pack.Sources, *reason), chatUsageRecord{}, nil
	}

	resp, err := s.llm.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are FenixCRM Copilot. Always answer only from the provided evidence and cite sources."},
			{Role: "user", Content: buildPrompt(query, pack.Sources)},
		},
		Temperature: 0.2,
		MaxTokens:   512,
	})
	if err != nil {
		return ChatResult{}, chatUsageRecord{}, err
	}

	modelName := strings.TrimSpace(s.llm.ModelInfo().ID)
	record := chatUsageRecord{
		inputUnits:  int64(resp.Tokens),
		outputUnits: int64(resp.Tokens),
		cost:        0,
	}
	if modelName != "" {
		record.modelName = &modelName
	}

	return ChatResult{
		AnswerType: AnswerTypeGrounded,
		Content:    redactOutputPII(resp.Content),
		Sources:    pack.Sources,
	}, record, nil
}

func newAbstentionResult(sources []knowledge.Evidence, reason AbstentionReason) ChatResult {
	return ChatResult{
		AnswerType:       AnswerTypeAbstention,
		Content:          abstentionMessage(reason),
		Sources:          sources,
		AbstentionReason: &reason,
	}
}

func abstentionMessage(reason AbstentionReason) string {
	switch reason {
	case AbstentionReasonIrrelevantEvidence:
		return "No puedo responder de forma grounded porque la evidencia recuperada no es suficientemente relevante para tu consulta."
	default:
		return "No puedo responder de forma grounded porque no hay evidencia suficiente y trazable para sostener una respuesta."
	}
}

func evaluateAbstention(pack *knowledge.EvidencePack, query string) *AbstentionReason {
	if !hasTraceableEvidence(pack) {
		reason := AbstentionReasonInsufficientEvidence
		return &reason
	}
	if !hasRelevantEvidence(query, pack.Sources) {
		reason := AbstentionReasonIrrelevantEvidence
		return &reason
	}
	return nil
}

func hasTraceableEvidence(pack *knowledge.EvidencePack) bool {
	if pack == nil || pack.Confidence == knowledge.ConfidenceLow {
		return false
	}
	for _, source := range pack.Sources {
		if source.ID != "" && source.Snippet != nil && strings.TrimSpace(*source.Snippet) != "" {
			return true
		}
	}
	return false
}

func hasRelevantEvidence(query string, sources []knowledge.Evidence) bool {
	queryTerms := normalizedTerms(query)
	if len(queryTerms) == 0 {
		return false
	}
	for _, source := range sources {
		if source.Snippet == nil {
			continue
		}
		if sharesRelevantTerm(queryTerms, normalizedTerms(*source.Snippet)) {
			return true
		}
	}
	return false
}

func sharesRelevantTerm(queryTerms, sourceTerms []string) bool {
	if len(sourceTerms) == 0 {
		return false
	}
	sourceSet := make(map[string]struct{}, len(sourceTerms))
	for _, term := range sourceTerms {
		sourceSet[term] = struct{}{}
	}
	for _, term := range queryTerms {
		if _, ok := sourceSet[term]; ok {
			return true
		}
	}
	return false
}

func normalizedTerms(text string) []string {
	raw := tokenSplitter.Split(strings.ToLower(text), -1)
	terms := make([]string, 0, len(raw))
	for _, term := range raw {
		if len(term) < 4 || ignoredTerms[term] {
			continue
		}
		terms = append(terms, term)
	}
	return terms
}

func (s *ChatService) auditChat(ctx context.Context, in ChatInput, filter policy.Filter, pack *knowledge.EvidencePack, result ChatResult) {
	metadata := map[string]any{
		"query":           in.Query,
		"permissionWhere": filter.Where,
		"filteredCount":   pack.FilteredCount,
		"confidence":      string(pack.Confidence),
		"answerType":      string(result.AnswerType),
		"sourceCount":     len(result.Sources),
	}
	if result.AbstentionReason != nil {
		metadata["abstentionReason"] = string(*result.AbstentionReason)
	}

	_ = s.audit.LogWithDetails(ctx, in.WorkspaceID, in.UserID, audit.ActorTypeUser, "copilot.chat", in.EntityType, in.EntityID, &audit.EventDetails{
		Metadata: metadata,
	}, audit.OutcomeSuccess)
}

func (s *ChatService) recordUsage(ctx context.Context, in ChatInput, record chatUsageRecord, elapsed time.Duration) {
	if s.usage == nil {
		return
	}

	latencyMs := elapsed.Milliseconds()
	_, _ = s.usage.RecordEvent(ctx, usage.RecordEventInput{
		WorkspaceID:   in.WorkspaceID,
		ActorID:       in.UserID,
		ActorType:     string(audit.ActorTypeUser),
		ModelName:     record.modelName,
		InputUnits:    record.inputUnits,
		OutputUnits:   record.outputUnits,
		EstimatedCost: record.cost,
		LatencyMs:     &latencyMs,
	})
}

func streamChatResult(result ChatResult) <-chan StreamChunk {
	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		ch <- StreamChunk{Type: "evidence", Sources: result.Sources, Meta: evidenceMeta(result.Sources)}
		for _, tk := range strings.Fields(result.Content) {
			ch <- StreamChunk{Type: "token", Delta: tk + " "}
		}
		ch <- StreamChunk{Type: "done", Done: true, Meta: doneMeta(result)}
	}()

	return ch
}

func evidenceMeta(sources []knowledge.Evidence) map[string]any {
	methods := make([]string, 0, len(sources))
	seen := make(map[knowledge.EvidenceMethod]struct{}, len(sources))
	for _, source := range sources {
		if _, ok := seen[source.Method]; ok {
			continue
		}
		seen[source.Method] = struct{}{}
		methods = append(methods, string(source.Method))
	}

	return map[string]any{
		"schema_version":         knowledge.EvidencePackSchemaVersion,
		"source_count":           len(sources),
		"retrieval_methods_used": methods,
		"built_at":               time.Now().UTC().Format(time.RFC3339),
	}
}

func doneMeta(result ChatResult) map[string]any {
	meta := map[string]any{
		"at":          time.Now().UTC().Format(time.RFC3339),
		"answer_type": string(result.AnswerType),
		"source_ids":  sourceIDs(result.Sources),
	}
	if result.AbstentionReason != nil {
		meta["abstention_reason"] = string(*result.AbstentionReason)
	}
	return meta
}

func sourceIDs(sources []knowledge.Evidence) []string {
	ids := make([]string, 0, len(sources))
	for _, source := range sources {
		if source.ID != "" {
			ids = append(ids, source.ID)
		}
	}
	return ids
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
var tokenSplitter = regexp.MustCompile(`[^a-z0-9]+`)
var ignoredTerms = map[string]bool{
	"about": true, "como": true, "con": true, "cual": true, "del": true, "desde": true,
	"donde": true, "esta": true, "este": true, "estos": true, "from": true, "para": true,
	"that": true, "this": true, "what": true, "when": true, "where": true, "which": true,
}

func redactOutputPII(content string) string {
	return piiOutRe.ReplaceAllString(content, "[REDACTED]")
}
