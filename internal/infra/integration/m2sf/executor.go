// Package m2sf provides the HTTP adapter from Fenix governed capabilities to Mermaid2SF.
package m2sf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/integration/telemetry"
)

const maxResponseBytes = 4 << 20

var (
	errInvalidConfig   = errors.New("invalid Mermaid2SF integration configuration")
	errInvalidResponse = errors.New("invalid Mermaid2SF provider response")
)

// Executor invokes exactly one versioned Mermaid2SF semantic operation.
type Executor struct {
	baseURL   string
	token     string
	operation flowinterop.Operation
	client    *http.Client
}

// NewExecutor builds one operation-specific M2SF executor.
func NewExecutor(
	baseURL string,
	token string,
	operation flowinterop.Operation,
	client *http.Client,
) (*Executor, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return nil, errInvalidConfig
	}
	if _, ok := flowinterop.Lookup(operation); !ok {
		return nil, errInvalidConfig
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Executor{baseURL: baseURL, token: token, operation: operation, client: client}, nil
}

// Execute validates the Fenix semantic request, invokes M2SF and returns the normalized result JSON.
func (e *Executor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	request, err := e.decodeRequest(params)
	if err != nil {
		return nil, err
	}
	httpRequest, err := e.newHTTPRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	responseRaw, err := e.executeHTTPRequest(httpRequest)
	if err != nil {
		return nil, err
	}
	if validationErr := decodeAndValidateResult(e.operation, responseRaw); validationErr != nil {
		return nil, validationErr
	}
	return json.RawMessage(responseRaw), nil
}

func (e *Executor) newHTTPRequest(
	ctx context.Context,
	request flowinterop.Request,
) (*http.Request, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal Mermaid2SF request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		e.baseURL+"/api/fenix/flow",
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, fmt.Errorf("create Mermaid2SF request: %w", err)
	}
	e.decorateRequest(ctx, httpRequest)
	return httpRequest, nil
}

func (e *Executor) executeHTTPRequest(request *http.Request) (responseRaw []byte, err error) {
	started := time.Now()
	defer func() {
		outcome := "success"
		if err != nil {
			outcome = "error"
		}
		telemetry.Default.ObserveProvider("m2sf", string(e.operation), outcome, time.Since(started))
	}()

	response, err := e.client.Do(request)
	if err != nil {
		return nil, tool.NewRetryableCapabilityError(
			fmt.Errorf("call Mermaid2SF provider: %w", err),
		)
	}
	defer response.Body.Close()

	responseRaw, err = io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, tool.NewRetryableCapabilityError(
			fmt.Errorf("read Mermaid2SF response: %w", err),
		)
	}
	if statusErr := validateHTTPResponse(response.StatusCode, responseRaw); statusErr != nil {
		return nil, statusErr
	}
	return responseRaw, nil
}

func validateHTTPResponse(statusCode int, responseRaw []byte) error {
	if statusCode >= http.StatusInternalServerError {
		return tool.NewRetryableCapabilityError(
			fmt.Errorf("Mermaid2SF provider status %d", statusCode),
		)
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Mermaid2SF provider status %d: %s", statusCode, bodySummary(responseRaw))
	}
	return nil
}

func decodeAndValidateResult(operation flowinterop.Operation, responseRaw []byte) error {
	var result flowinterop.Result
	if err := json.Unmarshal(responseRaw, &result); err != nil {
		return fmt.Errorf("%w: decode JSON: %w", errInvalidResponse, err)
	}
	return validateResult(operation, result)
}

func (e *Executor) decodeRequest(params json.RawMessage) (flowinterop.Request, error) {
	var request flowinterop.Request
	if err := json.Unmarshal(params, &request); err != nil {
		return flowinterop.Request{}, fmt.Errorf("decode Mermaid2SF request: %w", err)
	}
	if request.Operation != e.operation {
		return flowinterop.Request{}, fmt.Errorf(
			"Mermaid2SF operation mismatch: executor=%s request=%s",
			e.operation,
			request.Operation,
		)
	}
	if err := request.Validate(); err != nil {
		return flowinterop.Request{}, fmt.Errorf("validate Mermaid2SF request: %w", err)
	}
	return request, nil
}

func (e *Executor) decorateRequest(ctx context.Context, request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+e.token)
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

func validateResult(operation flowinterop.Operation, result flowinterop.Result) error {
	if !validResultIdentity(operation, result) || !validResultStatus(result.Status) {
		return errInvalidResponse
	}
	if err := result.Fidelity.Validate(); err != nil {
		return fmt.Errorf("%w: fidelity: %w", errInvalidResponse, err)
	}
	if err := validateDiagnostics(result.Diagnostics); err != nil {
		return err
	}
	return validateSemanticDiff(result.Diff)
}

func validResultIdentity(operation flowinterop.Operation, result flowinterop.Result) bool {
	return result.ContractVersion == flowinterop.ContractVersion && result.Operation == operation
}

func validResultStatus(status flowinterop.ResultStatus) bool {
	switch status {
	case flowinterop.ResultSucceeded, flowinterop.ResultRejected, flowinterop.ResultUnsupported:
		return true
	default:
		return false
	}
}

func validateDiagnostics(diagnostics []flowinterop.Diagnostic) error {
	for _, diagnostic := range diagnostics {
		if err := diagnostic.Validate(); err != nil {
			return fmt.Errorf("%w: diagnostic: %w", errInvalidResponse, err)
		}
	}
	return nil
}

func validateSemanticDiff(diff *flowinterop.SemanticDiff) error {
	if diff == nil {
		return nil
	}
	if err := diff.Validate(); err != nil {
		return fmt.Errorf("%w: semantic diff: %w", errInvalidResponse, err)
	}
	return nil
}

func bodySummary(raw []byte) string {
	const max = 256
	text := strings.TrimSpace(string(raw))
	if len(text) <= max {
		return text
	}
	return text[:max]
}
