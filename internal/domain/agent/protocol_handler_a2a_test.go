package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
)

type stubA2AClientFactory struct {
	client   a2aDispatchClient
	err      error
	lastCard *a2a.AgentCard
	lastOpts []a2aclient.FactoryOption
}

func (f *stubA2AClientFactory) NewFromCard(_ context.Context, card *a2a.AgentCard, opts ...a2aclient.FactoryOption) (a2aDispatchClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastCard = card
	f.lastOpts = append([]a2aclient.FactoryOption(nil), opts...)
	return f.client, nil
}

type stubA2AClient struct {
	result       a2a.SendMessageResult
	err          error
	destroyCalls int
	lastMessage  *a2a.MessageSendParams
	deadlineSeen bool
}

func (c *stubA2AClient) SendMessage(ctx context.Context, message *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	c.lastMessage = message
	_, c.deadlineSeen = ctx.Deadline()
	if c.err != nil {
		return nil, c.err
	}
	return c.result, nil
}

func (c *stubA2AClient) Destroy() error {
	c.destroyCalls++
	return nil
}

func TestA2AProtocolHandlerDispatchRequiresAgentCard(t *testing.T) {
	t.Parallel()

	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{})
	_, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
	})
	if !errors.Is(err, ErrA2AAgentCardRequired) {
		t.Fatalf("expected ErrA2AAgentCardRequired, got %v", err)
	}
}

func TestA2AProtocolHandlerDispatchRejectsCircularDelegation(t *testing.T) {
	t.Parallel()

	client := &stubA2AClient{result: &a2a.Message{ID: "msg-0"}}
	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{client: client})

	resp, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		CallChain:    []string{"insights", "support_agent"},
		AgentCard:    testDispatchAgentCard(),
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if resp.Status != DispatchStatusRejected || resp.Reason != dispatchRejectLoop {
		t.Fatalf("unexpected response = %#v", resp)
	}
	if client.lastMessage != nil {
		t.Fatal("expected no remote call on circular delegation")
	}
}

func TestA2AProtocolHandlerDispatchBuildsA2AMessage(t *testing.T) {
	t.Parallel()

	client := &stubA2AClient{result: &a2a.Message{ID: "msg-1"}}
	factory := &stubA2AClientFactory{client: client}
	handler := NewA2AProtocolHandlerWithFactory(factory)

	card := testDispatchAgentCard()
	_, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		TraceID:      "trace-1",
		DispatchID:   "disp-1",
		CallChain:    []string{"insights"},
		TimeoutSec:   5,
		AgentCard:    card,
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if client.lastMessage == nil || client.lastMessage.Message == nil {
		t.Fatal("expected a2a message")
	}
	if client.lastMessage.Metadata[dispatchMetaMode] != dispatchMetaModeValue {
		t.Fatalf("unexpected mode metadata = %#v", client.lastMessage.Metadata)
	}
	if client.lastMessage.Metadata[dispatchMetaDispatchID] != "disp-1" {
		t.Fatalf("unexpected dispatch id metadata = %#v", client.lastMessage.Metadata)
	}
	if !client.deadlineSeen {
		t.Fatal("expected timeout deadline in context")
	}
}

func TestA2AProtocolHandlerDispatchRequiresAuthForSecuredCard(t *testing.T) {
	t.Parallel()

	client := &stubA2AClient{result: &a2a.Message{ID: "msg-1"}}
	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{client: client})

	_, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		AgentCard:    testSecuredDispatchAgentCard(),
	})
	if !errors.Is(err, ErrDispatchAuthRequired) {
		t.Fatalf("expected ErrDispatchAuthRequired, got %v", err)
	}
}

