// Package relationship implements the Relationship Memory Engine domain (ADR-101, Task B.1).
// Agents use this layer to persist stakeholder intelligence, interaction history,
// trust evolution, and influence graphs across CRM entities.
package relationship

import "time"

// EntityType identifies which CRM domain an entity belongs to.
// References are loose (no FK) to survive CRM record deletion.
type EntityType string

const (
	EntityTypeAccount EntityType = "account"
	EntityTypeContact EntityType = "contact"
	EntityTypeLead    EntityType = "lead"
	EntityTypeDeal    EntityType = "deal"
	EntityTypeCase    EntityType = "case"
)

// ToneType classifies the inferred communication tone with an entity.
type ToneType string

const (
	TonePositive ToneType = "positive"
	ToneNeutral  ToneType = "neutral"
	ToneNegative ToneType = "negative"
	ToneMixed    ToneType = "mixed"
)

// TrajectoryType describes the direction of a relationship over time.
type TrajectoryType string

const (
	TrajectoryImproving TrajectoryType = "improving"
	TrajectoryStable    TrajectoryType = "stable"
	TrajectortyDeclining TrajectoryType = "declining"
)

// SignalType classifies the CRM interaction that produced an interaction_signal.
type SignalType string

const (
	SignalEmail      SignalType = "email"
	SignalCall       SignalType = "call"
	SignalMeeting    SignalType = "meeting"
	SignalNote       SignalType = "note"
	SignalCaseUpdate SignalType = "case_update"
	SignalDealUpdate SignalType = "deal_update"
)

// SentimentType is the inferred emotional valence of an interaction signal.
type SentimentType string

const (
	SentimentPositive SentimentType = "positive"
	SentimentNeutral  SentimentType = "neutral"
	SentimentNegative SentimentType = "negative"
)

// InfluenceType describes the nature of a directed stakeholder edge.
type InfluenceType string

const (
	InfluenceReportsTo   InfluenceType = "reports_to"
	InfluenceInfluences  InfluenceType = "influences"
	InfluenceBlocks      InfluenceType = "blocks"
	InfluenceCollaborates InfluenceType = "collaborates"
	InfluenceApproves    InfluenceType = "approves"
)

// ConfidenceLevel represents the certainty tier of a trust score.
type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceLow    ConfidenceLevel = "low"
)

// Memory is the anchor record for a CRM entity's relationship history.
// One record per (workspace_id, entity_type, entity_id). CRM references are loose — no FK.
type Memory struct {
	ID             string
	WorkspaceID    string
	EntityType     EntityType
	EntityID       string
	Summary        string
	InferredIntent *string
	Tone           *ToneType
	Trajectory     *TrajectoryType
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// InteractionSignal records a single CRM interaction that contributes to relationship intelligence.
// SourceEntityType and SourceEntityID are loose back-references to the originating CRM record.
type InteractionSignal struct {
	ID                   string
	RelationshipMemoryID string
	SignalType           SignalType
	Sentiment            *SentimentType
	Summary              string
	SourceEntityType     *string
	SourceEntityID       *string
	OccurredAt           time.Time
	CreatedAt            time.Time
}

// StakeholderGraph is a directed influence edge between two CRM entities within a workspace.
// Both entity references are loose (no FK). Strength is in [0.0, 1.0].
type StakeholderGraph struct {
	ID             string
	WorkspaceID    string
	FromEntityType EntityType
	FromEntityID   string
	ToEntityType   EntityType
	ToEntityID     string
	InfluenceType  InfluenceType
	Strength       float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TrustScore holds the computed trust metrics for a relationship Memory.
// Enforces a strict 1:1 relationship with Memory via UNIQUE constraint.
// Score and DecayFactor are in [0.0, 1.0].
type TrustScore struct {
	ID                   string
	RelationshipMemoryID string
	Score                float64
	Confidence           ConfidenceLevel
	DecayFactor          float64
	LastScoredAt         time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
