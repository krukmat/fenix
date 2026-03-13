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

const (
	errScheduleWorkspaceRequired = "%w: workspace_id is required"
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
		return marshalScheduledPayload(job.Payload)
	}

	switch payload := job.Payload.(type) {
	case WorkflowResumePayload:
		return encodeResumePayload(payload)
	case json.RawMessage:
		return decodeAndEncodeResumePayload(payload)
	case []byte:
		return decodeAndEncodeResumePayload(payload)
	default:
		return normalizeGenericResumePayload(job.Payload)
	}
}

func marshalScheduledPayload(payload any) (json.RawMessage, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, invalidSchedulePayload(err)
	}
	return raw, nil
}

func encodeResumePayload(payload WorkflowResumePayload) (json.RawMessage, error) {
	raw, err := EncodeWorkflowResumePayload(payload)
	if err != nil {
		return nil, invalidSchedulePayload(err)
	}
	return raw, nil
}

func decodeAndEncodeResumePayload(payload []byte) (json.RawMessage, error) {
	decoded, err := DecodeWorkflowResumePayload(payload)
	if err != nil {
		return nil, invalidSchedulePayload(err)
	}
	return encodeResumePayload(decoded)
}

func normalizeGenericResumePayload(payload any) (json.RawMessage, error) {
	raw, err := marshalScheduledPayload(payload)
	if err != nil {
		return nil, err
	}
	return decodeAndEncodeResumePayload(raw)
}

func invalidSchedulePayload(err error) error {
	return fmt.Errorf("%w: payload: %w", ErrInvalidScheduleInput, err)
}

func (s *Service) Cancel(ctx context.Context, workspaceID, jobID string) (*ScheduledJob, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf(errScheduleWorkspaceRequired, ErrInvalidScheduleInput)
	}
	if jobID == "" {
		return nil, fmt.Errorf("%w: job_id is required", ErrInvalidScheduleInput)
	}
	return s.repo.Cancel(ctx, workspaceID, jobID)
}

func (s *Service) CancelBySource(ctx context.Context, workspaceID, sourceID string) (int64, error) {
	if workspaceID == "" {
		return 0, fmt.Errorf(errScheduleWorkspaceRequired, ErrInvalidScheduleInput)
	}
	if sourceID == "" {
		return 0, fmt.Errorf("%w: source_id is required", ErrInvalidScheduleInput)
	}
	return s.repo.CancelBySource(ctx, workspaceID, sourceID)
}

func validateScheduleJobInput(job ScheduleJobInput) error {
	switch {
	case job.WorkspaceID == "":
		return fmt.Errorf(errScheduleWorkspaceRequired, ErrInvalidScheduleInput)
	case job.JobType == "":
		return fmt.Errorf("%w: job_type is required", ErrInvalidScheduleInput)
	case job.SourceID == "":
		return fmt.Errorf("%w: source_id is required", ErrInvalidScheduleInput)
	default:
		return nil
	}
}
