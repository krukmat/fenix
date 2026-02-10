// Package crm provides domain logic for CRM entities.
// Task 1.4: Contact service layer.
package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// Contact domain model.
type Contact struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	AccountID   string     `json:"accountId"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Status      string     `json:"status"`
	OwnerID     string     `json:"ownerId"`
	Metadata    *string    `json:"metadata,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// CreateContactInput defines required + optional fields for contact creation.
type CreateContactInput struct {
	WorkspaceID string
	AccountID   string
	FirstName   string
	LastName    string
	Email       string
	Phone       string
	Title       string
	Status      string
	OwnerID     string
	Metadata    string
}

// UpdateContactInput defines fields that can be updated.
type UpdateContactInput struct {
	AccountID string
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Title     string
	Status    string
	OwnerID   string
	Metadata  string
}

// ListContactsInput defines pagination for contact listings.
type ListContactsInput struct {
	Limit  int
	Offset int
}

// ContactService provides contact operations scoped to a workspace.
type ContactService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

// NewContactService creates a ContactService instance.
func NewContactService(db *sql.DB) *ContactService {
	return &ContactService{
		db:      db,
		querier: sqlcgen.New(db),
	}
}

// Create inserts a new contact into the database.
func (s *ContactService) Create(ctx context.Context, input CreateContactInput) (*Contact, error) {
	contactID := uuid.NewV7().String()
	now := time.Now().UTC()
	status := input.Status
	if status == "" {
		status = "active"
	}

	err := s.querier.CreateContact(ctx, sqlcgen.CreateContactParams{
		ID:          contactID,
		WorkspaceID: input.WorkspaceID,
		AccountID:   input.AccountID,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Email:       nullString(input.Email),
		Phone:       nullString(input.Phone),
		Title:       nullString(input.Title),
		Status:      status,
		OwnerID:     input.OwnerID,
		Metadata:    nullString(input.Metadata),
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}

	return s.Get(ctx, input.WorkspaceID, contactID)
}

// Get retrieves a contact by ID (excludes soft-deleted).
func (s *ContactService) Get(ctx context.Context, workspaceID, contactID string) (*Contact, error) {
	row, err := s.querier.GetContactByID(ctx, sqlcgen.GetContactByIDParams{
		ID:          contactID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return rowToContact(row), nil
}

// List retrieves active contacts in a workspace with pagination.
func (s *ContactService) List(ctx context.Context, workspaceID string, input ListContactsInput) ([]*Contact, int, error) {
	total, err := s.querier.CountContactsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count contacts: %w", err)
	}

	rows, err := s.querier.ListContactsByWorkspace(ctx, sqlcgen.ListContactsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list contacts: %w", err)
	}

	contacts := make([]*Contact, len(rows))
	for i, row := range rows {
		contacts[i] = rowToContact(row)
	}

	return contacts, int(total), nil
}

// ListByAccount retrieves active contacts for an account in a workspace.
func (s *ContactService) ListByAccount(ctx context.Context, workspaceID, accountID string) ([]*Contact, error) {
	rows, err := s.querier.ListContactsByAccount(ctx, sqlcgen.ListContactsByAccountParams{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
	})
	if err != nil {
		return nil, fmt.Errorf("list contacts by account: %w", err)
	}

	contacts := make([]*Contact, len(rows))
	for i, row := range rows {
		contacts[i] = rowToContact(row)
	}

	return contacts, nil
}

// Update modifies a contact (excludes soft-deleted).
func (s *ContactService) Update(ctx context.Context, workspaceID, contactID string, input UpdateContactInput) (*Contact, error) {
	now := time.Now().UTC()

	err := s.querier.UpdateContact(ctx, sqlcgen.UpdateContactParams{
		AccountID:   input.AccountID,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Email:       nullString(input.Email),
		Phone:       nullString(input.Phone),
		Title:       nullString(input.Title),
		Status:      input.Status,
		OwnerID:     input.OwnerID,
		Metadata:    nullString(input.Metadata),
		UpdatedAt:   now.Format(time.RFC3339),
		ID:          contactID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update contact: %w", err)
	}

	return s.Get(ctx, workspaceID, contactID)
}

// Delete performs a soft delete (sets deleted_at).
func (s *ContactService) Delete(ctx context.Context, workspaceID, contactID string) error {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	err := s.querier.SoftDeleteContact(ctx, sqlcgen.SoftDeleteContactParams{
		DeletedAt:   &nowStr,
		UpdatedAt:   nowStr,
		ID:          contactID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete contact: %w", err)
	}

	return nil
}

// rowToContact converts a sqlcgen row to a Contact domain model.
func rowToContact(row sqlcgen.Contact) *Contact {
	deletedAt := row.DeletedAt
	var deletedAtTime *time.Time
	if deletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *deletedAt)
		deletedAtTime = &t
	}

	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	return &Contact{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		AccountID:   row.AccountID,
		FirstName:   row.FirstName,
		LastName:    row.LastName,
		Email:       row.Email,
		Phone:       row.Phone,
		Title:       row.Title,
		Status:      row.Status,
		OwnerID:     row.OwnerID,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   deletedAtTime,
	}
}
