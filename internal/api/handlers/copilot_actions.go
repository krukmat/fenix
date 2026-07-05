package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
)

type CopilotActionsService interface {
	SuggestActions(ctx context.Context, in copilot.SuggestActionsInput) ([]copilot.SuggestedAction, error)
	Summarize(ctx context.Context, in copilot.SummarizeInput) (string, error)
	SalesBrief(ctx context.Context, in copilot.SalesBriefInput) (*copilot.SalesBriefResult, error)
}

type CopilotActionsHandler struct {
	service CopilotActionsService
}

func NewCopilotActionsHandler(service CopilotActionsService) *CopilotActionsHandler {
	return &CopilotActionsHandler{service: service}
}

type copilotEntityRequest struct {
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
}

type copilotSuggestActionsResponse struct {
	Data struct {
		Actions []copilot.SuggestedAction `json:"actions"`
	} `json:"data"`
}

type copilotSummarizeResponse struct {
	Data struct {
		Summary string `json:"summary"`
	} `json:"data"`
}

type copilotSalesBriefResponse struct {
	Data struct {
		Outcome          string                    `json:"outcome"`
		EntityType       string                    `json:"entityType"`
		EntityID         string                    `json:"entityId"`
		Summary          string                    `json:"summary"`
		Risks            []string                  `json:"risks"`
		NextBestActions  []copilot.SuggestedAction `json:"nextBestActions"`
		Confidence       string                    `json:"confidence"`
		AbstentionReason *string                   `json:"abstentionReason,omitempty"`
		EvidencePack     evidenceData              `json:"evidencePack"`
		// UIX-22A: evidenceIds is brief-scoped (grounds summary, risks, and
		// nextBestActions together), not attributed per individual item —
		// the brief generation step does not produce a finer-grained mapping.
		EvidenceIDs []string `json:"evidenceIds,omitempty"`
		// UIX-22A: runId is a correlation-only identifier (audit log +
		// response), NOT a usage_event foreign key — usage_event.run_id
		// references agent_run(id) and this synchronous call has no backing
		// AgentRun row. Usage is therefore returned inline below rather than
		// via a GET /api/v1/usage?run_id= lookup, which could never find it.
		RunID string                  `json:"runId,omitempty"`
		Usage *copilotSalesBriefUsage `json:"usage,omitempty"`
	} `json:"data"`
}

type copilotSalesBriefUsage struct {
	ModelName   *string `json:"modelName,omitempty"`
	InputUnits  int64   `json:"inputUnits"`
	OutputUnits int64   `json:"outputUnits"`
	Cost        float64 `json:"cost"`
	LatencyMs   int64   `json:"latencyMs"`
}

type actionRequestError struct {
	status  int
	message string
}

func (e actionRequestError) Error() string { return e.message }

func (h *CopilotActionsHandler) SuggestActions(w http.ResponseWriter, r *http.Request) {
	handleCopilotEntityAction(w, r,
		func(ctx context.Context, in copilotEntityInput) (copilotSuggestActionsResponse, error) {
			actions, err := h.service.SuggestActions(ctx, copilot.SuggestActionsInput{
				WorkspaceID: in.WorkspaceID,
				UserID:      in.UserID,
				EntityType:  in.EntityType,
				EntityID:    in.EntityID,
			})
			return mapCopilotEntityAction(
				actions,
				err,
				func(resp *copilotSuggestActionsResponse, actions []copilot.SuggestedAction) {
					resp.Data.Actions = actions
				},
			)
		},
		"suggest-actions failed",
	)
}

func (h *CopilotActionsHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	handleCopilotEntityAction(w, r,
		func(ctx context.Context, in copilotEntityInput) (copilotSummarizeResponse, error) {
			summary, err := h.service.Summarize(ctx, copilot.SummarizeInput{
				WorkspaceID: in.WorkspaceID,
				UserID:      in.UserID,
				EntityType:  in.EntityType,
				EntityID:    in.EntityID,
			})
			return mapCopilotEntityAction(
				summary,
				err,
				func(resp *copilotSummarizeResponse, summary string) {
					resp.Data.Summary = summary
				},
			)
		},
		"summarize failed",
	)
}