func TestA2AProtocolHandlerDispatchUsesAuthSessionAndHeaders(t *testing.T) {
	t.Parallel()

	client := &stubA2AClient{result: &a2a.Message{ID: "msg-1"}}
	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{client: client})

	_, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		DispatchID:   "disp-2",
		AgentCard:    testSecuredDispatchAgentCard(),
		Auth: &DispatchAuthConfig{
			SessionID: "session-1",
			Credentials: map[string]string{
				"bearerAuth": "jwt-123",
			},
			Headers: map[string]string{
				"x-customer-id": "cust-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
}

func TestBuildA2ACallMetaIncludesTraceAndHeaders(t *testing.T) {
	t.Parallel()

	meta := buildA2ACallMeta(DispatchInput{
		CallChain: []string{"insights", "support"},
		Auth: &DispatchAuthConfig{
			Headers: map[string]string{"x-customer-id": "cust-1"},
		},
	}, "disp-1", "trace-1")

	if got := meta.Get(dispatchHeaderTraceID); len(got) != 1 || got[0] != "trace-1" {
		t.Fatalf("unexpected trace header = %#v", meta)
	}
	if got := meta.Get(dispatchHeaderDispatchID); len(got) != 1 || got[0] != "disp-1" {
		t.Fatalf("unexpected dispatch header = %#v", meta)
	}
	if got := meta.Get(dispatchHeaderCallChain); len(got) != 1 || got[0] != "insights,support" {
		t.Fatalf("unexpected call chain header = %#v", meta)
	}
	if got := meta.Get("x-customer-id"); len(got) != 1 || got[0] != "cust-1" {
		t.Fatalf("unexpected custom header = %#v", meta)
	}
}

func TestDispatchContextSetsSessionID(t *testing.T) {
	t.Parallel()

	ctx, cancel := dispatchContext(context.Background(), 0, &DispatchAuthConfig{SessionID: "session-1"}, "disp-2")
	defer cancel()

	sid, ok := a2aclient.SessionIDFrom(ctx)
	if !ok || string(sid) != "session-1" {
		t.Fatalf("unexpected session id = %q, ok=%v", sid, ok)
	}
}

func TestA2AProtocolHandlerDispatchUsesEndpointOverride(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	client := &stubA2AClient{
		result: &a2a.Task{
			ID:        a2a.TaskID("task-1"),
			ContextID: "ctx-1",
			Status: a2a.TaskStatus{
				State:     a2a.TaskStateWorking,
				Timestamp: &now,
			},
		},
	}
	factory := &stubA2AClientFactory{client: client}
	handler := NewA2AProtocolHandlerWithFactory(factory)

	resp, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		DispatchID:   "disp-3",
		AgentCard:    testDispatchAgentCard(),
		Endpoint: &DispatchEndpoint{
			URL:       "https://dispatch.example.com/a2a/grpc",
			Transport: "GRPC",
		},
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if factory.lastCard == nil || factory.lastCard.URL != "https://dispatch.example.com/a2a/grpc" || factory.lastCard.PreferredTransport != a2a.TransportProtocolGRPC {
		t.Fatalf("unexpected card override = %#v", factory.lastCard)
	}
	if resp.Metadata[dispatchMetaEndpointTransport] != "GRPC" {
		t.Fatalf("unexpected endpoint metadata = %#v", resp.Metadata)
	}
}

func TestA2AProtocolHandlerDispatchMapsRejectedTask(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	client := &stubA2AClient{
		result: &a2a.Task{
			ID:        a2a.TaskID("task-2"),
			ContextID: "ctx-2",
			Status: a2a.TaskStatus{
				State:     a2a.TaskStateRejected,
				Timestamp: &now,
				Message: &a2a.Message{
					ID:    "m-1",
					Role:  a2a.MessageRoleAgent,
					Parts: a2a.ContentParts{a2a.TextPart{Text: "policy denied"}},
				},
			},
		},
	}
	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{client: client})

	resp, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		DispatchID:   "disp-4",
		TraceID:      "trace-2",
		AgentCard:    testDispatchAgentCard(),
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if resp.Status != DispatchStatusRejected || resp.Reason != "policy denied" {
		t.Fatalf("unexpected response = %#v", resp)
	}
	if resp.Metadata[dispatchMetaDispatchID] != "disp-4" || resp.Metadata[dispatchMetaTraceID] != "trace-2" {
		t.Fatalf("unexpected response metadata = %#v", resp.Metadata)
	}
	if resp.Metadata["reason"] != "policy denied" {
		t.Fatalf("expected canonical rejection reason in metadata, got %#v", resp.Metadata)
	}
}

