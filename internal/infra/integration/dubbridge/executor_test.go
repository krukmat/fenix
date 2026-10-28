package dubbridge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

const sampleOrg = "550e8400-e29b-41d4-a716-446655440000"
const sampleProject = "550e8400-e29b-41d4-a716-446655440001"

func TestExecutorRejectsUnauthorizedAndMalformedWithoutNetwork(t *testing.T) {
	e, err := NewExecutor(Config{BaseURL: "http://localhost:7000", BearerToken: "not-real", AllowedOrgID: sampleOrg, AllowedProjectID: sampleProject})
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, "t-local")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.ExecutionID, "e-local")
	for _, body := range []string{
		`{}`, `{"contract_version":"2","org_id":"` + sampleOrg + `","project_id":"` + sampleProject + `"}`,
		`{"contract_version":"1","org_id":"bad","project_id":"` + sampleProject + `"}`,
		`{"contract_version":"1","org_id":"` + sampleOrg + `","project_id":"` + sampleProject + `","url":"https://evil.invalid"}`,
		`{"contract_version":"1","org_id":"` + sampleOrg + `","project_id":"` + sampleProject + `"}{};`,
	} {
		_, err := e.Execute(ctx, []byte(body))
		if !errors.Is(err, ErrConfiguration) {
			t.Errorf("wanted config failure for input %q got %v", body, err)
		}
	}
	unauthorized := `{"contract_version":"1","org_id":"` + sampleOrg + `","project_id":"550e8400-e29b-41d4-a716-446655440099"}`
	_, err = e.Execute(ctx, []byte(unauthorized))
	if !errors.Is(err, ErrUnauthorizedTarget) {
		t.Fatalf("missing target rejection: %v", err)
	}
	ctx = context.Background()
	approved := `{"contract_version":"1","org_id":"` + sampleOrg + `","project_id":"` + sampleProject + `"}`
	_, err = e.Execute(ctx, []byte(approved))
	if !errors.Is(err, ErrConfiguration) {
		t.Fatalf("missing trace not rejected: %v", err)
	}
}
func TestInputSchemaExists(t *testing.T) {
	if len(InputSchema) == 0 || !json.Valid(InputSchema) {
		t.Fatal("schema missing or invalid")
	}
	var shape map[string]any
	if err := json.Unmarshal(InputSchema, &shape); err != nil {
		t.Fatal(err)
	}
	if shape["additionalProperties"] != false {
		t.Fatal("schema allows arbitrary input")
	}
	fields, ok := shape["required"].([]any)
	if !ok || len(fields) != 3 {
		t.Fatalf("required fields unexpected: %#v", shape["required"])
	}
}
