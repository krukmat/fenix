package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func TestRepository_CreateAndGetByID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	executeAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)

	created, err := repo.Create(context.Background(), CreateInput{
		ID:          "job-1",
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{"workflow_id":"wf-1","run_id":"run-1","resume_step_index":3}`),
		ExecuteAt:   executeAt,
		Status:      StatusPending,
		SourceID:    "wf-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("status = %s, want %s", created.Status, StatusPending)
	}

	got, err := repo.GetByID(context.Background(), "ws_test", "job-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.JobType != JobTypeWorkflowResume {
		t.Fatalf("jobType = %s, want %s", got.JobType, JobTypeWorkflowResume)
	}
	if got.SourceID != "wf-1" {
		t.Fatalf("sourceID = %s, want wf-1", got.SourceID)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.GetByID(context.Background(), "ws_test", "missing")
	if !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("expected ErrScheduledJobNotFound, got %v", err)
	}
}

func TestRepository_ListDueHonorsStatusAndLimit(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	inputs := []CreateInput{
		{
			ID:          "job-due-1",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-1"}`),
			ExecuteAt:   now.Add(-1 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-1",
		},
		{
			ID:          "job-due-2",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-2"}`),
			ExecuteAt:   now.Add(-30 * time.Second),
			Status:      StatusPending,
			SourceID:    "wf-2",
		},
		{
			ID:          "job-future",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-3"}`),
			ExecuteAt:   now.Add(1 * time.Hour),
			Status:      StatusPending,
			SourceID:    "wf-3",
		},
		{
			ID:          "job-executed",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-4"}`),
			ExecuteAt:   now.Add(-2 * time.Minute),
			Status:      StatusExecuted,
			SourceID:    "wf-4",
			ExecutedAt:  ptrTime(now.Add(-1 * time.Minute)),
		},
	}

	for _, input := range inputs {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("Create(%s) error = %v", input.ID, err)
		}
	}

	due, err := repo.ListDue(context.Background(), now, 1)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("len(due) = %d, want 1", len(due))
	}
	if due[0].ID != "job-due-1" {
		t.Fatalf("first due job = %s, want job-due-1", due[0].ID)
	}
}

func TestRepository_MarkExecuted(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := repo.Create(context.Background(), CreateInput{
		ID:          "job-exec",
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{"run_id":"run-1"}`),
		ExecuteAt:   now,
		Status:      StatusPending,
		SourceID:    "wf-1",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	executed, err := repo.MarkExecuted(context.Background(), "ws_test", "job-exec", now.Add(5*time.Second))
	if err != nil {
		t.Fatalf("MarkExecuted() error = %v", err)
	}
	if executed.Status != StatusExecuted {
		t.Fatalf("status = %s, want %s", executed.Status, StatusExecuted)
	}
	if executed.ExecutedAt == nil {
		t.Fatal("executedAt = nil, want non-nil")
	}
}

func TestRepository_MarkExecuted_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.MarkExecuted(context.Background(), "ws_test", "missing", time.Now().UTC())
	if !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("expected ErrScheduledJobNotFound, got %v", err)
	}
}

func TestRepository_Cancel(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := repo.Create(context.Background(), CreateInput{
		ID:          "job-cancel",
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{"run_id":"run-1"}`),
		ExecuteAt:   now.Add(10 * time.Minute),
		Status:      StatusPending,
		SourceID:    "wf-1",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	cancelled, err := repo.Cancel(context.Background(), "ws_test", "job-cancel")
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("status = %s, want %s", cancelled.Status, StatusCancelled)
	}
}

func TestRepository_Cancel_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Cancel(context.Background(), "ws_test", "missing")
	if !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("expected ErrScheduledJobNotFound, got %v", err)
	}
}

func TestRepository_CancelBySource(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	for _, input := range []CreateInput{
		{
			ID:          "job-source-1",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-1"}`),
			ExecuteAt:   now.Add(10 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-source",
		},
		{
			ID:          "job-source-2",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-2"}`),
			ExecuteAt:   now.Add(11 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-source",
		},
		{
			ID:          "job-other",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-3"}`),
			ExecuteAt:   now.Add(12 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-other",
		},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("Create(%s) error = %v", input.ID, err)
		}
	}

	rows, err := repo.CancelBySource(context.Background(), "ws_test", "wf-source")
	if err != nil {
		t.Fatalf("CancelBySource() error = %v", err)
	}
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}

	other, err := repo.GetByID(context.Background(), "ws_test", "job-other")
	if err != nil {
		t.Fatalf("GetByID(job-other) error = %v", err)
	}
	if other.Status != StatusPending {
		t.Fatalf("other status = %s, want %s", other.Status, StatusPending)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = isqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	if _, err = db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws_test', 'Scheduler Test', 'scheduler-test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
