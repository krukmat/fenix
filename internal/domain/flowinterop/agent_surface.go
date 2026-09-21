package flowinterop

// AgentDisposition controls whether an agent may consume a semantic result automatically.
type AgentDisposition string

const (
	AgentUse        AgentDisposition = "use"
	AgentReviewOnly AgentDisposition = "review_only"
	AgentAbstain    AgentDisposition = "abstain"
)

// AgentPolicyDecision explains how an agent may consume a capability result.
type AgentPolicyDecision struct {
	Disposition AgentDisposition `json:"disposition"`
	Reason      string           `json:"reason"`
}

// AgentCallable reports whether the operation is safe for autonomous invocation.
// W2 deliberately exposes only VERIFY and TRANSFORM capabilities; no Salesforce mutation is present.
func AgentCallable(operation Operation) bool {
	spec, ok := Lookup(operation)
	if !ok {
		return false
	}
	return spec.Descriptor.SideEffectClass == "verify" ||
		spec.Descriptor.SideEffectClass == "transform"
}

// EvaluateAgentResult separates capability invocation safety from result trust.
func EvaluateAgentResult(result Result) AgentPolicyDecision {
	if result.Status == ResultRejected {
		return AgentPolicyDecision{Disposition: AgentAbstain, Reason: "semantic operation was rejected"}
	}
	if result.Status == ResultUnsupported || result.Fidelity.Level == FidelityUnsupported {
		return AgentPolicyDecision{Disposition: AgentAbstain, Reason: "semantic boundary is unsupported"}
	}
	if result.Fidelity.Level == FidelityPartial {
		return AgentPolicyDecision{Disposition: AgentReviewOnly, Reason: "semantic fidelity is partial"}
	}
	if result.Status == ResultSucceeded && result.Fidelity.Level == FidelityGuaranteed {
		return AgentPolicyDecision{Disposition: AgentUse, Reason: "semantic fidelity is guaranteed"}
	}
	return AgentPolicyDecision{Disposition: AgentReviewOnly, Reason: "result requires explicit review"}
}
