package eval

import (
	"encoding/json"
	"time"
)

// ReplayMode identifies how an eval run was seeded.
type ReplayMode string

const (
	ReplayModeAdHoc     ReplayMode = "adhoc"
	ReplayModeBenchmark ReplayMode = "benchmark"
	ReplayModeReplay    ReplayMode = "replay"
)

// SyntheticOrg is a deterministic workspace fixture used to seed eval scenarios.
type SyntheticOrg struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspaceId"`
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	Version     int             `json:"version"`
	Seed        int64           `json:"seed"`
	FixtureData json.RawMessage `json:"fixtureData"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// BenchmarkCase is a versioned, queryable eval artifact anchored to a workspace.
type BenchmarkCase struct {
	ID              string          `json:"id"`
	WorkspaceID     string          `json:"workspaceId"`
	SyntheticOrgID  *string         `json:"syntheticOrgId,omitempty"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Domain          string          `json:"domain"`
	Version         int             `json:"version"`
	InputPayload    json.RawMessage `json:"inputPayload"`
	ExpectedOutcome json.RawMessage `json:"expectedOutcome"`
	Tags            []string        `json:"tags"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// ReplayProvenance captures the structured source of an eval run.
type ReplayProvenance struct {
	Mode                       ReplayMode `json:"mode"`
	BenchmarkCaseID            *string    `json:"benchmarkCaseId,omitempty"`
	SyntheticOrgID             *string    `json:"syntheticOrgId,omitempty"`
	SourceAgentRunID           *string    `json:"sourceAgentRunId,omitempty"`
	SourceCognitiveWorkspaceID *string    `json:"sourceCognitiveWorkspaceId,omitempty"`
	SourceTraceID              *string    `json:"sourceTraceId,omitempty"`
}
