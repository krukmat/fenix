package agent

import "testing"

func TestDispatchInputValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   DispatchInput
		wantErr bool
	}{
		{
			name: "valid with workflow name",
			input: DispatchInput{
				TargetAgent:  "support_agent",
				WorkflowName: "delegate_case",
			},
		},
		{
			name: "valid with dsl source",
			input: DispatchInput{
				TargetAgent: "support_agent",
				DSLSource:   "WORKFLOW delegate_case\nON case.created\nSET case.status = \"resolved\"",
			},
		},
		{
			name: "missing target",
			input: DispatchInput{
				WorkflowName: "delegate_case",
			},
			wantErr: true,
		},
		{
			name: "missing workflow and dsl",
			input: DispatchInput{
				TargetAgent: "support_agent",
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			input: DispatchInput{
				TargetAgent:  "support_agent",
				WorkflowName: "delegate_case",
				TimeoutSec:   -1,
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint override missing transport",
			input: DispatchInput{
				TargetAgent:  "support_agent",
				WorkflowName: "delegate_case",
				Endpoint: &DispatchEndpoint{
					URL: "https://example.com/a2a",
				},
			},
			wantErr: true,
		},
		{
			name: "call chain does not invalidate local input contract",
			input: DispatchInput{
				TargetAgent:  "support_agent",
				WorkflowName: "delegate_case",
				CallChain:    []string{"support_agent"},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.input.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDispatchTargetInCallChain(t *testing.T) {
	t.Parallel()

	if !dispatchTargetInCallChain("support_agent", []string{"insights", "support_agent"}) {
		t.Fatal("expected target in call chain")
	}
	if dispatchTargetInCallChain("support_agent", []string{"insights", "kb"}) {
		t.Fatal("did not expect target in call chain")
	}
}

func TestDispatchResponseValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		resp    DispatchResponse
		wantErr bool
	}{
		{name: "accepted valid", resp: DispatchResponse{Status: DispatchStatusAccepted}},
		{name: "rejected valid", resp: DispatchResponse{Status: DispatchStatusRejected, Reason: "denied"}},
		{name: "delegated valid", resp: DispatchResponse{Status: DispatchStatusDelegated, Target: "product_agent"}},
		{name: "rejected missing reason", resp: DispatchResponse{Status: DispatchStatusRejected}, wantErr: true},
		{name: "delegated missing target", resp: DispatchResponse{Status: DispatchStatusDelegated}, wantErr: true},
		{name: "unsupported status", resp: DispatchResponse{Status: "PENDING"}, wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.resp.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDispatchResponseConstructors(t *testing.T) {
	t.Parallel()

	accepted := NewDispatchAcceptedResponse("run-1", map[string]any{"trace_id": "tr-1"})
	if accepted.Status != DispatchStatusAccepted || accepted.RunID != "run-1" || accepted.Adapter != dispatchAdapterA2A {
		t.Fatalf("unexpected accepted response = %#v", accepted)
	}
	if accepted.Metadata["dispatch_status"] != string(DispatchStatusAccepted) {
		t.Fatalf("expected accepted metadata status, got %#v", accepted.Metadata)
	}

	rejected, err := NewDispatchRejectedResponse("denied", map[string]any{"reason_code": "policy"})
	if err != nil {
		t.Fatalf("NewDispatchRejectedResponse() error = %v", err)
	}
	if rejected.Status != DispatchStatusRejected || rejected.Reason != "denied" {
		t.Fatalf("unexpected rejected response = %#v", rejected)
	}
	if rejected.Metadata["dispatch_status"] != string(DispatchStatusRejected) || rejected.Metadata["reason"] != "denied" {
		t.Fatalf("expected rejected metadata reason, got %#v", rejected.Metadata)
	}

	delegated, err := NewDispatchDelegatedResponse("specialist_agent", "run-2", map[string]any{"trace_id": "tr-2"})
	if err != nil {
		t.Fatalf("NewDispatchDelegatedResponse() error = %v", err)
	}
	if delegated.Status != DispatchStatusDelegated || delegated.Target != "specialist_agent" || delegated.RunID != "run-2" {
		t.Fatalf("unexpected delegated response = %#v", delegated)
	}
	if delegated.Metadata["dispatch_status"] != string(DispatchStatusDelegated) {
		t.Fatalf("expected delegated metadata status, got %#v", delegated.Metadata)
	}
}
