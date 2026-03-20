package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerRunCycleProcessesDueJobsOnly(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, 3, 12, 22, 0, 0, 0, time.UTC)

	for _, input := range []CreateInput{
		{
			ID:          "job-due-worker",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-1"}`),
			ExecuteAt:   now.Add(-1 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-1",
		},
		{
			ID:          "job-future-worker",
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run-2"}`),
			ExecuteAt:   now.Add(1 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-2",
		},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	var processed []string
	var mu sync.Mutex
	worker := NewWorker(repo, func(_ context.Context, job *ScheduledJob) error {
		mu.Lock()
		defer mu.Unlock()
		processed = append(processed, job.ID)
		return nil
	})
	worker.nowFn = func() time.Time { return now }
	worker.maxConcurrency = 10

	if err := worker.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error = %v", err)
	}

	if len(processed) != 1 || processed[0] != "job-due-worker" {
		t.Fatalf("processed = %v, want [job-due-worker]", processed)
	}
}

func TestWorkerRunCycleRespectsConcurrencyLimit(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, 3, 12, 22, 30, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		if _, err := repo.Create(context.Background(), CreateInput{
			ID:          "job-limit-" + string(rune('a'+i)),
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run"}`),
			ExecuteAt:   now.Add(-1 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-limit",
		}); err != nil {
			t.Fatalf("repo.Create() error = %v", err)
		}
	}

	var active int32
	var maxSeen int32
	worker := NewWorker(repo, func(_ context.Context, _ *ScheduledJob) error {
		current := atomic.AddInt32(&active, 1)
		for {
			prev := atomic.LoadInt32(&maxSeen)
			if current <= prev || atomic.CompareAndSwapInt32(&maxSeen, prev, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return nil
	})
	worker.nowFn = func() time.Time { return now }
	worker.maxConcurrency = 2

	if err := worker.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error = %v", err)
	}
	if maxSeen > 2 {
		t.Fatalf("max concurrency = %d, want <= 2", maxSeen)
	}
}

func TestWorkerRunCycleReturnsHandlerError(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, 3, 12, 23, 0, 0, 0, time.UTC)

	if _, err := repo.Create(context.Background(), CreateInput{
		ID:          "job-error",
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{"run_id":"run-error"}`),
		ExecuteAt:   now.Add(-1 * time.Minute),
		Status:      StatusPending,
		SourceID:    "wf-error",
	}); err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	wantErr := errors.New("handler boom")
	worker := NewWorker(repo, func(_ context.Context, _ *ScheduledJob) error {
		return wantErr
	})
	worker.nowFn = func() time.Time { return now }

	err := worker.RunCycle(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunCycle() err = %v, want %v", err, wantErr)
	}

	stored, err := repo.GetByID(context.Background(), "ws_test", "job-error")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Status != StatusExecuted {
		t.Fatalf("status = %s, want %s", stored.Status, StatusExecuted)
	}
	if stored.ExecutedAt == nil {
		t.Fatal("ExecutedAt = nil, want timestamp")
	}
}

func TestWorkerStartRequiresDependencies(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	worker := NewWorker(nil, func(context.Context, *ScheduledJob) error { return nil })
	if err := worker.Start(context.Background()); !errors.Is(err, ErrWorkerMissingRepository) {
		t.Fatalf("expected ErrWorkerMissingRepository, got %v", err)
	}

	worker = NewWorker(repo, nil)
	if err := worker.Start(context.Background()); !errors.Is(err, ErrWorkerMissingHandler) {
		t.Fatalf("expected ErrWorkerMissingHandler, got %v", err)
	}
}

func TestWorkerStartReturnsSleepErrorAfterCycle(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	worker := NewWorker(repo, func(context.Context, *ScheduledJob) error { return nil })
	worker.pollInterval = 0
	worker.maxConcurrency = 0
	worker.sleepFn = func(context.Context, time.Duration) error {
		return context.Canceled
	}

	err := worker.Start(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start() err = %v, want context.Canceled", err)
	}
	if worker.pollInterval != 10*time.Second {
		t.Fatalf("pollInterval = %s, want 10s", worker.pollInterval)
	}
	if worker.maxConcurrency != 10 {
		t.Fatalf("maxConcurrency = %d, want 10", worker.maxConcurrency)
	}
}

func TestWaitForJobsReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForJobs(ctx, []*ScheduledJob{{ID: "job-1"}}, make(chan struct{}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForJobs() err = %v, want context.Canceled", err)
	}
}

func TestWorkerRunCycleDrainsDueJobsAcrossMultipleCycles(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		if _, err := repo.Create(context.Background(), CreateInput{
			ID:          "job-drain-" + string(rune('a'+i)),
			WorkspaceID: "ws_test",
			JobType:     JobTypeWorkflowResume,
			Payload:     json.RawMessage(`{"run_id":"run"}`),
			ExecuteAt:   now.Add(-1 * time.Minute),
			Status:      StatusPending,
			SourceID:    "wf-drain",
		}); err != nil {
			t.Fatalf("repo.Create() error = %v", err)
		}
	}

	var processed []string
	var mu sync.Mutex
	worker := NewWorker(repo, func(_ context.Context, job *ScheduledJob) error {
		mu.Lock()
		defer mu.Unlock()
		processed = append(processed, job.ID)
		return nil
	})
	worker.nowFn = func() time.Time { return now }
	worker.maxConcurrency = 2

	for cycle := 1; cycle <= 3; cycle++ {
		if err := worker.RunCycle(context.Background()); err != nil {
			t.Fatalf("RunCycle(%d) error = %v", cycle, err)
		}
	}

	if len(processed) != 5 {
		t.Fatalf("len(processed) = %d, want 5", len(processed))
	}

	due, err := repo.ListDue(context.Background(), now.Add(1*time.Minute), 10)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("len(due) = %d, want 0 after draining cycles", len(due))
	}
}
