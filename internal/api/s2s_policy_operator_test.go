package api

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

type b2b4PermissionStub struct {
	allowed bool
	calls int
	lastResource string
	lastAction string
	lastOperator string
	lastWorkspace string
}

func (s *b2b4PermissionStub) CheckActionPermission(
	_ context.Context, operatorID, resource, action string, attrs map[string]string,
) (bool, error) {
	s.calls++
	s.lastOperator = operatorID
	s.lastResource = resource
	s.lastAction = action
	s.lastWorkspace = attrs["workspace_id"]
	return s.allowed, nil
}

func b2b4InsertTestOperator(t *testing.T, db *sql.DB, id, workspace, status string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO user_account
        (id,workspace_id,email,display_name,status,created_at,updated_at)
        VALUES(?,?,?,?,?,'2026-01-01','2026-01-01')`,
		id,workspace,id+"@example.invalid",id,status); err != nil {
		t.Fatal(err)
	}
}

func TestB2B4FenixPolicyOperatorRBACAndScope(t *testing.T) {
	db := mustOpenAPITestDB(t)
	if _, err := db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES('op-w1','Workspace One','op-w1','2026-01-01','2026-01-01'),
              ('op-w2','Workspace Two','op-w2','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	b2b4InsertTestOperator(t,db,"operator-w1","op-w1","active")
	b2b4InsertTestOperator(t,db,"operator-w2","op-w2","active")
	b2b4InsertTestOperator(t,db,"operator-suspended","op-w1","suspended")
	allow := &b2b4PermissionStub{allowed:true}
	authorizer := NewS2SPolicyOperatorAuthorizer(db,allow)
	if err:=authorizer.AuthorizeS2SPolicyChange(
		context.Background(),"operator-w1","op-w1");err!=nil {
		t.Fatalf("allowed operator denied: %v",err)
	}
	if allow.calls!=1||allow.lastResource!="s2s_capabilities"||
		allow.lastAction!="manage"||allow.lastOperator!="operator-w1"||
		allow.lastWorkspace!="op-w1" {
		t.Fatalf("incorrect policy action requested: %+v",allow)
	}
	for _, tc := range []struct{name,actor,workspace string}{
		{"cross workspace","operator-w1","op-w2"},
		{"inactive operator","operator-suspended","op-w1"},
		{"missing account","unknown-operator","op-w1"},
	} {
		t.Run(tc.name,func(t *testing.T){
			before:=allow.calls
			err:=authorizer.AuthorizeS2SPolicyChange(
				context.Background(),tc.actor,tc.workspace)
			if !errors.Is(err,integration.ErrCapabilityConfiguration)||allow.calls!=before {
				t.Fatalf("untrusted operator reached policy grant: err=%v calls=%d",err,allow.calls)
			}
		})
	}
	allow.allowed=false
	if err:=authorizer.AuthorizeS2SPolicyChange(context.Background(),
		"operator-w1","op-w1");!errors.Is(err,integration.ErrCapabilityConfiguration) {
		t.Fatalf("policy denial not enforced: %v",err)
	}
	for _, unsafe := range []*S2SPolicyOperatorAuthorizer{
		nil,
		NewS2SPolicyOperatorAuthorizer(nil,allow),
		NewS2SPolicyOperatorAuthorizer(db,nil),
	} {
		if err:=unsafe.AuthorizeS2SPolicyChange(context.Background(),
			"operator-w1","op-w1");!errors.Is(err,integration.ErrCapabilityConfiguration) {
			t.Fatalf("missing operator policy component granted access: %v",err)
		}
	}
}

func TestB2B4ReadOnlyStoreCannotManagePolicy(t *testing.T) {
	db := mustOpenAPITestDB(t)
	readOnly:=integration.NewS2SCapabilityPolicyStore(db)
	now:=time.Now().UTC()
	reg:=integration.S2SCapabilityPolicyRegistration{
		Binding:integration.S2SCapabilityBinding{
			PrincipalID:"unknown",WorkspaceID:"unknown",CapabilityKey:"integration.diagnostics",
			InputSchemaVersion:"1",HandlerID:"test-only",
		},
		SchemaJSON:[]byte(`{"type":"object","properties":{"v":{"type":"integer"}},"additionalProperties":false}`),
		OperatorID:"operator-w1",
	}
	if err:=readOnly.Provision(context.Background(),reg,now);
		!errors.Is(err,integration.ErrCapabilityConfiguration) {
		t.Fatalf("read-only policy store provisioned capability: %v",err)
	}
	if err:=readOnly.SetEnabled(context.Background(),
		"unknown","unknown","integration.diagnostics","1",true,"operator-w1",now);
		!errors.Is(err,integration.ErrCapabilityConfiguration) {
		t.Fatalf("read-only policy store enabled capability: %v",err)
	}
}
