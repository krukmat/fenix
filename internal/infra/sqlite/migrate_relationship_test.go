// Tests for migration 032: relationship memory schema (Task B.1)
package sqlite_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// --- Table existence ---

func TestMigrate_RelationshipMemoryTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "relationship_memory")
}

func TestMigrate_InteractionSignalTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "interaction_signal")
}

func TestMigrate_StakeholderGraphTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "stakeholder_graph")
}

func TestMigrate_TrustScoreTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "trust_score")
}

// --- relationship_memory CHECK constraints ---

func TestMigrate_RelationshipMemoryEntityTypeCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-rm-et', 'RM ET WS', 'rm-et-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-et', 'ws-rm-et', 'invalid_entity', 'ent-001', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("relationship_memory insert with invalid entity_type succeeded; want CHECK constraint error")
	}
}

func TestMigrate_RelationshipMemoryToneCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-rm-tone', 'RM Tone WS', 'rm-tone-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, tone, created_at, updated_at)
		VALUES ('rm-tone', 'ws-rm-tone', 'contact', 'ent-002', 'happy', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("relationship_memory insert with invalid tone succeeded; want CHECK constraint error")
	}
}

func TestMigrate_RelationshipMemoryTrajectoryCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-rm-traj', 'RM Traj WS', 'rm-traj-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, trajectory, created_at, updated_at)
		VALUES ('rm-traj', 'ws-rm-traj', 'contact', 'ent-003', 'sideways', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("relationship_memory insert with invalid trajectory succeeded; want CHECK constraint error")
	}
}

// --- relationship_memory UNIQUE constraint ---

func TestMigrate_RelationshipMemoryUniqueConstraint(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-rm-uniq', 'RM Uniq WS', 'rm-uniq-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-uniq-1', 'ws-rm-uniq', 'contact', 'ent-010', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("first relationship_memory insert failed: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-uniq-2', 'ws-rm-uniq', 'contact', 'ent-010', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("duplicate (workspace_id, entity_type, entity_id) insert succeeded; want UNIQUE constraint error")
	}
}

// --- interaction_signal CHECK constraints ---

func TestMigrate_InteractionSignalTypeCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-is-type', 'IS Type WS', 'is-type-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-is-type', 'ws-is-type', 'account', 'ent-020', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO interaction_signal (id, relationship_memory_id, signal_type, occurred_at, created_at)
		VALUES ('is-type', 'rm-is-type', 'invalid_type', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("interaction_signal insert with invalid signal_type succeeded; want CHECK constraint error")
	}
}

func TestMigrate_InteractionSignalSentimentCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-is-sent', 'IS Sent WS', 'is-sent-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-is-sent', 'ws-is-sent', 'account', 'ent-021', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO interaction_signal (id, relationship_memory_id, signal_type, sentiment, occurred_at, created_at)
		VALUES ('is-sent', 'rm-is-sent', 'email', 'happy', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("interaction_signal insert with invalid sentiment succeeded; want CHECK constraint error")
	}
}

// --- stakeholder_graph CHECK constraints ---

func TestMigrate_StakeholderGraphInfluenceTypeCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-inf', 'SG Inf WS', 'sg-inf-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, created_at, updated_at)
		VALUES ('sg-inf', 'ws-sg-inf', 'contact', 'ent-030', 'contact', 'ent-031', 'boss_of', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("stakeholder_graph insert with invalid influence_type succeeded; want CHECK constraint error")
	}
}

func TestMigrate_StakeholderGraphStrengthCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-str', 'SG Str WS', 'sg-str-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-str', 'ws-sg-str', 'contact', 'ent-032', 'contact', 'ent-033', 'reports_to', 1.5, datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("stakeholder_graph insert with strength > 1.0 succeeded; want CHECK constraint error")
	}
}

func TestMigrate_StakeholderGraphFromEntityTypeCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-fet', 'SG FET WS', 'sg-fet-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, created_at, updated_at)
		VALUES ('sg-fet', 'ws-sg-fet', 'invalid_type', 'ent-034', 'contact', 'ent-035', 'influences', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("stakeholder_graph insert with invalid from_entity_type succeeded; want CHECK constraint error")
	}
}

