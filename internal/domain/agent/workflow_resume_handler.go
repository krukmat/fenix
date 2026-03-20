package agent

import (
	"context"
	"errors"

	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
)

var (
	ErrWorkflowResumeHandlerMissingRunner = errors.New("workflow resume handler requires dsl runner")
	ErrWorkflowResumeJobInvalid           = errors.New("workflow resume job is invalid")
)

type WorkflowResumeHandler struct {
	runner *DSLRunner
	rc     *RunContext
}

func NewWorkflowResumeHandler(runner *DSLRunner, rc *RunContext) *WorkflowResumeHandler {
	return &WorkflowResumeHandler{
		runner: runner,
		rc:     rc,
	}
}

func (h *WorkflowResumeHandler) Handle(ctx context.Context, job *schedulerdomain.ScheduledJob) error {
	if h == nil || h.runner == nil {
		return ErrWorkflowResumeHandlerMissingRunner
	}
	if job == nil || job.JobType != schedulerdomain.JobTypeWorkflowResume || job.WorkspaceID == "" {
		return ErrWorkflowResumeJobInvalid
	}

	payload, err := schedulerdomain.DecodeWorkflowResumePayload(job.Payload)
	if err != nil {
		return err
	}
	_, err = h.runner.Resume(ctx, h.rc, job.WorkspaceID, payload)
	return err
}
