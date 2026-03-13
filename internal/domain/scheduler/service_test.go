package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestServiceScheduleCreatesPendingJob(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	fixedNow := time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC)
	svc.nowFn = func() time.Time { return fixedNow }
	svc.idFn = func() string { return "job-service-1" }

	job, err := svc.Schedule(context.Background(), ScheduleJobInput{
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload: WorkflowResumePayload{
			WorkflowID:      "wf-1",
			RunID:           "run-1",
			ResumeStepIndex: 3,
		},
		ExecuteAt: fixedNow.Add(48 * time.Hour),
		SourceID:  "wf-1",
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if job.ID != "job-service-1" {
		t.Fatalf("id = %s, want job-service-1", job.ID)
	}
	if job.Status != StatusPending {
		t.Fatalf("status = %s, want %s", job.Status, StatusPending)
	}
	if job.JobType != JobTypeWorkflowResume {
		t.Fatalf("jobType = %s, want %s", job.JobType, JobTypeWorkflowResume)
	}
}

func TestServiceScheduleSupportsZeroTimeAsYield(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	fixedNow := time.Date(2026, 3, 12, 21, 0, 0, 0, time.UTC)
	svc.nowFn = func() time.Time { return fixedNow }
	svc.idFn = func() string { return "job-yield" }

	job, err := svc.Schedule(context.Background(), ScheduleJobInput{
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload: WorkflowResumePayload{
			WorkflowID:      "wf-1",
			RunID:           "run-1",
			ResumeStepIndex: 4,
		},
		SourceID: "wf-1",
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if !job.ExecuteAt.Equal(fixedNow) {
		t.Fatalf("executeAt = %s, want %s", job.ExecuteAt, fixedNow)
	}
}

func TestServiceScheduleRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	_, err := svc.Schedule(context.Background(), ScheduleJobInput{
		WorkspaceID: "",
		JobType:     JobTypeWorkflowResume,
		SourceID:    "wf-1",
		Payload:     json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrInvalidScheduleInput) {
		t.Fatalf("expected ErrInvalidScheduleInput, got %v", err)
	}
}

func TestServiceScheduleRejectsInvalidWorkflowResumePayload(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	_, err := svc.Schedule(context.Background(), ScheduleJobInput{
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		SourceID:    "wf-1",
		Payload: WorkflowResumePayload{
			WorkflowID:      "",
			RunID:           "run-1",
			ResumeStepIndex: 0,
		},
	})
	if !errors.Is(err, ErrInvalidScheduleInput) {
		t.Fatalf("expected ErrInvalidScheduleInput, got %v", err)
	}
}

func TestServiceCancelCancelsPendingJob(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	if _, err := repo.Create(context.Background(), CreateInput{
		ID:          "job-cancel-service",
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{"run_id":"run-1"}`),
		ExecuteAt:   time.Now().UTC().Add(1 * time.Hour),
		Status:      StatusPending,
		SourceID:    "wf-1",
	}); err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	job, err := svc.Cancel(context.Background(), "ws_test", "job-cancel-service")
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if job.Status != StatusCancelled {
		t.Fatalf("status = %s, want %s", job.Status, StatusCancelled)
	}
}

func TestServiceCancelBySourceCancelsPendingJobs(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	for _, input := range []CreateInput{
		{
			ID:          "job-cancel-source-1",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-1"}`),
			ExecuteAt:   time.Now().UTC().Add(1 * time.Hour),
			Status:      StatusPending,
			SourceID:    "wf-cancel-source",
		},
		{
			ID:          "job-cancel-source-2",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-2"}`),
			ExecuteAt:   time.Now().UTC().Add(2 * time.Hour),
			Status:      StatusPending,
			SourceID:    "wf-cancel-source",
		},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	rows, err := svc.CancelBySource(context.Background(), "ws_test", "wf-cancel-source")
	if err != nil {
		t.Fatalf("CancelBySource() error = %v", err)
	}
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestServiceCancelRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	_, err := svc.Cancel(context.Background(), "", "job-1")
	if !errors.Is(err, ErrInvalidScheduleInput) {
		t.Fatalf("expected ErrInvalidScheduleInput, got %v", err)
	}
}
