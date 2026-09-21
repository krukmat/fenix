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

	response, err := e.client.Do(httpRequest)
	if err != nil {
		return nil, tool.NewRetryableCapabilityError(
			fmt.Errorf("call Mermaid2SF provider: %w", err),
		)
	}
	defer response.Body.Close()

	responseRaw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, tool.NewRetryableCapabilityError(
			fmt.Errorf("read Mermaid2SF response: %w", err),
		)
	}
	if response.StatusCode >= http.StatusInternalServerError {
		return nil, tool.NewRetryableCapabilityError(
			fmt.Errorf("Mermaid2SF provider status %d", response.StatusCode),
		)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Mermaid2SF provider status %d: %s", response.StatusCode, bodySummary(responseRaw))
	}

	var result flowinterop.Result
	if err := json.Unmarshal(responseRaw, &result); err != nil {
		return nil, fmt.Errorf("%w: decode JSON: %v", errInvalidResponse, err)
	}
	if err := validateResult(e.operation, result); err != nil {
		return nil, err
	}
	return json.RawMessage(responseRaw), nil
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
	if result.ContractVersion != flowinterop.ContractVersion || result.Operation != operation {
		return errInvalidResponse
	}
	switch result.Status {
	case flowinterop.ResultSucceeded, flowinterop.ResultRejected, flowinterop.ResultUnsupported:
	default:
		return errInvalidResponse
	}
	if err := result.Fidelity.Validate(); err != nil {
		return fmt.Errorf("%w: fidelity: %v", errInvalidResponse, err)
	}
	for _, diagnostic := range result.Diagnostics {
		if err := diagnostic.Validate(); err != nil {
			return fmt.Errorf("%w: diagnostic: %v", errInvalidResponse, err)
		}
	}
	if result.Diff != nil {
		if err := result.Diff.Validate(); err != nil {
			return fmt.Errorf("%w: semantic diff: %v", errInvalidResponse, err)
		}
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
