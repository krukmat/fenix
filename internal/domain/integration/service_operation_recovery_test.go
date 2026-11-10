package integration_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

func TestB2CLeaseIsSingleClaimAndCrashBecomesIndeterminate(t *testing.T) {
	f := newB2CFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000010")
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	var before, after int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&before); err != nil {
		t.Fatal(err)
	}
	claimed, err := f.store.AcquireLease(ctx, req.Scope, req.OperationID,
		"worker1", f.registry, now, time.Minute)
	if err != nil || !claimed {
		t.Fatalf("expected one claim: %v %v", claimed, err)
	}
	claimed, err = f.store.AcquireLease(ctx, req.Scope, req.OperationID,
		"worker2", f.registry, now.Add(2*time.Minute), time.Minute)
	if err != nil || claimed {
		t.Fatalf("expired lease was reacquired: %v %v", claimed, err)
	}
	record, err := f.store.Lookup(ctx, req.Scope, req.OperationID)
	if err != nil || record.RunID != "" || record.Status != "accepted" || record.LeaseOwner != "worker1" {
		t.Fatalf("lease unexpectedly dispatched: %+v err=%v", record, err)
	}
	changed, err := f.store.ReconcileExpiredLeases(ctx, now.Add(2*time.Minute))
	if err != nil || changed != 1 {
		t.Fatalf("recover expired lease: changed=%d err=%v", changed, err)
	}
	record, err = f.store.Lookup(ctx, req.Scope, req.OperationID)
	if err != nil || record.Status != "indeterminate" || record.LeaseOwner != "" || record.RunID != "" {
		t.Fatalf("ambiguous operation not isolated: %+v err=%v", record, err)
	}
	claimed, err = f.store.AcquireLease(ctx, req.Scope, req.OperationID,
		"worker3", f.registry, now.Add(3*time.Minute), time.Minute)
	if err != nil || claimed {
		t.Fatalf("indeterminate state redispatched: %v %v", claimed, err)
	}
	if err := f.db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&after); err != nil || before != after {
		t.Fatalf("operation lease spawned agent_run: before=%d after=%d err=%v", before, after, err)
	}
}

func TestB2CLeaseConcurrentExactlyOneWinner(t *testing.T) {
	f := newB2CFixture(t)
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000011")
	if _, _, err := f.store.Reserve(context.Background(), req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	var winners int64
	var wg sync.WaitGroup
	failures := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := f.store.AcquireLease(context.Background(), req.Scope, req.OperationID,
				"worker-concurrent", f.registry, now, time.Minute)
			if err != nil {
				failures <- err
			}
			if ok {
				atomic.AddInt64(&winners, 1)
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Errorf("lease failed under concurrent readers: %v", err)
	}
	if winners != 1 {
		t.Fatalf("expected one lease winner, got %d", winners)
	}
}

func TestB2CLeaseScopedLookupAndExplicitAmbiguity(t *testing.T) {
	f := newB2CFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000012")
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []integration.ServiceOperationScope{
		{PrincipalID: "b2c-p2", WorkspaceID: "b2c-w2", OrgID: "org2"},
		{PrincipalID: "b2c-p1", WorkspaceID: "b2c-w1", OrgID: "org2", ProjectID: "project1"},
		{PrincipalID: "b2c-p1", WorkspaceID: "b2c-w1", OrgID: "org1"},
	} {
		if _, err := f.store.Lookup(ctx, invalid, req.OperationID); !errors.Is(err, integration.ErrServiceOperationDenied) {
			t.Fatalf("scope escaped: %+v err=%v", invalid, err)
		}
	}
	if ok, err := f.store.AcquireLease(ctx, req.Scope, req.OperationID, "worker1",
		f.registry, now, time.Minute); err != nil || !ok {
		t.Fatalf("claim: %v %v", ok, err)
	}
	if ok, err := f.store.MarkIndeterminate(ctx, req.Scope, req.OperationID,
		"wrong-worker", now.Add(time.Second)); err != nil || ok {
		t.Fatalf("wrong worker could change state: %v %v", ok, err)
	}
	if ok, err := f.store.MarkIndeterminate(ctx, req.Scope, req.OperationID,
		"worker1", now.Add(time.Second)); err != nil || !ok {
		t.Fatalf("real worker failed safe transition: %v %v", ok, err)
	}
	if ok, err := f.store.MarkIndeterminate(ctx, req.Scope, req.OperationID,
		"worker1", now.Add(2*time.Second)); err != nil || ok {
		t.Fatalf("second indeterminate transition accepted: %v %v", ok, err)
	}
}

func TestB2CRevokedPrincipalCannotLookupOrLease(t *testing.T) {
	f := newB2CFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000013")
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	if err := domainauth.NewServiceCredentialStore(f.db).RevokeServicePrincipal(
		ctx, req.Scope.PrincipalID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Lookup(ctx, req.Scope, req.OperationID); !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("revoked principal read operation: %v", err)
	}
	if ok, err := f.store.AcquireLease(ctx, req.Scope, req.OperationID,
		"worker1", f.registry, now.Add(time.Second), time.Minute); ok || !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("revoked principal claimed: %v %v", ok, err)
	}
}

func TestB2CLeaseRejectsHandlerChange(t *testing.T) {
	f := newB2CFixture(t)
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000014")
	if _, _, err := f.store.Reserve(context.Background(), req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	if err := f.registry.Register(integration.S2SCapabilityBinding{
		PrincipalID: "b2c-p1", WorkspaceID: "b2c-w1",
		CapabilityKey: "integration.diagnostics", InputSchemaVersion: "1",
		HandlerID: "changed-handler", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := f.store.AcquireLease(context.Background(), req.Scope, req.OperationID,
		"worker1", f.registry, now, time.Minute)
	if claimed || !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("handler change accepted old operation: claimed=%v err=%v", claimed, err)
	}
}
