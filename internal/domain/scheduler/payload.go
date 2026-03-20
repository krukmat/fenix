package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidWorkflowResumePayload = errors.New("invalid workflow resume payload")

type WorkflowResumePayload struct {
	WorkflowID      string `json:"workflow_id"`
	RunID           string `json:"run_id"`
	ResumeStepIndex int    `json:"resume_step_index"`
}

func (p WorkflowResumePayload) Validate() error {
	switch {
	case strings.TrimSpace(p.WorkflowID) == "":
		return fmt.Errorf("%w: workflow_id is required", ErrInvalidWorkflowResumePayload)
	case strings.TrimSpace(p.RunID) == "":
		return fmt.Errorf("%w: run_id is required", ErrInvalidWorkflowResumePayload)
	case p.ResumeStepIndex < 0:
		return fmt.Errorf("%w: resume_step_index must be >= 0", ErrInvalidWorkflowResumePayload)
	default:
		return nil
	}
}

func EncodeWorkflowResumePayload(payload WorkflowResumePayload) (json.RawMessage, error) {
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func DecodeWorkflowResumePayload(raw json.RawMessage) (WorkflowResumePayload, error) {
	var payload WorkflowResumePayload
	if len(raw) == 0 {
		return payload, fmt.Errorf("%w: payload is required", ErrInvalidWorkflowResumePayload)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, err
	}
	if err := payload.Validate(); err != nil {
		return payload, err
	}
	return payload, nil
}
