package eval

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// F1-T1/F1-T2: Golden scenario registry — deterministic expected behavior fixtures.
// Distinct from eval/suite.go (keyword-based TestCases). Both coexist.

// validDomains is the closed set of valid scenario domains.
var validDomains = map[string]struct{}{
	"support": {},
	"sales":   {},
	"general": {},
}

// validPolicyOutcomes is the closed set of valid policy decision outcomes.
var validPolicyOutcomes = map[string]struct{}{
	"allow":            {},
	"deny":             {},
	"require_approval": {},
}

// GoldenScenario represents a deterministic expected behavior fixture.
// Loaded from YAML; used by the comparator (Wave F3) and regression suite (Wave F7).
// Structure mirrors the Example Scenario Contract in the Wave F1 spec.
type GoldenScenario struct {
	ID           string             `yaml:"id"`
	Title        string             `yaml:"title"`
	Description  string             `yaml:"description"`
	Domain       string             `yaml:"domain"` // "support" | "sales" | "general"
	Tags         []string           `yaml:"tags"`
	InputEvent   ScenarioInputEvent `yaml:"input_event"`
	InitialState map[string]any     `yaml:"initial_state"`
	Expected     ScenarioExpected   `yaml:"expected"`
	Thresholds   ScenarioThresholds `yaml:"thresholds"`
}

// ScenarioInputEvent describes the triggering event for the scenario.
type ScenarioInputEvent struct {
	Type    string         `yaml:"type"`    // e.g. "case.created", "manual"
	Payload map[string]any `yaml:"payload"` // arbitrary event payload
}

// ScenarioExpected groups all expected-behavior fields for comparison by Wave F3.
type ScenarioExpected struct {
	FinalOutcome       string                    `yaml:"final_outcome"`     // e.g. "success", "abstained", "awaiting_approval", "escalated"
	RequiredEvidence   []string                  `yaml:"required_evidence"` // e.g. ["case:CASE-001", "account:ACC-001"]
	ForbiddenEvidence  []string                  `yaml:"forbidden_evidence"`
	PolicyDecisions    []ExpectedPolicyDecision  `yaml:"expected_policy_decisions"`
	ToolCalls          []ExpectedToolCall        `yaml:"expected_tool_calls"`
	ForbiddenToolCalls []ForbiddenToolCall       `yaml:"forbidden_tool_calls"`
	ApprovalBehavior   *ExpectedApprovalBehavior `yaml:"approval_behavior,omitempty"`
	AuditEvents        []string                  `yaml:"expected_audit_events"` // e.g. "agent.run.started", "tool.executed"
	FinalState         map[string]any            `yaml:"expected_final_state"`
	ShouldAbstain      bool                      `yaml:"should_abstain"`
	AbstainReason      string                    `yaml:"abstain_reason"`
}

// ExpectedToolCall describes a tool call the agent must produce.
type ExpectedToolCall struct {
	ToolName string         `yaml:"tool_name"`
	Params   map[string]any `yaml:"params"`   // expected param values (partial match allowed)
	Required bool           `yaml:"required"` // if true, absence is a hard failure
}

// ForbiddenToolCall describes a tool the agent must NOT call.
type ForbiddenToolCall struct {
	ToolName string `yaml:"tool_name"`
	Reason   string `yaml:"reason"` // human-readable rationale
}

// ExpectedPolicyDecision describes the expected outcome of a policy evaluation.
type ExpectedPolicyDecision struct {
	Action          string `yaml:"action"`           // e.g. "tool:send_email"
	ExpectedOutcome string `yaml:"expected_outcome"` // "allow" | "deny" | "require_approval"
}

// ExpectedApprovalBehavior describes whether an approval request is expected and its outcome.
type ExpectedApprovalBehavior struct {
	Required        bool   `yaml:"required"`         // true = an approval request must be created
	ExpectedOutcome string `yaml:"expected_outcome"` // "approved" | "rejected" | "pending"
}

// ScenarioThresholds defines performance acceptance gates for the scenario.
type ScenarioThresholds struct {
	MinScore     int `yaml:"min_score" json:"min_score"`           // minimum composite score (0-100)
	MaxLatencyMs int `yaml:"max_latency_ms" json:"max_latency_ms"` // maximum allowed latency in milliseconds
	MaxToolCalls int `yaml:"max_tool_calls" json:"max_tool_calls"` // maximum number of tool calls allowed
	MaxRetries   int `yaml:"max_retries" json:"max_retries"`       // maximum number of retry attempts allowed
}

// LoadGoldenScenario reads and validates a YAML fixture from disk.
// F1-T1/F1-T2: used by tests and the regression runner (Wave F7).
func LoadGoldenScenario(path string) (*GoldenScenario, error) {
	data, readErr := os.ReadFile(path) //nolint:gosec // path comes from trusted internal testdata
	if readErr != nil {
		return nil, fmt.Errorf("read scenario file %q: %w", path, readErr)
	}

	var sc GoldenScenario
	if parseErr := yaml.Unmarshal(data, &sc); parseErr != nil {
		return nil, fmt.Errorf("parse scenario YAML %q: %w", path, parseErr)
	}

	if valErr := sc.Validate(); valErr != nil {
		return nil, fmt.Errorf("invalid scenario %q: %w", path, valErr)
	}

	return &sc, nil
}

// Validate checks that the scenario has the minimum required fields and valid values.
func (s *GoldenScenario) Validate() error {
	if s.ID == "" {
		return errors.New("scenario id is required")
	}
	if _, ok := validDomains[s.Domain]; !ok {
		return fmt.Errorf("invalid domain %q: must be one of support, sales, general", s.Domain)
	}
	if s.InputEvent.Type == "" {
		return errors.New("input_event.type is required")
	}
	for i, pd := range s.Expected.PolicyDecisions {
		if pd.Action == "" {
			return fmt.Errorf("expected_policy_decisions[%d].action is required", i)
		}
		if _, ok := validPolicyOutcomes[pd.ExpectedOutcome]; !ok {
			return fmt.Errorf("expected_policy_decisions[%d].expected_outcome %q: must be allow, deny, or require_approval", i, pd.ExpectedOutcome)
		}
	}
	return nil
}
