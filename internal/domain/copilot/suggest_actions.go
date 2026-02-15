package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

var (
	errInvalidEntityInput        = errors.New("entity_type and entity_id are required")
	errSuggestedActionsParseFail = errors.New("could not parse suggested actions")
	jsonFenceRe                  = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
)

var allowedActionTools = map[string]struct{}{
	tool.BuiltinCreateTask: {},
	tool.BuiltinUpdateCase: {},
	tool.BuiltinSendReply:  {},
}

type SuggestedAction struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Tool        string         `json:"tool"`
	Params      map[string]any `json:"params"`
}

type SuggestActionsInput struct {
	WorkspaceID string
	UserID      string
	EntityType  string
	EntityID    string
}

type SummarizeInput struct {
	WorkspaceID string
	UserID      string
	EntityType  string
	EntityID    string
}

type ActionService struct {
	evidence EvidencePackBuilder
	llm      llm.LLMProvider
	policy   PolicyEnforcer
	audit    AuditLogger
}

func NewActionService(e EvidencePackBuilder, l llm.LLMProvider, p PolicyEnforcer, a AuditLogger) *ActionService {
	return &ActionService{evidence: e, llm: l, policy: p, audit: a}
}

func (s *ActionService) SuggestActions(ctx context.Context, in SuggestActionsInput) ([]SuggestedAction, error) {
	if err := validateEntityInput(in.EntityType, in.EntityID); err != nil {
		return nil, err
	}

	prepared, err := s.prepareSuggestActionsContext(ctx, in)
	if err != nil {
		return nil, err
	}

	actions, err := s.generateSuggestedActions(ctx, in.EntityType, in.EntityID, prepared.redactedSources)
	if err != nil {
		return nil, err
	}
	s.logSuggestActionsAudit(ctx, in, prepared, len(actions))

	return actions, nil
}

type suggestActionsContext struct {
	filter          policy.Filter
	evidencePack    *knowledge.EvidencePack
	redactedSources []knowledge.Evidence
}

func (s *ActionService) prepareSuggestActionsContext(ctx context.Context, in SuggestActionsInput) (*suggestActionsContext, error) {
	filter, err := s.policy.BuildPermissionFilter(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	pack, err := s.evidence.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       buildEntityEvidenceQuery(in.EntityType, in.EntityID),
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

	return &suggestActionsContext{filter: filter, evidencePack: pack, redactedSources: redacted}, nil
}

func (s *ActionService) generateSuggestedActions(
	ctx context.Context,
	entityType, entityID string,
	sources []knowledge.Evidence,
) ([]SuggestedAction, error) {
	prompt := buildSuggestActionsPrompt(entityType, entityID, sources)
	resp, err := s.llm.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are FenixCRM Copilot. Return only valid JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   600,
	})
	if err != nil {
		return nil, err
	}

	actions, err := parseSuggestedActions(resp.Content)
	if err != nil {
		return nil, err
	}
	actions = normalizeActions(actions, 3)
	if len(actions) == 0 {
		return nil, errSuggestedActionsParseFail
	}
	return actions, nil
}

func (s *ActionService) logSuggestActionsAudit(ctx context.Context, in SuggestActionsInput, prepared *suggestActionsContext, actionCount int) {
	entityType := in.EntityType
	entityID := in.EntityID

	_ = s.audit.LogWithDetails(ctx, in.WorkspaceID, in.UserID, audit.ActorTypeUser, "copilot.suggest_actions", &entityType, &entityID, &audit.EventDetails{
		Metadata: map[string]any{
			"permissionWhere":   prepared.filter.Where,
			"filteredCount":     prepared.evidencePack.FilteredCount,
			"confidence":        string(prepared.evidencePack.Confidence),
			"generated_actions": actionCount,
		},
	}, audit.OutcomeSuccess)
}

func (s *ActionService) Summarize(ctx context.Context, in SummarizeInput) (string, error) {
	if err := validateEntityInput(in.EntityType, in.EntityID); err != nil {
		return "", err
	}

	filter, err := s.policy.BuildPermissionFilter(ctx, in.UserID)
	if err != nil {
		return "", err
	}

	pack, err := s.evidence.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       buildEntitySummaryQuery(in.EntityType, in.EntityID),
		WorkspaceID: in.WorkspaceID,
		Limit:       10,
	})
	if err != nil {
		return "", err
	}

	redacted, err := s.policy.RedactPII(ctx, pack.Sources)
	if err != nil {
		return "", err
	}

	prompt := buildSummarizePrompt(in.EntityType, in.EntityID, redacted)
	resp, err := s.llm.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are FenixCRM Copilot. Write concise, factual summaries."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   400,
	})
	if err != nil {
		return "", err
	}

	summary := strings.TrimSpace(redactOutputPII(resp.Content))
	if summary == "" {
		return "", errors.New("empty summary response")
	}

	entityType := in.EntityType
	entityID := in.EntityID
	_ = s.audit.LogWithDetails(ctx, in.WorkspaceID, in.UserID, audit.ActorTypeUser, "copilot.summarize", &entityType, &entityID, &audit.EventDetails{
		Metadata: map[string]any{
			"permissionWhere": filter.Where,
			"filteredCount":   pack.FilteredCount,
			"confidence":      string(pack.Confidence),
		},
	}, audit.OutcomeSuccess)

	return summary, nil
}

