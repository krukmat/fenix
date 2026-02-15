// Traces: FR-001
package server

import (
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "0.0.0.0" {
		t.Fatalf("Host = %q; want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d; want %d", cfg.Port, 8080)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Fatalf("ReadTimeout = %v; want %v", cfg.ReadTimeout, 15*time.Second)
	}
	if cfg.WriteTimeout != 15*time.Second {
		t.Fatalf("WriteTimeout = %v; want %v", cfg.WriteTimeout, 15*time.Second)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %v; want %v", cfg.IdleTimeout, 60*time.Second)
	}
}

func TestNewServer_ConfiguresAddressAndHandler(t *testing.T) {
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("sqlite.NewDB error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("sqlite.MigrateUp error = %v", err)
	}

	cfg := Config{Host: "127.0.0.1", Port: 18080, ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 3 * time.Second}
	s := NewServer(db, cfg)

	if s == nil {
		t.Fatal("NewServer() returned nil")
	}
	if s.http == nil {
		t.Fatal("server.http should not be nil")
	}
	if s.http.Addr != "127.0.0.1:18080" {
		t.Fatalf("Addr = %q; want %q", s.http.Addr, "127.0.0.1:18080")
	}
	if s.http.Handler == nil {
		t.Fatal("Handler should not be nil")
	}
}
