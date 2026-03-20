package agent

import (
	"context"
	"strings"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type Judge interface {
	Verify(ctx context.Context, workflow *workflowdomain.Workflow) (*JudgeResult, error)
}

type JudgeResult struct {
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations,omitempty"`
	Warnings   []Warning   `json:"warnings,omitempty"`
}

type Violation struct {
	CheckID     int    `json:"checkId,omitempty"`
	Code        string `json:"code,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
}

type Warning struct {
	CheckID     int    `json:"checkId,omitempty"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
}

func NewJudgeResult(violations []Violation, warnings []Warning) *JudgeResult {
	result := &JudgeResult{
		Violations: cloneViolations(violations),
		Warnings:   cloneWarnings(warnings),
	}
	result.RecomputePassed()
	return result
}

func (r *JudgeResult) RecomputePassed() {
	if r == nil {
		return
	}
	r.Passed = len(r.Violations) == 0
}

func (r *JudgeResult) AddViolation(v Violation) {
	if r == nil {
		return
	}
	r.Violations = append(r.Violations, normalizeViolation(v))
	r.Passed = false
}

func (r *JudgeResult) AddWarning(w Warning) {
	if r == nil {
		return
	}
	r.Warnings = append(r.Warnings, normalizeWarning(w))
	if len(r.Violations) == 0 {
		r.Passed = true
	}
}

func normalizeViolation(v Violation) Violation {
	v.Code = strings.TrimSpace(v.Code)
	v.Type = strings.TrimSpace(v.Type)
	v.Description = strings.TrimSpace(v.Description)
	v.Location = strings.TrimSpace(v.Location)
	return v
}

func normalizeWarning(w Warning) Warning {
	w.Code = strings.TrimSpace(w.Code)
	w.Description = strings.TrimSpace(w.Description)
	w.Location = strings.TrimSpace(w.Location)
	return w
}

func cloneViolations(in []Violation) []Violation {
	if len(in) == 0 {
		return nil
	}
	out := make([]Violation, len(in))
	copy(out, in)
	for i := range out {
		out[i] = normalizeViolation(out[i])
	}
	return out
}

func cloneWarnings(in []Warning) []Warning {
	if len(in) == 0 {
		return nil
	}
	out := make([]Warning, len(in))
	copy(out, in)
	for i := range out {
		out[i] = normalizeWarning(out[i])
	}
	return out
}
