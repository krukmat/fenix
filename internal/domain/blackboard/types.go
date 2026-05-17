// Package blackboard implements the shared cognitive workspace domain (ADR-100, Task A.1).
// Agents publish hypotheses, observations, and signals into a cognitive workspace,
// enabling collaborative reasoning and multi-agent coordination.
package blackboard

import "time"

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
