package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

type b2b4TestOperatorAuthorization struct{}

func (b2b4TestOperatorAuthorization) AuthorizeS2SPolicyChange(
	_ context.Context, actorID, workspaceID string,
) error {
	if workspaceID != "b2c-w1" ||
		(actorID != "b2b4-operator" && actorID != "operator") {
		return integration.ErrCapabilityConfiguration
	}
	return nil
}

func b2b4AuthorizedPolicyStore(db *sql.DB) *integration.S2SCapabilityPolicyStore {
	return integration.NewS2SCapabilityPolicyStoreWithOperator(
		db, b2b4TestOperatorAuthorization{},
	)
}

const b2b4SchemaV1 = `{
  "type":"object",
  "properties":{
    "v":{"type":"integer"},
    "message":{"type":"string","minLength":2,"maxLength":12},
    "tags":{"type":"array","maxItems":2,"items":{"type":"string","maxLength":8}}
  },
  "required":["v"],
  "additionalProperties":false
}`

func b2b4Registration(version, schema string) integration.S2SCapabilityPolicyRegistration {
	return integration.S2SCapabilityPolicyRegistration{
		Binding: integration.S2SCapabilityBinding{
			PrincipalID: "b2e-principal",
			WorkspaceID: "b2c-w1",
			CapabilityKey: "integration.diagnostics",
			InputSchemaVersion: version,
			HandlerID: "test-diagnostic-only",
		},
		SchemaJSON: []byte(schema),
		OperatorID: "b2b4-operator",
	}
}

func TestB2B4PersistedPolicyStartsDisabledAndValidatesSchema(t *testing.T) {
	f := newB2EFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	policies := b2b4AuthorizedPolicyStore(f.db)
	registry := integration.NewS2SCapabilityRegistryFromPolicies(policies)
	validator := integration.NewPersistedS2SInputValidator(policies)

	if _, err := registry.Resolve("b2e-principal","b2c-w1","integration.diagnostics","1"); !errors.Is(err, integration.ErrCapabilityUnavailable) {
		t.Fatalf("unconfigured policy allowed: %v", err)
	}
	if err := policies.Provision(ctx, b2b4Registration("1", b2b4SchemaV1), now); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve("b2e-principal","b2c-w1","integration.diagnostics","1"); !errors.Is(err, integration.ErrCapabilityUnavailable) {
		t.Fatalf("provision silently enabled capability: %v", err)
	}
	if err := registry.Register(b2b4Registration("1",b2b4SchemaV1).Binding); !errors.Is(err,integration.ErrCapabilityConfiguration) {
		t.Fatalf("managed registry bypassed policy store: %v", err)
	}
	if err := policies.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","1",
		true,"b2b4-operator",now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	binding, err := registry.ResolveContext(ctx,"b2e-principal","b2c-w1","integration.diagnostics","1")
	if err != nil || binding.SchemaSHA256 == [32]byte{} {
		t.Fatalf("persisted schema hash missing: %+v %v", binding, err)
	}
	for _, tc := range []struct {
		name string
		input map[string]any
		valid bool
	}{
		{"valid integer",map[string]any{"v":json.Number("1")},true},
		{"valid optional",map[string]any{"v":json.Number("2"),"message":"hola","tags":[]any{"one","two"}},true},
		{"no required field",map[string]any{"message":"hola"},false},
		{"wrong numeric type",map[string]any{"v":"1"},false},
		{"decimal is not integer",map[string]any{"v":json.Number("1.0")},false},
		{"unrecognized field",map[string]any{"v":json.Number("1"),"secret":"forbidden"},false},
		{"wrong string minimum",map[string]any{"v":json.Number("1"),"message":"a"},false},
		{"wrong array item type",map[string]any{"v":json.Number("1"),"tags":[]any{json.Number("3")}},false},
		{"too many array items",map[string]any{"v":json.Number("1"),"tags":[]any{"one","two","three"}},false},
	} {
		t.Run(tc.name,func(t *testing.T){
			err := validator.ValidateS2SInput(ctx,binding,tc.input)
			if (err==nil)!=tc.valid {
				t.Fatalf("schema verdict incorrect: %v",err)
			}
		})
	}
}

