// Task 2.1: Unit tests for domain model methods in models.go.
// These tests exercise Go business logic only — no database, no migrations.
package knowledge

import (
	"testing"
	"time"
)

// ============================================================================
// KnowledgeItem.IsDeleted
// ============================================================================

func TestKnowledgeItem_IsDeleted_WhenDeletedAtIsNil(t *testing.T) {
	item := &KnowledgeItem{DeletedAt: nil}
	if item.IsDeleted() {
		t.Error("IsDeleted() should return false when DeletedAt is nil")
	}
}

func TestKnowledgeItem_IsDeleted_WhenDeletedAtIsSet(t *testing.T) {
	now := time.Now()
	item := &KnowledgeItem{DeletedAt: &now}
	if !item.IsDeleted() {
		t.Error("IsDeleted() should return true when DeletedAt is set")
	}
}

// ============================================================================
// EmbeddingDocument.IsPending
// ============================================================================

func TestEmbeddingDocument_IsPending_WhenStatusIsPending(t *testing.T) {
	doc := &EmbeddingDocument{EmbeddingStatus: EmbeddingStatusPending}
	if !doc.IsPending() {
		t.Error("IsPending() should return true when status is 'pending'")
	}
}

func TestEmbeddingDocument_IsPending_WhenStatusIsEmbedded(t *testing.T) {
	doc := &EmbeddingDocument{EmbeddingStatus: EmbeddingStatusEmbedded}
	if doc.IsPending() {
		t.Error("IsPending() should return false when status is 'embedded'")
	}
}

func TestEmbeddingDocument_IsPending_WhenStatusIsFailed(t *testing.T) {
	doc := &EmbeddingDocument{EmbeddingStatus: EmbeddingStatusFailed}
	if doc.IsPending() {
		t.Error("IsPending() should return false when status is 'failed'")
	}
}

// ============================================================================
// EmbeddingDocument.IsEmbedded
// ============================================================================

func TestEmbeddingDocument_IsEmbedded_WhenStatusIsEmbedded(t *testing.T) {
	doc := &EmbeddingDocument{EmbeddingStatus: EmbeddingStatusEmbedded}
	if !doc.IsEmbedded() {
		t.Error("IsEmbedded() should return true when status is 'embedded'")
	}
}

func TestEmbeddingDocument_IsEmbedded_WhenStatusIsPending(t *testing.T) {
	doc := &EmbeddingDocument{EmbeddingStatus: EmbeddingStatusPending}
	if doc.IsEmbedded() {
		t.Error("IsEmbedded() should return false when status is 'pending'")
	}
}

func TestEmbeddingDocument_IsEmbedded_WhenStatusIsFailed(t *testing.T) {
	doc := &EmbeddingDocument{EmbeddingStatus: EmbeddingStatusFailed}
	if doc.IsEmbedded() {
		t.Error("IsEmbedded() should return false when status is 'failed'")
	}
}

// ============================================================================
// SourceType / EmbeddingStatus / EvidenceMethod constants
// ============================================================================

func TestSourceType_Constants(t *testing.T) {
	cases := []struct {
		name  string
		value SourceType
		want  string
	}{
		{"document", SourceTypeDocument, "document"},
		{"email", SourceTypeEmail, "email"},
		{"call", SourceTypeCall, "call"},
		{"note", SourceTypeNote, "note"},
		{"case", SourceTypeCase, "case"},
		{"ticket", SourceTypeTicket, "ticket"},
		{"kb_article", SourceTypeKBArticle, "kb_article"},
		{"api", SourceTypeAPI, "api"},
		{"other", SourceTypeOther, "other"},
	}
	for _, c := range cases {
		if string(c.value) != c.want {
			t.Errorf("SourceType %s: expected %q, got %q", c.name, c.want, string(c.value))
		}
	}
}

func TestEmbeddingStatus_Constants(t *testing.T) {
	cases := []struct {
		name  string
		value EmbeddingStatus
		want  string
	}{
		{"pending", EmbeddingStatusPending, "pending"},
		{"embedded", EmbeddingStatusEmbedded, "embedded"},
		{"failed", EmbeddingStatusFailed, "failed"},
	}
	for _, c := range cases {
		if string(c.value) != c.want {
			t.Errorf("EmbeddingStatus %s: expected %q, got %q", c.name, c.want, string(c.value))
		}
	}
}

func TestEvidenceMethod_Constants(t *testing.T) {
	cases := []struct {
		name  string
		value EvidenceMethod
		want  string
	}{
		{"bm25", EvidenceMethodBM25, "bm25"},
		{"vector", EvidenceMethodVector, "vector"},
		{"hybrid", EvidenceMethodHybrid, "hybrid"},
	}
	for _, c := range cases {
		if string(c.value) != c.want {
			t.Errorf("EvidenceMethod %s: expected %q, got %q", c.name, c.want, string(c.value))
		}
	}
}
