package m2sf

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func TestExecutorExecutePropagatesIdentityAndReturnsNormalizedResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/fenix/flow" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("X-Fenix-Trace-ID"); got != "trace-1" {
			t.Fatalf("trace header = %q", got)
		}
		if got := r.Header.Get("X-Fenix-Execution-ID"); got != "exec-1" {
			t.Fatalf("execution header = %q", got)
		}

		result := flowinterop.Result{
			ContractVersion: flowinterop.ContractVersion,
			Operation:       flowinterop.OperationValidate,
			Status:          flowinterop.ResultSucceeded,
			Fidelity: flowinterop.FidelityReport{
				ContractVersion: flowinterop.ContractVersion,
				Level:           flowinterop.FidelityGuaranteed,
				FlowFamily:      flowinterop.FlowFamilyAutolaunched,
			},
			ProviderReference: "provider-1",
		}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	executor, err := NewExecutor(
		server.URL,
		"test-token",
		flowinterop.OperationValidate,
		server.Client(),
	)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}

	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, "trace-1")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.ExecutionID, "exec-1")
	out, err := executor.Execute(ctx, validValidateRequest())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var result flowinterop.Result
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ProviderReference != "provider-1" {
		t.Fatalf("provider reference = %q", result.ProviderReference)
	}
}

func TestExecutorExecuteMarksServerFailureRetryable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	executor, err := NewExecutor(
		server.URL,
		"test-token",
		flowinterop.OperationValidate,
		server.Client(),
	)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}

	_, err = executor.Execute(context.Background(), validValidateRequest())
	var capabilityErr *tool.CapabilityExecutionError
	if !errors.As(err, &capabilityErr) {
		t.Fatalf("expected CapabilityExecutionError, got %v", err)
	}
	if capabilityErr.Disposition != tool.CapabilityRetryable {
		t.Fatalf("disposition = %q", capabilityErr.Disposition)
	}
}

func validValidateRequest() json.RawMessage {
	return json.RawMessage(`{
		"contract_version":"1",
		"operation":"salesforce.flow.validate",
		"input":{"format":"mermaid","name":"Demo","content":"flowchart TD\nStart --> End"}
	}`)
}
