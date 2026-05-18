// Task B.2.1 — SignalRepository interface and event bus topic constants.
// Task B.2.2 — SQLiteSignalRepository concrete implementation.
package relationship

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event bus topic constants consumed by the Summarizer (Task B.2.3).
const (
	TopicActivityCreated          = "activity.created"
	TopicNoteCreated              = "note.created"
	TopicCaseUpdated              = "case.updated"
	TopicDealUpdated              = "deal.updated"
	TopicInteractionSignalCreated = "interaction_signal.created"
)

// SignalRepository is the persistence contract for the Summarizer.
// Implemented by SQLiteSignalRepository (Task B.2.2); faked in unit tests (Task B.2.4).
// Callers never import internal/infra/sqlite directly.
type SignalRepository interface {
	// UpsertMemory creates or updates the relationship_memory anchor for a CRM entity.
	// Conflict key: (workspace_id, entity_type, entity_id). Returns the persisted row.
	UpsertMemory(ctx context.Context, workspaceID string, entityType EntityType, entityID, summary string) (*Memory, error)

	// InsertSignal appends one interaction_signal row linked to an existing memory.
	// Returns the persisted signal ID for downstream event publication.
	InsertSignal(ctx context.Context, memoryID string, signalType SignalType, sentiment SentimentType,
		summary, sourceEntityType, sourceEntityID string, occurredAt time.Time) (string, error)
}

// LifecycleRepository owns persistence for stale-memory updates and GDPR erasure flows.
type LifecycleRepository interface {
	ListStaleMemories(ctx context.Context, workspaceID string, cutoff time.Time) ([]Memory, error)
	ListStaleSignals(ctx context.Context, workspaceID string, cutoff time.Time) ([]InteractionSignal, error)
	UpdateMemorySummary(ctx context.Context, memoryID, summary string) error
	UpdateSignalSummary(ctx context.Context, signalID, summary string) error
	EraseEntityArtifacts(ctx context.Context, workspaceID string, entityType EntityType, entityID string) error
}

// compile-time interface check.
var _ SignalRepository = (*SQLiteSignalRepository)(nil)
var _ LifecycleRepository = (*SQLiteSignalRepository)(nil)
var _ TrustRepository = (*SQLiteSignalRepository)(nil)
var _ GraphRepository = (*SQLiteSignalRepository)(nil)

// SQLiteSignalRepository is the SQLite-backed implementation of SignalRepository.
type SQLiteSignalRepository struct {
	db           *sql.DB
	embeddingDim int
}

// NewSQLiteSignalRepository returns a repository backed by the given db connection.
func NewSQLiteSignalRepository(db *sql.DB) *SQLiteSignalRepository {
	return &SQLiteSignalRepository{db: db}
}

// NewSQLiteSignalRepositoryWithEmbeddingDim returns a repository with vector dimension validation enabled.
func NewSQLiteSignalRepositoryWithEmbeddingDim(db *sql.DB, embeddingDim int) *SQLiteSignalRepository {
	return &SQLiteSignalRepository{db: db, embeddingDim: embeddingDim}
}

// UpsertMemory inserts or updates the relationship_memory anchor row for a CRM entity.
// On conflict (workspace_id, entity_type, entity_id) the summary and updated_at are refreshed.
// The persisted row is always read back so the returned ID reflects the winner row.
func (r *SQLiteSignalRepository) UpsertMemory(ctx context.Context, workspaceID string, entityType EntityType, entityID, summary string) (*Memory, error) {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO relationship_memory
			(id, workspace_id, entity_type, entity_id, summary, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, entity_type, entity_id)
		DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at
	`, id, workspaceID, string(entityType), entityID, summary, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("relationship.UpsertMemory exec: %w", err)
	}

	var mem Memory
	var updatedStr, createdStr string
	err = r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, entity_type, entity_id, summary, updated_at, created_at
		FROM relationship_memory
		WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?
	`, workspaceID, string(entityType), entityID).Scan(
		&mem.ID, &mem.WorkspaceID, &mem.EntityType, &mem.EntityID,
		&mem.Summary, &updatedStr, &createdStr,
	)
	if err != nil {
		return nil, fmt.Errorf("relationship.UpsertMemory select: %w", err)
	}
	mem.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	mem.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return &mem, nil
}

// InsertSignal appends one interaction_signal row linked to the given memory.
func (r *SQLiteSignalRepository) InsertSignal(ctx context.Context, memoryID string, signalType SignalType, sentiment SentimentType,
	summary, sourceEntityType, sourceEntityID string, occurredAt time.Time) (string, error) {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO interaction_signal
			(id, relationship_memory_id, signal_type, sentiment, summary,
			 source_entity_type, source_entity_id, occurred_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, memoryID, string(signalType), string(sentiment), summary,
		sourceEntityType, sourceEntityID,
		occurredAt.UTC().Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("relationship.InsertSignal: %w", err)
	}
	return id, nil
}