func (h *CopilotActionsHandler) SalesBrief(w http.ResponseWriter, r *http.Request) {
	handleCopilotEntityAction(w, r,
		func(ctx context.Context, in copilotEntityInput) (copilotSalesBriefResponse, error) {
			result, err := h.service.SalesBrief(ctx, copilot.SalesBriefInput{
				WorkspaceID: in.WorkspaceID,
				UserID:      in.UserID,
				EntityType:  in.EntityType,
				EntityID:    in.EntityID,
			})
			return mapCopilotEntityAction(
				result,
				err,
				func(resp *copilotSalesBriefResponse, result *copilot.SalesBriefResult) {
					resp.Data.Outcome = result.Outcome
					resp.Data.EntityType = result.EntityType
					resp.Data.EntityID = result.EntityID
					resp.Data.Summary = result.Summary
					resp.Data.Risks = result.Risks
					resp.Data.NextBestActions = result.NextBestActions
					resp.Data.Confidence = string(result.Confidence)
					if result.AbstentionReason != nil {
						reason := string(*result.AbstentionReason)
						resp.Data.AbstentionReason = &reason
					}
					resp.Data.EvidencePack = newEvidenceData(result.EvidencePack)
					resp.Data.EvidenceIDs = result.EvidenceIDs
					resp.Data.RunID = result.RunID
					resp.Data.Usage = &copilotSalesBriefUsage{
						ModelName:   result.Usage.ModelName,
						InputUnits:  result.Usage.InputUnits,
						OutputUnits: result.Usage.OutputUnits,
						Cost:        result.Usage.Cost,
						LatencyMs:   result.Usage.LatencyMs,
					}
				},
			)
		},
		"sales brief failed",
	)
}

func handleCopilotEntityAction[T any](
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, copilotEntityInput) (T, error),
	errorMsg string,
) {
	common, err := buildCopilotEntityInput(r)
	if err != nil {
		writeCopilotActionsError(w, err)
		return
	}

	resp, err := action(r.Context(), common)
	if err != nil {
		log.Printf("%s: %v", errorMsg, err)
		writeError(w, http.StatusInternalServerError, errorMsg)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type copilotEntityInput struct {
	WorkspaceID string
	UserID      string
	EntityType  string
	EntityID    string
}

func buildCopilotEntityInput(r *http.Request) (copilotEntityInput, error) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		return copilotEntityInput{}, actionRequestError{status: http.StatusUnauthorized, message: "missing workspace context"}
	}

	userID, _ := ctx.Value(ctxkeys.UserID).(string)
	if userID == "" {
		return copilotEntityInput{}, actionRequestError{status: http.StatusUnauthorized, message: "missing user context"}
	}

	var req copilotEntityRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		return copilotEntityInput{}, actionRequestError{status: http.StatusBadRequest, message: "invalid request body"}
	}
	if req.EntityType == "" || req.EntityID == "" {
		return copilotEntityInput{}, actionRequestError{status: http.StatusBadRequest, message: "entityType and entityId are required"}
	}

	return copilotEntityInput{
		WorkspaceID: wsID,
		UserID:      userID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	}, nil
}

func writeCopilotActionsError(w http.ResponseWriter, err error) {
	var reqErr actionRequestError
	if ok := errorAsAction(err, &reqErr); ok {
		writeError(w, reqErr.status, reqErr.message)
		return
	}
	writeError(w, http.StatusInternalServerError, "request failed")
}

func errorAsAction(err error, target *actionRequestError) bool {
	return errors.As(err, target)
}

func mapCopilotEntityAction[T any, R any](value T, err error, assign func(*R, T)) (R, error) {
	var resp R
	if err != nil {
		return resp, err
	}
	assign(&resp, value)
	return resp, nil
}
