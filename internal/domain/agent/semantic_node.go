package agent

type SemanticNodeID string

type SemanticNodeKind string

const (
	SemanticNodeWorkflow  SemanticNodeKind = "workflow"
	SemanticNodeTrigger   SemanticNodeKind = "trigger"
	SemanticNodeAction    SemanticNodeKind = "action"
	SemanticNodeDecision  SemanticNodeKind = "decision"
	SemanticNodeGrounds   SemanticNodeKind = "grounds"
	SemanticNodePermit    SemanticNodeKind = "permit"
	SemanticNodeDelegate  SemanticNodeKind = "delegate"
	SemanticNodeInvariant SemanticNodeKind = "invariant"
	SemanticNodeBudget    SemanticNodeKind = "budget"

	// DSL v1 node kinds — extended profile until runtime exists. // CLSF-54
	SemanticNodeCall    SemanticNodeKind = "call"
	SemanticNodeApprove SemanticNodeKind = "approve"
)

type SemanticSourceKind string

const (
	SemanticSourceDSL    SemanticSourceKind = "dsl"
	SemanticSourceCarta  SemanticSourceKind = "carta"
	SemanticSourceLegacy SemanticSourceKind = "legacy"
	SemanticSourceSystem SemanticSourceKind = "system"
)

type SemanticEffectKind string

const (
	SemanticEffectNone       SemanticEffectKind = "none"
	SemanticEffectRead       SemanticEffectKind = "read"
	SemanticEffectWrite      SemanticEffectKind = "write"
	SemanticEffectNotify     SemanticEffectKind = "notify"
	SemanticEffectDelegate   SemanticEffectKind = "delegate"
	SemanticEffectGovernance SemanticEffectKind = "governance"
)

type WorkflowSemanticNode struct {
	ID         SemanticNodeID     `json:"id"`
	Kind       SemanticNodeKind   `json:"kind"`
	Label      string             `json:"label,omitempty"`
	Source     SemanticSourceKind `json:"source"`
	Position   Position           `json:"position,omitempty"`
	Effect     SemanticEffectKind `json:"effect,omitempty"`
	Properties map[string]any     `json:"properties,omitempty"`
}

func NewWorkflowSemanticNode(id SemanticNodeID, kind SemanticNodeKind, source SemanticSourceKind) WorkflowSemanticNode {
	return WorkflowSemanticNode{
		ID:     id,
		Kind:   kind,
		Source: source,
		Effect: SemanticEffectNone,
	}
}

func SupportedSemanticNodeKinds() []SemanticNodeKind {
	return []SemanticNodeKind{
		SemanticNodeWorkflow,
		SemanticNodeTrigger,
		SemanticNodeAction,
		SemanticNodeDecision,
		SemanticNodeGrounds,
		SemanticNodePermit,
		SemanticNodeDelegate,
		SemanticNodeInvariant,
		SemanticNodeBudget,
	}
}
