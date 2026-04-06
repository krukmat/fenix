package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

var (
	errInvalidEntityInput        = errors.New("entity_type and entity_id are required")
	errSuggestedActionsParseFail = errors.New("could not parse suggested actions")
	jsonFenceRe                  = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
)

type ConfidenceLevel string

const (
	ConfidenceLevelLow    ConfidenceLevel = "low"
	ConfidenceLevelMedium ConfidenceLevel = "medium"
	ConfidenceLevelHigh   ConfidenceLevel = "high"
	entityTypeCase        string          = "case"
	entityTypeLead        string          = "lead"
	entityTypeAccount     string          = "account"
	entityTypeDeal        string          = "deal"
)

var allowedActionTools = map[string]struct{}{
	tool.BuiltinCreateTask: {},
	tool.BuiltinUpdateCase: {},
	tool.BuiltinUpdateDeal: {},
	tool.BuiltinSendReply:  {},
}

type SuggestedAction struct {
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Tool            string          `json:"tool"`
	Params          map[string]any  `json:"params"`
	ConfidenceScore float64         `json:"confidence_score"`
	ConfidenceLevel ConfidenceLevel `json:"confidence_level"`
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

type SalesBriefInput struct {
	WorkspaceID string
	UserID      string
	EntityType  string
	EntityID    string
}

type SalesBriefResult struct {
	Outcome          string
	EntityType       string
	EntityID         string
	Summary          string
	Risks            []string
	NextBestActions  []SuggestedAction
	Confidence       ConfidenceLevel
	AbstentionReason *AbstentionReason
	EvidencePack     *knowledge.EvidencePack
}

type ActionService struct {
	evidence EvidencePackBuilder
	llm      llm.LLMProvider
	policy   PolicyEnforcer
	audit    AuditLogger
	usage    UsageRecorder
}

type suggestActionsContext struct {
	filter          policy.Filter
	evidencePack    *knowledge.EvidencePack
	redactedSources []knowledge.Evidence
}

type suggestActionsMetrics struct {
	generated      int
	returned       int
	discardReasons map[string]int
}

func NewActionService(e EvidencePackBuilder, l llm.LLMProvider, p PolicyEnforcer, a AuditLogger) *ActionService {
	return NewActionServiceWithUsage(e, l, p, a, nil)
}

func NewActionServiceWithUsage(e EvidencePackBuilder, l llm.LLMProvider, p PolicyEnforcer, a AuditLogger, u UsageRecorder) *ActionService {
	return &ActionService{evidence: e, llm: l, policy: p, audit: a, usage: u}
}

func (s *ActionService) SuggestActions(ctx context.Context, in SuggestActionsInput) ([]SuggestedAction, error) {
	if err := validateEntityInput(in.EntityType, in.EntityID); err != nil {
		return nil, err
	}

	prepared, err := s.prepareSuggestActionsContext(ctx, in)
	if err != nil {
		return nil, err
	}

	actions, metrics, err := s.generateSuggestedActions(ctx, in.EntityType, in.EntityID, prepared.evidencePack, prepared.redactedSources)
	if err != nil {
		return nil, err
	}

	s.logSuggestActionsAudit(ctx, in, prepared, metrics)
	return actions, nil
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

	return &suggestActionsContext{
		filter:          filter,
		evidencePack:    pack,
		redactedSources: redacted,
	}, nil
}

func (s *ActionService) generateSuggestedActions(
	ctx context.Context,
	entityType, entityID string,
	pack *knowledge.EvidencePack,
	sources []knowledge.Evidence,
) ([]SuggestedAction, suggestActionsMetrics, error) {
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
		return nil, suggestActionsMetrics{}, err
	}

	actions, err := parseSuggestedActions(resp.Content)
	if err != nil {
		return nil, suggestActionsMetrics{}, err
	}

	actions = normalizeActions(actions, 3)
	if len(actions) == 0 {
		return nil, suggestActionsMetrics{}, errSuggestedActionsParseFail
	}

	filtered, metrics := scoreAndFilterSuggestedActions(actions, entityType, entityID, pack)
	return filtered, metrics, nil
}