func TestA2AProtocolHandlerDispatchMapsDelegatedFromMetadata(t *testing.T) {
	t.Parallel()

	client := &stubA2AClient{
		result: &a2a.Message{
			ID:   "msg-2",
			Role: a2a.MessageRoleAgent,
			Metadata: map[string]any{
				"dispatch_status": "DELEGATED",
				"target_agent":    "product_agent",
			},
		},
	}
	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{client: client})

	resp, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		AgentCard:    testDispatchAgentCard(),
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if resp.Status != DispatchStatusDelegated || resp.Target != "product_agent" {
		t.Fatalf("unexpected response = %#v", resp)
	}
}

func TestA2AProtocolHandlerDispatchRejectsRejectedMetadataWithoutReason(t *testing.T) {
	t.Parallel()

	client := &stubA2AClient{
		result: &a2a.Message{
			ID:   "msg-3",
			Role: a2a.MessageRoleAgent,
			Metadata: map[string]any{
				"dispatch_status": "REJECTED",
			},
		},
	}
	handler := NewA2AProtocolHandlerWithFactory(&stubA2AClientFactory{client: client})

	_, err := handler.Dispatch(context.Background(), DispatchInput{
		TargetAgent:  "support_agent",
		WorkflowName: "delegate_case",
		AgentCard:    testDispatchAgentCard(),
	})
	if err == nil {
		t.Fatal("expected error for rejected metadata without reason")
	}
}

func TestMapA2ATaskVariantsAndFirstTextPart(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	completed, err := mapA2ATask(&a2a.Task{
		ID: "task-completed",
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateCompleted,
			Timestamp: &now,
		},
	})
	if err != nil {
		t.Fatalf("mapA2ATask(completed) error = %v", err)
	}
	if completed.Status != DispatchStatusDelegated || completed.Target != "task-completed" {
		t.Fatalf("unexpected completed response = %#v", completed)
	}

	accepted, err := mapA2ATask(&a2a.Task{
		ID: "task-working",
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateWorking,
			Timestamp: &now,
		},
	})
	if err != nil {
		t.Fatalf("mapA2ATask(working) error = %v", err)
	}
	if accepted.Status != DispatchStatusAccepted {
		t.Fatalf("unexpected accepted response = %#v", accepted)
	}

	rejected, err := mapA2ATask(&a2a.Task{
		ID: "task-rejected",
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateRejected,
			Timestamp: &now,
		},
	})
	if err != nil {
		t.Fatalf("mapA2ATask(rejected) error = %v", err)
	}
	if rejected.Status != DispatchStatusRejected || rejected.Reason == "" {
		t.Fatalf("unexpected rejected response = %#v", rejected)
	}

	msg := &a2a.Message{Parts: a2a.ContentParts{a2a.TextPart{Text: "  first  "}}}
	if got := firstTextPart(msg); got != "first" {
		t.Fatalf("firstTextPart() = %q", got)
	}
	if got := firstTextPart(&a2a.Message{}); got != "" {
		t.Fatalf("firstTextPart(empty) = %q", got)
	}
}

func testDispatchAgentCard() *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:               "ext",
		URL:                "https://example.com/a2a",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		ProtocolVersion:    string(a2a.Version),
	}
}

func testSecuredDispatchAgentCard() *a2a.AgentCard {
	card := testDispatchAgentCard()
	card.Security = []a2a.SecurityRequirements{
		{"bearerAuth": {}},
	}
	card.SecuritySchemes = a2a.NamedSecuritySchemes{
		"bearerAuth": a2a.HTTPAuthSecurityScheme{
			Scheme: "Bearer",
		},
	}
	return card
}

