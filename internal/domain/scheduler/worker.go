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
	if err := w.validate(); err != nil {
		return err
	}
	w.normalizeConfig()

	for {
		if cycleErr := w.RunCycle(ctx); cycleErr != nil {
			if errors.Is(cycleErr, context.Canceled) {
				return cycleErr
			}
		}
		if err := w.sleepFn(ctx, w.pollInterval); err != nil {
			return err
		}
	}
}

func (w *Worker) RunCycle(ctx context.Context) error {
	limit, jobs, err := w.loadDueJobs(ctx)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}

	return w.runJobs(ctx, jobs, limit)
}

func (w *Worker) validate() error {
	if w.repo == nil {
		return ErrWorkerMissingRepository
	}
	if w.handler == nil {
		return ErrWorkerMissingHandler
	}
	return nil
}

func (w *Worker) normalizeConfig() {
	if w.maxConcurrency <= 0 {
		w.maxConcurrency = 10
	}
	if w.pollInterval <= 0 {
		w.pollInterval = 10 * time.Second
	}
}

func (w *Worker) loadDueJobs(ctx context.Context) (int, []*ScheduledJob, error) {
	if err := w.validate(); err != nil {
		return 0, nil, err
	}
	limit := w.maxConcurrency
	if limit <= 0 {
		limit = 10
	}
	jobs, err := w.repo.ListDue(ctx, w.nowFn(), limit)
	if err != nil {
		return 0, nil, err
	}
	return limit, jobs, nil
}

func (w *Worker) runJobs(ctx context.Context, jobs []*ScheduledJob, limit int) error {
	resultCh := make(chan workerJobResult, len(jobs))
	sem := make(chan struct{}, limit)
	done := make(chan struct{}, len(jobs))

	for _, job := range jobs {
		w.startJob(ctx, job, sem, done, resultCh)
	}

	if err := waitForJobs(ctx, jobs, done); err != nil {
		return err
	}
	return collectJobResults(resultCh)
}

func (w *Worker) startJob(ctx context.Context, job *ScheduledJob, sem chan struct{}, done chan struct{}, resultCh chan workerJobResult) {
	sem <- struct{}{}
	go func(currentJob *ScheduledJob) {
		defer func() {
			<-sem
			done <- struct{}{}
		}()
		resultCh <- w.processJob(ctx, currentJob)
	}(job)
}

func waitForJobs(ctx context.Context, jobs []*ScheduledJob, done chan struct{}) error {
	for range jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
		}
	}
	return nil
}

func collectJobResults(resultCh chan workerJobResult) error {
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