// UpsertTrustScore inserts or updates the trust score row for one relationship memory.
func (r *SQLiteSignalRepository) UpsertTrustScore(ctx context.Context, memoryID string, score float64,
	confidence ConfidenceLevel, decayFactor float64, lastScoredAt time.Time) error {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trust_score
			(id, relationship_memory_id, score, confidence, decay_factor, last_scored_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(relationship_memory_id) DO UPDATE SET
			score = excluded.score,
			confidence = excluded.confidence,
			decay_factor = excluded.decay_factor,
			last_scored_at = excluded.last_scored_at,
			updated_at = excluded.updated_at
	`, id, memoryID, score, string(confidence), decayFactor, lastScoredAt.UTC().Format(time.RFC3339), now, now)
	if err != nil {
		return fmt.Errorf("relationship.UpsertTrustScore: %w", err)
	}
	return nil
}

// UpsertEdge inserts or updates one stakeholder graph edge keyed by its logical identity.
func (r *SQLiteSignalRepository) UpsertEdge(ctx context.Context,
	workspaceID, fromEntityType, fromEntityID,
	toEntityType, toEntityID string,
	influenceType InfluenceType,
	strength float64,
) error {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO stakeholder_graph
			(id, workspace_id, from_entity_type, from_entity_id,
			 to_entity_type, to_entity_id, influence_type,
			 strength, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, from_entity_type, from_entity_id,
		            to_entity_type, to_entity_id, influence_type)
		DO UPDATE SET
			strength = excluded.strength,
			updated_at = excluded.updated_at
	`, id, workspaceID, fromEntityType, fromEntityID, toEntityType, toEntityID, string(influenceType), strength, now, now)
	if err != nil {
		return fmt.Errorf("relationship.UpsertEdge: %w", err)
	}
	return nil
}

// UpsertSignalEmbedding stores one embedding vector per interaction signal.
func (r *SQLiteSignalRepository) UpsertSignalEmbedding(ctx context.Context, workspaceID, signalID string, vector []float32) error {
	if r.embeddingDim > 0 && len(vector) != r.embeddingDim {
		return fmt.Errorf("relationship.UpsertSignalEmbedding dim mismatch: want %d got %d", r.embeddingDim, len(vector))
	}

	embeddingJSON, err := encodeEmbeddingVector(vector)
	if err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding encode vector: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var existingVecID sql.NullString
	queryErr := tx.QueryRowContext(ctx, `
		SELECT vec_embedding_id
		FROM interaction_signal_embedding
		WHERE workspace_id = ? AND signal_id = ?
	`, workspaceID, signalID).Scan(&existingVecID)
	if queryErr != nil && queryErr != sql.ErrNoRows {
		return fmt.Errorf("relationship.UpsertSignalEmbedding select existing vector: %w", queryErr)
	}

	if existingVecID.Valid {
		if err = deleteSyntheticEmbeddingArtifacts(ctx, tx, existingVecID.String, workspaceID); err != nil {
			return err
		}
	}

	vecID := uuid.Must(uuid.NewV7()).String()
	knowledgeItemID := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC().Format(time.RFC3339)

	if err = insertSyntheticEmbeddingBackingRows(ctx, tx, workspaceID, signalID, knowledgeItemID, vecID, now); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO vec_embedding (id, workspace_id, embedding, created_at)
		VALUES (?, ?, ?, ?)
	`, vecID, workspaceID, embeddingJSON, now); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding insert vec_embedding: %w", err)
	}

	linkID := uuid.Must(uuid.NewV7()).String()
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO interaction_signal_embedding
			(id, workspace_id, signal_id, vec_embedding_id, dim, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(signal_id) DO UPDATE SET
			vec_embedding_id = excluded.vec_embedding_id,
			dim = excluded.dim,
			updated_at = excluded.updated_at
	`, linkID, workspaceID, signalID, vecID, len(vector), now, now); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding upsert link: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding commit: %w", err)
	}
	return nil
}

