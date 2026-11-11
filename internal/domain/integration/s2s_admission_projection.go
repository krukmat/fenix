package integration

import "errors"

// S2SAdmissionProblem is a bounded, transport-neutral error classification.
// A future authenticated HTTP adapter must make a separate status-code
// decision. No internal error, principal ID, SQL text or digest is exposed.
type S2SAdmissionProblem struct {
	Code    string
	Message string
}

// ProjectS2SAdmissionError is safe for a future allowlisted rejection body,
// but is not itself an HTTP handler or an indication that a route is active.
// Unknown errors fail closed to an opaque unavailable response.
func ProjectS2SAdmissionError(err error) S2SAdmissionProblem {
	switch {
	case errors.Is(err, ErrS2SAdmissionInvalid):
		return S2SAdmissionProblem{Code: "INVALID_REQUEST", Message: "Request could not be accepted"}
	case errors.Is(err, ErrS2SAdmissionConflict):
		return S2SAdmissionProblem{Code: "OPERATION_CONFLICT", Message: "Request could not be accepted"}
	case errors.Is(err, ErrS2SAdmissionDenied):
		return S2SAdmissionProblem{Code: "REQUEST_DENIED", Message: "Request could not be accepted"}
	default:
		return S2SAdmissionProblem{Code: "PROVIDER_UNAVAILABLE", Message: "Request could not be accepted"}
	}
}
