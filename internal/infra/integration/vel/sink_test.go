package vel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func TestSinkRecordEvidenceBuildsVELExternalAuthorityRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/external/events" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("X-Fenix-Execution-ID"); got != "exec-1" {
			t.Fatalf("execution header = %q", got)
		}

		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request["idempotency_key"] != "exec-1" {
			t.Fatalf("idempotency key = %#v", request["idempotency_key"])
		}
		auth := request["authorization"].(map[string]any)
		contextValue := auth["context"].(map[string]any)
		if contextValue["execution_id"] != "exec-1" {
			t.Fatalf("authorization context = %#v", contextValue)
		}

		_, _ = w.Write([]byte(`{
			"event_id":"018f4f0a-0000-7000-8000-000000000001",
			"stream_id":"workspace/ws-1",
			"sequence":1,
			"key_id":"key-1",
			"event_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"signature":"signature"
		}`))
	}))
	defer server.Close()

	sink, err := NewSink(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("NewSink: %v", err)
	}
	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.ExecutionID, "exec-1")
	ref, err := sink.RecordEvidence(ctx, validEnvelope())
	if err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}
	if ref.ExecutionID != "exec-1" || ref.VerificationStatus != evidence.VerificationRecorded {
		t.Fatalf("proof = %#v", ref)
	}
}

func TestSinkLookupEvidenceHandlesSlashStreamIDAndNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("stream_id") != "workspace/ws-1" {
			t.Fatalf("stream_id = %q", r.URL.Query().Get("stream_id"))
		}
		if r.URL.Query().Get("idempotency_key") != "exec-1" {
			t.Fatalf("idempotency_key = %q", r.URL.Query().Get("idempotency_key"))
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	sink, err := NewSink(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("NewSink: %v", err)
	}
	ref, err := sink.LookupEvidence(context.Background(), "workspace/ws-1", "exec-1")
	if err != nil {
		t.Fatalf("LookupEvidence: %v", err)
	}
	if ref != nil {
		t.Fatalf("expected nil reference, got %#v", ref)
	}
}

func TestSinkRecordEvidenceMarksServerFailureIndeterminate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporary failure", http.StatusInternalServerError)
	}))
	defer server.Close()

	sink, err := NewSink(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("NewSink: %v", err)
	}
	_, err = sink.RecordEvidence(context.Background(), validEnvelope())
	var indeterminate *evidence.IndeterminateRecordError
	if !errors.As(err, &indeterminate) {
		t.Fatalf("expected IndeterminateRecordError, got %v", err)
	}
}

func validEnvelope() evidence.Envelope {
	return evidence.Envelope{
		SchemaVersion: evidence.SchemaVersion,
		StreamID:      "workspace/ws-1",
		WorkspaceID:   "ws-1",
		TraceID:       "trace-1",
		ExecutionID:   "exec-1",
		Actor:         evidence.ActorRef{ID: "agent-1", Type: "agent"},
		Capability: evidence.CapabilityRef{
			Name:            "salesforce.flow.export",
			Version:         "1",
			Operation:       "export",
			SideEffectClass: tool.SideEffectTransform,
		},
		Authorization: evidence.AuthorizationEvidence{
			PolicyID:      "fenix:w1-policy-gate",
			PolicyVersion: "runtime",
			Decision:      evidence.PolicyAllow,
			Reason:        "authorized by Fenix governance",
			ContextHash:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Outcome:    evidence.ExecutionOutcome{Status: evidence.OutcomeSucceeded},
		OccurredAt: time.Date(2026, 9, 21, 18, 0, 0, 0, time.UTC),
	}
}
