package flowinterop

import "errors"

var ErrDiagnostic = errors.New("invalid flow diagnostic")

// DiagnosticSeverity describes the impact of one normalized provider diagnostic.
type DiagnosticSeverity string

const (
	DiagnosticInfo    DiagnosticSeverity = "info"
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticError   DiagnosticSeverity = "error"
)

// DiagnosticStage identifies where the semantic pipeline produced a diagnostic.
type DiagnosticStage string

const (
	DiagnosticStageParse     DiagnosticStage = "parse"
	DiagnosticStageNormalize DiagnosticStage = "normalize"
	DiagnosticStageValidate  DiagnosticStage = "validate"
	DiagnosticStageGenerate  DiagnosticStage = "generate"
	DiagnosticStageCompare   DiagnosticStage = "compare"
	DiagnosticStageFidelity  DiagnosticStage = "fidelity"
	DiagnosticStageProvider  DiagnosticStage = "provider"
)

// Diagnostic preserves stable Mermaid2SF codes such as M2SF-SF-* while
// normalizing their shape for Fenix agents and audit consumers.
type Diagnostic struct {
	Code        string             `json:"code"`
	Severity    DiagnosticSeverity `json:"severity"`
	Stage       DiagnosticStage    `json:"stage"`
	Message     string             `json:"message"`
	ElementID   string             `json:"element_id,omitempty"`
	Feature     string             `json:"feature,omitempty"`
	Recoverable bool               `json:"recoverable"`
}

// Validate checks diagnostic structure without interpreting provider semantics.
func (d Diagnostic) Validate() error {
	if d.Code == "" || d.Message == "" {
		return ErrDiagnostic
	}
	if !isDiagnosticSeverity(d.Severity) || !isDiagnosticStage(d.Stage) {
		return ErrDiagnostic
	}
	return nil
}

func isDiagnosticSeverity(severity DiagnosticSeverity) bool {
	return severity == DiagnosticInfo ||
		severity == DiagnosticWarning ||
		severity == DiagnosticError
}

var diagnosticStages = map[DiagnosticStage]struct{}{
	DiagnosticStageParse:     {},
	DiagnosticStageNormalize: {},
	DiagnosticStageValidate:  {},
	DiagnosticStageGenerate:  {},
	DiagnosticStageCompare:   {},
	DiagnosticStageFidelity:  {},
	DiagnosticStageProvider:  {},
}

func isDiagnosticStage(stage DiagnosticStage) bool {
	_, ok := diagnosticStages[stage]
	return ok
}
