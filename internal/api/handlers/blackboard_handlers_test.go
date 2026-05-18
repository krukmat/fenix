package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
)

type blackboardPipelineStub struct {
	outcome *blackboard.ExecutionOutcome
	err     error
	calls   int
	lastCW  string
}

func (s *blackboardPipelineStub) RunPipeline(_ context.Context, cognitiveWorkspaceID string) (*blackboard.ExecutionOutcome, error) {
	s.calls++
	s.lastCW = cognitiveWorkspaceID
	return s.outcome, s.err
}

func TestBlackboardHandler_RunPipeline_ForbiddenByAuthorizer(t *testing.T) {
	t.Parallel()

	handler := NewBlackboardHandlerWithAuthorizer(&blackboardPipelineStub{}, &toolAuthzStub{allow: false})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blackboard/cw-1/plan", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, "user-1"))
	req = withRouteParam(req, "cwID", "cw-1")
	rr := httptest.NewRecorder()

	handler.RunPipeline(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

func TestBlackboardHandler_RunPipeline_ReturnsJSONOutcome(t *testing.T) {
	t.Parallel()

	pipeline := &blackboardPipelineStub{outcome: &blackboard.ExecutionOutcome{
		CognitiveWorkspaceID: "cw-1",
		WorkspaceID:          "ws-1",
		ProposalID:           "proposal-1",
		Executed: []blackboard.ExecutedStep{
			{Step: blackboard.ToolSequenceStep{Sequence: 1, ToolName: "tool-1"}},
			{Step: blackboard.ToolSequenceStep{Sequence: 2, ToolName: "tool-2"}},
			{Step: blackboard.ToolSequenceStep{Sequence: 3, ToolName: "tool-3"}},
		},
	}}
	handler := NewBlackboardHandlerWithAuthorizer(pipeline, &toolAuthzStub{allow: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blackboard/cw-1/plan", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), "ws-1"))
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, "user-1"))
	req = withRouteParam(req, "cwID", "cw-1")
	rr := httptest.NewRecorder()

	handler.RunPipeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if pipeline.calls != 1 || pipeline.lastCW != "cw-1" {
		t.Fatalf("calls=%d lastCW=%q", pipeline.calls, pipeline.lastCW)
	}

	var outcome blackboard.ExecutionOutcome
	if err := json.NewDecoder(rr.Body).Decode(&outcome); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(outcome.Executed) != 3 {
		t.Fatalf("Executed len = %d, want 3", len(outcome.Executed))
	}
}

func TestBlackboardHandler_RunPipeline_ErrorMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "already running", err: blackboard.ErrPipelineAlreadyRunning, wantStatus: http.StatusConflict},
		{name: "workspace not found", err: blackboard.ErrCognitiveWorkspaceNotFound, wantStatus: http.StatusNotFound},
		{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewBlackboardHandlerWithAuthorizer(&blackboardPipelineStub{err: tc.err}, &toolAuthzStub{allow: true})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/blackboard/cw-1/plan", nil)
			req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, "user-1"))
			req = withRouteParam(req, "cwID", "cw-1")
			rr := httptest.NewRecorder()

			handler.RunPipeline(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tc.wantStatus, rr.Body.String())
			}
		})
	}
}
