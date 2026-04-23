package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const semanticIDHashLength = 16

type SemanticIDInput struct {
	Kind    SemanticNodeKind
	Source  SemanticSourceKind
	Scope   string
	Ordinal int
	Parts   []string
}

func NewSemanticNodeID(input SemanticIDInput) SemanticNodeID {
	normalized := normalizeSemanticIDInput(input)
	sum := sha256.Sum256([]byte(normalized))
	return SemanticNodeID(fmt.Sprintf("%s:%s", input.Kind, hex.EncodeToString(sum[:])[:semanticIDHashLength]))
}

func normalizeSemanticIDInput(input SemanticIDInput) string {
	parts := make([]string, 0, len(input.Parts)+4)
	parts = append(parts,
		normalizeSemanticIDPart(string(input.Kind)),
		normalizeSemanticIDPart(string(input.Source)),
		normalizeSemanticIDPart(input.Scope),
		fmt.Sprintf("%d", input.Ordinal),
	)
	for _, part := range input.Parts {
		parts = append(parts, normalizeSemanticIDPart(part))
	}
	return strings.Join(parts, "\x00")
}

func normalizeSemanticIDPart(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