func TestMigrate_StakeholderGraphUserEntityTypeAllowed(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-user', 'SG User WS', 'sg-user-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-user', 'ws-sg-user', 'user', 'user-001', 'contact', 'ent-036', 'reports_to', 0.7, datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("stakeholder_graph insert with user entity_type failed: %v", err)
	}
}

func TestMigrate_StakeholderGraphUserMigrationPreservesRowsAndIndexes(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	applyUpMigrationsThrough(t, db, 32)

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-preserve', 'SG Preserve WS', 'sg-preserve-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-before-034', 'ws-sg-preserve', 'contact', 'ent-037', 'account', 'ent-038', 'approves', 0.8, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("stakeholder_graph pre-034 insert failed: %v", err)
	}

	applyNamedMigration(t, db, "034_stakeholder_graph_user_actor.up.sql")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stakeholder_graph WHERE id = 'sg-before-034'`).Scan(&count); err != nil {
		t.Fatalf("count preserved stakeholder_graph row: %v", err)
	}
	if count != 1 {
		t.Fatalf("preserved stakeholder_graph row count = %d; want 1", count)
	}

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-after-034', 'ws-sg-preserve', 'user', 'user-002', 'contact', 'ent-039', 'reports_to', 0.6, datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("stakeholder_graph insert with user after 034 failed: %v", err)
	}

	assertIndexExists(t, db, "idx_stakeholder_graph_workspace")
	assertIndexExists(t, db, "idx_stakeholder_graph_from")
	assertIndexExists(t, db, "idx_stakeholder_graph_to")
}

func TestMigrate_StakeholderGraphUserDownMigrationRestoresOriginalConstraint(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	applyUpMigrationsThrough(t, db, 32)

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-down', 'SG Down WS', 'sg-down-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-down-before', 'ws-sg-down', 'contact', 'ent-040', 'account', 'ent-041', 'approves', 0.9, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("stakeholder_graph pre-down insert failed: %v", err)
	}

	applyNamedMigration(t, db, "034_stakeholder_graph_user_actor.up.sql")
	applyNamedMigration(t, db, "034_stakeholder_graph_user_actor.down.sql")

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, created_at, updated_at)
		VALUES ('sg-down-invalid', 'ws-sg-down', 'user', 'user-003', 'contact', 'ent-042', 'influences', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Fatal("stakeholder_graph insert with user succeeded after down migration; want CHECK constraint error")
	}
}

func TestMigrate_StakeholderGraphUserDownMigrationFailsWhenUserRowsExist(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	applyUpMigrationsThrough(t, db, 32)

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-down-user', 'SG Down User WS', 'sg-down-user-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	applyNamedMigration(t, db, "034_stakeholder_graph_user_actor.up.sql")

	if _, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-down-user-row', 'ws-sg-down-user', 'user', 'user-004', 'contact', 'ent-043', 'reports_to', 0.4, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("stakeholder_graph insert with user before down migration failed: %v", err)
	}

	err := execSQLFile(db, "034_stakeholder_graph_user_actor.down.sql")
	if err == nil {
		t.Fatal("down migration succeeded with user rows present; want error")
	}
	if !strings.Contains(err.Error(), "user") {
		t.Fatalf("down migration error = %v; want message mentioning user rows", err)
	}
}

// --- trust_score CHECK and UNIQUE constraints ---

func TestMigrate_TrustScoreScoreCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-ts-score', 'TS Score WS', 'ts-score-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-ts-score', 'ws-ts-score', 'contact', 'ent-040', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO trust_score (id, relationship_memory_id, score, last_scored_at, created_at, updated_at)
		VALUES ('ts-score', 'rm-ts-score', 1.5, datetime('now'), datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("trust_score insert with score > 1.0 succeeded; want CHECK constraint error")
	}
}

func TestMigrate_TrustScoreConfidenceCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-ts-conf', 'TS Conf WS', 'ts-conf-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-ts-conf', 'ws-ts-conf', 'deal', 'ent-041', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO trust_score (id, relationship_memory_id, score, confidence, last_scored_at, created_at, updated_at)
		VALUES ('ts-conf', 'rm-ts-conf', 0.5, 'very_high', datetime('now'), datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("trust_score insert with invalid confidence succeeded; want CHECK constraint error")
	}
}

func TestMigrate_TrustScoreDecayFactorCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-ts-decay', 'TS Decay WS', 'ts-decay-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-ts-decay', 'ws-ts-decay', 'case', 'ent-042', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO trust_score (id, relationship_memory_id, score, decay_factor, last_scored_at, created_at, updated_at)
		VALUES ('ts-decay', 'rm-ts-decay', 0.5, 1.5, datetime('now'), datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("trust_score insert with decay_factor > 1.0 succeeded; want CHECK constraint error")
	}
}

func TestMigrate_TrustScoreUniqueConstraint(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-ts-uniq', 'TS Uniq WS', 'ts-uniq-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-ts-uniq', 'ws-ts-uniq', 'lead', 'ent-050', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO trust_score (id, relationship_memory_id, score, last_scored_at, created_at, updated_at)
		VALUES ('ts-uniq-1', 'rm-ts-uniq', 0.7, datetime('now'), datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("first trust_score insert failed: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO trust_score (id, relationship_memory_id, score, last_scored_at, created_at, updated_at)
		VALUES ('ts-uniq-2', 'rm-ts-uniq', 0.8, datetime('now'), datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("second trust_score for same relationship_memory_id succeeded; want UNIQUE constraint error")
	}
}

// --- Valid inserts (happy path) ---

func TestMigrate_RelationshipMemoryValidInsert(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-rm-valid', 'RM Valid WS', 'rm-valid-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, summary, tone, trajectory, created_at, updated_at)
		VALUES ('rm-valid', 'ws-rm-valid', 'contact', 'ent-100', 'Strong relationship', 'positive', 'improving', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("valid relationship_memory insert failed: %v", err)
	}
}

func TestMigrate_TrustScoreValidInsert(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-ts-valid', 'TS Valid WS', 'ts-valid-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-ts-valid', 'ws-ts-valid', 'account', 'ent-101', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO trust_score (id, relationship_memory_id, score, confidence, decay_factor, last_scored_at, created_at, updated_at)
		VALUES ('ts-valid', 'rm-ts-valid', 0.85, 'high', 0.9, datetime('now'), datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("valid trust_score insert failed: %v", err)
	}
}

func TestMigrate_InteractionSignalValidInsert(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-is-valid', 'IS Valid WS', 'is-valid-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO relationship_memory (id, workspace_id, entity_type, entity_id, created_at, updated_at)
		VALUES ('rm-is-valid', 'ws-is-valid', 'contact', 'ent-102', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("relationship_memory insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO interaction_signal (id, relationship_memory_id, signal_type, sentiment, summary, source_entity_type, source_entity_id, occurred_at, created_at)
		VALUES ('is-valid', 'rm-is-valid', 'email', 'positive', 'Customer replied positively', 'case', 'case-001', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("valid interaction_signal insert failed: %v", err)
	}
}

func TestMigrate_StakeholderGraphValidInsert(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sg-valid', 'SG Valid WS', 'sg-valid-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO stakeholder_graph (id, workspace_id, from_entity_type, from_entity_id, to_entity_type, to_entity_id, influence_type, strength, created_at, updated_at)
		VALUES ('sg-valid', 'ws-sg-valid', 'contact', 'ent-103', 'account', 'ent-104', 'approves', 0.8, datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("valid stakeholder_graph insert failed: %v", err)
	}
}

func applyUpMigrationsThrough(t *testing.T, db execer, maxVersion int) {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	sort.Strings(paths)

	for _, path := range paths {
		name := filepath.Base(path)
		if version := migrationVersionFromName(t, name); version > maxVersion {
			continue
		}
		if execErr := execSQLFile(db, name); execErr != nil {
			t.Fatalf("apply migration %s: %v", name, execErr)
		}
	}
}

func applyNamedMigration(t *testing.T, db execer, name string) {
	t.Helper()
	if err := execSQLFile(db, name); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func execSQLFile(db execer, name string) error {
	content, err := os.ReadFile(filepath.Join("migrations", name))
	if err != nil {
		return err
	}
	_, err = db.Exec(string(content))
	return err
}

func migrationVersionFromName(t *testing.T, name string) int {
	t.Helper()

	var version int
	if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
		t.Fatalf("parse migration version from %q: %v", name, err)
	}
	return version
}

func assertIndexExists(t *testing.T, db *sql.DB, name string) {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&count); err != nil {
		t.Fatalf("query index %s: %v", name, err)
	}
	if count != 1 {
		t.Fatalf("index %s count = %d; want 1", name, count)
	}
}
