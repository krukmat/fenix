package crm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrInvalidDealInput = errors.New("invalid deal input")
	ErrInvalidCaseInput = errors.New("invalid case input")
)

var (
	validDealStatuses = map[string]struct{}{
		"open":   {},
		"won":    {},
		"lost":   {},
		"closed": {},
	}
	validCasePriorities = map[string]struct{}{
		"low":    {},
		"medium": {},
		"high":   {},
		"urgent": {},
	}
	validCaseStatuses = map[string]struct{}{
		"open":        {},
		"in_progress": {},
		"resolved":    {},
		"closed":      {},
		"escalated":   {},
	}
)

func validateDealInput(ctx context.Context, db *sql.DB, workspaceID string, input CreateDealInput) error {
	if err := validateDealRelations(ctx, db, workspaceID, input); err != nil {
		return err
	}
	return validateDealValues(input)
}

func validateCaseInput(ctx context.Context, db *sql.DB, workspaceID string, input CreateCaseInput) error {
	if err := validateCaseRelations(ctx, db, workspaceID, input); err != nil {
		return err
	}
	return validateCaseValues(input)
}

func validateDealRelations(ctx context.Context, db *sql.DB, workspaceID string, input CreateDealInput) error {
	if err := ensureUserExists(ctx, db, workspaceID, input.OwnerID); err != nil {
		return invalidDealInput("owner_id is invalid", err)
	}
	if err := ensureAccountExists(ctx, db, workspaceID, input.AccountID); err != nil {
		return invalidDealInput("account_id is invalid", err)
	}
	if input.ContactID != "" {
		if err := ensureContactExists(ctx, db, workspaceID, input.ContactID); err != nil {
			return invalidDealInput("contact_id is invalid", err)
		}
	}
	if err := ensurePipelineExists(ctx, db, workspaceID, input.PipelineID); err != nil {
		return invalidDealInput("pipeline_id is invalid", err)
	}
	if err := ensureStageBelongsToPipeline(ctx, db, input.StageID, input.PipelineID); err != nil {
		return invalidDealInput("stage_id does not belong to pipeline_id", err)
	}
	return nil
}

func validateDealValues(input CreateDealInput) error {
	if input.Amount != nil && *input.Amount < 0 {
		return invalidDealInput("amount cannot be negative", nil)
	}
	if input.Status != "" && !isValidEnum(input.Status, validDealStatuses) {
		return invalidDealInput("status is invalid", nil)
	}
	return nil
}

func validateCaseRelations(ctx context.Context, db *sql.DB, workspaceID string, input CreateCaseInput) error {
	if err := ensureUserExists(ctx, db, workspaceID, input.OwnerID); err != nil {
		return invalidCaseInput("owner_id is invalid", err)
	}
	if err := validateOptionalCaseAccount(ctx, db, workspaceID, input.AccountID); err != nil {
		return err
	}
	if err := validateOptionalCaseContact(ctx, db, workspaceID, input.ContactID); err != nil {
		return err
	}
	if err := validateOptionalCasePipeline(ctx, db, workspaceID, input.PipelineID); err != nil {
		return err
	}
	return validateOptionalCaseStage(ctx, db, input.StageID, input.PipelineID)
}

func validateOptionalCaseAccount(ctx context.Context, db *sql.DB, workspaceID, accountID string) error {
	if accountID == "" {
		return nil
	}
	if err := ensureAccountExists(ctx, db, workspaceID, accountID); err != nil {
		return invalidCaseInput("account_id is invalid", err)
	}
	return nil
}

func validateOptionalCaseContact(ctx context.Context, db *sql.DB, workspaceID, contactID string) error {
	if contactID == "" {
		return nil
	}
	if err := ensureContactExists(ctx, db, workspaceID, contactID); err != nil {
		return invalidCaseInput("contact_id is invalid", err)
	}
	return nil
}

func validateOptionalCasePipeline(ctx context.Context, db *sql.DB, workspaceID, pipelineID string) error {
	if pipelineID == "" {
		return nil
	}
	if err := ensurePipelineExists(ctx, db, workspaceID, pipelineID); err != nil {
		return invalidCaseInput("pipeline_id is invalid", err)
	}
	return nil
}

func validateOptionalCaseStage(ctx context.Context, db *sql.DB, stageID, pipelineID string) error {
	if stageID == "" {
		return nil
	}
	if pipelineID == "" {
		return invalidCaseInput("stage_id requires pipeline_id", nil)
	}
	if err := ensureStageBelongsToPipeline(ctx, db, stageID, pipelineID); err != nil {
		return invalidCaseInput("stage_id does not belong to pipeline_id", err)
	}
	return nil
}

func validateCaseValues(input CreateCaseInput) error {
	if input.Priority != "" && !isValidEnum(input.Priority, validCasePriorities) {
		return invalidCaseInput("priority is invalid", nil)
	}
	if input.Status != "" && !isValidEnum(input.Status, validCaseStatuses) {
		return invalidCaseInput("status is invalid", nil)
	}
	return nil
}

func ensureUserExists(ctx context.Context, db *sql.DB, workspaceID, userID string) error {
	return ensureExists(ctx, db, `SELECT 1 FROM user_account WHERE id = ? AND workspace_id = ? LIMIT 1`, userID, workspaceID)
}

func ensureAccountExists(ctx context.Context, db *sql.DB, workspaceID, accountID string) error {
	return ensureExists(ctx, db, `SELECT 1 FROM account WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1`, accountID, workspaceID)
}

func ensureContactExists(ctx context.Context, db *sql.DB, workspaceID, contactID string) error {
	return ensureExists(ctx, db, `SELECT 1 FROM contact WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1`, contactID, workspaceID)
}

func ensurePipelineExists(ctx context.Context, db *sql.DB, workspaceID, pipelineID string) error {
	return ensureExists(ctx, db, `SELECT 1 FROM pipeline WHERE id = ? AND workspace_id = ? LIMIT 1`, pipelineID, workspaceID)
}

func ensureStageBelongsToPipeline(ctx context.Context, db *sql.DB, stageID, pipelineID string) error {
	var stagePipelineID string
	err := db.QueryRowContext(ctx, `SELECT pipeline_id FROM pipeline_stage WHERE id = ? LIMIT 1`, stageID).Scan(&stagePipelineID)
	if err != nil {
		return err
	}
	if stagePipelineID != pipelineID {
		return fmt.Errorf("stage %s belongs to pipeline %s", stageID, stagePipelineID)
	}
	return nil
}

func ensureExists(ctx context.Context, db *sql.DB, query string, args ...any) error {
	var exists int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&exists); err != nil {
		return err
	}
	return nil
}

func invalidDealInput(reason string, err error) error {
	return wrapValidationError(ErrInvalidDealInput, reason, err)
}

func invalidCaseInput(reason string, err error) error {
	return wrapValidationError(ErrInvalidCaseInput, reason, err)
}

func wrapValidationError(base error, reason string, err error) error {
	if err == nil || errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", base, reason)
	}
	return fmt.Errorf("%w: %s: %w", base, reason, err)
}

func isValidEnum(value string, allowed map[string]struct{}) bool {
	_, ok := allowed[value]
	return ok
}