// ListStaleMemories returns relationship_memory rows whose updated_at is at or before cutoff.
func (r *SQLiteSignalRepository) ListStaleMemories(ctx context.Context, workspaceID string, cutoff time.Time) ([]Memory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, entity_type, entity_id, summary, updated_at, created_at
		FROM relationship_memory
		WHERE workspace_id = ? AND updated_at <= ?
		ORDER BY updated_at ASC
	`, workspaceID, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("relationship.ListStaleMemories query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make([]Memory, 0, 8)
	for rows.Next() {
		var item Memory
		var updatedStr, createdStr string
		if scanErr := rows.Scan(
			&item.ID, &item.WorkspaceID, &item.EntityType, &item.EntityID,
			&item.Summary, &updatedStr, &createdStr,
		); scanErr != nil {
			return nil, fmt.Errorf("relationship.ListStaleMemories scan: %w", scanErr)
		}
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		out = append(out, item)
	}
	rowsErr := rows.Err()
	if rowsErr != nil {
		return nil, fmt.Errorf("relationship.ListStaleMemories rows: %w", rowsErr)
	}
	return out, nil
}

// ListStaleSignals returns interaction_signal rows whose occurred_at is at or before cutoff.
func (r *SQLiteSignalRepository) ListStaleSignals(ctx context.Context, workspaceID string, cutoff time.Time) ([]InteractionSignal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.relationship_memory_id, s.signal_type, s.sentiment, s.summary,
		       s.source_entity_type, s.source_entity_id, s.occurred_at, s.created_at
		FROM interaction_signal s
		JOIN relationship_memory m ON m.id = s.relationship_memory_id
		WHERE m.workspace_id = ? AND s.occurred_at <= ?
		ORDER BY s.occurred_at ASC
	`, workspaceID, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("relationship.ListStaleSignals query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make([]InteractionSignal, 0, 8)
	for rows.Next() {
		item, scanErr := scanStaleSignal(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("relationship.ListStaleSignals scan: %w", scanErr)
		}
		out = append(out, item)
	}
	rowsErr := rows.Err()
	if rowsErr != nil {
		return nil, fmt.Errorf("relationship.ListStaleSignals rows: %w", rowsErr)
	}
	return out, nil
}

// ListSignalsByMemory returns all interaction signals linked to one memory in chronological order.
func (r *SQLiteSignalRepository) ListSignalsByMemory(ctx context.Context, memoryID string) ([]InteractionSignal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, relationship_memory_id, signal_type, sentiment, summary,
		       source_entity_type, source_entity_id, occurred_at, created_at
		FROM interaction_signal
		WHERE relationship_memory_id = ?
		ORDER BY occurred_at ASC, created_at ASC
	`, memoryID)
	if err != nil {
		return nil, fmt.Errorf("relationship.ListSignalsByMemory query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make([]InteractionSignal, 0, 8)
	for rows.Next() {
		item, scanErr := scanInteractionSignal(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("relationship.ListSignalsByMemory scan: %w", scanErr)
		}
		out = append(out, item)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("relationship.ListSignalsByMemory rows: %w", rowsErr)
	}
	return out, nil
}

func (r *SQLiteSignalRepository) UpdateMemorySummary(ctx context.Context, memoryID, summary string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE relationship_memory
		SET summary = ?, updated_at = ?
		WHERE id = ?
	`, summary, time.Now().UTC().Format(time.RFC3339), memoryID)
	if err != nil {
		return fmt.Errorf("relationship.UpdateMemorySummary: %w", err)
	}
	return nil
}

