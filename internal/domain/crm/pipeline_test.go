// Traces: FR-001, FR-002
package crm_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestPipelineService_CRUD(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewPipelineService(db)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	created, err := svc.Create(context.Background(), crm.CreatePipelineInput{
		WorkspaceID: wsID,
		Name:        "Sales",
		EntityType:  "deal",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created.ID")
	}

	got, err := svc.Get(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != "Sales" {
		t.Fatalf("expected name Sales, got %q", got.Name)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListPipelinesInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("expected at least one pipeline, got total=%d len=%d", total, len(items))
	}

	updated, err := svc.Update(context.Background(), wsID, created.ID, crm.UpdatePipelineInput{
		Name:       "Sales Updated",
		EntityType: "deal",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Sales Updated" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}

	if err := svc.Delete(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = svc.Get(context.Background(), wsID, created.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestPipelineService_StageCRUD(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewPipelineService(db)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	p, err := svc.Create(context.Background(), crm.CreatePipelineInput{
		WorkspaceID: wsID,
		Name:        "Support",
		EntityType:  "case",
	})
	if err != nil {
		t.Fatalf("seed pipeline Create() error = %v", err)
	}

	stage, err := svc.CreateStage(context.Background(), crm.CreatePipelineStageInput{
		PipelineID: p.ID,
		Name:       "Open",
		Position:   1,
	})
	if err != nil {
		t.Fatalf("CreateStage() error = %v", err)
	}
	if stage.ID == "" {
		t.Fatalf("expected stage.ID")
	}

	got, err := svc.GetStage(context.Background(), stage.ID)
	if err != nil {
		t.Fatalf("GetStage() error = %v", err)
	}
	if got.Name != "Open" {
		t.Fatalf("expected stage name Open, got %q", got.Name)
	}

	stages, err := svc.ListStages(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListStages() error = %v", err)
	}
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(stages))
	}

	updated, err := svc.UpdateStage(context.Background(), stage.ID, crm.UpdatePipelineStageInput{
		Name:     "In Progress",
		Position: 2,
	})
	if err != nil {
		t.Fatalf("UpdateStage() error = %v", err)
	}
	if updated.Name != "In Progress" || updated.Position != 2 {
		t.Fatalf("unexpected updated stage: %+v", updated)
	}

	if err := svc.DeleteStage(context.Background(), stage.ID); err != nil {
		t.Fatalf("DeleteStage() error = %v", err)
	}

	_, err = svc.GetStage(context.Background(), stage.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after stage delete, got %v", err)
	}
}
