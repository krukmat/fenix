package agent

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
)

const (
	dispatchMetaMode              = "mode"
	dispatchMetaModeValue         = "dispatch"
	dispatchMetaTargetAgent       = "target_agent"
	dispatchMetaWorkflowName      = "workflow_name"
	dispatchMetaDSLSource         = "dsl_source"
	dispatchMetaCallChain         = "call_chain"
	dispatchMetaTraceID           = "trace_id"
	dispatchMetaDispatchID        = "dispatch_id"
	dispatchMetaTimeoutSec        = "timeout_sec"
	dispatchMetaEndpointURL       = "endpoint_url"
	dispatchMetaEndpointTransport = "endpoint_transport"
	dispatchHeaderTraceID         = "x-fenix-trace-id"
	dispatchHeaderDispatchID      = "x-fenix-dispatch-id"
	dispatchHeaderCallChain       = "x-fenix-call-chain"
)

var ErrA2AAgentCardRequired = fmt.Errorf("%w: agent_card is required for a2a dispatch", ErrInvalidDispatchInput)

type a2aDispatchClient interface {
	SendMessage(ctx context.Context, message *a2a.MessageSendParams) (a2a.SendMessageResult, error)
	Destroy() error
}

type a2aClientFactory interface {
	NewFromCard(ctx context.Context, card *a2a.AgentCard, opts ...a2aclient.FactoryOption) (a2aDispatchClient, error)
}

type defaultA2AClientFactory struct{}

func (defaultA2AClientFactory) NewFromCard(ctx context.Context, card *a2a.AgentCard, opts ...a2aclient.FactoryOption) (a2aDispatchClient, error) {
	return a2aclient.NewFromCard(ctx, card, opts...)
}

type A2AProtocolHandler struct {
	factory a2aClientFactory
}

func NewA2AProtocolHandler() *A2AProtocolHandler {
	return NewA2AProtocolHandlerWithFactory(defaultA2AClientFactory{})
}

func NewA2AProtocolHandlerWithFactory(factory a2aClientFactory) *A2AProtocolHandler {
	if factory == nil {
		factory = defaultA2AClientFactory{}
	}
	return &A2AProtocolHandler{factory: factory}
}

func (h *A2AProtocolHandler) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResponse, error) {
	if rejected, ok, err := rejectedCircularDispatchResponse(input); ok || err != nil {
		return rejected, err
	}

	card, endpoint, dispatchID, traceID, dispatchCtx, cancel, err := prepareA2ADispatch(ctx, input)
	if err != nil {
		return nil, err
	}
	defer cancel()

	opts, err := buildA2AFactoryOptions(input, card, dispatchID, traceID)
	if err != nil {
		return nil, err
	}

	result, err := h.sendA2ADispatch(dispatchCtx, card, opts, input, endpoint, dispatchID, traceID)
	if err != nil {
		return nil, err
	}
	return finalizeA2ADispatch(result, endpoint, dispatchID, traceID, input.CallChain)
}

func rejectedCircularDispatchResponse(input DispatchInput) (*DispatchResponse, bool, error) {
	if !dispatchTargetInCallChain(input.TargetAgent, input.CallChain) {
		return nil, false, nil
	}
	resp, err := NewDispatchRejectedResponse(dispatchRejectLoop, map[string]any{
		dispatchMetaCallChain:   append([]string(nil), input.CallChain...),
		dispatchMetaTargetAgent: strings.TrimSpace(input.TargetAgent),
	})
	return resp, true, err
}

func prepareA2ADispatch(ctx context.Context, input DispatchInput) (*a2a.AgentCard, DispatchEndpoint, string, string, context.Context, context.CancelFunc, error) {
	if err := input.Validate(); err != nil {
		return nil, DispatchEndpoint{}, "", "", nil, nil, err
	}
	if input.AgentCard == nil {
		return nil, DispatchEndpoint{}, "", "", nil, nil, ErrA2AAgentCardRequired
	}

	endpoint, card, err := resolveDispatchEndpoint(input)
	if err != nil {
		return nil, DispatchEndpoint{}, "", "", nil, nil, err
	}

	dispatchID := normalizeDispatchID(input)
	traceID := normalizeTraceID(input)
	dispatchCtx, cancel := dispatchContext(ctx, input.TimeoutSec, input.Auth, dispatchID)
	return card, endpoint, dispatchID, traceID, dispatchCtx, cancel, nil
}

