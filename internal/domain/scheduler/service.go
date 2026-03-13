package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrInvalidScheduleInput = errors.New("invalid schedule input")
)

type ScheduleJobInput struct {
	WorkspaceID string
	JobType     JobType
	Payload     any
	ExecuteAt   time.Time
	SourceID    string
}

type Scheduler interface {
	Schedule(ctx context.Context, job ScheduleJobInput) (*ScheduledJob, error)
	Cancel(ctx context.Context, workspaceID, jobID string) (*ScheduledJob, error)
	CancelBySource(ctx context.Context, workspaceID, sourceID string) (int64, error)
}

type Service struct {
	repo  *Repository
	nowFn func() time.Time
	idFn  func() string
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn:  func() string { return uuid.NewV7().String() },
	}
}

func (s *Service) Schedule(ctx context.Context, job ScheduleJobInput) (*ScheduledJob, error) {
	if err := validateScheduleJobInput(job); err != nil {
		return nil, err
	}

	payload, err := normalizeScheduledPayload(job)
	if err != nil {
		return nil, err
	}

	executeAt := job.ExecuteAt.UTC()
	if executeAt.IsZero() {
		executeAt = s.nowFn()
	}

	return s.repo.Create(ctx, CreateInput{
		ID:          s.idFn(),
		WorkspaceID: job.WorkspaceID,
		JobType:     job.JobType,
		Payload:     payload,
		ExecuteAt:   executeAt,
		Status:      StatusPending,
		SourceID:    job.SourceID,
	})
}

func normalizeScheduledPayload(job ScheduleJobInput) (json.RawMessage, error) {
	if job.JobType != JobTypeWorkflowResume {
		payload, err := json.Marshal(job.Payload)
		if err != nil {
			return nil, fmt.Errorf("%w: payload: %v", ErrInvalidScheduleInput, err)
		}
		return payload, nil
	}

	switch payload := job.Payload.(type) {
	case WorkflowResumePayload:
		raw, err := EncodeWorkflowResumePayload(payload)
		if err != nil {
			return nil, fmt.Errorf("%w: payload: %v", ErrInvalidScheduleInput, err)
		}
		return raw, nil
	case json.RawMessage:
		decoded, err := DecodeWorkflowResumePayload(payload)
		if err != nil {
			return nil, fmt.Errorf("%w: payload: %v", ErrInvalidScheduleInput, err)
		}
		return EncodeWorkflowResumePayload(decoded)
	case []byte:
		decoded, err := DecodeWorkflowResumePayload(payload)
		if err != nil {
			return nil, fmt.Errorf("%w: payload: %v", ErrInvalidScheduleInput, err)
		}
		return EncodeWorkflowResumePayload(decoded)
	default:
		raw, err := json.Marshal(job.Payload)
		if err != nil {
			return nil, fmt.Errorf("%w: payload: %v", ErrInvalidScheduleInput, err)
		}
		decoded, err := DecodeWorkflowResumePayload(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: payload: %v", ErrInvalidScheduleInput, err)
		}
		return EncodeWorkflowResumePayload(decoded)
	}
}

func (s *Service) Cancel(ctx context.Context, workspaceID, jobID string) (*ScheduledJob, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace_id is required", ErrInvalidScheduleInput)
	}
	if jobID == "" {
		return nil, fmt.Errorf("%w: job_id is required", ErrInvalidScheduleInput)
	}
	return s.repo.Cancel(ctx, workspaceID, jobID)
}

func (s *Service) CancelBySource(ctx context.Context, workspaceID, sourceID string) (int64, error) {
	if workspaceID == "" {
		return 0, fmt.Errorf("%w: workspace_id is required", ErrInvalidScheduleInput)
	}
	if sourceID == "" {
		return 0, fmt.Errorf("%w: source_id is required", ErrInvalidScheduleInput)
	}
	return s.repo.CancelBySource(ctx, workspaceID, sourceID)
}

func validateScheduleJobInput(job ScheduleJobInput) error {
	switch {
	case job.WorkspaceID == "":
		return fmt.Errorf("%w: workspace_id is required", ErrInvalidScheduleInput)
	case job.JobType == "":
		return fmt.Errorf("%w: job_type is required", ErrInvalidScheduleInput)
	case job.SourceID == "":
		return fmt.Errorf("%w: source_id is required", ErrInvalidScheduleInput)
	default:
		return nil
	}
}