func TestA2AProtocolHandlerHelperCoverage(t *testing.T) {
	t.Parallel()

	base := []a2a.AgentInterface{{
		URL:       "https://example.com/jsonrpc",
		Transport: a2a.TransportProtocolJSONRPC,
	}}
	extra := []a2a.AgentInterface{
		{URL: "https://example.com/jsonrpc", Transport: a2a.TransportProtocolJSONRPC},
		{URL: "https://example.com/grpc", Transport: a2a.TransportProtocolGRPC},
		{URL: "", Transport: a2a.TransportProtocolJSONRPC},
	}
	got := appendUniqueInterfaces(base, extra)
	if len(got) != 2 {
		t.Fatalf("appendUniqueInterfaces() len = %d, want 2", len(got))
	}
	if !containsAgentInterface(got, a2a.AgentInterface{URL: "https://example.com/grpc", Transport: a2a.TransportProtocolGRPC}) {
		t.Fatal("containsAgentInterface() expected true")
	}

	auth := &DispatchAuthConfig{SessionID: " sess-1 "}
	if got := dispatchSessionID(auth, "dispatch-1"); got != "sess-1" {
		t.Fatalf("dispatchSessionID(auth) = %q", got)
	}
	if got := dispatchSessionID(nil, " dispatch-1 "); got != "dispatch-1" {
		t.Fatalf("dispatchSessionID(dispatch) = %q", got)
	}
	if got := dispatchSessionID(nil, ""); got != "fenix-dispatch" {
		t.Fatalf("dispatchSessionID(default) = %q", got)
	}

	if !agentCardRequiresAuth(testSecuredDispatchAgentCard()) {
		t.Fatal("agentCardRequiresAuth() expected true")
	}
	if agentCardRequiresAuth(testDispatchAgentCard()) {
		t.Fatal("agentCardRequiresAuth() expected false")
	}

	task := &a2a.Task{
		ID: "task-1",
		Metadata: map[string]any{
			"target_agent": "support_agent",
		},
	}
	if got := taskTarget(task); got != "support_agent" {
		t.Fatalf("taskTarget(metadata) = %q", got)
	}
	if got := taskTarget(&a2a.Task{ID: "task-2"}); got != "task-2" {
		t.Fatalf("taskTarget(id) = %q", got)
	}
	if got := taskTarget(nil); got != "" {
		t.Fatalf("taskTarget(nil) = %q", got)
	}
}

func TestA2AProtocolHandlerConstructorAndEndpointHelpers(t *testing.T) {
	t.Parallel()

	if handler := NewA2AProtocolHandler(); handler == nil || handler.factory == nil {
		t.Fatal("NewA2AProtocolHandler() expected non-nil factory")
	}
	if handler := NewA2AProtocolHandlerWithFactory(nil); handler == nil || handler.factory == nil {
		t.Fatal("NewA2AProtocolHandlerWithFactory(nil) expected default factory")
	}

	card := testDispatchAgentCard()
	if _, cloned, err := resolveDispatchEndpoint(DispatchInput{AgentCard: card}); err != nil || cloned == nil {
		t.Fatalf("resolveDispatchEndpoint(card) = %v, %v", cloned, err)
	}

	if err := validateDispatchEndpoint(DispatchEndpoint{}); err == nil {
		t.Fatal("validateDispatchEndpoint(empty) expected error")
	}
	if err := validateDispatchEndpoint(DispatchEndpoint{URL: "https://example.com/a2a", Transport: string(a2a.TransportProtocolJSONRPC)}); err != nil {
		t.Fatalf("validateDispatchEndpoint(valid) error = %v", err)
	}
	if _, err := parseDispatchTransport(string(a2a.TransportProtocolHTTPJSON)); err != nil {
		t.Fatalf("parseDispatchTransport(HTTPJSON) error = %v", err)
	}
	if _, err := parseDispatchTransport("weird"); err == nil {
		t.Fatal("parseDispatchTransport(weird) expected error")
	}

	if got := dispatchPromptText(DispatchInput{TargetAgent: "support", WorkflowName: "delegate_case"}); got == "" {
		t.Fatal("dispatchPromptText(workflow) expected text")
	}
	if got := dispatchPromptText(DispatchInput{TargetAgent: "support", DSLSource: "WORKFLOW x"}); got == "" {
		t.Fatal("dispatchPromptText(dsl) expected text")
	}

	reason := taskStatusReason(a2a.TaskStatus{State: a2a.TaskStateFailed})
	if reason == "" {
		t.Fatal("taskStatusReason(no message) expected fallback")
	}
	reason = taskStatusReason(a2a.TaskStatus{
		State: a2a.TaskStateRejected,
		Message: &a2a.Message{
			Parts: a2a.ContentParts{a2a.TextPart{Text: "policy denied"}},
		},
	})
	if reason != "policy denied" {
		t.Fatalf("taskStatusReason(message) = %q", reason)
	}
}
