package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrScheduledJobNotFound = errors.New("scheduled job not found")
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusExecuted  Status = "executed"
	StatusCancelled Status = "cancelled"
)

type JobType string

const (
	JobTypeWorkflowResume JobType = "workflow_resume"
)

type ScheduledJob struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspaceId"`
	JobType     JobType         `json:"jobType"`
	Payload     json.RawMessage `json:"payload"`
	ExecuteAt   time.Time       `json:"executeAt"`
	Status      Status          `json:"status"`
	SourceID    string          `json:"sourceId"`
	CreatedAt   time.Time       `json:"createdAt"`
	ExecutedAt  *time.Time      `json:"executedAt,omitempty"`
}

type CreateInput struct {
	ID          string
	WorkspaceID string
	JobType     JobType
	Payload     json.RawMessage
	ExecuteAt   time.Time
	Status      Status
	SourceID    string
	ExecutedAt  *time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (*ScheduledJob, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_job (
			id, workspace_id, job_type, payload, execute_at, status, source_id, created_at, executed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, workspace_id, job_type, payload, execute_at, status, source_id, created_at, executed_at
	`,
		input.ID,
		input.WorkspaceID,
		string(input.JobType),
		normalizeJSON(input.Payload, []byte("{}")),
		formatTime(input.ExecuteAt),
		string(input.Status),
		input.SourceID,
		nowRFC3339(),
		formatOptionalTime(input.ExecutedAt),
	)

	out, err := scanScheduledJob(row)
	if err != nil {
		return nil, fmt.Errorf("create scheduled job: %w", err)
	}
	return out, nil
}

func (r *Repository) GetByID(ctx context.Context, workspaceID, jobID string) (*ScheduledJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, job_type, payload, execute_at, status, source_id, created_at, executed_at
		FROM scheduled_job
		WHERE id = ? AND workspace_id = ?
		LIMIT 1
	`, jobID, workspaceID)

	out, err := scanScheduledJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScheduledJobNotFound
		}
		return nil, fmt.Errorf("get scheduled job by id: %w", err)
	}
	return out, nil
}

func (r *Repository) ListDue(ctx context.Context, now time.Time, limit int) ([]*ScheduledJob, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, workspace_id, job_type, payload, execute_at, status, source_id, created_at, executed_at
		FROM scheduled_job
		WHERE status = 'pending' AND execute_at <= ?
		ORDER BY execute_at ASC, created_at ASC
		LIMIT ?
	`, formatTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("list due scheduled jobs: %w", err)
	}
	defer rows.Close()

	return scanScheduledJobRows(rows)
}

func (r *Repository) MarkExecuted(ctx context.Context, workspaceID, jobID string, executedAt time.Time) (*ScheduledJob, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE scheduled_job
		SET status = 'executed', executed_at = ?
		WHERE id = ? AND workspace_id = ?
		RETURNING
			id, workspace_id, job_type, payload, execute_at, status, source_id, created_at, executed_at
	`, formatTime(executedAt), jobID, workspaceID)

	out, err := scanScheduledJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScheduledJobNotFound
		}
		return nil, fmt.Errorf("mark scheduled job executed: %w", err)
	}
	return out, nil
}

func (r *Repository) Cancel(ctx context.Context, workspaceID, jobID string) (*ScheduledJob, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE scheduled_job
		SET status = 'cancelled'
		WHERE id = ? AND workspace_id = ? AND status = 'pending'
		RETURNING
			id, workspace_id, job_type, payload, execute_at, status, source_id, created_at, executed_at
	`, jobID, workspaceID)

	out, err := scanScheduledJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScheduledJobNotFound
		}
		return nil, fmt.Errorf("cancel scheduled job: %w", err)
	}
	return out, nil
}

func (r *Repository) CancelBySource(ctx context.Context, workspaceID, sourceID string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_job
		SET status = 'cancelled'
		WHERE workspace_id = ? AND source_id = ? AND status = 'pending'
	`, workspaceID, sourceID)
	if err != nil {
		return 0, fmt.Errorf("cancel scheduled jobs by source: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("cancel scheduled jobs by source rows affected: %w", err)
	}
	return rows, nil
}

func scanScheduledJob(row interface{ Scan(dest ...any) error }) (*ScheduledJob, error) {
	var (
		out            ScheduledJob
		jobType        string
		status         string
		payload        []byte
		executeAt      string
		createdAt      string
		executedAtText sql.NullString
	)

	err := row.Scan(
		&out.ID,
		&out.WorkspaceID,
		&jobType,
		&payload,
		&executeAt,
		&status,
		&out.SourceID,
		&createdAt,
		&executedAtText,
	)
	if err != nil {
		return nil, err
	}

	executeParsed, err := parseTime(executeAt)
	if err != nil {
		return nil, fmt.Errorf("parse scheduled job execute_at: %w", err)
	}
	createdParsed, err := parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse scheduled job created_at: %w", err)
	}

	out.JobType = JobType(jobType)
	out.Payload = normalizeJSON(payload, []byte("{}"))
	out.ExecuteAt = executeParsed
	out.Status = Status(status)
	out.CreatedAt = createdParsed
	if executedAtText.Valid {
		parsed, parseErr := parseTime(executedAtText.String)
		if parseErr != nil {
			return nil, fmt.Errorf("parse scheduled job executed_at: %w", parseErr)
		}
		out.ExecutedAt = &parsed
	}

	return &out, nil
}

func scanScheduledJobRows(rows *sql.Rows) ([]*ScheduledJob, error) {
	out := make([]*ScheduledJob, 0)
	for rows.Next() {
		job, err := scanScheduledJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