func (h *A2AProtocolHandler) sendA2ADispatch(
	ctx context.Context,
	card *a2a.AgentCard,
	opts []a2aclient.FactoryOption,
	input DispatchInput,
	endpoint DispatchEndpoint,
	dispatchID, traceID string,
) (a2a.SendMessageResult, error) {
	client, err := h.factory.NewFromCard(ctx, card, opts...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Destroy() }()

	params := buildA2AMessageSendParams(input, endpoint, dispatchID, traceID)
	return client.SendMessage(ctx, params)
}

func finalizeA2ADispatch(
	result a2a.SendMessageResult,
	endpoint DispatchEndpoint,
	dispatchID, traceID string,
	callChain []string,
) (*DispatchResponse, error) {
	resp, err := mapA2ASendResult(result)
	if err != nil {
		return nil, err
	}

	resp.Adapter = dispatchAdapterA2A
	resp.Metadata = mergeDispatchResponseMetadata(resp.Metadata, dispatchResponseMetadata(endpoint, dispatchID, traceID, callChain))
	return resp, nil
}

func buildA2AFactoryOptions(input DispatchInput, card *a2a.AgentCard, dispatchID, traceID string) ([]a2aclient.FactoryOption, error) {
	interceptors, err := buildA2AInterceptors(input, card, dispatchID, traceID)
	if err != nil {
		return nil, err
	}
	if len(interceptors) == 0 {
		return nil, nil
	}
	return []a2aclient.FactoryOption{a2aclient.WithInterceptors(interceptors...)}, nil
}

func buildA2AInterceptors(input DispatchInput, card *a2a.AgentCard, dispatchID, traceID string) ([]a2aclient.CallInterceptor, error) {
	var interceptors []a2aclient.CallInterceptor

	meta := buildA2ACallMeta(input, dispatchID, traceID)
	if len(meta) > 0 {
		interceptors = append(interceptors, a2aclient.NewStaticCallMetaInjector(meta))
	}

	if !agentCardRequiresAuth(card) {
		return interceptors, nil
	}
	if input.Auth == nil || len(input.Auth.Credentials) == 0 {
		return nil, ErrDispatchAuthRequired
	}

	store := a2aclient.NewInMemoryCredentialsStore()
	sessionID := a2aclient.SessionID(dispatchSessionID(input.Auth, dispatchID))
	for name, credential := range input.Auth.Credentials {
		store.Set(sessionID, a2a.SecuritySchemeName(name), a2aclient.AuthCredential(strings.TrimSpace(credential)))
	}

	interceptors = append(interceptors, &a2aclient.AuthInterceptor{Service: store})
	return interceptors, nil
}

func buildA2ACallMeta(input DispatchInput, dispatchID, traceID string) a2aclient.CallMeta {
	meta := a2aclient.CallMeta{}

	appendDispatchTraceHeaders(meta, traceID, dispatchID, input.CallChain)
	appendDispatchCustomHeaders(meta, input.Auth)

	if len(meta) == 0 {
		return nil
	}
	return meta
}

func appendDispatchTraceHeaders(meta a2aclient.CallMeta, traceID, dispatchID string, callChain []string) {
	if traceID != "" {
		meta.Append(dispatchHeaderTraceID, traceID)
	}
	if dispatchID != "" {
		meta.Append(dispatchHeaderDispatchID, dispatchID)
	}
	if len(callChain) > 0 {
		meta.Append(dispatchHeaderCallChain, strings.Join(callChain, ","))
	}
}

func appendDispatchCustomHeaders(meta a2aclient.CallMeta, auth *DispatchAuthConfig) {
	if auth == nil {
		return
	}
	for key, value := range auth.Headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		meta.Append(key, value)
	}
}

