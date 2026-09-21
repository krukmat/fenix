package evidence

import "strings"

// DeliveryState describes where verifiable evidence is in its independent lifecycle.
type DeliveryState string

const (
	DeliveryPendingRecord       DeliveryState = "pending_record"
	DeliveryRecorded            DeliveryState = "recorded"
	DeliveryPendingCheckpoint   DeliveryState = "pending_checkpoint"
	DeliveryVerified            DeliveryState = "verified"
	DeliveryVerificationFailed  DeliveryState = "verification_failed"
	DeliveryIndeterminate       DeliveryState = "indeterminate"
)

// ReconciliationAction describes the next safe evidence-only recovery step.
type ReconciliationAction string

const (
	ReconcileNone            ReconciliationAction = "none"
	ReconcileLookupExisting  ReconciliationAction = "lookup_existing"
	ReconcileRetryRecord     ReconciliationAction = "retry_record"
	ReconcileAwaitCheckpoint ReconciliationAction = "await_checkpoint"
	ReconcileVerify          ReconciliationAction = "verify"
	ReconcileEscalate        ReconciliationAction = "escalate"
)

// ReconciliationDecision deliberately separates evidence recovery from business execution.
type ReconciliationDecision struct {
	Action                  ReconciliationAction `json:"action"`
	Reason                  string               `json:"reason"`
	MayRepeatBusinessAction bool                 `json:"may_repeat_business_action"`
}

// IdempotencyKey returns the stable VEL append key for one logical Fenix execution.
func IdempotencyKey(envelope Envelope) string {
	executionID := strings.TrimSpace(envelope.ExecutionID)
	if executionID == "" {
		return ""
	}
	return "fenix:evidence:v1:" + executionID
}

// DecideReconciliation returns the safe next evidence-only action.
// No evidence state is allowed to repeat the governed business capability.
func DecideReconciliation(state DeliveryState) ReconciliationDecision {
	switch state {
	case DeliveryPendingRecord:
		return reconciliationDecision(ReconcileRetryRecord, "recording has not been attempted or was safely rejected")
	case DeliveryRecorded:
		return reconciliationDecision(ReconcileAwaitCheckpoint, "event is recorded and awaits checkpoint coverage")
	case DeliveryPendingCheckpoint:
		return reconciliationDecision(ReconcileAwaitCheckpoint, "event is recorded but not yet checkpointed")
	case DeliveryVerified:
		return reconciliationDecision(ReconcileNone, "evidence is verified")
	case DeliveryVerificationFailed:
		return reconciliationDecision(ReconcileEscalate, "cryptographic verification failed")
	case DeliveryIndeterminate:
		return reconciliationDecision(ReconcileLookupExisting, "append outcome is uncertain; resolve by idempotency key before retry")
	default:
		return reconciliationDecision(ReconcileEscalate, "unknown evidence lifecycle state")
	}
}

func reconciliationDecision(action ReconciliationAction, reason string) ReconciliationDecision {
	return ReconciliationDecision{
		Action:                  action,
		Reason:                  reason,
		MayRepeatBusinessAction: false,
	}
}
