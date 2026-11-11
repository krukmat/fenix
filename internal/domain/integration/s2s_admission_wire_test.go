package integration_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

func TestB4ProposedAdmissionWireAcknowledgement(t *testing.T) {
	receipt := integration.S2SAdmissionReceipt{
		OperationID: "operation_00000001",
		TraceID:     "trace_b4_000001",
	}
	for _, tc := range []struct {
		replayed bool
		status   int
	}{
		{false, 202},
		{true, 200},
	} {
		receipt.Replayed = tc.replayed
		status, ack, err := integration.ProjectS2SAdmissionWireAck(receipt)
		if err != nil || status != tc.status {
			t.Fatalf("invalid admission response: status=%d ack=%+v err=%v", status, ack, err)
		}
		buf, err := json.Marshal(ack)
		if err != nil {
			t.Fatal(err)
		}
		var obj map[string]any
		if err := json.Unmarshal(buf, &obj); err != nil {
			t.Fatal(err)
		}
		if len(obj) != 4 || obj["contract_version"] != "1" ||
			obj["operation_id"] != receipt.OperationID ||
			obj["trace_id"] != receipt.TraceID || obj["replayed"] != tc.replayed {
			t.Fatalf("wire ack contract violated: %s", buf)
		}
	}
	for _, malformed := range []integration.S2SAdmissionReceipt{
		{OperationID: "", TraceID: "trace_b4_000001"},
		{OperationID: "operation_00000001", TraceID: "bad"},
		{OperationID: "invalid operation with spaces", TraceID: "trace_b4_000001"},
	} {
		status, ack, err := integration.ProjectS2SAdmissionWireAck(malformed)
		if status != 0 || ack.OperationID != "" || !errors.Is(err, integration.ErrS2SAdmissionInvalid) {
			t.Fatalf("forged acknowledgement accepted: status=%d ack=%+v err=%v", status, ack, err)
		}
	}
}

func TestB4ProposedAdmissionWireErrorsRedactInternals(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{integration.ErrS2SAdmissionInvalid, 400, "INVALID_REQUEST"},
		{integration.ErrS2SAdmissionDenied, 403, "REQUEST_DENIED"},
		{integration.ErrS2SAdmissionConflict, 409, "OPERATION_CONFLICT"},
		{integration.ErrS2SAdmissionUnavailable, 503, "PROVIDER_UNAVAILABLE"},
		{errors.New("internal SQL principal_id=secret"), 503, "PROVIDER_UNAVAILABLE"},
	} {
		status, projected := integration.ProjectS2SAdmissionWireFailure(
			fmt.Errorf("secret-bearing error: %w", tc.err),
		)
		if status != tc.status || projected.ErrorCode != tc.code {
			t.Fatalf("unexpected failure contract: status=%d projected=%+v", status, projected)
		}
		payload, err := json.Marshal(projected)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatal(err)
		}
		if len(fields) != 3 || fields["contract_version"] != "1" ||
			fields["error_code"] != tc.code ||
			fields["message"] != "Request could not be accepted" {
			t.Fatalf("unexpected error wire fields: %s", payload)
		}
		if strings.Contains(string(payload), "principal") || strings.Contains(string(payload), "secret") ||
			strings.Contains(string(payload), "SQL") {
			t.Fatalf("error leaks internal failure: %s", payload)
		}
	}
}
