// Package crm provides domain logic for CRM entities (accounts, contacts, deals, cases, activities, etc.)
// Task 1.3.5: Account service layer — orchestrates sqlc queries + business logic
package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// Account domain model — represents a customer/organization account.
type Account struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	Name        string     `json:"name"`
	Domain      *string    `json:"domain,omitempty"`
	Industry    *string    `json:"industry,omitempty"`
	SizeSegment *string    `json:"sizeSegment,omitempty"` // smb|mid|enterprise
	OwnerID     string     `json:"ownerId"`
	Address     *string    `json:"address,omitempty"`     // JSON blob
	Metadata    *string    `json:"metadata,omitempty"`    // JSON blob
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// CreateAccountInput defines required + optional fields for account creation.
type CreateAccountInput struct {
	WorkspaceID string
	Name        string
	Domain      string
	Industry    string
	SizeSegment string // smb|mid|enterprise
	OwnerID     string
	Address     string // JSON
	Metadata    string // JSON
}

// UpdateAccountInput defines fields that can be updated.
type UpdateAccountInput struct {
	Name        string
	Domain      string
	Industry    string
	SizeSegment string
	OwnerID     string
	Address     string // JSON
	Metadata    string // JSON
}

// ListAccountsInput defines pagination for account listings.
type ListAccountsInput struct {
	Limit  int
	Offset int
}

// AccountService provides account operations scoped to a workspace.
type AccountService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

// NewAccountService creates an AccountService instance.
func NewAccountService(db *sql.DB) *AccountService {
	return &AccountService{
		db:      db,
		querier: sqlcgen.New(db),
	}
}

// Create inserts a new account into the database.
// Task 1.3.5: TDD red → green
func (s *AccountService) Create(ctx context.Context, input CreateAccountInput) (*Account, error) {
	accountID := uuid.NewV7().String()
	now := time.Now().UTC()

	// Convert empty strings to nil for nullable fields (sqlc handles this)
	domain := nullString(input.Domain)
	industry := nullString(input.Industry)
	sizeSegment := nullString(input.SizeSegment)
	address := nullString(input.Address)
	metadata := nullString(input.Metadata)

	err := s.querier.CreateAccount(ctx, sqlcgen.CreateAccountParams{
		ID:          accountID,
		WorkspaceID: input.WorkspaceID,
		Name:        input.Name,
		Domain:      domain,
		Industry:    industry,
		SizeSegment: sizeSegment,
		OwnerID:     input.OwnerID,
		Address:     address,
		Metadata:    metadata,
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	// Return the created account by fetching it
	return s.Get(ctx, input.WorkspaceID, accountID)
}

// Get retrieves an account by ID (excludes soft-deleted).
func (s *AccountService) Get(ctx context.Context, workspaceID, accountID string) (*Account, error) {
	row, err := s.querier.GetAccountByID(ctx, sqlcgen.GetAccountByIDParams{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return rowToAccount(row), nil
}

// List retrieves active accounts in a workspace with pagination.
func (s *AccountService) List(ctx context.Context, workspaceID string, input ListAccountsInput) ([]*Account, int, error) {
	// Get total count
	total, err := s.querier.CountAccountsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count accounts: %w", err)
	}

	// Fetch paginated results
	rows, err := s.querier.ListAccountsByWorkspace(ctx, sqlcgen.ListAccountsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list accounts: %w", err)
	}

	accounts := make([]*Account, len(rows))
	for i, row := range rows {
		accounts[i] = rowToAccount(row)
	}

	return accounts, int(total), nil
}

// ListByOwner retrieves all accounts owned by a user.
func (s *AccountService) ListByOwner(ctx context.Context, workspaceID, ownerID string) ([]*Account, error) {
	rows, err := s.querier.ListAccountsByOwner(ctx, sqlcgen.ListAccountsByOwnerParams{
		WorkspaceID: workspaceID,
		OwnerID:     ownerID,
	})
	if err != nil {
		return nil, fmt.Errorf("list accounts by owner: %w", err)
	}

	accounts := make([]*Account, len(rows))
	for i, row := range rows {
		accounts[i] = rowToAccount(row)
	}

	return accounts, nil
}

// Update modifies an account (excludes soft-deleted).
func (s *AccountService) Update(ctx context.Context, workspaceID, accountID string, input UpdateAccountInput) (*Account, error) {
	now := time.Now().UTC()

	domain := nullString(input.Domain)
	industry := nullString(input.Industry)
	sizeSegment := nullString(input.SizeSegment)
	address := nullString(input.Address)
	metadata := nullString(input.Metadata)

	err := s.querier.UpdateAccount(ctx, sqlcgen.UpdateAccountParams{
		Name:        input.Name,
		Domain:      domain,
		Industry:    industry,
		SizeSegment: sizeSegment,
		OwnerID:     input.OwnerID,
		Address:     address,
		Metadata:    metadata,
		UpdatedAt:   now.Format(time.RFC3339),
		ID:          accountID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	return s.Get(ctx, workspaceID, accountID)
}

// Delete performs a soft delete (sets deleted_at).
func (s *AccountService) Delete(ctx context.Context, workspaceID, accountID string) error {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	err := s.querier.SoftDeleteAccount(ctx, sqlcgen.SoftDeleteAccountParams{
		DeletedAt:   &nowStr,
		UpdatedAt:   nowStr,
		ID:          accountID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete account: %w", err)
	}

	return nil
}

// --- internal helpers ---

// rowToAccount converts a sqlcgen row to an Account domain model.
func rowToAccount(row sqlcgen.Account) *Account {
	deletedAt := row.DeletedAt
	var deletedAtTime *time.Time
	if deletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *deletedAt)
		deletedAtTime = &t
	}

	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	return &Account{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Name:        row.Name,
		Domain:      row.Domain,
		Industry:    row.Industry,
		SizeSegment: row.SizeSegment,
		OwnerID:     row.OwnerID,
		Address:     row.Address,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   deletedAtTime,
	}
}

// nullString converts an empty string to nil, non-empty to pointer.
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
