package evidence

import "errors"

var ErrVerificationProgressInvalid = errors.New("invalid evidence verification progress")

// VerificationProgress is the compact lifecycle result returned by an evidence provider.
// Full verification bundles remain provider-owned and must not be persisted by Fenix.
type VerificationProgress struct {
	Checkpoint CheckpointReference
	Status     VerificationStatus
	Issues     []string
}

// Validate enforces the compact provider-to-Fenix verification contract.
func (p VerificationProgress) Validate() error {
	if !validCheckpointReference(p.Checkpoint) {
		return ErrVerificationProgressInvalid
	}
	if p.Status != VerificationVerified && p.Status != VerificationFailed {
		return ErrVerificationProgressInvalid
	}
	return nil
}
