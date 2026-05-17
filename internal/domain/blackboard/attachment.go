// Package blackboard — Attachment bundles the three blackboard components (Task A.5, ADR-100).
// Injected into RunContext when a cognitive workspace is attached to an agent run.
package blackboard

// Attachment groups the shared cognitive workspace components for a single agent run.
// All three components are scoped to the same CognitiveWorkspaceID.
type Attachment struct {
	CognitiveWorkspaceID string
	Bus                  WorkspaceBus
	Memory               MemoryStore
	Timeline             ReasoningTimeline
}