func TestB2B4OperatorPolicyRejectsUnknownOrPermissiveSchemas(t *testing.T) {
	f := newB2EFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	store := b2b4AuthorizedPolicyStore(f.db)
	for _, raw := range []string{
		`{}`,
		`{"type":"object","properties":{"v":{"type":"string"}},"additionalProperties":false}`,
		`{"type":"object","properties":{"v":{"type":"integer"}},"additionalProperties":true}`,
		`{"type":"object","properties":{"v":{"type":"integer"}},"additionalProperties":false,"unexpected":true}`,
		`{"type":"object","properties":{"v":{"type":"integer"}},"required":["missing"],"additionalProperties":false}`,
		`{"type":"object","properties":{"v":{"type":"integer"},"v":{"type":"boolean"}},"additionalProperties":false}`,
		`{"type":"object","properties":{"v":{"type":"integer"}},"additionalProperties":false,"additionalProperties":false}`,
		`{"type":"object","properties":{"v":{"type":"integer"}},"additionalProperties":false,"items":{"type":"string"}}`,
		`{"type":"object","properties":{"v":{"type":"string","maxLength":20,"pattern":".*"}},"additionalProperties":false}`,
		`{"type":"object","properties":{"v":{"type":"string","maxLength":1,"minLength":2}},"additionalProperties":false}`,
		`{"type":"object","properties":{"v":{"type":"integer"}},"additionalProperties":false,"required":["v","v"]}`,
	} {
		err:=store.Provision(ctx,b2b4Registration("1",raw),now)
		if !errors.Is(err,integration.ErrCapabilityConfiguration) {
			t.Fatalf("unsafe operator schema accepted: %s err=%v",raw,err)
		}
	}
	if err:=store.Provision(ctx,b2b4Registration("1",b2b4SchemaV1),now);err!=nil {
		t.Fatal(err)
	}
	if err:=store.Provision(ctx,b2b4Registration("1",b2b4SchemaV1),now);err==nil {
		t.Fatal("duplicate schema version overwrote immutable policy")
	}
	if _,err:=f.db.Exec(`UPDATE service_s2s_capability_policy SET schema_json='{}'`);err==nil {
		t.Fatal("out-of-band schema mutation bypassed DB immutable identity")
	}
}

func TestB2B4VersionSwitchAndRevocation(t *testing.T) {
	f:=newB2EFixture(t)
	ctx:=context.Background()
	now:=time.Now().UTC()
	store:=b2b4AuthorizedPolicyStore(f.db)
	reg:=integration.NewS2SCapabilityRegistryFromPolicies(store)
	for _,v:=range []string{"1","2"} {
		if err:=store.Provision(ctx,b2b4Registration(v,b2b4SchemaV1),now);err!=nil {t.Fatal(err)}
	}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","1",true,"operator",now);err!=nil {
		t.Fatal(err)
	}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","2",true,"operator",now);err==nil {
		t.Fatal("enabled two versions concurrently")
	}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","1",false,"operator",now.Add(time.Second));err!=nil {
		t.Fatal(err)
	}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","2",true,"operator",now.Add(2*time.Second));err!=nil {
		t.Fatal(err)
	}
	if _,err:=reg.ResolveContext(ctx,"b2e-principal","b2c-w1","integration.diagnostics","1");!errors.Is(err,integration.ErrCapabilityUnavailable) {
		t.Fatalf("old version remains enabled: %v",err)
	}
	if _,err:=reg.ResolveContext(ctx,"b2e-principal","b2c-w1","integration.diagnostics","2");err!=nil {
		t.Fatalf("new version unavailable: %v",err)
	}
	if err:=f.creds.RevokeServicePrincipal(ctx,"b2e-principal",now.Add(3*time.Second));err!=nil {
		t.Fatal(err)
	}
	if _,err:=reg.ResolveContext(ctx,"b2e-principal","b2c-w1","integration.diagnostics","2");!errors.Is(err,integration.ErrCapabilityUnavailable) {
		t.Fatalf("revoked principal retains enabled capability: %v",err)
	}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","2",false,"operator",now.Add(4*time.Second));err!=nil {
		t.Fatalf("cannot disable revoked principal's policy: %v",err)
	}
	var actions string
	rows,err:=f.db.Query("SELECT action FROM service_s2s_capability_audit ORDER BY id")
	if err!=nil {t.Fatal(err)}
	defer rows.Close()
	for rows.Next(){
		var action string
		if err:=rows.Scan(&action);err!=nil {t.Fatal(err)}
		actions += action+","
	}
	if err:=rows.Err();err!=nil {t.Fatal(err)}
	if !strings.Contains(actions,"provision,")||!strings.Contains(actions,"disable,")||!strings.Contains(actions,"enable,") {
		t.Fatalf("policy transitions not audited: %s",actions)
	}
}

type b2b4DisablingValidator struct {
	delegate *integration.PersistedS2SInputValidator
	policies *integration.S2SCapabilityPolicyStore
	disabled bool
}