func dispatchContext(ctx context.Context, timeoutSec int, auth *DispatchAuthConfig, dispatchID string) (context.Context, context.CancelFunc) {
	if auth != nil {
		ctx = a2aclient.WithSessionID(ctx, a2aclient.SessionID(dispatchSessionID(auth, dispatchID)))
	}
	if timeoutSec <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
}

func resolveDispatchEndpoint(input DispatchInput) (DispatchEndpoint, *a2a.AgentCard, error) {
	card := cloneAgentCard(input.AgentCard)
	if input.Endpoint == nil {
		endpoint := DispatchEndpoint{
			URL:       strings.TrimSpace(card.URL),
			Transport: strings.TrimSpace(string(card.PreferredTransport)),
		}
		if err := validateDispatchEndpoint(endpoint); err != nil {
			return DispatchEndpoint{}, nil, err
		}
		return endpoint, card, nil
	}

	endpoint := DispatchEndpoint{
		URL:       strings.TrimSpace(input.Endpoint.URL),
		Transport: strings.TrimSpace(input.Endpoint.Transport),
	}
	if err := validateDispatchEndpoint(endpoint); err != nil {
		return DispatchEndpoint{}, nil, err
	}

	protocol, err := parseDispatchTransport(endpoint.Transport)
	if err != nil {
		return DispatchEndpoint{}, nil, err
	}

	card.URL = endpoint.URL
	card.PreferredTransport = protocol
	card.AdditionalInterfaces = appendUniqueInterfaces(
		[]a2a.AgentInterface{{Transport: protocol, URL: endpoint.URL}},
		card.AdditionalInterfaces,
	)

	return endpoint, card, nil
}

func validateDispatchEndpoint(endpoint DispatchEndpoint) error {
	if strings.TrimSpace(endpoint.URL) == "" {
		return fmt.Errorf("%w: endpoint url is required", ErrDispatchEndpointInvalid)
	}
	if _, err := url.ParseRequestURI(endpoint.URL); err != nil {
		return fmt.Errorf("%w: invalid endpoint url: %s", ErrDispatchEndpointInvalid, err.Error())
	}
	if _, err := parseDispatchTransport(endpoint.Transport); err != nil {
		return err
	}
	return nil
}

func parseDispatchTransport(transport string) (a2a.TransportProtocol, error) {
	switch strings.ToUpper(strings.TrimSpace(transport)) {
	case string(a2a.TransportProtocolJSONRPC):
		return a2a.TransportProtocolJSONRPC, nil
	case string(a2a.TransportProtocolGRPC):
		return a2a.TransportProtocolGRPC, nil
	case string(a2a.TransportProtocolHTTPJSON):
		return a2a.TransportProtocolHTTPJSON, nil
	default:
		return "", fmt.Errorf("%w: unsupported endpoint transport %q", ErrDispatchEndpointInvalid, transport)
	}
}

func cloneAgentCard(card *a2a.AgentCard) *a2a.AgentCard {
	if card == nil {
		return nil
	}
	cloned := *card
	cloned.AdditionalInterfaces = append([]a2a.AgentInterface(nil), card.AdditionalInterfaces...)
	cloned.Security = append([]a2a.SecurityRequirements(nil), card.Security...)
	cloned.Skills = append([]a2a.AgentSkill(nil), card.Skills...)
	return &cloned
}

func appendUniqueInterfaces(base []a2a.AgentInterface, extra []a2a.AgentInterface) []a2a.AgentInterface {
	out := append([]a2a.AgentInterface(nil), base...)
	for _, candidate := range extra {
		if candidate.URL == "" {
			continue
		}
		if !containsAgentInterface(out, candidate) {
			out = append(out, candidate)
		}
	}
	return out
}

func containsAgentInterface(in []a2a.AgentInterface, candidate a2a.AgentInterface) bool {
	for _, existing := range in {
		if existing.URL == candidate.URL && existing.Transport == candidate.Transport {
			return true
		}
	}
	return false
}