func (r *SQLiteSignalRepository) UpdateSignalSummary(ctx context.Context, signalID, summary string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE interaction_signal
		SET summary = ?
		WHERE id = ?
	`, summary, signalID)
	if err != nil {
		return fmt.Errorf("relationship.UpdateSignalSummary: %w", err)
	}
	return nil
}

func (r *SQLiteSignalRepository) EraseEntityArtifacts(ctx context.Context, workspaceID string, entityType EntityType, entityID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	execErr := runEraseSteps(ctx, tx, workspaceID, entityType, entityID)
	if execErr != nil {
		return execErr
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts commit: %w", commitErr)
	}
	return nil
}

func runEraseSteps(ctx context.Context, tx *sql.Tx, workspaceID string, entityType EntityType, entityID string) error {
	steps := []func(context.Context, *sql.Tx, string, EntityType, string) error{
		eraseEntityEmbeddings,
		eraseEntityGraphEdges,
		eraseEntityTrustScores,
		eraseEntitySignals,
		eraseEntityMemoryRow,
	}
	for _, step := range steps {
		if err := step(ctx, tx, workspaceID, entityType, entityID); err != nil {
			return err
		}
	}
	return nil
}

func scanStaleSignal(rows *sql.Rows) (InteractionSignal, error) {
	return scanInteractionSignal(rows)
}

func scanInteractionSignal(rows *sql.Rows) (InteractionSignal, error) {
	var item InteractionSignal
	var sentiment sql.NullString
	var sourceEntityType sql.NullString
	var sourceEntityID sql.NullString
	var occurredAtStr, createdAtStr string

	if scanErr := rows.Scan(
		&item.ID, &item.RelationshipMemoryID, &item.SignalType, &sentiment, &item.Summary,
		&sourceEntityType, &sourceEntityID, &occurredAtStr, &createdAtStr,
	); scanErr != nil {
		return InteractionSignal{}, fmt.Errorf("scan interaction signal row: %w", scanErr)
	}

	item.Sentiment = nullableSentiment(sentiment)
	item.SourceEntityType = nullableString(sourceEntityType)
	item.SourceEntityID = nullableString(sourceEntityID)
	item.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return item, nil
}

func nullableSentiment(value sql.NullString) *SentimentType {
	if !value.Valid {
		return nil
	}
	sentiment := SentimentType(value.String)
	return &sentiment
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func encodeEmbeddingVector(vec []float32) (string, error) {
	raw, err := json.Marshal(vec)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func insertSyntheticEmbeddingBackingRows(ctx context.Context, tx *sql.Tx, workspaceID, signalID, knowledgeItemID, embeddingDocumentID, now string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_item (
			id, workspace_id, source_type, title, raw_content, normalized_content, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, knowledgeItemID, workspaceID, "other", "relationship-signal-"+signalID, signalID, signalID, now, now); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding insert knowledge_item: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO embedding_document (
			id, knowledge_item_id, workspace_id, chunk_index, chunk_text, embedding_status, embedded_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, embeddingDocumentID, knowledgeItemID, workspaceID, 0, signalID, "embedded", now, now); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding insert embedding_document: %w", err)
	}
	return nil
}

func deleteSyntheticEmbeddingArtifacts(ctx context.Context, tx *sql.Tx, embeddingDocumentID, workspaceID string) error {
	var knowledgeItemID string
	err := tx.QueryRowContext(ctx, `
		SELECT knowledge_item_id
		FROM embedding_document
		WHERE id = ? AND workspace_id = ?
	`, embeddingDocumentID, workspaceID).Scan(&knowledgeItemID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("relationship.UpsertSignalEmbedding select old embedding_document: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM vec_embedding WHERE id = ? AND workspace_id = ?`, embeddingDocumentID, workspaceID); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding delete old vector: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM embedding_document WHERE id = ? AND workspace_id = ?`, embeddingDocumentID, workspaceID); err != nil {
		return fmt.Errorf("relationship.UpsertSignalEmbedding delete old embedding_document: %w", err)
	}
	if knowledgeItemID != "" {
		if _, err = tx.ExecContext(ctx, `DELETE FROM knowledge_item WHERE id = ? AND workspace_id = ?`, knowledgeItemID, workspaceID); err != nil {
			return fmt.Errorf("relationship.UpsertSignalEmbedding delete old knowledge_item: %w", err)
		}
	}
	return nil
}

func eraseEntityEmbeddings(ctx context.Context, tx *sql.Tx, workspaceID string, entityType EntityType, entityID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM vec_embedding
		WHERE workspace_id = ?
		  AND id IN (
			SELECT s.id
			FROM interaction_signal s
			JOIN relationship_memory m ON m.id = s.relationship_memory_id
			WHERE m.workspace_id = ? AND m.entity_type = ? AND m.entity_id = ?
		  )
	`, workspaceID, workspaceID, string(entityType), entityID)
	if err != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts delete vec_embedding: %w", err)
	}
	return nil
}

func eraseEntityGraphEdges(ctx context.Context, tx *sql.Tx, workspaceID string, entityType EntityType, entityID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM stakeholder_graph
		WHERE workspace_id = ?
		  AND (
			(from_entity_type = ? AND from_entity_id = ?)
			OR
			(to_entity_type = ? AND to_entity_id = ?)
		  )
	`, workspaceID, string(entityType), entityID, string(entityType), entityID)
	if err != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts delete stakeholder_graph: %w", err)
	}
	return nil
}

func eraseEntityTrustScores(ctx context.Context, tx *sql.Tx, workspaceID string, entityType EntityType, entityID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM trust_score
		WHERE relationship_memory_id IN (
			SELECT id
			FROM relationship_memory
			WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?
		)
	`, workspaceID, string(entityType), entityID)
	if err != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts delete trust_score: %w", err)
	}
	return nil
}

func eraseEntitySignals(ctx context.Context, tx *sql.Tx, workspaceID string, entityType EntityType, entityID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM interaction_signal
		WHERE relationship_memory_id IN (
			SELECT id
			FROM relationship_memory
			WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?
		)
	`, workspaceID, string(entityType), entityID)
	if err != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts delete interaction_signal: %w", err)
	}
	return nil
}

func eraseEntityMemoryRow(ctx context.Context, tx *sql.Tx, workspaceID string, entityType EntityType, entityID string) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM relationship_memory
		WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?
	`, workspaceID, string(entityType), entityID)
	if err != nil {
		return fmt.Errorf("relationship.EraseEntityArtifacts delete relationship_memory: %w", err)
	}
	return nil
}
