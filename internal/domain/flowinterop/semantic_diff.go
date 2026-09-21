package flowinterop

import (
	"encoding/json"
	"errors"
)

var ErrSemanticDiff = errors.New("invalid semantic diff")

// SemanticChangeKind classifies one semantic difference between normalized Flows.
type SemanticChangeKind string

const (
	SemanticChangeAdded   SemanticChangeKind = "added"
	SemanticChangeRemoved SemanticChangeKind = "removed"
	SemanticChangeChanged SemanticChangeKind = "changed"
)

// SemanticChange identifies one stable semantic difference without exposing FlowIR internals.
type SemanticChange struct {
	Path   string             `json:"path"`
	Kind   SemanticChangeKind `json:"kind"`
	Before json.RawMessage    `json:"before,omitempty"`
	After  json.RawMessage    `json:"after,omitempty"`
}

// SemanticDiff is the provider-neutral semantic comparison result.
type SemanticDiff struct {
	Equal   bool             `json:"equal"`
	Changes []SemanticChange `json:"changes,omitempty"`
}

// Validate ensures compare results do not contradict themselves.
func (d SemanticDiff) Validate() error {
	if d.Equal && len(d.Changes) > 0 {
		return ErrSemanticDiff
	}
	if !d.Equal && len(d.Changes) == 0 {
		return ErrSemanticDiff
	}
	for _, change := range d.Changes {
		if change.Path == "" || !isSemanticChangeKind(change.Kind) {
			return ErrSemanticDiff
		}
	}
	return nil
}

func isSemanticChangeKind(kind SemanticChangeKind) bool {
	return kind == SemanticChangeAdded ||
		kind == SemanticChangeRemoved ||
		kind == SemanticChangeChanged
}
