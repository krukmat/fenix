package agent

import (
	"fmt"
	"sync"
	"testing"
)

func TestRunnerRegistryRegisterAndGet(t *testing.T) {
	registry := NewRunnerRegistry()
	runner := stubRunner{}

	if err := registry.Register("support", runner); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, ok := registry.Get("support")
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got == nil {
		t.Fatal("Get() returned nil runner")
	}
}

func TestRunnerRegistryRejectsDuplicateType(t *testing.T) {
	registry := NewRunnerRegistry()
	runner := stubRunner{}

	if err := registry.Register("support", runner); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := registry.Register("support", runner); err != ErrRunnerAlreadyExists {
		t.Fatalf("second Register() error = %v, want %v", err, ErrRunnerAlreadyExists)
	}
}

func TestRunnerRegistryRejectsInvalidRegistration(t *testing.T) {
	registry := NewRunnerRegistry()

	if err := registry.Register("   ", stubRunner{}); err != ErrRunnerTypeEmpty {
		t.Fatalf("Register(empty) error = %v, want %v", err, ErrRunnerTypeEmpty)
	}
	if err := registry.Register("support", nil); err != ErrRunnerNil {
		t.Fatalf("Register(nil) error = %v, want %v", err, ErrRunnerNil)
	}
}

func TestRunnerRegistryResolveUnknownType(t *testing.T) {
	registry := NewRunnerRegistry()

	if _, err := registry.Resolve("missing"); err != ErrRunnerNotFound {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrRunnerNotFound)
	}
}

func TestRunnerRegistryGetUnknownType(t *testing.T) {
	registry := NewRunnerRegistry()

	got, ok := registry.Get("missing")
	if ok {
		t.Fatal("Get() ok = true, want false")
	}
	if got != nil {
		t.Fatal("Get() runner != nil, want nil")
	}
}

func TestRunnerRegistryConcurrentGet(t *testing.T) {
	registry := NewRunnerRegistry()
	if err := registry.Register("support", stubRunner{}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, ok := registry.Get("support")
			if !ok || got == nil {
				errCh <- fmt.Errorf("Get() returned ok=%v runner=%v", ok, got)
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}
}
