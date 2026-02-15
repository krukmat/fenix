// Package knowledge defines domain types for the knowledge layer (Task 2.1).
// These types represent the business model for knowledge ingestion, FTS5 search,
// vector embedding, and evidence packs used by the AI/retrieval pipeline.
//
// Related tasks: 2.1 (tables), 2.2 (ingest), 2.4 (embed), 2.5 (search), 2.6 (evidence)
package knowledge

import "time"

// ============================================================================
// ENUMERATIONS
// ============================================================================

// SourceType identifies the origin of a KnowledgeItem (Task 2.2).
type SourceType string

const (
	SourceTypeDocument  SourceType = "document"
	SourceTypeEmail     SourceType = "email"
	SourceTypeCall      SourceType = "call"
	SourceTypeNote      SourceType = "note"
	SourceTypeCase      SourceType = "case"
	SourceTypeTicket    SourceType = "ticket"
	SourceTypeKBArticle SourceType = "kb_article"
	SourceTypeAPI       SourceType = "api"
	SourceTypeOther     SourceType = "other"
)

// EmbeddingStatus tracks the lifecycle of a chunk through the embedding pipeline (Task 2.4).
type EmbeddingStatus string

const (
	EmbeddingStatusPending  EmbeddingStatus = "pending"
	EmbeddingStatusEmbedded EmbeddingStatus = "embedded"
	EmbeddingStatusFailed   EmbeddingStatus = "failed"
)

// EvidenceMethod identifies how a piece of evidence was retrieved (Task 2.5/2.6).
type EvidenceMethod string

const (
	EvidenceMethodBM25   EvidenceMethod = "bm25"
	EvidenceMethodVector EvidenceMethod = "vector"
	EvidenceMethodHybrid EvidenceMethod = "hybrid"
)

// ============================================================================
// DOMAIN TYPES
// ============================================================================

// KnowledgeItem is the top-level ingested unit in the knowledge layer (Task 2.1/2.2).
// It stores raw content (from documents, emails, calls) and the normalized form
// used for FTS5 indexing. An optional link to a CRM entity (entity_type, entity_id)
// supports CDC-driven reindex in Task 2.7.
//
// DB table: knowledge_item (migration 011)
// FTS5 sync: automatic via triggers knowledge_item_ai/au/ad (migration 012)
//nolint:revive // término de dominio principal de Knowledge Layer
type KnowledgeItem struct {
	ID                string
	WorkspaceID       string
	SourceType        SourceType
	Title             string
	RawContent        string
	NormalizedContent *string
	EntityType        *string
	EntityID          *string
	Metadata          *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// IsDeleted returns true if the knowledge item has been soft-deleted.
func (k *KnowledgeItem) IsDeleted() bool {
	return k.DeletedAt != nil
}

// EmbeddingDocument is a chunk of a KnowledgeItem prepared for vector embedding (Task 2.2/2.4).
// Large documents are split into 512-token chunks with 50-token overlap.
// Each chunk is embedded independently via LLM.Embed() and stored in the
// vec_embedding virtual table (Task 2.4, migration 013).
//
// SECURITY: workspace_id is stored on EmbeddingDocument to allow secure multi-tenant
// vector search (JOIN on workspace_id prevents cross-tenant leaks, since sqlite-vec
// has no native multi-column index support).
//
// DB table: embedding_document (migration 011)
type EmbeddingDocument struct {
	ID              string
	KnowledgeItemID string
	WorkspaceID     string
	ChunkIndex      int64
	ChunkText       string
	TokenCount      *int64
	EmbeddingStatus EmbeddingStatus
	EmbeddedAt      *time.Time
	CreatedAt       time.Time
}

// IsPending returns true if the chunk has not yet been embedded.
func (e *EmbeddingDocument) IsPending() bool {
	return e.EmbeddingStatus == EmbeddingStatusPending
}

// IsEmbedded returns true if the vector embedding has been computed and stored.
func (e *EmbeddingDocument) IsEmbedded() bool {
	return e.EmbeddingStatus == EmbeddingStatusEmbedded
}

// Evidence is a single retrieved and scored search result (Task 2.6).
// Evidence records are assembled into EvidencePacks per query.
// The PiiRedacted flag must be set if any PII was removed before the snippet
// was shown to the LLM (policy enforcement, Task 3).
//
// DB table: evidence (migration 011)
type Evidence struct {
	ID              string
	KnowledgeItemID string
	WorkspaceID     string
	Method          EvidenceMethod
	Score           float64
	Snippet         *string
	PiiRedacted     bool
	Metadata        *string
	CreatedAt       time.Time
}

// ConfidenceLevel categorises the overall confidence of an EvidencePack (Task 2.6).
type ConfidenceLevel string

const (
	ConfidenceLow    ConfidenceLevel = "low"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceHigh   ConfidenceLevel = "high"
)

// EvidencePack is the assembled result returned from the hybrid search pipeline (Task 2.6).
// It groups deduplicated Evidence records, computes overall confidence, and carries
// any warnings (stale data, filtered results, etc.) for the calling agent/copilot.
type EvidencePack struct {
	Sources         []Evidence
	Confidence      ConfidenceLevel
	TotalCandidates int // total results from hybrid search before filtering/dedup
	FilteredCount   int // how many were removed by permissions/dedup/freshness
	Warnings        []string
}

// ============================================================================
// INPUT TYPES (for service layer, Task 2.2/2.6)
// ============================================================================

// CreateKnowledgeItemInput carries the fields required to create a new KnowledgeItem.
type CreateKnowledgeItemInput struct {
	WorkspaceID       string
	SourceType        SourceType
	Title             string
	RawContent        string
	NormalizedContent *string
	EntityType        *string
	EntityID          *string
	Metadata          *string
}

// CreateEmbeddingDocumentInput carries the fields required to create a new chunk.
type CreateEmbeddingDocumentInput struct {
	KnowledgeItemID string
	WorkspaceID     string
	ChunkIndex      int64
	ChunkText       string
	TokenCount      *int64
}

// CreateEvidenceInput carries the fields required to record a search result.
type CreateEvidenceInput struct {
	KnowledgeItemID string
	WorkspaceID     string
	Method          EvidenceMethod
	Score           float64
	Snippet         *string
	PiiRedacted     bool
	Metadata        *string
}

// BuildEvidencePackInput carries parameters for building an evidence pack (Task 2.6).
type BuildEvidencePackInput struct {
	Query       string
	WorkspaceID string
	Limit       int // 0 uses default (10), capped at 50
}
