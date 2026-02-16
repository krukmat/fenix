// Package knowledge — Task 2.6: EvidencePackService.
//
// EvidencePackService transforms raw hybrid search results into curated,
// deduplicated, confidence-scored evidence packs for downstream AI consumers.
package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	defaultEvidenceCandidateLimit = 50
	maxEvidenceTopK               = 50
)

// EvidenceConfig configures EvidencePackService behavior.
type EvidenceConfig struct {
	DefaultTopK            int
	FreshnessWarning       time.Duration
	DedupThreshold         float64
	HighConfidenceMin      float64
	MediumConfidenceMin    float64
	PermissionCheckStubbed bool
}

// DefaultEvidenceConfig returns sane defaults for Task 2.6.
func DefaultEvidenceConfig() EvidenceConfig {
	return EvidenceConfig{
		DefaultTopK:            10,
		FreshnessWarning:       30 * 24 * time.Hour,
		DedupThreshold:         0.95,
		HighConfidenceMin:      0.8,
		MediumConfidenceMin:    0.5,
		PermissionCheckStubbed: true,
	}
}

// calculateConfidence maps top score to low/medium/high.
func (c EvidenceConfig) calculateConfidence(topScore float64) ConfidenceLevel {
	if topScore >= c.HighConfidenceMin {
		return ConfidenceHigh
	}
	if topScore >= c.MediumConfidenceMin {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

// EvidencePackService builds evidence packs from hybrid search results.
type EvidencePackService struct {
	db     *sql.DB
	q      *sqlcgen.Queries
	search *SearchService
	cfg    EvidenceConfig
}

// NewEvidencePackService creates a new service instance.
func NewEvidencePackService(db *sql.DB, searchSvc *SearchService, cfg EvidenceConfig) *EvidencePackService {
	if cfg.DefaultTopK <= 0 {
		cfg.DefaultTopK = 10
	}
	if cfg.DedupThreshold <= 0 {
		cfg.DedupThreshold = 0.95
	}
	if cfg.HighConfidenceMin <= 0 {
		cfg.HighConfidenceMin = 0.8
	}
	if cfg.MediumConfidenceMin <= 0 {
		cfg.MediumConfidenceMin = 0.5
	}
	if cfg.FreshnessWarning <= 0 {
		cfg.FreshnessWarning = 30 * 24 * time.Hour
	}

	return &EvidencePackService{
		db:     db,
		q:      sqlcgen.New(db),
		search: searchSvc,
		cfg:    cfg,
	}
}

// BuildEvidencePack executes hybrid search and returns curated evidence.
func (s *EvidencePackService) BuildEvidencePack(ctx context.Context, input BuildEvidencePackInput) (*EvidencePack, error) {
	topK := s.resolveTopK(input.Limit)

	searchRes, err := s.search.HybridSearch(ctx, SearchInput{
		Query:       input.Query,
		WorkspaceID: input.WorkspaceID,
		Limit:       defaultEvidenceCandidateLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("evidence: hybrid search: %w", err)
	}

	totalCandidates := len(searchRes.Items)
	if totalCandidates == 0 {
		return s.emptyEvidencePack(), nil
	}

	representativeVectors, _ := s.getRepresentativeVectors(ctx, input.WorkspaceID)
	selected, dedupCount, staleCount := s.selectCandidates(ctx, input.WorkspaceID, searchRes.Items, representativeVectors, topK)
	warnings := s.buildWarnings(dedupCount, staleCount)

	evidenceRows, err := s.persistEvidence(ctx, input.WorkspaceID, selected)
	if err != nil {
		return nil, err
	}

	return &EvidencePack{
		Sources:         evidenceRows,
		Confidence:      s.packConfidence(selected),
		TotalCandidates: totalCandidates,
		FilteredCount:   s.filteredCount(totalCandidates, len(selected)),
		Warnings:        warnings,
	}, nil
}

func (s *EvidencePackService) emptyEvidencePack() *EvidencePack {
	return &EvidencePack{
		Sources:         []Evidence{},
		Confidence:      ConfidenceLow,
		TotalCandidates: 0,
		FilteredCount:   0,
		Warnings:        []string{"no sources found"},
	}
}

func (s *EvidencePackService) selectCandidates(
	ctx context.Context,
	wsID string,
	candidates []SearchResult,
	representativeVectors map[string][]float32,
	topK int,
) ([]SearchResult, int, int) {
	selected := make([]SearchResult, 0, topK)
	selectedVectors := make([][]float32, 0, topK)
	dedupCount := 0
	staleCount := 0

	for _, candidate := range candidates {
		if len(selected) >= topK {
			break
		}
		if s.isStale(ctx, candidate.KnowledgeItemID, wsID) {
			staleCount++
		}
		vec, hasVec := representativeVectors[candidate.KnowledgeItemID]
		if hasVec && s.isNearDuplicate(vec, selectedVectors) {
			dedupCount++
			continue
		}
		selected = append(selected, candidate)
		if hasVec {
			selectedVectors = append(selectedVectors, vec)
		}
	}

	return selected, dedupCount, staleCount
}

func (s *EvidencePackService) isNearDuplicate(vec []float32, selectedVectors [][]float32) bool {
	for _, existing := range selectedVectors {
		if nearDuplicateVectors(vec, existing, s.cfg.DedupThreshold) {
			return true
		}
	}
	return false
}

func (s *EvidencePackService) buildWarnings(dedupCount, staleCount int) []string {
	warnings := make([]string, 0, 2)
	if dedupCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d items deduplicated", dedupCount))
	}
	if staleCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d items stale", staleCount))
	}
	return warnings
}

func (s *EvidencePackService) packConfidence(selected []SearchResult) ConfidenceLevel {
	if len(selected) == 0 {
		return ConfidenceLow
	}
	return s.cfg.calculateConfidence(s.normalizeConfidenceScore(selected[0].Score))
}

func (s *EvidencePackService) filteredCount(total, selected int) int {
	filtered := total - selected
	if filtered < 0 {
		return 0
	}
	return filtered
}

func (s *EvidencePackService) resolveTopK(limit int) int {
	if limit <= 0 {
		limit = s.cfg.DefaultTopK
	}
	if limit > maxEvidenceTopK {
		return maxEvidenceTopK
	}
	return limit
}

func (s *EvidencePackService) persistEvidence(ctx context.Context, wsID string, selected []SearchResult) ([]Evidence, error) {
	now := time.Now()
	rows := make([]Evidence, 0, len(selected))

	for _, item := range selected {
		id := uuid.NewV7().String()
		snippet := item.Snippet
		snippetPtr := &snippet
		if snippet == "" {
			snippetPtr = nil
		}

		if err := s.q.CreateEvidence(ctx, sqlcgen.CreateEvidenceParams{
			ID:              id,
			KnowledgeItemID: item.KnowledgeItemID,
			WorkspaceID:     wsID,
			Method:          string(item.Method),
			Score:           item.Score,
			Snippet:         snippetPtr,
			PiiRedacted:     false,
			Metadata:        nil,
			CreatedAt:       now,
		}); err != nil {
			return nil, fmt.Errorf("evidence: create evidence: %w", err)
		}

		rows = append(rows, Evidence{
			ID:              id,
			KnowledgeItemID: item.KnowledgeItemID,
			WorkspaceID:     wsID,
			Method:          item.Method,
			Score:           item.Score,
			Snippet:         snippetPtr,
			PiiRedacted:     false,
			Metadata:        nil,
			CreatedAt:       now,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Score > rows[j].Score })
	return rows, nil
}

func (s *EvidencePackService) isStale(ctx context.Context, itemID, wsID string) bool {
	if s.cfg.FreshnessWarning <= 0 {
		return false
	}
	item, err := s.q.GetKnowledgeItemByID(ctx, sqlcgen.GetKnowledgeItemByIDParams{
		ID:          itemID,
		WorkspaceID: wsID,
	})
	if err != nil {
		return false
	}
	return time.Since(item.UpdatedAt) > s.cfg.FreshnessWarning
}

func (s *EvidencePackService) getRepresentativeVectors(ctx context.Context, wsID string) (map[string][]float32, error) {
	rows, rowsErr := s.q.GetAllEmbeddedVectorsByWorkspace(ctx, wsID)
	if rowsErr != nil {
		return nil, rowsErr
	}

	out := make(map[string][]float32, len(rows))
	for _, row := range rows {
		if _, exists := out[row.KnowledgeItemID]; exists {
			continue
		}
		vec, decodeErr := decodeEmbedding(row.Embedding)
		if decodeErr != nil {
			continue
		}
		out[row.KnowledgeItemID] = vec
	}
	return out, nil
}

// nearDuplicateVectors returns true if cosine similarity is above threshold.
func nearDuplicateVectors(a, b []float32, threshold float64) bool {
	return float64(cosineSimilarity(a, b)) >= threshold
}

// normalizeConfidenceScore maps RRF score to [0,1] for confidence thresholds.
// RRF absolute values are tiny (e.g. ~0.01-0.03 with k=60), so we normalize
// against the theoretical max with two retrieval methods (BM25 + vector).
func (s *EvidencePackService) normalizeConfidenceScore(raw float64) float64 {
	if raw <= 0 {
		return 0
	}
	maxScore := 2.0 / float64(rrfK+1)
	if maxScore <= 0 {
		return 0
	}
	return math.Min(1.0, raw/maxScore)
}
