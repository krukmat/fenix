package eval_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

func TestSyntheticOrgService_Create(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "synthetic-create")
	service := eval.NewSyntheticOrgService(db)

	org, err := service.Create(context.Background(), eval.CreateSyntheticOrgInput{
		WorkspaceID: wsID,
		Slug:        "acme-support",
		Name:        "Acme Support",
		Version:     2,
		Seed:        42,
		FixtureData: json.RawMessage("{\n  \"accounts\": [{\"id\": \"acc-1\"}]\n}"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if org.ID == "" {
		t.Fatal("expected synthetic org ID to be populated")
	}
	if org.Version != 2 {
		t.Fatalf("Version = %d; want 2", org.Version)
	}
	if org.Seed != 42 {
		t.Fatalf("Seed = %d; want 42", org.Seed)
	}
	if string(org.FixtureData) != `{"accounts":[{"id":"acc-1"}]}` {
		t.Fatalf("FixtureData = %s; want normalized compact JSON", org.FixtureData)
	}
}

func TestSyntheticOrgService_Generate_Deterministic(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "synthetic-generate")
	service := eval.NewSyntheticOrgService(db)

	org, err := service.Create(context.Background(), eval.CreateSyntheticOrgInput{
		WorkspaceID: wsID,
		Slug:        "seeded-acme",
		Name:        "Seeded Acme",
		Version:     1,
		Seed:        7,
		FixtureData: json.RawMessage(`{"contacts":[{"id":"c-1"}],"accounts":[{"id":"a-1"}]}`),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	first, err := service.Generate(context.Background(), wsID, org.ID)
	if err != nil {
		t.Fatalf("Generate first: %v", err)
	}
	second, err := service.Generate(context.Background(), wsID, org.ID)
	if err != nil {
		t.Fatalf("Generate second: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("Generate not deterministic:\nfirst=%s\nsecond=%s", first, second)
	}
}

func TestSyntheticOrgService_GetByID(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "synthetic-get")
	service := eval.NewSyntheticOrgService(db)

	created, err := service.Create(context.Background(), eval.CreateSyntheticOrgInput{
		WorkspaceID: wsID,
		Slug:        "seeded-beta",
		Name:        "Seeded Beta",
		Version:     3,
		Seed:        99,
		FixtureData: json.RawMessage(`{"pipelines":[{"id":"p-1"}]}`),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fetched, err := service.GetByID(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("ID = %q; want %q", fetched.ID, created.ID)
	}
	if fetched.Slug != "seeded-beta" {
		t.Fatalf("Slug = %q; want %q", fetched.Slug, "seeded-beta")
	}
}
