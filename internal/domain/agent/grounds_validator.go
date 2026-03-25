package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

type groundsEvidencePackBuilder interface {
	BuildEvidencePack(ctx context.Context, input knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error)
}

type GroundsValidator struct {
	evidence groundsEvidencePackBuilder
	now      func() time.Time
}

type GroundsResult struct {
	Met         bool
	Reason      string
	Query       string
	EvidencePack *knowledge.EvidencePack
}

func NewGroundsValidator(evidence groundsEvidencePackBuilder) *GroundsValidator {
	return &GroundsValidator{
		evidence: evidence,
		now:      time.Now,
	}
}

func (v *GroundsValidator) Validate(ctx context.Context, grounds *CartaGrounds, input TriggerAgentInput) (*GroundsResult, error) {
	if grounds == nil {
		return &GroundsResult{Met: true}, nil
	}
	if v == nil || v.evidence == nil {
		return nil, fmt.Errorf("grounds validator requires evidence service")
	}

	query := buildGroundsQuery(input)
	pack, err := v.evidence.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       query,
		WorkspaceID: input.WorkspaceID,
		Limit:       max(grounds.MinSources, 1),
	})
	if err != nil {
		return nil, err
	}

	return applyGroundsConstraints(v.now(), grounds, pack, query), nil
}

func applyGroundsConstraints(now time.Time, grounds *CartaGrounds, pack *knowledge.EvidencePack, query string) *GroundsResult {
	if failed := validateGroundsSources(grounds, pack); failed != nil {
		failed.Query = query
		return failed
	}
	if failed := validateGroundsConfidence(grounds, pack); failed != nil {
		failed.Query = query
		return failed
	}
	if failed := validateGroundsStaleness(now, grounds, pack); failed != nil {
		failed.Query = query
		return failed
	}
	return &GroundsResult{Met: true, Query: query, EvidencePack: pack}
}

func validateGroundsSources(grounds *CartaGrounds, pack *knowledge.EvidencePack) *GroundsResult {
	if len(pack.Sources) < grounds.MinSources {
		return &GroundsResult{
			Met:          false,
			Reason:       fmt.Sprintf("insufficient evidence: %d source(s) (need %d)", len(pack.Sources), grounds.MinSources),
			EvidencePack: pack,
		}
	}
	return nil
}

func validateGroundsConfidence(grounds *CartaGrounds, pack *knowledge.EvidencePack) *GroundsResult {
	if !groundsConfidenceMet(pack.Confidence, grounds.MinConfidence) {
		return &GroundsResult{
			Met:          false,
			Reason:       fmt.Sprintf("insufficient evidence: confidence=%s (need %s)", pack.Confidence, grounds.MinConfidence),
			EvidencePack: pack,
		}
	}
	return nil
}

func validateGroundsStaleness(now time.Time, grounds *CartaGrounds, pack *knowledge.EvidencePack) *GroundsResult {
	if grounds.MaxStaleness <= 0 || grounds.MaxAgeUnit == "" {
		return nil
	}
	allowedAge := cartaDuration(grounds.MaxStaleness, grounds.MaxAgeUnit)
	if allowedAge > 0 && evidencePackIsStale(now, pack, allowedAge) {
		return &GroundsResult{
			Met:          false,
			Reason:       fmt.Sprintf("insufficient evidence: stale sources exceed %d %s", grounds.MaxStaleness, grounds.MaxAgeUnit),
			EvidencePack: pack,
		}
	}
	return nil
}

func groundsConfidenceMet(actual, required knowledge.ConfidenceLevel) bool {
	return confidenceRank(actual) >= confidenceRank(required)
}

func confidenceRank(level knowledge.ConfidenceLevel) int {
	switch strings.ToLower(strings.TrimSpace(string(level))) {
	case string(knowledge.ConfidenceHigh):
		return 3
	case string(knowledge.ConfidenceMedium):
		return 2
	case string(knowledge.ConfidenceLow):
		return 1
	default:
		return 0
	}
}

func cartaDuration(value int, unit string) time.Duration {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "days":
		return time.Duration(value) * 24 * time.Hour
	case "hours":
		return time.Duration(value) * time.Hour
	case "minutes":
		return time.Duration(value) * time.Minute
	default:
		return 0
	}
}

func evidencePackIsStale(now time.Time, pack *knowledge.EvidencePack, allowedAge time.Duration) bool {
	if pack == nil || len(pack.Sources) == 0 {
		return true
	}
	for _, source := range pack.Sources {
		if source.CreatedAt.IsZero() {
			return true
		}
		if now.Sub(source.CreatedAt) > allowedAge {
			return true
		}
	}
	return false
}

func buildGroundsQuery(input TriggerAgentInput) string {
	values := collectGroundsQueryValues(input.TriggerContext)
	values = append(values, collectGroundsQueryValues(input.Inputs)...)
	if len(values) == 0 {
		return "workflow evidence"
	}
	return strings.Join(dedupeGroundsValues(values), " ")
}

func collectGroundsQueryValues(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	var values []string
	visitGroundsQueryValue(payload, &values)
	return values
}

func visitGroundsQueryValue(value any, out *[]string) {
	switch node := value.(type) {
	case map[string]any:
		visitGroundsMap(node, out)
	case []any:
		visitGroundsList(node, out)
	case string:
		trimmed := strings.TrimSpace(node)
		if trimmed != "" && len(*out) < 6 {
			*out = append(*out, trimmed)
		}
	}
}

func visitGroundsMap(node map[string]any, out *[]string) {
	for key, child := range node {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		switch lowerKey {
		case "query", "customer_query", "message", "title", "subject", "summary", "description", "id":
			if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
				*out = append(*out, text)
				continue
			}
		}
		visitGroundsQueryValue(child, out)
	}
}

func visitGroundsList(node []any, out *[]string) {
	for _, child := range node {
		visitGroundsQueryValue(child, out)
	}
}

func dedupeGroundsValues(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
