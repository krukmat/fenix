package integration

// This file defines an inert, transport-neutral B4 admission wire projection.
// The Fenix S2S route is NOT mounted, and no credential, capability or agent
// execution is created. HTTP status integers are proposed for future review.

const s2sAdmissionWireVersion = "1"

type S2SAdmissionWireAck struct {
	ContractVersion string `json:"contract_version"`
	OperationID     string `json:"operation_id"`
	TraceID         string `json:"trace_id"`
	Replayed        bool   `json:"replayed"`
}

type S2SAdmissionWireFailure struct {
	ContractVersion string `json:"contract_version"`
	ErrorCode       string `json:"error_code"`
	Message         string `json:"message"`
}

// ProjectS2SAdmissionWireAck exposes neither the authenticated principal,
// capability registry nor request/config hashes, run IDs or internal state.
// Return errors rather than manufacture a valid-looking acknowledgement.
func ProjectS2SAdmissionWireAck(
	receipt S2SAdmissionReceipt,
) (int, S2SAdmissionWireAck, error) {
	if !validS2SOperationID(receipt.OperationID) || !validS2STraceID(receipt.TraceID) {
		return 0, S2SAdmissionWireAck{}, ErrS2SAdmissionInvalid
	}
	status := 202
	if receipt.Replayed {
		status = 200
	}
	return status, S2SAdmissionWireAck{
		ContractVersion: s2sAdmissionWireVersion,
		OperationID:     receipt.OperationID,
		TraceID:         receipt.TraceID,
		Replayed:        receipt.Replayed,
	}, nil
}

// ProjectS2SAdmissionWireFailure returns bounded, uniform public text and a
// static code. It has no access to raw credentials, tenant IDs or SQL details.
// HTTP 401/403/404 differentiation is intentionally not attempted here.
func ProjectS2SAdmissionWireFailure(err error) (int, S2SAdmissionWireFailure) {
	projected := ProjectS2SAdmissionError(err)
	status := 503
	switch projected.Code {
	case "INVALID_REQUEST":
		status = 400
	case "REQUEST_DENIED":
		status = 403
	case "OPERATION_CONFLICT":
		status = 409
	}
	return status, S2SAdmissionWireFailure{
		ContractVersion: s2sAdmissionWireVersion,
		ErrorCode:       projected.Code,
		Message:         projected.Message,
	}
}
