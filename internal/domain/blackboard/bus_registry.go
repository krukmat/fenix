// Package blackboard — shared bus registry keyed by cognitive_workspace_id (Task R.12).
package blackboard

import (
	"database/sql"
	"sync"
)

// BusRegistry caches one WorkspaceBus per cognitive_workspace_id so multiple agents
// on the same workspace share subscribers. Evict must be called when a workspace
// transitions to closed or expired to release resources.
type BusRegistry struct {
	db    *sql.DB
	mu    sync.Mutex
	buses map[string]WorkspaceBus
}

// NewBusRegistry constructs an empty registry backed by db.
func NewBusRegistry(db *sql.DB) *BusRegistry {
	return &BusRegistry{
		db:    db,
		buses: make(map[string]WorkspaceBus),
	}
}

// GetOrCreate returns the cached bus for cwID, creating one if absent.
func (r *BusRegistry) GetOrCreate(cwID string) WorkspaceBus {
	r.mu.Lock()
	defer r.mu.Unlock()
	if bus, ok := r.buses[cwID]; ok {
		return bus
	}
	bus := NewWorkspaceBus(cwID, r.db)
	r.buses[cwID] = bus
	return bus
}

// Evict closes and removes the bus for cwID. Safe to call on unknown IDs.
func (r *BusRegistry) Evict(cwID string) {
	r.mu.Lock()
	bus, ok := r.buses[cwID]
	delete(r.buses, cwID)
	r.mu.Unlock()
	if ok {
		bus.Close()
	}
}

// Len returns the number of currently cached buses.
func (r *BusRegistry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.buses)
}
