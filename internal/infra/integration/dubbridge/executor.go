package dubbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

// Executor is the real (non-mock) Fenix ToolExecutor implementation. It never
// issues tokens and never performs provider mutations.
type Executor struct{ client *Client }

func NewExecutor(config Config) (*Executor, error) {
	c, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Executor{client: c}, nil
}

type QueueRequest struct {
	ContractVersion string `json:"contract_version"`
	OrgID           string `json:"org_id"`
	ProjectID       string `json:"project_id"`
}

func (e *Executor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	if e == nil || e.client == nil {
		return nil, ErrConfiguration
	}
	var input QueueRequest
	decoder := json.NewDecoder(bytes.NewReader(params))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return nil, ErrConfiguration
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, ErrConfiguration
	}
	if input.ContractVersion != "1" || !ValidUUID(input.OrgID) || !ValidUUID(input.ProjectID) {
		return nil, ErrConfiguration
	}
	traceID, _ := ctx.Value(ctxkeys.TraceID).(string)
	executionID, _ := ctx.Value(ctxkeys.ExecutionID).(string)
	queue, err := e.client.ListReviewQueue(ctx, input.OrgID, input.ProjectID, traceID, executionID)
	if err != nil {
		return nil, err
	}
	result, err := ProjectForAgent(queue)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, ErrInvalidResponse
	}
	return json.RawMessage(payload), nil
}

// RequiresRetry advises Fenix's governed retry implementation of a provider
// transport failure. The caller must still use tool.NewRetryableCapabilityError
// and respect the capability's bounded attempt count.
func RequiresRetry(err error) bool {
	var p *ProviderError
	return errors.As(err, &p) && p.Retryable
}

// InputSchema is validated by Fenix's existing minimal-schema validator.
// Ownership remains Fenix; DubBridge never interprets this semantic envelope.
var InputSchema = json.RawMessage(strings.Join([]string{
	`{"type":"object",`,
	`"required":["contract_version","org_id","project_id"],`,
	`"properties":{`,
	`"contract_version":{"type":"string"},`,
	`"org_id":{"type":"string"},`,
	`"project_id":{"type":"string"}`,
	`},"additionalProperties":false}`,
}, ""))
