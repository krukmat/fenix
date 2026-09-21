package evidence

import (
	"errors"
	"strings"
)

var ErrProofReferenceInvalid = errors.New("invalid evidence proof reference")

type VerificationStatus string

const (
	VerificationRecorded         VerificationStatus = "recorded"
	VerificationVerified         VerificationStatus = "verified"
	VerificationFailed           VerificationStatus = "verification_failed"
	VerificationPendingCheckpoint VerificationStatus = "pending_checkpoint"
)

type CheckpointReference struct {
	CheckpointID   string `json:"checkpoint_id"`
	CheckpointHash string `json:"checkpoint_hash"`
	MerkleRoot     string `json:"merkle_root"`
	TreeSize       int64  `json:"tree_size"`
}

type ProofReference struct {
	SchemaVersion      string               `json:"schema_version"`
	Provider           string               `json:"provider"`
	ExecutionID        string               `json:"execution_id"`
	StreamID           string               `json:"stream_id"`
	EventID            string               `json:"event_id"`
	EventHash          string               `json:"event_hash"`
	KeyID              string               `json:"key_id"`
	SignatureRef       string               `json:"signature_ref"`
	Sequence           int64                `json:"sequence"`
	Checkpoint         *CheckpointReference `json:"checkpoint,omitempty"`
	VerificationStatus VerificationStatus   `json:"verification_status"`
	VerificationIssues []string             `json:"verification_issues,omitempty"`
}

func (p ProofReference) Validate() error {
	if !p.hasValidCore() || !p.hasValidCheckpoint() {
		return ErrProofReferenceInvalid
	}
	return nil
}

func (p ProofReference) hasValidCore() bool {
	return p.SchemaVersion == SchemaVersion &&
		p.hasValidIdentity() &&
		p.hasValidEventEvidence() &&
		validVerificationStatus(p.VerificationStatus)
}

func (p ProofReference) hasValidIdentity() bool {
	return strings.TrimSpace(p.Provider) != "" &&
		strings.TrimSpace(p.ExecutionID) != "" &&
		strings.TrimSpace(p.StreamID) != ""
}

func (p ProofReference) hasValidEventEvidence() bool {
	return strings.TrimSpace(p.EventID) != "" &&
		isSHA256Hex(p.EventHash) &&
		strings.TrimSpace(p.KeyID) != "" &&
		strings.TrimSpace(p.SignatureRef) != "" &&
		p.Sequence >= 1
}

func (p ProofReference) hasValidCheckpoint() bool {
	return p.Checkpoint == nil || validCheckpointReference(*p.Checkpoint)
}

func validCheckpointReference(checkpoint CheckpointReference) bool {
	return strings.TrimSpace(checkpoint.CheckpointID) != "" &&
		isSHA256Hex(checkpoint.CheckpointHash) &&
		isSHA256Hex(checkpoint.MerkleRoot) &&
		checkpoint.TreeSize >= 1
}

func validVerificationStatus(status VerificationStatus) bool {
	switch status {
	case VerificationRecorded,
		VerificationVerified,
		VerificationFailed,
		VerificationPendingCheckpoint:
		return true
	default:
		return false
	}
}
