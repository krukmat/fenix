// Package vel provides the HTTP adapter from Fenix evidence runtime to VEL.
package vel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
)

const maxResponseBytes = 4 << 20

var errInvalidConfig = errors.New("invalid VEL integration configuration")

// Sink implements evidence.RuntimeSink over the authenticated VEL HTTP surface.
type Sink struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewSink creates a VEL HTTP evidence sink.
func NewSink(baseURL, token string, client *http.Client) (*Sink, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return nil, errInvalidConfig
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Sink{baseURL: baseURL, token: token, client: client}, nil
}

// RecordEvidence appends one externally authorized Fenix evidence event.
func (s *Sink) RecordEvidence(
	ctx context.Context,
	envelope evidence.Envelope,
) (evidence.ProofReference, error) {
	request, err := s.newAppendRequest(ctx, envelope)
	if err != nil {
		return evidence.ProofReference{}, err
	}
	responseRaw, err := s.executeAppend(request)
	if err != nil {
		return evidence.ProofReference{}, err
	}
	event, err := decodeStoredEvent(responseRaw)
	if err != nil {
		return evidence.ProofReference{}, err
	}
	return proofFromEvent(envelope.ExecutionID, event)
}

func (s *Sink) newAppendRequest(
	ctx context.Context,
	envelope evidence.Envelope,
) (*http.Request, error) {
	raw, err := json.Marshal(externalEventFromEnvelope(envelope))
	if err != nil {
		return nil, fmt.Errorf("marshal VEL external event: %w", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/v1/external/events",
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, fmt.Errorf("create VEL append request: %w", err)
	}
	s.decorateRequest(ctx, request)
	return request, nil
}

func (s *Sink) executeAppend(request *http.Request) ([]byte, error) {
	response, err := s.client.Do(request)
	if err != nil {
		return nil, evidence.NewIndeterminateRecordError(
			fmt.Errorf("append VEL evidence: %w", err),
		)
	}
	defer response.Body.Close()

	responseRaw, err := readResponse(response)
	if err != nil {
		return nil, evidence.NewIndeterminateRecordError(err)
	}
	if err := validateAppendStatus(response.StatusCode, responseRaw); err != nil {
		return nil, err
	}
	return responseRaw, nil
}

func validateAppendStatus(statusCode int, responseRaw []byte) error {
	if statusCode >= http.StatusInternalServerError {
		return evidence.NewIndeterminateRecordError(
			fmt.Errorf("VEL append status %d", statusCode),
		)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"VEL append status %d: %s",
			statusCode,
			bodySummary(responseRaw),
		)
	}
	return nil
}