func buildA2AMessageSendParams(input DispatchInput, endpoint DispatchEndpoint, dispatchID, traceID string) *a2a.MessageSendParams {
	metadata := buildA2ADispatchMetadata(input, endpoint, dispatchID, traceID)
	return &a2a.MessageSendParams{
		Message:  buildA2ADispatchMessage(input),
		Metadata: metadata,
	}
}

func buildA2ADispatchMetadata(input DispatchInput, endpoint DispatchEndpoint, dispatchID, traceID string) map[string]any {
	metadata := cloneDispatchMetadata(input.Metadata)
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata[dispatchMetaMode] = dispatchMetaModeValue
	metadata[dispatchMetaTargetAgent] = input.TargetAgent
	appendDispatchWorkflowMetadata(metadata, input)
	appendDispatchTraceMetadata(metadata, input, dispatchID, traceID)
	metadata[dispatchMetaEndpointURL] = endpoint.URL
	metadata[dispatchMetaEndpointTransport] = endpoint.Transport
	return metadata
}

func appendDispatchWorkflowMetadata(metadata map[string]any, input DispatchInput) {
	if input.WorkflowName != "" {
		metadata[dispatchMetaWorkflowName] = input.WorkflowName
	}
	if input.DSLSource != "" {
		metadata[dispatchMetaDSLSource] = input.DSLSource
	}
}

func appendDispatchTraceMetadata(metadata map[string]any, input DispatchInput, dispatchID, traceID string) {
	if len(input.CallChain) > 0 {
		metadata[dispatchMetaCallChain] = append([]string(nil), input.CallChain...)
	}
	if traceID != "" {
		metadata[dispatchMetaTraceID] = traceID
	}
	if dispatchID != "" {
		metadata[dispatchMetaDispatchID] = dispatchID
	}
	if input.TimeoutSec > 0 {
		metadata[dispatchMetaTimeoutSec] = input.TimeoutSec
	}
}

func buildA2ADispatchMessage(input DispatchInput) *a2a.Message {
	return &a2a.Message{
		ID:    a2a.NewMessageID(),
		Role:  a2a.MessageRoleUser,
		Parts: a2a.ContentParts{a2a.TextPart{Text: dispatchPromptText(input)}},
	}
}

func dispatchPromptText(input DispatchInput) string {
	if strings.TrimSpace(input.WorkflowName) != "" {
		return fmt.Sprintf("Dispatch workflow %s to agent %s", input.WorkflowName, input.TargetAgent)
	}
	return fmt.Sprintf("Dispatch DSL workflow to agent %s", input.TargetAgent)
}

func mapA2ASendResult(result a2a.SendMessageResult) (*DispatchResponse, error) {
	switch item := result.(type) {
	case *a2a.Task:
		return mapA2ATask(item)
	case *a2a.Message:
		return mapA2AMessage(item)
	default:
		return nil, fmt.Errorf("%w: unsupported A2A send result %T", ErrInvalidDispatchResponse, result)
	}
}

func mapA2ATask(task *a2a.Task) (*DispatchResponse, error) {
	if task == nil {
		return nil, fmt.Errorf("%w: task result is nil", ErrInvalidDispatchResponse)
	}
	if delegated, ok, err := dispatchResponseFromMetadata(task.Metadata, string(task.ID)); ok || err != nil {
		return delegated, err
	}
	switch task.Status.State {
	case a2a.TaskStateRejected, a2a.TaskStateFailed, a2a.TaskStateCanceled:
		return NewDispatchRejectedResponse(taskStatusReason(task.Status), map[string]any{
			"task_id": string(task.ID),
			"state":   string(task.Status.State),
		})
	case a2a.TaskStateCompleted:
		return NewDispatchDelegatedResponse(taskTarget(task), string(task.ID), map[string]any{
			"task_id": string(task.ID),
			"state":   string(task.Status.State),
		})
	default:
		return NewDispatchAcceptedResponse(string(task.ID), map[string]any{
			"task_id": string(task.ID),
			"state":   string(task.Status.State),
		}), nil
	}
}

