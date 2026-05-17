// Package blackboard — append-only read path over reasoning_event table (Task A.4, ADR-100).
// ReasoningTimeline provides ordered event retrieval for replay (Phase C) and
// direct append for isolated workspace injection without a live bus.
package blackboard

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TimelineFilter narrows a List query. Zero values mean "no filter applied".
type TimelineFilter struct {
	EventType EventType // filter to a single event type; empty = all types
	Limit     int       // max rows returned; 0 = no limit
}

// ReasoningTimeline is the append-only query interface over reasoning_event.
// Append coexists safely with WorkspaceBus.Publish — both write to the same table.
type ReasoningTimeline interface {
	// Append inserts one event directly into reasoning_event.
	// Returns an error on duplicate ID (PK violation) or FK failure.
	Append(ctx context.Context, event ReasoningEvent) error

	// List returns events for cognitiveWorkspaceID in ascending created_at order.
	// Returns a non-nil empty slice when no rows match.
	List(ctx context.Context, cognitiveWorkspaceID string, filter TimelineFilter) ([]ReasoningEvent, error)
}

type sqliteReasoningTimeline struct {
	db *sql.DB
}

// NewReasoningTimeline returns a ReasoningTimeline backed by the given SQLite database.
func NewReasoningTimeline(db *sql.DB) ReasoningTimeline {
	return &sqliteReasoningTimeline{db: db}
}

// Append inserts a ReasoningEvent row. Payload defaults to `{}` when nil or empty.
func (t *sqliteReasoningTimeline) Append(ctx context.Context, event ReasoningEvent) error {
	payload := event.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	_, err := t.db.ExecContext(ctx,
		`INSERT INTO reasoning_event (id, cognitive_workspace_id, actor_agent_id, event_type, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.CognitiveWorkspaceID,
		event.ActorAgentID,
		string(event.EventType),
		string(payload),
		event.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("reasoning timeline Append: %w", err)
	}
	return nil
}

// List queries reasoning_event for the given workspace, applying optional type filter and limit.
// Results are ordered by created_at ASC, id ASC for deterministic replay ordering.
func (t *sqliteReasoningTimeline) List(ctx context.Context, cognitiveWorkspaceID string, filter TimelineFilter) ([]ReasoningEvent, error) {
	query, args := buildListQuery(cognitiveWorkspaceID, filter)

	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("reasoning timeline List: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

// buildListQuery constructs the SELECT statement and argument slice for List.
func buildListQuery(cognitiveWorkspaceID string, filter TimelineFilter) (string, []any) {
	args := []any{cognitiveWorkspaceID}
	where := "WHERE cognitive_workspace_id = ?"

	if filter.EventType != "" {
		where += " AND event_type = ?"
		args = append(args, string(filter.EventType))
	}

	q := "SELECT id, cognitive_workspace_id, actor_agent_id, event_type, payload, created_at " +
		"FROM reasoning_event " + where +
		" ORDER BY created_at ASC, id ASC"

	if filter.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	return q, args
}

// scanEvents reads all rows into a ReasoningEvent slice.
// Returns a non-nil empty slice when no rows are found.
func scanEvents(rows *sql.Rows) ([]ReasoningEvent, error) {
	events := []ReasoningEvent{}

	for rows.Next() {
		var e ReasoningEvent
		var actorAgentID sql.NullString
		var eventTypeStr, payloadStr, createdAtStr string

		if err := rows.Scan(&e.ID, &e.CognitiveWorkspaceID, &actorAgentID,
			&eventTypeStr, &payloadStr, &createdAtStr); err != nil {
			return nil, fmt.Errorf("reasoning timeline scan row: %w", err)
		}

		if actorAgentID.Valid {
			e.ActorAgentID = &actorAgentID.String
		}
		e.EventType = EventType(eventTypeStr)
		e.Payload = []byte(payloadStr)

		t, err := parseTime(createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("reasoning timeline parse created_at: %w", err)
		}
		e.CreatedAt = t

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reasoning timeline rows error: %w", err)
	}

	return events, nil
}
