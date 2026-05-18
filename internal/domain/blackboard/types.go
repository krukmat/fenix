// Package blackboard implements the shared cognitive workspace domain (ADR-100, Task A.1).
// Agents publish hypotheses, observations, and signals into a cognitive workspace,
// enabling collaborative reasoning and multi-agent coordination.
package blackboard

import (
	"encoding/json"
	"time"
)

// WorkspaceStatus represents the lifecycle state of a cognitive workspace.
type WorkspaceStatus string

const (
	WorkspaceStatusActive  WorkspaceStatus = "active"
	WorkspaceStatusClosed  WorkspaceStatus = "closed"
	WorkspaceStatusExpired WorkspaceStatus = "expired"
)

// EventType classifies what an agent published into the reasoning timeline.
type EventType string

const (
	EventTypeHypothesis     EventType = "hypothesis"
	EventTypeObservation    EventType = "observation"
	EventTypeRisk           EventType = "risk"
	EventTypeRecommendation EventType = "recommendation"
	EventTypeIntent         EventType = "intent"
)

// HypothesisStatus tracks the arbitration state of a signal hypothesis.
type HypothesisStatus string

const (
	HypothesisStatusOpen       HypothesisStatus = "open"
	HypothesisStatusAccepted   HypothesisStatus = "accepted"
	HypothesisStatusRejected   HypothesisStatus = "rejected"
	HypothesisStatusSuperseded HypothesisStatus = "superseded"
)

// MemoryScope determines the lifetime of an agent_memory entry.
type MemoryScope string

const (
	MemoryScopeSession    MemoryScope = "session"
	MemoryScopePersistent MemoryScope = "persistent"
)

// CognitiveWorkspace is the shared reasoning container for a multi-agent session.
// Optionally attached to an agent_run via AgentRunID.
type CognitiveWorkspace struct {
	ID          string
	WorkspaceID string
	AgentRunID  *string
	Status      WorkspaceStatus
	CreatedAt   time.Time
	ClosedAt    *time.Time
}

// ReasoningEvent is an append-only entry in the reasoning timeline.
// It is the source of truth for replay (Phase C).
type ReasoningEvent struct {
	ID                   string
	CognitiveWorkspaceID string
	ActorAgentID         *string
	EventType            EventType
	Payload              []byte
	CreatedAt            time.Time
}

// SignalHypothesis is a hypothesis posted by an agent, subject to confidence arbitration (Phase D).
type SignalHypothesis struct {
	ID                   string
	CognitiveWorkspaceID string
	SourceAgentID        *string
	Content              string
	Confidence           float64
	Status               HypothesisStatus
	CreatedAt            time.Time
	ResolvedAt           *time.Time
}

// ArbitrationConfig defines how confidence arbitration scores hypotheses.
type ArbitrationConfig struct {
	Now                    time.Time
	RecencyHalfLife        time.Duration
	SourceAgentReliability map[string]float64
	PersistResult          bool
	MemoryKey              string
}

// ArbitrationScoreBreakdown explains how one ranked hypothesis received its score.
type ArbitrationScoreBreakdown struct {
	Confidence  float64 `json:"confidence"`
	Recency     float64 `json:"recency"`
	Reliability float64 `json:"reliability"`
	Final       float64 `json:"final"`
}

// RankedHypothesis is one arbitration output row with explicit score details.
type RankedHypothesis struct {
	Rank       int                       `json:"rank"`
	Hypothesis SignalHypothesis          `json:"hypothesis"`
	Score      float64                   `json:"score"`
	Breakdown  ArbitrationScoreBreakdown `json:"breakdown"`
}

// ArbitrationResult is the deterministic ranking output for one workspace.
type ArbitrationResult struct {
	CognitiveWorkspaceID string             `json:"cognitive_workspace_id"`
	GeneratedAt          time.Time          `json:"generated_at"`
	Ranked               []RankedHypothesis `json:"ranked"`
}

// PlanningState represents the planner decision after combining ranked hypotheses,
// evidence, and policy constraints.
type PlanningState string

const (
	PlanningStateNoAction         PlanningState = "no_action"
	PlanningStateAwaitingEvidence PlanningState = "awaiting_evidence"
	PlanningStateNeedsReview      PlanningState = "needs_review"
	PlanningStatePendingApproval  PlanningState = "pending_approval"
	PlanningStateReady            PlanningState = "ready"
)

// PlanningConfig defines how collaborative planning reads and persists its outputs.
type PlanningConfig struct {
	Now                  time.Time
	ArbitrationMemoryKey string
	ResultMemoryKey      string
	MinReadyScore        float64
	PersistResult        bool
}

// ToolSequenceStep is one planned governed action in execution order.
type ToolSequenceStep struct {
	Sequence         int             `json:"sequence"`
	ToolName         string          `json:"tool_name"`
	Reason           string          `json:"reason"`
	RequiresApproval bool            `json:"requires_approval"`
	Params           json.RawMessage `json:"params,omitempty"`
}

// CollaborativePlanProposal is one deterministic proposal synthesized from a ranked hypothesis.
type CollaborativePlanProposal struct {
	ProposalID     string             `json:"proposal_id"`
	HypothesisID   string             `json:"hypothesis_id"`
	HypothesisRank int                `json:"hypothesis_rank"`
	Summary        string             `json:"summary"`
	Score          float64            `json:"score"`
	State          PlanningState      `json:"state"`
	Constraints    []string           `json:"constraints"`
	Contributors   []string           `json:"contributors"`
	Steps          []ToolSequenceStep `json:"steps"`
}

// CollaborativePlanningResult is the planner output persisted for downstream governed execution.
type CollaborativePlanningResult struct {
	CognitiveWorkspaceID string                      `json:"cognitive_workspace_id"`
	GeneratedAt          time.Time                   `json:"generated_at"`
	State                PlanningState               `json:"state"`
	SelectedProposal     *CollaborativePlanProposal  `json:"selected_proposal,omitempty"`
	Proposals            []CollaborativePlanProposal `json:"proposals"`
}

// AgentMemory is a shared key-value entry accessible by all agents in a workspace.
// TTL is enforced lazily on read via ExpiresAt.
type AgentMemory struct {
	ID                   string
	CognitiveWorkspaceID string
	Key                  string
	Value                []byte
	Scope                MemoryScope
	ExpiresAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