func (s *ActionService) logSuggestActionsAudit(
	ctx context.Context,
	in SuggestActionsInput,
	prepared *suggestActionsContext,
	metrics suggestActionsMetrics,
) {
	entityType := in.EntityType
	entityID := in.EntityID

	_ = s.audit.LogWithDetails(ctx, in.WorkspaceID, in.UserID, audit.ActorTypeUser, "copilot.suggest_actions", &entityType, &entityID, &audit.EventDetails{
		Metadata: map[string]any{
			"permissionWhere":    prepared.filter.Where,
			"filteredCount":      prepared.evidencePack.FilteredCount,
			"confidence":         string(prepared.evidencePack.Confidence),
			"generated_actions":  metrics.generated,
			"returned_actions":   metrics.returned,
			"discarded_actions":  metrics.generated - metrics.returned,
			"discard_categories": metrics.discardReasons,
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

type salesBriefPayload struct {
	Summary string   `json:"summary"`
	Risks   []string `json:"risks"`
}

type salesBriefUsageRecord struct {
	modelName   *string
	inputUnits  int64
	outputUnits int64
	cost        float64
}

func (s *ActionService) SalesBrief(ctx context.Context, in SalesBriefInput) (*SalesBriefResult, error) {
	if err := validateSalesEntityInput(in.EntityType, in.EntityID); err != nil {
		return nil, err
	}

	startedAt := time.Now()
	prepared, err := s.prepareSuggestActionsContext(ctx, SuggestActionsInput{
		WorkspaceID: in.WorkspaceID,
		UserID:      in.UserID,
		EntityType:  in.EntityType,
		EntityID:    in.EntityID,
	})
	if err != nil {
		return nil, err
	}

	if reason := salesBriefAbstentionReason(prepared.evidencePack); reason != nil {
		result := &SalesBriefResult{
			Outcome:          "abstained",
			EntityType:       in.EntityType,
			EntityID:         in.EntityID,
			Summary:          abstentionMessage(*reason),
			Risks:            []string{},
			NextBestActions:  []SuggestedAction{},
			Confidence:       ConfidenceLevelLow,
			AbstentionReason: reason,
			EvidencePack:     prepared.evidencePack,
		}
		s.logSalesBriefAudit(ctx, in, prepared, result, suggestActionsMetrics{})
		s.recordSalesBriefUsage(ctx, in, salesBriefUsageRecord{}, time.Since(startedAt))
		return result, nil
	}

	brief, record, err := s.generateSalesBrief(ctx, in.EntityType, in.EntityID, prepared.evidencePack, prepared.redactedSources)
	if err != nil {
		return nil, err
	}
	actions, metrics, err := s.generateSuggestedActions(ctx, in.EntityType, in.EntityID, prepared.evidencePack, prepared.redactedSources)
	if err != nil {
		return nil, err
	}

	result := &SalesBriefResult{
		Outcome:         "completed",
		EntityType:      in.EntityType,
		EntityID:        in.EntityID,
		Summary:         brief.Summary,
		Risks:           brief.Risks,
		NextBestActions: actions,
		Confidence:      scoreToConfidenceLevel(baseScoreFromConfidence(prepared.evidencePack)),
		EvidencePack:    prepared.evidencePack,
	}
	s.logSalesBriefAudit(ctx, in, prepared, result, metrics)
	s.recordSalesBriefUsage(ctx, in, record, time.Since(startedAt))
	return result, nil
}

func (s *ActionService) generateSalesBrief(
	ctx context.Context,
	entityType, entityID string,
	pack *knowledge.EvidencePack,
	sources []knowledge.Evidence,
) (salesBriefPayload, salesBriefUsageRecord, error) {
	resp, err := s.llm.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are FenixCRM Sales Copilot. Return only valid JSON."},
			{Role: "user", Content: buildSalesBriefPrompt(entityType, entityID, sources)},
		},
		Temperature: 0.1,
		MaxTokens:   500,
	})
	if err != nil {
		return salesBriefPayload{}, salesBriefUsageRecord{}, err
	}

	payload, err := parseSalesBriefPayload(resp.Content)
	if err != nil {
		return salesBriefPayload{}, salesBriefUsageRecord{}, err
	}

	modelName := strings.TrimSpace(s.llm.ModelInfo().ID)
	record := salesBriefUsageRecord{
		inputUnits:  int64(resp.Tokens),
		outputUnits: int64(resp.Tokens),
		cost:        0,
	}
	if modelName != "" {
		record.modelName = &modelName
	}

	payload.Risks = filterNonEmptyStrings(payload.Risks)
	if payload.Summary == "" {
		payload.Summary = fallbackSalesSummary(entityType, entityID, pack)
	}
	return payload, record, nil
}

func (s *ActionService) logSalesBriefAudit(
	ctx context.Context,
	in SalesBriefInput,
	prepared *suggestActionsContext,
	result *SalesBriefResult,
	metrics suggestActionsMetrics,
) {
	entityType := in.EntityType
	entityID := in.EntityID
	metadata := map[string]any{
		"permissionWhere":  prepared.filter.Where,
		"filteredCount":    prepared.evidencePack.FilteredCount,
		"confidence":       string(prepared.evidencePack.Confidence),
		"outcome":          result.Outcome,
		"returned_actions": len(result.NextBestActions),
	}
	if metrics.generated > 0 {
		metadata["generated_actions"] = metrics.generated
		metadata["discard_categories"] = metrics.discardReasons
	}
	if result.AbstentionReason != nil {
		metadata["abstention_reason"] = string(*result.AbstentionReason)
	}
	_ = s.audit.LogWithDetails(ctx, in.WorkspaceID, in.UserID, audit.ActorTypeUser, "copilot.sales_brief", &entityType, &entityID, &audit.EventDetails{
		Metadata: metadata,
	}, audit.OutcomeSuccess)
}

func (s *ActionService) recordSalesBriefUsage(ctx context.Context, in SalesBriefInput, record salesBriefUsageRecord, duration time.Duration) {
	if s.usage == nil {
		return
	}

	latencyMs := duration.Milliseconds()
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

func salesBriefAbstentionReason(pack *knowledge.EvidencePack) *AbstentionReason {
	if !hasTraceableEvidence(pack) {
		reason := AbstentionReasonInsufficientEvidence
		return &reason
	}
	return nil
}

func validateEntityInput(entityType, entityID string) error {
	if strings.TrimSpace(entityType) == "" || strings.TrimSpace(entityID) == "" {
		return errInvalidEntityInput
	}
	return nil
}

func validateSalesEntityInput(entityType, entityID string) error {
	if err := validateEntityInput(entityType, entityID); err != nil {
		return err
	}
	if entityType != entityTypeAccount && entityType != entityTypeDeal {
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
	b.WriteString(` {"actions":[{"title":"...","description":"...","tool":"create_task|update_case|update_deal|send_reply","params":{}}]}`)
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

func buildSalesBriefPrompt(entityType, entityID string, sources []knowledge.Evidence) string {
	b := strings.Builder{}
	b.WriteString("Entity type: ")
	b.WriteString(entityType)
	b.WriteString("\nEntity id: ")
	b.WriteString(entityID)
	b.WriteString("\n\nEvidence:\n")
	b.WriteString(renderEvidenceForPrompt(sources))
	b.WriteString("\nTask: Return a grounded sales brief.")
	b.WriteString("\nRespond ONLY with JSON in this format:")
	b.WriteString(` {"summary":"...","risks":["..."]}`)
	b.WriteString("\nThe summary must be concise and operational. Risks must list objections, blockers, or gaps visible in the evidence.")
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

func appendRangeCandidate(candidates []string, input, open, closeToken string) []string {
	start, end := strings.Index(input, open), strings.LastIndex(input, closeToken)
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

func normalizeActions(actions []SuggestedAction, limit int) []SuggestedAction {
	if limit <= 0 {
		return []SuggestedAction{}
	}
	out := make([]SuggestedAction, 0, limit)
	seen := map[string]struct{}{}
	for _, action := range actions {
		if len(out) >= limit {
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

func parseSalesBriefPayload(raw string) (salesBriefPayload, error) {
	candidates := extractJSONCandidates(raw)
	for _, candidate := range candidates {
		var payload salesBriefPayload
		if err := json.Unmarshal([]byte(candidate), &payload); err == nil && strings.TrimSpace(payload.Summary) != "" {
			payload.Summary = strings.TrimSpace(payload.Summary)
			return payload, nil
		}

		var envelope struct {
			Data salesBriefPayload `json:"data"`
		}
		if err := json.Unmarshal([]byte(candidate), &envelope); err == nil && strings.TrimSpace(envelope.Data.Summary) != "" {
			envelope.Data.Summary = strings.TrimSpace(envelope.Data.Summary)
			return envelope.Data, nil
		}
	}
	return salesBriefPayload{}, errors.New("could not parse sales brief")
}

func scoreAndFilterSuggestedActions(
	actions []SuggestedAction,
	entityType, entityID string,
	pack *knowledge.EvidencePack,
) ([]SuggestedAction, suggestActionsMetrics) {
	metrics := suggestActionsMetrics{
		generated:      len(actions),
		discardReasons: map[string]int{},
	}
	filtered := make([]SuggestedAction, 0, len(actions))

	for _, action := range actions {
		reason := eligibilityDiscardReason(action, entityType, entityID)
		if reason != "" {
			metrics.discardReasons[reason]++
			continue
		}
		action.ConfidenceScore = scoreSuggestedAction(pack, action)
		action.ConfidenceLevel = scoreToConfidenceLevel(action.ConfidenceScore)
		filtered = append(filtered, action)
	}

	metrics.returned = len(filtered)
	return filtered, metrics
}

func eligibilityDiscardReason(action SuggestedAction, entityType, entityID string) string {
	switch action.Tool {
	case tool.BuiltinCreateTask:
		return validateCreateTaskEligibility(action.Params, entityType, entityID)
	case tool.BuiltinUpdateCase:
		return validateUpdateCaseEligibility(action.Params, entityType, entityID)
	case tool.BuiltinUpdateDeal:
		return validateUpdateDealEligibility(action.Params, entityType, entityID)
	case tool.BuiltinSendReply:
		return validateSendReplyEligibility(action.Params, entityType, entityID)
	default:
		return "tool_not_allowed"
	}
}

func validateCreateTaskEligibility(params map[string]any, entityType, entityID string) string {
	if !isActionEntityTypeSupported(entityType) {
		return "entity_not_supported"
	}
	if !matchesStringParam(params, "entity_type", entityType) {
		return "missing_or_mismatched_entity_type"
	}
	if !matchesStringParam(params, "entity_id", entityID) {
		return "missing_or_mismatched_entity_id"
	}
	return ""
}

func validateUpdateDealEligibility(params map[string]any, entityType, entityID string) string {
	if entityType != entityTypeDeal {
		return "tool_entity_mismatch"
	}
	if !matchesStringParam(params, "deal_id", entityID) {
		return "missing_or_mismatched_deal_id"
	}
	return ""
}

func validateUpdateCaseEligibility(params map[string]any, entityType, entityID string) string {
	if entityType != entityTypeCase {
		return "tool_entity_mismatch"
	}
	if !matchesStringParam(params, "case_id", entityID) {
		return "missing_or_mismatched_case_id"
	}
	return ""
}

func validateSendReplyEligibility(params map[string]any, entityType, entityID string) string {
	if entityType != entityTypeCase {
		return "tool_entity_mismatch"
	}
	if !matchesStringParam(params, "case_id", entityID) {
		return "missing_or_mismatched_case_id"
	}
	if !hasRequiredStringParam(params, "body") {
		return "missing_reply_body"
	}
	return ""
}

func matchesStringParam(params map[string]any, key, expected string) bool {
	value, ok := params[key]
	if !ok {
		return false
	}
	asString, ok := value.(string)
	return ok && strings.TrimSpace(asString) == expected
}

func hasRequiredStringParam(params map[string]any, key string) bool {
	value, ok := params[key]
	if !ok {
		return false
	}
	asString, ok := value.(string)
	return ok && strings.TrimSpace(asString) != ""
}

func scoreSuggestedAction(pack *knowledge.EvidencePack, action SuggestedAction) float64 {
	score := baseScoreFromConfidence(pack)
	if hasTraceableActionEvidence(pack) {
		score += 0.15
	}
	if len(action.Params) > 0 {
		score += 0.10
	}
	if action.Description != "" {
		score += 0.05
	}
	if score > 1 {
		return 1
	}
	return score
}

func baseScoreFromConfidence(pack *knowledge.EvidencePack) float64 {
	if pack == nil {
		return 0.2
	}
	switch pack.Confidence {
	case knowledge.ConfidenceHigh:
		return 0.75
	case knowledge.ConfidenceMedium:
		return 0.55
	default:
		return 0.35
	}
}

func hasTraceableActionEvidence(pack *knowledge.EvidencePack) bool {
	if pack == nil {
		return false
	}
	for _, source := range pack.Sources {
		if source.ID != "" || (source.Snippet != nil && strings.TrimSpace(*source.Snippet) != "") {
			return true
		}
	}
	return false
}

func scoreToConfidenceLevel(score float64) ConfidenceLevel {
	switch {
	case score >= 0.75:
		return ConfidenceLevelHigh
	case score >= 0.5:
		return ConfidenceLevelMedium
	default:
		return ConfidenceLevelLow
	}
}

func fallbackSalesSummary(entityType, entityID string, pack *knowledge.EvidencePack) string {
	return fmt.Sprintf("Grounded %s summary ready for %s with %d evidence sources.", entityType, entityID, len(pack.Sources))
}

func filterNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
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

func isActionEntityTypeSupported(entityType string) bool {
	switch entityType {
	case entityTypeCase, entityTypeLead, entityTypeAccount, entityTypeDeal:
		return true
	default:
		return false
	}
}
