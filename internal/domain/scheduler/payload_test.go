package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkflowResumePayloadEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()

	payload := WorkflowResumePayload{
		WorkflowID:      "wf-1",
		RunID:           "run-1",
		ResumeStepIndex: 3,
	}

	raw, err := EncodeWorkflowResumePayload(payload)
	if err != nil {
		t.Fatalf("EncodeWorkflowResumePayload() error = %v", err)
	}
	decoded, err := DecodeWorkflowResumePayload(raw)
	if err != nil {
		t.Fatalf("DecodeWorkflowResumePayload() error = %v", err)
	}
	if decoded != payload {
		t.Fatalf("decoded = %#v, want %#v", decoded, payload)
	}
}

func TestWorkflowResumePayloadRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := EncodeWorkflowResumePayload(WorkflowResumePayload{
		WorkflowID:      "",
		RunID:           "run-1",
		ResumeStepIndex: 0,
	})
	if !errors.Is(err, ErrInvalidWorkflowResumePayload) {
		t.Fatalf("expected ErrInvalidWorkflowResumePayload, got %v", err)
	}
}

func TestDecodeWorkflowResumePayloadRejectsInvalidRawPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  []byte
	}{
		{name: "empty", raw: nil},
		{name: "invalid json", raw: []byte(`{`)},
		{name: "missing run id", raw: []byte(`{"workflow_id":"wf-1","resume_step_index":1}`)},
		{name: "negative index", raw: []byte(`{"workflow_id":"wf-1","run_id":"run-1","resume_step_index":-1}`)},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := DecodeWorkflowResumePayload(tc.raw)
			if err == nil {
				t.Fatal("DecodeWorkflowResumePayload() expected error")
			}
		})
	}
}

func TestWorkflowResumePayloadPersistsThroughScheduledJob(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	svc.idFn = func() string { return "job-payload-1" }
	svc.nowFn = func() time.Time { return time.Date(2026, 3, 12, 22, 0, 0, 0, time.UTC) }

	payload := WorkflowResumePayload{
		WorkflowID:      "wf-persisted",
		RunID:           "run-persisted",
		ResumeStepIndex: 4,
	}
	raw, err := EncodeWorkflowResumePayload(payload)
	if err != nil {
		t.Fatalf("EncodeWorkflowResumePayload() error = %v", err)
	}

	job, err := svc.Schedule(context.Background(), ScheduleJobInput{
		WorkspaceID: "ws_test",
		JobType:     JobTypeWorkflowResume,
		Payload:     raw,
		ExecuteAt:   svc.nowFn().Add(1 * time.Hour),
		SourceID:    payload.WorkflowID,
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	decoded, err := DecodeWorkflowResumePayload(job.Payload)
	if err != nil {
		t.Fatalf("DecodeWorkflowResumePayload(job.Payload) error = %v", err)
	}
	if decoded != payload {
		t.Fatalf("decoded = %#v, want %#v", decoded, payload)
	}
}