func (v *b2b4DisablingValidator) ValidateS2SInput(
	ctx context.Context,b integration.S2SCapabilityBinding,input map[string]any,
) error {
	if err:=v.delegate.ValidateS2SInput(ctx,b,input);err!=nil {return err}
	if !v.disabled {
		if err:=v.policies.SetEnabled(ctx,b.PrincipalID,b.WorkspaceID,
			b.CapabilityKey,b.InputSchemaVersion,false,"operator",time.Now().UTC());err!=nil {
			return err
		}
		v.disabled=true
	}
	return nil
}

func TestB2B4PolicyDisableBetweenValidateAndReserveDenies(t *testing.T) {
	f:=newB2EFixture(t)
	ctx:=context.Background()
	now:=time.Now().UTC()
	store:=b2b4AuthorizedPolicyStore(f.db)
	if err:=store.Provision(ctx,b2b4Registration("1",b2b4SchemaV1),now);err!=nil {t.Fatal(err)}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1",
		"integration.diagnostics","1",true,"operator",now.Add(time.Second));err!=nil {t.Fatal(err)}
	reg:=integration.NewS2SCapabilityRegistryFromPolicies(store)
	validator:=&b2b4DisablingValidator{
		delegate:integration.NewPersistedS2SInputValidator(store),policies:store,
	}
	admit:=&integration.S2SAdmission{
		Credentials:f.creds,Registry:reg,InputSchema:validator,Operations:f.store,
	}
	receipt,err:=admit.Admit(ctx,f.token,[]byte(b2eEnvelope()),now.Add(2*time.Second))
	if !errors.Is(err,integration.ErrS2SAdmissionDenied)||receipt.OperationID!=""||b2eCount(t,f)!=0 {
		t.Fatalf("disabled policy raced into reservation: receipt=%+v err=%v",receipt,err)
	}
}

func TestB2B4PersistedAdmissionCreatesOneInertRecord(t *testing.T) {
	f:=newB2EFixture(t)
	ctx:=context.Background()
	now:=time.Now().UTC()
	store:=b2b4AuthorizedPolicyStore(f.db)
	if err:=store.Provision(ctx,b2b4Registration("1",b2b4SchemaV1),now);err!=nil {t.Fatal(err)}
	reg:=integration.NewS2SCapabilityRegistryFromPolicies(store)
	admit:=&integration.S2SAdmission{
		Credentials:f.creds,Registry:reg,
		InputSchema:integration.NewPersistedS2SInputValidator(store),Operations:f.store,
	}
	if _,err:=admit.Admit(ctx,f.token,[]byte(b2eEnvelope()),now);!errors.Is(err,integration.ErrS2SAdmissionDenied) {
		t.Fatalf("disabled capability accepted: %v",err)
	}
	if err:=store.SetEnabled(ctx,"b2e-principal","b2c-w1","integration.diagnostics","1",
		true,"operator",now.Add(time.Second));err!=nil {t.Fatal(err)}
	first,err:=admit.Admit(ctx,f.token,[]byte(b2eEnvelope()),now.Add(2*time.Second))
	if err!=nil||first.Replayed||first.OperationID!="operation_00000001" {
		t.Fatalf("enabled policy failed to reserve: %+v err=%v",first,err)
	}
	replay,err:=admit.Admit(ctx,f.token,[]byte(b2eEnvelope()),now.Add(3*time.Second))
	if err!=nil||!replay.Replayed||b2eCount(t,f)!=1 {
		t.Fatalf("durable replay mismatch: %+v err=%v",replay,err)
	}
	var agentRuns int
	if err:=f.db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&agentRuns);err!=nil||agentRuns!=0 {
		t.Fatalf("admission executed agent: runs=%d err=%v",agentRuns,err)
	}
}

func TestB2B4ManagedRegistryNilStoreCannotFallbackToMemory(t *testing.T) {
	reg := integration.NewS2SCapabilityRegistryFromPolicies(nil)
	binding := b2b4Registration("1", b2b4SchemaV1).Binding
	binding.Enabled = true
	if err := reg.Register(binding); !errors.Is(err, integration.ErrCapabilityConfiguration) {
		t.Fatalf("nil managed store accepted in-memory policy registration: %v", err)
	}
	if _, err := reg.Resolve(binding.PrincipalID, binding.WorkspaceID,
		binding.CapabilityKey, binding.InputSchemaVersion); !errors.Is(err, integration.ErrCapabilityUnavailable) {
		t.Fatalf("nil managed store resolved capability: %v", err)
	}
	if err := reg.Disable(binding.PrincipalID, binding.WorkspaceID, binding.CapabilityKey);
		!errors.Is(err, integration.ErrCapabilityUnavailable) {
		t.Fatalf("managed registry incorrectly accepted direct disable: %v", err)
	}
}
