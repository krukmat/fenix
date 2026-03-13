package scheduler

import (
	"context"
	"errors"
	"time"
)

var (
	ErrWorkerMissingRepository = errors.New("scheduler worker requires repository")
	ErrWorkerMissingHandler    = errors.New("scheduler worker requires job handler")
)

type JobHandler func(ctx context.Context, job *ScheduledJob) error

type workerJobResult struct {
	err error
}

type Worker struct {
	repo           *Repository
	handler        JobHandler
	pollInterval   time.Duration
	maxConcurrency int
	nowFn          func() time.Time
	sleepFn        func(context.Context, time.Duration) error
}

func NewWorker(repo *Repository, handler JobHandler) *Worker {
	return &Worker{
		repo:           repo,
		handler:        handler,
		pollInterval:   10 * time.Second,
		maxConcurrency: 10,
		nowFn:          func() time.Time { return time.Now().UTC() },
		sleepFn: func(ctx context.Context, delay time.Duration) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				return nil
			}
		},
	}
}

func (w *Worker) Start(ctx context.Context) error {
	if w.repo == nil {
		return ErrWorkerMissingRepository
	}
	if w.handler == nil {
		return ErrWorkerMissingHandler
	}
	if w.maxConcurrency <= 0 {
		w.maxConcurrency = 10
	}
	if w.pollInterval <= 0 {
		w.pollInterval = 10 * time.Second
	}

	for {
		if err := w.RunCycle(ctx); err != nil && !errors.Is(err, context.Canceled) {
			// worker is best-effort at this phase; next cycle may still progress
		}
		if err := w.sleepFn(ctx, w.pollInterval); err != nil {
			return err
		}
	}
}

func (w *Worker) RunCycle(ctx context.Context) error {
	if w.repo == nil {
		return ErrWorkerMissingRepository
	}
	if w.handler == nil {
		return ErrWorkerMissingHandler
	}

	limit := w.maxConcurrency
	if limit <= 0 {
		limit = 10
	}

	jobs, err := w.repo.ListDue(ctx, w.nowFn(), limit)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}

	resultCh := make(chan workerJobResult, len(jobs))
	sem := make(chan struct{}, limit)
	done := make(chan struct{}, len(jobs))

	for _, job := range jobs {
		job := job
		sem <- struct{}{}
		go func() {
			defer func() {
				<-sem
				done <- struct{}{}
			}()
			resultCh <- w.processJob(ctx, job)
		}()
	}

	for range jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
		}
	}

	close(resultCh)
	for result := range resultCh {
		if result.err != nil {
			return result.err
		}
	}

	return nil
}

func (w *Worker) processJob(ctx context.Context, job *ScheduledJob) workerJobResult {
	handlerErr := w.handler(ctx, job)
	_, markErr := w.repo.MarkExecuted(ctx, job.WorkspaceID, job.ID, w.nowFn())
	if markErr != nil {
		return workerJobResult{err: markErr}
	}
	if handlerErr != nil {
		return workerJobResult{err: handlerErr}
	}
	return workerJobResult{}
}
