package crm

import (
	"context"
	"database/sql"

	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
)

type auditLogger interface {
	LogWithDetails(
		ctx context.Context,
		workspaceID string,
		actorID string,
		actorType domainaudit.ActorType,
		action string,
		entityType *string,
		entityID *string,
		details *domainaudit.EventDetails,
		outcome domainaudit.Outcome,
	) error
}

const (
	actionAccountCreated = "account.created"
	actionAccountUpdated = "account.updated"
	actionAccountDeleted = "account.deleted"
	actionContactCreated = "contact.created"
	actionContactUpdated = "contact.updated"
	actionContactDeleted = "contact.deleted"
	actionLeadCreated    = "lead.created"
	actionLeadUpdated    = "lead.updated"
	actionLeadDeleted    = "lead.deleted"
	actionDealCreated    = "deal.created"
	actionDealUpdated    = "deal.updated"
	actionDealDeleted    = "deal.deleted"
	actionCaseCreated    = "case.created"
	actionCaseUpdated    = "case.updated"
	actionCaseDeleted    = "case.deleted"
	actionNoteCreated    = "note.created"
	actionNoteUpdated    = "note.updated"
	actionNoteDeleted    = "note.deleted"
)

func newCRMAuditService(db *sql.DB) *domainaudit.AuditService {
	return domainaudit.NewAuditService(db)
}

func logCRMAudit(
	ctx context.Context,
	auditSvc auditLogger,
	workspaceID, actorID, action, entityType, entityID string,
) {
	if auditSvc == nil {
		return
	}

	actorType := domainaudit.ActorTypeSystem
	if actorID != "" {
		actorType = domainaudit.ActorTypeUser
	}

	_ = auditSvc.LogWithDetails(
		ctx,
		workspaceID,
		resolveAuditActorID(actorID),
		actorType,
		action,
		&entityType,
		&entityID,
		nil,
		domainaudit.OutcomeSuccess,
	)
}

func resolveAuditActorID(actorID string) string {
	if actorID == "" {
		return "system"
	}
	return actorID
}