// LookupEvidence resolves an earlier append by the stable Fenix execution id.
func (s *Sink) LookupEvidence(
	ctx context.Context,
	streamID, executionID string,
) (*evidence.ProofReference, error) {
	request, err := s.newLookupRequest(ctx, streamID, executionID)
	if err != nil {
		return nil, err
	}
	responseRaw, found, err := s.executeLookup(request)
	if err != nil || !found {
		return nil, err
	}
	event, err := decodeStoredEvent(responseRaw)
	if err != nil {
		return nil, err
	}
	ref, err := proofFromEvent(executionID, event)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (s *Sink) newLookupRequest(
	ctx context.Context,
	streamID, executionID string,
) (*http.Request, error) {
	query := url.Values{}
	query.Set("stream_id", streamID)
	query.Set("idempotency_key", executionID)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+"/v1/external/events/by-idempotency?"+query.Encode(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create VEL lookup request: %w", err)
	}
	s.decorateRequest(ctx, request)
	return request, nil
}

func (s *Sink) executeLookup(request *http.Request) ([]byte, bool, error) {
	response, err := s.client.Do(request)
	if err != nil {
		return nil, false, fmt.Errorf("lookup VEL evidence: %w", err)
	}
	defer response.Body.Close()

	responseRaw, err := readResponse(response)
	if err != nil {
		return nil, false, err
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if err := validateLookupStatus(response.StatusCode, responseRaw); err != nil {
		return nil, false, err
	}
	return responseRaw, true, nil
}

func validateLookupStatus(statusCode int, responseRaw []byte) error {
	if statusCode >= http.StatusInternalServerError {
		return fmt.Errorf("VEL lookup status %d", statusCode)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"VEL lookup status %d: %s",
			statusCode,
			bodySummary(responseRaw),
		)
	}
	return nil
}

type externalEvent struct {
	StreamID       string                       `json:"stream_id"`
	Actor          externalActor                `json:"actor"`
	Action         string                       `json:"action"`
	Resource       string                       `json:"resource"`
	Payload        map[string]any               `json:"payload"`
	Authorization externalAuthorization         `json:"authorization"`
	IdempotencyKey string                       `json:"idempotency_key"`
}

type externalActor struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type externalAuthorization struct {
	PolicyID      string         `json:"policy_id"`
	PolicyVersion string         `json:"policy_version"`
	Decision      string         `json:"decision"`
	Reason        string         `json:"reason"`
	Context       map[string]any `json:"context"`
	ContextHash   string         `json:"context_hash"`
}

type storedEvent struct {
	EventID   string `json:"event_id"`
	StreamID  string `json:"stream_id"`
	Sequence  int64  `json:"sequence"`
	KeyID     string `json:"key_id"`
	EventHash string `json:"event_hash"`
	Signature string `json:"signature"`
}

func externalEventFromEnvelope(envelope evidence.Envelope) externalEvent {
	return externalEvent{
		StreamID: envelope.StreamID,
		Actor: externalActor{
			ID:   envelope.Actor.ID,
			Type: envelope.Actor.Type,
		},
		Action:   "capability:" + envelope.Capability.Name + ":" + envelope.Capability.Operation,
		Resource: "execution/" + envelope.ExecutionID,
		Payload: map[string]any{
			"schema_version": envelope.SchemaVersion,
			"workspace_id":   envelope.WorkspaceID,
			"trace_id":       envelope.TraceID,
			"execution_id":   envelope.ExecutionID,
			"run_id":         envelope.RunID,
			"capability":     envelope.Capability,
			"approval":       envelope.Approval,
			"outcome":        envelope.Outcome,
			"input_digest":   envelope.InputDigest,
			"output_digest":  envelope.OutputDigest,
			"occurred_at":    envelope.OccurredAt,
		},
		Authorization: externalAuthorization{
			PolicyID:      envelope.Authorization.PolicyID,
			PolicyVersion: envelope.Authorization.PolicyVersion,
			Decision:      string(envelope.Authorization.Decision),
			Reason:        envelope.Authorization.Reason,
			Context:       authorizationContext(envelope),
			ContextHash:   envelope.Authorization.ContextHash,
		},
		IdempotencyKey: envelope.ExecutionID,
	}
}

func authorizationContext(envelope evidence.Envelope) map[string]any {
	context := map[string]any{
		"workspace_id":     envelope.WorkspaceID,
		"trace_id":         envelope.TraceID,
		"execution_id":     envelope.ExecutionID,
		"policy_reference": envelope.Authorization.PolicyID,
	}
	if envelope.Approval != nil && strings.TrimSpace(envelope.Approval.ApprovalID) != "" {
		context["approval_id"] = envelope.Approval.ApprovalID
	}
	return context
}

func proofFromEvent(
	executionID string,
	event storedEvent,
) (evidence.ProofReference, error) {
	ref := evidence.ProofReference{
		SchemaVersion:      evidence.SchemaVersion,
		Provider:           "vel-http",
		ExecutionID:        executionID,
		StreamID:           event.StreamID,
		EventID:            event.EventID,
		EventHash:          event.EventHash,
		KeyID:              event.KeyID,
		SignatureRef:       "vel://events/" + event.EventID + "#signature",
		Sequence:           event.Sequence,
		VerificationStatus: evidence.VerificationRecorded,
	}
	if err := ref.Validate(); err != nil {
		return evidence.ProofReference{}, fmt.Errorf("validate VEL proof reference: %w", err)
	}
	return ref, nil
}

func decodeStoredEvent(raw []byte) (storedEvent, error) {
	var event storedEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return storedEvent{}, fmt.Errorf("decode VEL event: %w", err)
	}
	return event, nil
}

func (s *Sink) decorateRequest(ctx context.Context, request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.token)
	setContextHeader(ctx, request, "X-Fenix-Trace-ID", ctxkeys.TraceID)
	setContextHeader(ctx, request, "X-Fenix-Execution-ID", ctxkeys.ExecutionID)
}

func setContextHeader(
	ctx context.Context,
	request *http.Request,
	header string,
	key ctxkeys.Key,
) {
	value, _ := ctx.Value(key).(string)
	if value = strings.TrimSpace(value); value != "" {
		request.Header.Set(header, value)
	}
}

func readResponse(response *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read VEL response: %w", err)
	}
	return raw, nil
}

func bodySummary(raw []byte) string {
	const max = 256
	text := strings.TrimSpace(string(raw))
	if len(text) <= max {
		return text
	}
	return text[:max]
}
