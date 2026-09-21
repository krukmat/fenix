package evidence

import "errors"

var ErrAuditProjectionInvalid = errors.New("invalid evidence audit projection")

// AuditProjection is the compact subset of a proof reference stored in Fenix operational audit.
type AuditProjection struct {
	Provider           string             `json:"provider"`
	ExecutionID        string             `json:"execution_id"`
	StreamID           string             `json:"stream_id"`
	EventID            string             `json:"event_id"`
	EventHash          string             `json:"event_hash"`
	KeyID              string             `json:"key_id"`
	SignatureRef       string             `json:"signature_ref"`
	Sequence           int64              `json:"sequence"`
	CheckpointID       string             `json:"checkpoint_id,omitempty"`
	MerkleRoot         string             `json:"merkle_root,omitempty"`
	TreeSize           int64              `json:"tree_size,omitempty"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	IssueCount         int                `json:"issue_count"`
}

// NewAuditProjection strips proof-bundle details and issue messages before operational audit storage.
func NewAuditProjection(ref ProofReference) (AuditProjection, error) {
	if err := ref.Validate(); err != nil {
		return AuditProjection{}, ErrAuditProjectionInvalid
	}
	projection := AuditProjection{
		Provider:           ref.Provider,
		ExecutionID:        ref.ExecutionID,
		StreamID:           ref.StreamID,
		EventID:            ref.EventID,
		EventHash:          ref.EventHash,
		KeyID:              ref.KeyID,
		SignatureRef:       ref.SignatureRef,
		Sequence:           ref.Sequence,
		VerificationStatus: ref.VerificationStatus,
		IssueCount:         len(ref.VerificationIssues),
	}
	if ref.Checkpoint != nil {
		projection.CheckpointID = ref.Checkpoint.CheckpointID
		projection.MerkleRoot = ref.Checkpoint.MerkleRoot
		projection.TreeSize = ref.Checkpoint.TreeSize
	}
	return projection, nil
}

// AgentEvidenceDisposition limits what an agent may claim about evidence integrity.
type AgentEvidenceDisposition string

const (
	AgentEvidenceUse     AgentEvidenceDisposition = "use"
	AgentEvidencePending AgentEvidenceDisposition = "pending"
	AgentEvidenceAbstain AgentEvidenceDisposition = "abstain"
)

// AgentEvidenceDecision explains how an agent may consume evidence state.
type AgentEvidenceDecision struct {
	Disposition AgentEvidenceDisposition `json:"disposition"`
	Reason      string                   `json:"reason"`
}

// EvaluateForAgent prevents agents from claiming cryptographic verification prematurely.
func EvaluateForAgent(state DeliveryState, ref *ProofReference) AgentEvidenceDecision {
	switch state {
	case DeliveryVerified:
		if ref != nil && ref.Validate() == nil && ref.VerificationStatus == VerificationVerified {
			return AgentEvidenceDecision{
				Disposition: AgentEvidenceUse,
				Reason:      "cryptographic evidence is verified",
			}
		}
		return AgentEvidenceDecision{
			Disposition: AgentEvidenceAbstain,
			Reason:      "verified lifecycle state lacks a valid verified proof reference",
		}
	case DeliveryPendingRecord, DeliveryRecorded, DeliveryPendingCheckpoint:
		return AgentEvidenceDecision{
			Disposition: AgentEvidencePending,
			Reason:      "evidence exists or is being recorded but is not yet verified",
		}
	case DeliveryVerificationFailed, DeliveryIndeterminate:
		return AgentEvidenceDecision{
			Disposition: AgentEvidenceAbstain,
			Reason:      "evidence integrity cannot currently be asserted",
		}
	default:
		return AgentEvidenceDecision{
			Disposition: AgentEvidenceAbstain,
			Reason:      "unknown evidence lifecycle state",
		}
	}
}
