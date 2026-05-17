// Package blackboard — shared key-value memory store over agent_memory table (Task A.3, ADR-100).
// Provides scoped read/write access for agents within a cognitive workspace.
// TTL is enforced lazily on Get: expired rows are deleted on first read.
package blackboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrMemoryNotFound is returned by Get when no entry exists for the given workspace + key.
var ErrMemoryNotFound = errors.New("agent memory entry not found")

// ErrMemoryExpired is returned by Get when the entry existed but its TTL has elapsed.
// The expired row is deleted as a side effect before this error is returned.
var ErrMemoryExpired = errors.New("agent memory entry expired")

// MemoryStore is the read/write interface over the agent_memory table.
// All operations are scoped to cognitive_workspace_id for multi-tenant isolation.
type MemoryStore interface {
	// Set upserts an AgentMemory entry. If a row with the same (cognitive_workspace_id, key)
	// already exists, value, scope, expires_at, and updated_at are overwritten.
	Set(ctx context.Context, entry AgentMemory) error

	// Get retrieves the entry for (cognitiveWorkspaceID, key).
	// Returns ErrMemoryNotFound if no row exists.
	// Returns ErrMemoryExpired if expires_at is set and in the past; the row is deleted.
	Get(ctx context.Context, cognitiveWorkspaceID, key string) (*AgentMemory, error)

	// Delete removes the entry for (cognitiveWorkspaceID, key). Idempotent — no error if missing.
	Delete(ctx context.Context, cognitiveWorkspaceID, key string) error

	// ClearSession deletes all scope='session' entries for the given cognitiveWorkspaceID.
	// Entries with scope='persistent' are untouched.
	ClearSession(ctx context.Context, cognitiveWorkspaceID string) error
}

type sqliteMemoryStore struct {
	db *sql.DB
}

// NewMemoryStore returns a MemoryStore backed by the given SQLite database.
func NewMemoryStore(db *sql.DB) MemoryStore {
	return &sqliteMemoryStore{db: db}
}

// Set upserts the AgentMemory entry.
// Uses INSERT ... ON CONFLICT DO UPDATE to preserve the original created_at and id.
func (s *sqliteMemoryStore) Set(ctx context.Context, entry AgentMemory) error {
	var expiresAt *string
	if entry.ExpiresAt != nil {
		formatted := entry.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &formatted
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_memory
			(id, cognitive_workspace_id, key, value, scope, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cognitive_workspace_id, key) DO UPDATE SET
			value      = excluded.value,
			scope      = excluded.scope,
			expires_at = excluded.expires_at,
			updated_at = excluded.updated_at`,
		entry.ID,
		entry.CognitiveWorkspaceID,
		entry.Key,
		string(entry.Value),
		string(entry.Scope),
		expiresAt,
		entry.CreatedAt.UTC().Format(time.RFC3339),
		entry.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("memory store Set: %w", err)
	}
	return nil
}

// Get retrieves the entry for the given workspace and key.
// Lazy TTL: if expires_at is set and in the past, the row is deleted and ErrMemoryExpired returned.
func (s *sqliteMemoryStore) Get(ctx context.Context, cognitiveWorkspaceID, key string) (*AgentMemory, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, cognitive_workspace_id, key, value, scope, expires_at, created_at, updated_at
		FROM agent_memory
		WHERE cognitive_workspace_id = ? AND key = ?`,
		cognitiveWorkspaceID, key,
	)

	m, err := scanMemory(row)
	if err != nil {
		return nil, err
	}

	if expired, delErr := s.checkExpiry(ctx, m, cognitiveWorkspaceID, key); expired {
		return nil, delErr
	}

	return m, nil
}

// scanMemory reads one agent_memory row into an AgentMemory struct.
func scanMemory(row *sql.Row) (*AgentMemory, error) {
	var m AgentMemory
	var valueStr, scopeStr, createdAtStr, updatedAtStr string
	var expiresAtRaw sql.NullString

	err := row.Scan(&m.ID, &m.CognitiveWorkspaceID, &m.Key,
		&valueStr, &scopeStr, &expiresAtRaw, &createdAtStr, &updatedAtStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMemoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memory store Get scan: %w", err)
	}

	m.Value = []byte(valueStr)
	m.Scope = MemoryScope(scopeStr)

	if expiresAtRaw.Valid {
		t, parseErr := parseTime(expiresAtRaw.String)
		if parseErr != nil {
			return nil, fmt.Errorf("memory store Get parse expires_at: %w", parseErr)
		}
		m.ExpiresAt = &t
	}

	if m.CreatedAt, err = parseTime(createdAtStr); err != nil {
		return nil, fmt.Errorf("memory store Get parse created_at: %w", err)
	}
	if m.UpdatedAt, err = parseTime(updatedAtStr); err != nil {
		return nil, fmt.Errorf("memory store Get parse updated_at: %w", err)
	}

	return &m, nil
}

// checkExpiry returns (true, ErrMemoryExpired) if the entry is expired and was deleted.
// Returns (false, nil) when the entry is valid or has no TTL.
func (s *sqliteMemoryStore) checkExpiry(ctx context.Context, m *AgentMemory, cwID, key string) (bool, error) {
	if m.ExpiresAt == nil || !time.Now().UTC().After(*m.ExpiresAt) {
		return false, nil
	}
	if err := s.deleteRow(ctx, cwID, key); err != nil {
		return true, fmt.Errorf("memory store Get delete expired: %w", err)
	}
	return true, ErrMemoryExpired
}

// Delete removes a single entry. Idempotent — no error if the row does not exist.
func (s *sqliteMemoryStore) Delete(ctx context.Context, cognitiveWorkspaceID, key string) error {
	return s.deleteRow(ctx, cognitiveWorkspaceID, key)
}

// ClearSession removes all scope='session' entries for the given cognitive workspace.
func (s *sqliteMemoryStore) ClearSession(ctx context.Context, cognitiveWorkspaceID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM agent_memory WHERE cognitive_workspace_id = ? AND scope = 'session'`,
		cognitiveWorkspaceID,
	)
	if err != nil {
		return fmt.Errorf("memory store ClearSession: %w", err)
	}
	return nil
}

// deleteRow executes the DELETE for a single (workspace, key) pair.
func (s *sqliteMemoryStore) deleteRow(ctx context.Context, cognitiveWorkspaceID, key string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM agent_memory WHERE cognitive_workspace_id = ? AND key = ?`,
		cognitiveWorkspaceID, key,
	)
	if err != nil {
		return fmt.Errorf("memory store deleteRow: %w", err)
	}
	return nil
}

// parseTime parses a SQLite datetime string in RFC3339 or SQLite's default format.
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format: %q", s)
}
