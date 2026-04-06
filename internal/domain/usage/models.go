package usage

import "time"

type Event struct {
	ID            string
	WorkspaceID   string
	ActorID       string
	ActorType     string
	RunID         *string
	ToolName      *string
	ModelName     *string
	InputUnits    int64
	OutputUnits   int64
	EstimatedCost float64
	LatencyMs     *int64
	CreatedAt     time.Time
}

type Policy struct {
	ID              string
	WorkspaceID     string
	PolicyType      string
	ScopeType       string
	ScopeID         *string
	MetricName      string
	LimitValue      float64
	ResetPeriod     string
	EnforcementMode string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type State struct {
	ID            string
	WorkspaceID   string
	QuotaPolicyID string
	CurrentValue  float64
	PeriodStart   time.Time
	PeriodEnd     time.Time
	LastEventAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RecordEventInput struct {
	WorkspaceID   string
	ActorID       string
	ActorType     string
	RunID         *string
	ToolName      *string
	ModelName     *string
	InputUnits    int64
	OutputUnits   int64
	EstimatedCost float64
	LatencyMs     *int64
}

type CreatePolicyInput struct {
	WorkspaceID     string
	PolicyType      string
	ScopeType       string
	ScopeID         *string
	MetricName      string
	LimitValue      float64
	ResetPeriod     string
	EnforcementMode string
	IsActive        bool
}

type UpsertStateInput struct {
	WorkspaceID   string
	QuotaPolicyID string
	CurrentValue  float64
	PeriodStart   time.Time
	PeriodEnd     time.Time
	LastEventAt   *time.Time
}
