package flowinterop

import (
	"errors"
	"strings"
)

var (
	ErrContractVersion = errors.New("unsupported flow interoperability contract version")
	ErrOperation       = errors.New("unsupported flow interoperability operation")
	ErrArtifact        = errors.New("invalid flow artifact")
	ErrArtifactFormat     = errors.New("artifact format is not supported for operation")
	ErrComparisonArtifact = errors.New("comparison artifact is required for operation")
)

// Artifact carries one representation across the Fenix/Mermaid2SF semantic boundary.
// FlowIR remains opaque to Fenix and is never re-modeled here.
type Artifact struct {
	Format  ArtifactFormat `json:"format"`
	Name    string         `json:"name,omitempty"`
	Content string         `json:"content"`
}

// Request is the transport-neutral input contract for one Mermaid2SF semantic operation.
type Request struct {
	ContractVersion string    `json:"contract_version"`
	Operation       Operation `json:"operation"`
	Input           Artifact  `json:"input"`
	CompareTo       *Artifact `json:"compare_to,omitempty"`
}

// ResultStatus describes the semantic outcome independently of transport status codes.
type ResultStatus string

const (
	ResultSucceeded   ResultStatus = "succeeded"
	ResultRejected    ResultStatus = "rejected"
	ResultUnsupported ResultStatus = "unsupported"
)

// SemanticMetadata contains stable facts Fenix may reason about without owning FlowIR.
type SemanticMetadata struct {
	FlowFamily  FlowFamily `json:"flow_family,omitempty"`
	FlowAPIName string     `json:"flow_api_name,omitempty"`
	APIVersion  string     `json:"api_version,omitempty"`
}

// Result is the transport-neutral response contract.
// Typed diagnostics are intentionally deferred to W2-T5.
type Result struct {
	ContractVersion   string           `json:"contract_version"`
	Operation         Operation        `json:"operation"`
	Status            ResultStatus     `json:"status"`
	Artifacts         []Artifact       `json:"artifacts,omitempty"`
	Fidelity          FidelityReport   `json:"fidelity"`
	SemanticMetadata  SemanticMetadata `json:"semantic_metadata,omitempty"`
	Diff              *SemanticDiff    `json:"diff,omitempty"`
	Diagnostics       []Diagnostic     `json:"diagnostics,omitempty"`
	ProviderReference string           `json:"provider_reference,omitempty"`
}

// Validate checks the semantic request contract before any provider/transport adapter runs.
func (r Request) Validate() error {
	if r.ContractVersion != ContractVersion {
		return ErrContractVersion
	}
	spec, ok := Lookup(r.Operation)
	if !ok {
		return ErrOperation
	}
	if strings.TrimSpace(r.Input.Content) == "" {
		return ErrArtifact
	}
	if !containsFormat(spec.InputFormats, r.Input.Format) {
		return ErrArtifactFormat
	}
	return r.validateComparison(spec)
}

func (r Request) validateComparison(spec CapabilitySpec) error {
	if r.Operation != OperationCompare {
		if r.CompareTo != nil {
			return ErrComparisonArtifact
		}
		return nil
	}
	if r.CompareTo == nil || strings.TrimSpace(r.CompareTo.Content) == "" {
		return ErrComparisonArtifact
	}
	if !containsFormat(spec.InputFormats, r.CompareTo.Format) {
		return ErrArtifactFormat
	}
	return nil
}

func containsFormat(formats []ArtifactFormat, target ArtifactFormat) bool {
	for _, format := range formats {
		if format == target {
			return true
		}
	}
	return false
}