func validateEntityInput(entityType, entityID string) error {
	if strings.TrimSpace(entityType) == "" || strings.TrimSpace(entityID) == "" {
		return errInvalidEntityInput
	}
	return nil
}

func buildEntityEvidenceQuery(entityType, entityID string) string {
	return fmt.Sprintf("entity_type:%s entity_id:%s latest updates timeline next steps", entityType, entityID)
}

func buildEntitySummaryQuery(entityType, entityID string) string {
	return fmt.Sprintf("entity_type:%s entity_id:%s timeline status history summary", entityType, entityID)
}

func buildSuggestActionsPrompt(entityType, entityID string, sources []knowledge.Evidence) string {
	b := strings.Builder{}
	b.WriteString("Entity type: ")
	b.WriteString(entityType)
	b.WriteString("\nEntity id: ")
	b.WriteString(entityID)
	b.WriteString("\n\nEvidence:\n")
	b.WriteString(renderEvidenceForPrompt(sources))
	b.WriteString("\nTask: Suggest exactly 3 actionable next steps.")
	b.WriteString("\nRespond ONLY with JSON in this format:")
	b.WriteString(` {"actions":[{"title":"...","description":"...","tool":"create_task|update_case|send_reply","params":{}}]}`)
	return b.String()
}

func buildSummarizePrompt(entityType, entityID string, sources []knowledge.Evidence) string {
	b := strings.Builder{}
	b.WriteString("Entity type: ")
	b.WriteString(entityType)
	b.WriteString("\nEntity id: ")
	b.WriteString(entityID)
	b.WriteString("\n\nEvidence:\n")
	b.WriteString(renderEvidenceForPrompt(sources))
	b.WriteString("\nTask: Write a concise operational summary in 4-6 sentences.")
	b.WriteString(" Include status, risks, and recommended immediate focus.")
	return b.String()
}

func renderEvidenceForPrompt(sources []knowledge.Evidence) string {
	b := strings.Builder{}
	for i, src := range sources {
		if src.Snippet == nil {
			continue
		}
		b.WriteString("[")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("] ")
		b.WriteString(strings.TrimSpace(*src.Snippet))
		b.WriteString("\n")
	}
	return b.String()
}

func parseSuggestedActions(raw string) ([]SuggestedAction, error) {
	candidates := extractJSONCandidates(raw)
	for _, candidate := range candidates {
		actions, err := decodeSuggestedActions(candidate)
		if err == nil && len(actions) > 0 {
			return actions, nil
		}
	}
	return nil, errSuggestedActionsParseFail
}

func extractJSONCandidates(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	candidates := make([]string, 0, 4)
	candidates = appendIfNonEmptyCandidate(candidates, trimmed)
	candidates = append(candidates, extractFencedCandidates(trimmed)...)
	candidates = appendRangeCandidate(candidates, trimmed, "[", "]")
	candidates = appendRangeCandidate(candidates, trimmed, "{", "}")
	return dedupeStrings(candidates)
}

func appendIfNonEmptyCandidate(candidates []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return candidates
	}
	return append(candidates, value)
}

func extractFencedCandidates(input string) []string {
	matches := jsonFenceRe.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		out = appendIfNonEmptyCandidate(out, match[1])
	}
	return out
}

func appendRangeCandidate(candidates []string, input, open, close string) []string {
	start, end := strings.Index(input, open), strings.LastIndex(input, close)
	if start < 0 || end <= start {
		return candidates
	}
	return appendIfNonEmptyCandidate(candidates, input[start:end+1])
}

func decodeSuggestedActions(candidate string) ([]SuggestedAction, error) {
	var list []SuggestedAction
	if err := json.Unmarshal([]byte(candidate), &list); err == nil {
		return sanitizeSuggestedActions(list), nil
	}

	var envelope struct {
		Actions []SuggestedAction `json:"actions"`
		Data    struct {
			Actions []SuggestedAction `json:"actions"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(candidate), &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Actions) > 0 {
		return sanitizeSuggestedActions(envelope.Actions), nil
	}
	return sanitizeSuggestedActions(envelope.Data.Actions), nil
}

func sanitizeSuggestedActions(actions []SuggestedAction) []SuggestedAction {
	clean := make([]SuggestedAction, 0, len(actions))
	for _, action := range actions {
		action.Title = strings.TrimSpace(action.Title)
		action.Description = strings.TrimSpace(action.Description)
		action.Tool = strings.TrimSpace(action.Tool)
		if action.Title == "" || action.Tool == "" || !isAllowedActionTool(action.Tool) {
			continue
		}
		if action.Params == nil {
			action.Params = map[string]any{}
		}
		clean = append(clean, action)
	}
	return clean
}

func normalizeActions(actions []SuggestedAction, max int) []SuggestedAction {
	if max <= 0 {
		return []SuggestedAction{}
	}
	out := make([]SuggestedAction, 0, max)
	seen := map[string]struct{}{}
	for _, action := range actions {
		if len(out) >= max {
			break
		}
		key := strings.ToLower(action.Title + "|" + action.Tool)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, action)
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isAllowedActionTool(name string) bool {
	_, ok := allowedActionTools[name]
	return ok
}
