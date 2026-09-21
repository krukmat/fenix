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
	Sequence           int64                `json:"sequence"`
	Checkpoint         *CheckpointReference `json:"checkpoint,omitempty"`
	VerificationStatus VerificationStatus   `json:"verification_status"`
	VerificationIssues []string             `json:"verification_issues,omitempty"`
}

func (p ProofReference) Validate() error {
	if p.SchemaVersion != SchemaVersion ||
		strings.TrimSpace(p.Provider) == "" ||
		strings.TrimSpace(p.ExecutionID) == "" ||
		strings.TrimSpace(p.StreamID) == "" ||
		strings.TrimSpace(p.EventID) == "" ||
		!isSHA256Hex(p.EventHash) ||
		strings.TrimSpace(p.KeyID) == "" ||
		p.Sequence < 1 ||
		!validVerificationStatus(p.VerificationStatus) {
		return ErrProofReferenceInvalid
	}
	if p.Checkpoint != nil && !validCheckpointReference(*p.Checkpoint) {
		return ErrProofReferenceInvalid
	}
	return nil
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
