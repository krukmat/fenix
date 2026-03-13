package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
)

var (
	ErrInvalidDispatchInput    = errors.New("invalid dispatch input")
	ErrInvalidDispatchResponse = errors.New("invalid dispatch response")
	ErrDispatchAuthRequired    = errors.New("dispatch auth is required")
	ErrDispatchEndpointInvalid = errors.New("dispatch endpoint is invalid")
)

type ProtocolHandler interface {
	Dispatch(ctx context.Context, input DispatchInput) (*DispatchResponse, error)
}

type DispatchStatus string

const (
	DispatchStatusAccepted  DispatchStatus = "ACCEPTED"
	DispatchStatusRejected  DispatchStatus = "REJECTED"
	DispatchStatusDelegated DispatchStatus = "DELEGATED"
)

const (
	dispatchAdapterA2A = "a2a-go"
)

type DispatchInput struct {
	TargetAgent  string
	WorkflowName string
	DSLSource    string
	CallChain    []string
	TimeoutSec   int
	TraceID      string
	DispatchID   string
	AgentCard    *a2a.AgentCard
	Endpoint     *DispatchEndpoint
	Auth         *DispatchAuthConfig
	Metadata     map[string]any
}

type DispatchAuthConfig struct {
	SessionID   string
	Credentials map[string]string
	Headers     map[string]string
}

type DispatchEndpoint struct {
	URL       string
	Transport string
}

type DispatchResponse struct {
	Status   DispatchStatus
	Reason   string
	Target   string
	RunID    string
	Adapter  string
	Metadata map[string]any
}

func (in DispatchInput) Validate() error {
	if err := validateDispatchCoreInput(in); err != nil {
		return err
	}
	return validateDispatchEndpointInput(in.Endpoint)
}

func (r DispatchResponse) Validate() error {
	switch r.Status {
	case DispatchStatusAccepted:
		return nil
	case DispatchStatusRejected:
		if strings.TrimSpace(r.Reason) == "" {
			return fmt.Errorf("%w: rejected response requires reason", ErrInvalidDispatchResponse)
		}
		return nil
	case DispatchStatusDelegated:
		if strings.TrimSpace(r.Target) == "" {
			return fmt.Errorf("%w: delegated response requires target", ErrInvalidDispatchResponse)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported dispatch status %q", ErrInvalidDispatchResponse, r.Status)
	}
}

func NewDispatchAcceptedResponse(runID string, metadata map[string]any) *DispatchResponse {
	metadata = cloneDispatchMetadata(metadata)
	ensureDispatchMetadataStatus(metadata, DispatchStatusAccepted)
	return &DispatchResponse{
		Status:   DispatchStatusAccepted,
		RunID:    strings.TrimSpace(runID),
		Adapter:  dispatchAdapterA2A,
		Metadata: metadata,
	}
}

func NewDispatchRejectedResponse(reason string, metadata map[string]any) (*DispatchResponse, error) {
	metadata = cloneDispatchMetadata(metadata)
	reason = strings.TrimSpace(reason)
	ensureDispatchMetadataStatus(metadata, DispatchStatusRejected)
	metadata["reason"] = reason
	resp := &DispatchResponse{
		Status:   DispatchStatusRejected,
		Reason:   reason,
		Adapter:  dispatchAdapterA2A,
		Metadata: metadata,
	}
	if err := resp.Validate(); err != nil {
		return nil, err
	}
	return resp, nil
}

func NewDispatchDelegatedResponse(target, runID string, metadata map[string]any) (*DispatchResponse, error) {
	metadata = cloneDispatchMetadata(metadata)
	ensureDispatchMetadataStatus(metadata, DispatchStatusDelegated)
	resp := &DispatchResponse{
		Status:   DispatchStatusDelegated,
		Target:   strings.TrimSpace(target),
		RunID:    strings.TrimSpace(runID),
		Adapter:  dispatchAdapterA2A,
		Metadata: metadata,
	}
	if err := resp.Validate(); err != nil {
		return nil, err
	}
	return resp, nil
}

func cloneDispatchMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ensureDispatchMetadataStatus(metadata map[string]any, status DispatchStatus) {
	if metadata == nil {
		return
	}
	metadata["dispatch_status"] = string(status)
}

func dispatchTargetInCallChain(target string, callChain []string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, candidate := range callChain {
		if strings.TrimSpace(candidate) == target {
			return true
		}
	}
	return false
}

func validateDispatchCoreInput(in DispatchInput) error {
	switch {
	case strings.TrimSpace(in.TargetAgent) == "":
		return fmt.Errorf("%w: target_agent is required", ErrInvalidDispatchInput)
	case strings.TrimSpace(in.WorkflowName) == "" && strings.TrimSpace(in.DSLSource) == "":
		return fmt.Errorf("%w: workflow_name or dsl_source is required", ErrInvalidDispatchInput)
	case in.TimeoutSec < 0:
		return fmt.Errorf("%w: timeout_sec must be >= 0", ErrInvalidDispatchInput)
	default:
		return nil
	}
}

func validateDispatchEndpointInput(endpoint *DispatchEndpoint) error {
	if endpoint == nil {
		return nil
	}
	switch {
	case strings.TrimSpace(endpoint.URL) == "":
		return fmt.Errorf("%w: endpoint.url is required", ErrInvalidDispatchInput)
	case strings.TrimSpace(endpoint.Transport) == "":
		return fmt.Errorf("%w: endpoint.transport is required", ErrInvalidDispatchInput)
	default:
		return nil
	}
}
