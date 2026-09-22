package evidence

import (
	"errors"
	"testing"
)

func TestVerificationProgressValidateAcceptsTerminalStates(t *testing.T) {
	checkpoint := *validProofReference().Checkpoint
	for _, status := range []VerificationStatus{VerificationVerified, VerificationFailed} {
		progress := VerificationProgress{Checkpoint: checkpoint, Status: status}
		if err := progress.Validate(); err != nil {
			t.Fatalf("status %q Validate: %v", status, err)
		}
	}
}

func TestVerificationProgressValidateRejectsInvalidContract(t *testing.T) {
	checkpoint := *validProofReference().Checkpoint
	invalidStatus := VerificationProgress{Checkpoint: checkpoint, Status: VerificationRecorded}
	if err := invalidStatus.Validate(); !errors.Is(err, ErrVerificationProgressInvalid) {
		t.Fatalf("invalid status error = %v", err)
	}

	checkpoint.CheckpointID = ""
	invalidCheckpoint := VerificationProgress{Checkpoint: checkpoint, Status: VerificationVerified}
	if err := invalidCheckpoint.Validate(); !errors.Is(err, ErrVerificationProgressInvalid) {
		t.Fatalf("invalid checkpoint error = %v", err)
	}
}
