// Tests for BusRegistry (Task R.12).
package blackboard_test

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func setupBusRegistryDB(t *testing.T) (*sql.DB, string) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-registry', 'Registry WS', 'registry-ws', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-registry', 'ws-registry', 'active', datetime('now'))`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, "cw-registry"
}

func TestBusRegistry_GetOrCreate_ReturnsSameInstance(t *testing.T) {
	db, cwID := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	b1 := reg.GetOrCreate(cwID)
	b2 := reg.GetOrCreate(cwID)

	if b1 != b2 {
		t.Error("GetOrCreate returned different instances for the same cwID; want same")
	}
}

func TestBusRegistry_GetOrCreate_DifferentIDs_DifferentInstances(t *testing.T) {
	db, _ := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	b1 := reg.GetOrCreate("cw-a")
	b2 := reg.GetOrCreate("cw-b")

	if b1 == b2 {
		t.Error("GetOrCreate returned same instance for different cwIDs; want distinct")
	}
}

func TestBusRegistry_Len_ReflectsCachedCount(t *testing.T) {
	db, _ := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	if reg.Len() != 0 {
		t.Fatalf("Len() = %d; want 0 on empty registry", reg.Len())
	}

	reg.GetOrCreate("cw-x")
	if reg.Len() != 1 {
		t.Errorf("Len() = %d; want 1 after one GetOrCreate", reg.Len())
	}

	reg.GetOrCreate("cw-y")
	if reg.Len() != 2 {
		t.Errorf("Len() = %d; want 2 after two distinct GetOrCreate", reg.Len())
	}
}

func TestBusRegistry_Evict_ClosesAndRemoves(t *testing.T) {
	db, cwID := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	reg.GetOrCreate(cwID)
	if reg.Len() != 1 {
		t.Fatalf("Len() = %d; want 1 before evict", reg.Len())
	}

	reg.Evict(cwID)

	if reg.Len() != 0 {
		t.Errorf("Len() = %d; want 0 after evict", reg.Len())
	}
}

func TestBusRegistry_Evict_NewInstanceAfterEviction(t *testing.T) {
	db, cwID := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	b1 := reg.GetOrCreate(cwID)
	reg.Evict(cwID)
	b2 := reg.GetOrCreate(cwID)

	if b1 == b2 {
		t.Error("GetOrCreate after Evict returned same instance; want a fresh bus")
	}
}

func TestBusRegistry_Evict_Idempotent(t *testing.T) {
	db, cwID := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	reg.GetOrCreate(cwID)
	reg.Evict(cwID)

	// second evict must not panic
	reg.Evict(cwID)

	if reg.Len() != 0 {
		t.Errorf("Len() = %d; want 0 after double evict", reg.Len())
	}
}

func TestBusRegistry_ConcurrentGetOrCreate_SameID_NoRace(t *testing.T) {
	db, _ := setupBusRegistryDB(t)
	reg := blackboard.NewBusRegistry(db)

	const goroutines = 100
	cwID := "cw-concurrent"

	startGun := make(chan struct{})
	results := make([]blackboard.WorkspaceBus, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startGun
			results[i] = reg.GetOrCreate(cwID)
		}(i)
	}

	close(startGun)
	wg.Wait()

	// all goroutines must have received the same instance
	first := results[0]
	for i, b := range results {
		if b != first {
			t.Errorf("goroutine %d got a different bus instance; want same", i)
		}
	}
}