func mapA2AMessage(message *a2a.Message) (*DispatchResponse, error) {
	if message == nil {
		return nil, fmt.Errorf("%w: message result is nil", ErrInvalidDispatchResponse)
	}
	if resp, ok, err := dispatchResponseFromMetadata(message.Metadata, ""); ok || err != nil {
		return resp, err
	}
	return NewDispatchAcceptedResponse("", map[string]any{
		"message_id": message.ID,
	}), nil
}

func dispatchResponseFromMetadata(metadata map[string]any, runID string) (*DispatchResponse, bool, error) {
	if len(metadata) == 0 {
		return nil, false, nil
	}
	rawStatus, _ := metadata["dispatch_status"].(string)
	switch strings.ToUpper(strings.TrimSpace(rawStatus)) {
	case string(DispatchStatusAccepted):
		return NewDispatchAcceptedResponse(runID, metadata), true, nil
	case string(DispatchStatusRejected):
		reason, _ := metadata["reason"].(string)
		resp, err := NewDispatchRejectedResponse(reason, metadata)
		return resp, true, err
	case string(DispatchStatusDelegated):
		target, _ := metadata["target_agent"].(string)
		resp, err := NewDispatchDelegatedResponse(target, runID, metadata)
		return resp, true, err
	default:
		return nil, false, nil
	}
}

func dispatchResponseMetadata(endpoint DispatchEndpoint, dispatchID, traceID string, callChain []string) map[string]any {
	metadata := map[string]any{
		dispatchMetaEndpointURL:       endpoint.URL,
		dispatchMetaEndpointTransport: endpoint.Transport,
	}
	if dispatchID != "" {
		metadata[dispatchMetaDispatchID] = dispatchID
	}
	if traceID != "" {
		metadata[dispatchMetaTraceID] = traceID
	}
	if len(callChain) > 0 {
		metadata[dispatchMetaCallChain] = append([]string(nil), callChain...)
	}
	return metadata
}

func mergeDispatchResponseMetadata(base map[string]any, extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return base
	}
	merged := cloneDispatchMetadata(base)
	if merged == nil {
		merged = make(map[string]any, len(extra))
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

func dispatchSessionID(auth *DispatchAuthConfig, dispatchID string) string {
	if auth != nil && strings.TrimSpace(auth.SessionID) != "" {
		return strings.TrimSpace(auth.SessionID)
	}
	if strings.TrimSpace(dispatchID) != "" {
		return strings.TrimSpace(dispatchID)
	}
	return "fenix-dispatch"
}

func normalizeDispatchID(input DispatchInput) string {
	if strings.TrimSpace(input.DispatchID) != "" {
		return strings.TrimSpace(input.DispatchID)
	}
	return a2a.NewMessageID()
}

func normalizeTraceID(input DispatchInput) string {
	return strings.TrimSpace(input.TraceID)
}

func agentCardRequiresAuth(card *a2a.AgentCard) bool {
	return card != nil && len(card.Security) > 0 && len(card.SecuritySchemes) > 0
}

func taskStatusReason(status a2a.TaskStatus) string {
	if status.Message == nil {
		return fmt.Sprintf("remote task returned %s", status.State)
	}
	if text := firstTextPart(status.Message); text != "" {
		return text
	}
	return fmt.Sprintf("remote task returned %s", status.State)
}

func taskTarget(task *a2a.Task) string {
	if task == nil {
		return ""
	}
	if raw, ok := task.Metadata["target_agent"].(string); ok && strings.TrimSpace(raw) != "" {
		return strings.TrimSpace(raw)
	}
	return string(task.ID)
}

func firstTextPart(message *a2a.Message) string {
	if message == nil {
		return ""
	}
	for _, part := range message.Parts {
		text, ok := part.(a2a.TextPart)
		if ok && strings.TrimSpace(text.Text) != "" {
			return strings.TrimSpace(text.Text)
		}
	}
	return ""
}
