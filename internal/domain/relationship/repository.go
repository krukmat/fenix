// Task B.2.1 — SignalRepository interface and event bus topic constants.
// Task B.2.2 — SQLiteSignalRepository concrete implementation.
package relationship

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event bus topic constants consumed by the Summarizer (Task B.2.3).
const (
	TopicActivityCreated          = "activity.created"
	TopicNoteCreated              = "note.created"
	TopicCaseUpdated              = "case.updated"
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

// SQLiteSignalRepository is the SQLite-backed implementation of SignalRepository.
type SQLiteSignalRepository struct {
	db *sql.DB
}

// NewSQLiteSignalRepository returns a repository backed by the given db connection.
func NewSQLiteSignalRepository(db *sql.DB) *SQLiteSignalRepository {
	return &SQLiteSignalRepository{db: db}
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
	var item InteractionSignal
	var sentiment sql.NullString
	var sourceEntityType sql.NullString
	var sourceEntityID sql.NullString
	var occurredAtStr, createdAtStr string

	if err := rows.Scan(
		&item.ID, &item.RelationshipMemoryID, &item.SignalType, &sentiment, &item.Summary,
		&sourceEntityType, &sourceEntityID, &occurredAtStr, &createdAtStr,
	); err != nil {
		return InteractionSignal{}, err
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
