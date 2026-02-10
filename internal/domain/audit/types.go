package audit

import (
	"encoding/json"
	"time"
)

// ActorType represents the type of actor performing an action
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeAgent  ActorType = "agent"
	ActorTypeSystem ActorType = "system"
)

// Outcome represents the result of an audited action
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeDenied  Outcome = "denied"
	OutcomeError   Outcome = "error"
)

// AuditEvent represents a single audit log entry
// This is immutable - once created, it should never be modified
type AuditEvent struct {
	ID                 string          `json:"id"`
	WorkspaceID        string          `json:"workspace_id"`
	ActorID            string          `json:"actor_id"`
	ActorType          ActorType       `json:"actor_type"`
	Action             string          `json:"action"`
	EntityType         *string         `json:"entity_type,omitempty"`
	EntityID           *string         `json:"entity_id,omitempty"`
	Details            json.RawMessage `json:"details,omitempty"`
	PermissionsChecked json.RawMessage `json:"permissions_checked,omitempty"`
	Outcome            Outcome         `json:"outcome"`
	TraceID            *string         `json:"trace_id,omitempty"`
	IPAddress          *string         `json:"ip_address,omitempty"`
	UserAgent          *string         `json:"user_agent,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// EventDetails captures the specifics of an audited action
type EventDetails struct {
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
	Changes  []Change    `json:"changes,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// Change represents a single field change
type Change struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// PermissionCheck represents a permission verification
type PermissionCheck struct {
	Permission string `json:"permission"`
	Granted    bool   `json:"granted"`
	Reason     string `json:"reason,omitempty"`
}
