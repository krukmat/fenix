package knowledge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	TopicRecordCreated = "record.created"
	TopicRecordUpdated = "record.updated"
	TopicRecordDeleted = "record.deleted"
)

type ChangeType string

const (
	ChangeTypeCreated ChangeType = "created"
	ChangeTypeUpdated ChangeType = "updated"
	ChangeTypeDeleted ChangeType = "deleted"
)

const (
	EntityTypeAccount    = "account"
	EntityTypeCaseTicket = "case_ticket"
)

// RecordChangedEvent is the CDC payload used by Task 2.7.
type RecordChangedEvent struct {
	EntityType  string
	EntityID    string
	WorkspaceID string
	ChangeType  ChangeType
	OccurredAt  time.Time
}

// TopicForChangeType resolves the event bus topic for a change type.
func TopicForChangeType(changeType ChangeType) string {
	switch changeType {
	case ChangeTypeCreated:
		return TopicRecordCreated
	case ChangeTypeDeleted:
		return TopicRecordDeleted
	default:
		return TopicRecordUpdated
	}
}

// ReindexService consumes CDC events and keeps knowledge indexes fresh.
type ReindexService struct {
	q      *sqlcgen.Queries
	bus    eventbus.EventBus
	ingest *IngestService
	audit  *audit.AuditService
}

func NewReindexService(db *sql.DB, bus eventbus.EventBus, ingest *IngestService, auditSvc *audit.AuditService) *ReindexService {
	return &ReindexService{
		q:      sqlcgen.New(db),
		bus:    bus,
		ingest: ingest,
		audit:  auditSvc,
	}
}

// Start subscribes to record.created|updated|deleted topics and handles events.
func (s *ReindexService) Start(ctx context.Context) {
	createdCh := s.bus.Subscribe(TopicRecordCreated)
	updatedCh := s.bus.Subscribe(TopicRecordUpdated)
	deletedCh := s.bus.Subscribe(TopicRecordDeleted)

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-createdCh:
			record, ok := evt.Payload.(RecordChangedEvent)
			if !ok {
				continue
			}
			record.ChangeType = ChangeTypeCreated
			_ = s.HandleRecordChange(ctx, record)
		case evt := <-updatedCh:
			record, ok := evt.Payload.(RecordChangedEvent)
			if !ok {
				continue
			}
			record.ChangeType = ChangeTypeUpdated
			_ = s.HandleRecordChange(ctx, record)
		case evt := <-deletedCh:
			record, ok := evt.Payload.(RecordChangedEvent)
			if !ok {
				continue
			}
			record.ChangeType = ChangeTypeDeleted
			_ = s.HandleRecordChange(ctx, record)
		}
	}
}

// QueueWorkspaceReindex publishes reindex update events for all linked entities.
func (s *ReindexService) QueueWorkspaceReindex(ctx context.Context, workspaceID string, entityType *string) (int, error) {
	const batchSize = 200
	offset := 0
	queued := 0

	for {
		var (
			items []sqlcgen.KnowledgeItem
			err   error
		)

		if entityType != nil && *entityType != "" {
			items, err = s.q.ListKnowledgeItemsByEntity(ctx, sqlcgen.ListKnowledgeItemsByEntityParams{
				WorkspaceID: workspaceID,
				EntityType:  entityType,
				Limit:       batchSize,
				Offset:      int64(offset),
			})
		} else {
			items, err = s.q.ListKnowledgeItemsByWorkspace(ctx, sqlcgen.ListKnowledgeItemsByWorkspaceParams{
				WorkspaceID: workspaceID,
				Limit:       batchSize,
				Offset:      int64(offset),
			})
		}
		if err != nil {
			return 0, err
		}
		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if item.EntityType == nil || item.EntityID == nil {
				continue
			}
			s.bus.Publish(TopicRecordUpdated, RecordChangedEvent{
				EntityType:  *item.EntityType,
				EntityID:    *item.EntityID,
				WorkspaceID: workspaceID,
				ChangeType:  ChangeTypeUpdated,
				OccurredAt:  time.Now(),
			})
			queued++
		}

		offset += len(items)
		if len(items) < batchSize {
			break
		}
	}

	return queued, nil
}

// HandleRecordChange updates or soft-deletes linked knowledge items after CRM changes.
func (s *ReindexService) HandleRecordChange(ctx context.Context, evt RecordChangedEvent) error {
	if evt.EntityType == "" || evt.EntityID == "" || evt.WorkspaceID == "" {
		return errors.New("invalid record change event")
	}

	entityType := evt.EntityType
	entityID := evt.EntityID

	item, err := s.q.GetKnowledgeItemByEntity(ctx, sqlcgen.GetKnowledgeItemByEntityParams{
		WorkspaceID: evt.WorkspaceID,
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	start := evt.OccurredAt
	if start.IsZero() {
		start = time.Now()
	}

	var opErr error
	switch evt.ChangeType {
	case ChangeTypeDeleted:
		opErr = s.handleDelete(ctx, item)
	default:
		opErr = s.handleUpsert(ctx, evt, item)
	}

	s.logReindexAudit(ctx, evt, opErr, time.Since(start))
	return opErr
}

func (s *ReindexService) handleDelete(ctx context.Context, item sqlcgen.KnowledgeItem) error {
	if err := s.q.DeleteVecEmbeddingsByKnowledgeItem(ctx, sqlcgen.DeleteVecEmbeddingsByKnowledgeItemParams{
		KnowledgeItemID: item.ID,
		WorkspaceID:     item.WorkspaceID,
	}); err != nil {
		return err
	}

	if err := s.q.DeleteEmbeddingDocumentsByKnowledgeItem(ctx, sqlcgen.DeleteEmbeddingDocumentsByKnowledgeItemParams{
		KnowledgeItemID: item.ID,
		WorkspaceID:     item.WorkspaceID,
	}); err != nil {
		return err
	}

	now := time.Now()
	return s.q.SoftDeleteKnowledgeItem(ctx, sqlcgen.SoftDeleteKnowledgeItemParams{
		DeletedAt:   &now,
		ID:          item.ID,
		WorkspaceID: item.WorkspaceID,
	})
}

func (s *ReindexService) handleUpsert(ctx context.Context, evt RecordChangedEvent, item sqlcgen.KnowledgeItem) error {
	title, rawContent, sourceType, err := s.buildKnowledgePayloadFromEntity(ctx, evt)
	if err != nil {
		return err
	}

	if err := s.q.DeleteVecEmbeddingsByKnowledgeItem(ctx, sqlcgen.DeleteVecEmbeddingsByKnowledgeItemParams{
		KnowledgeItemID: item.ID,
		WorkspaceID:     item.WorkspaceID,
	}); err != nil {
		return err
	}

	_, err = s.ingest.Ingest(ctx, CreateKnowledgeItemInput{
		WorkspaceID: evt.WorkspaceID,
		SourceType:  sourceType,
		Title:       title,
		RawContent:  rawContent,
		EntityType:  &evt.EntityType,
		EntityID:    &evt.EntityID,
	})
	return err
}

func (s *ReindexService) buildKnowledgePayloadFromEntity(ctx context.Context, evt RecordChangedEvent) (string, string, SourceType, error) {
	switch evt.EntityType {
	case EntityTypeCaseTicket:
		row, err := s.q.GetCaseByID(ctx, sqlcgen.GetCaseByIDParams{ID: evt.EntityID, WorkspaceID: evt.WorkspaceID})
		if err != nil {
			return "", "", "", err
		}
		desc := ""
		if row.Description != nil {
			desc = *row.Description
		}
		raw := strings.TrimSpace(strings.Join([]string{
			"Subject: " + row.Subject,
			"Description: " + desc,
			"Priority: " + row.Priority,
			"Status: " + row.Status,
		}, "\n"))
		return row.Subject, raw, SourceTypeCase, nil
	case EntityTypeAccount:
		row, err := s.q.GetAccountByID(ctx, sqlcgen.GetAccountByIDParams{ID: evt.EntityID, WorkspaceID: evt.WorkspaceID})
		if err != nil {
			return "", "", "", err
		}
		domain := ""
		if row.Domain != nil {
			domain = *row.Domain
		}
		industry := ""
		if row.Industry != nil {
			industry = *row.Industry
		}
		raw := strings.TrimSpace(strings.Join([]string{
			"Name: " + row.Name,
			"Domain: " + domain,
			"Industry: " + industry,
		}, "\n"))
		return row.Name, raw, SourceTypeDocument, nil
	default:
		return "", "", "", fmt.Errorf("unsupported entity_type for reindex: %s", evt.EntityType)
	}
}

func (s *ReindexService) logReindexAudit(ctx context.Context, evt RecordChangedEvent, opErr error, latency time.Duration) {
	if s.audit == nil {
		return
	}

	details, _ := json.Marshal(map[string]any{
		"change_type": evt.ChangeType,
		"latency_ms":  latency.Milliseconds(),
	})

	outcome := audit.OutcomeSuccess
	if opErr != nil {
		outcome = audit.OutcomeError
	}

	_ = s.audit.Log(ctx, &audit.AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: evt.WorkspaceID,
		ActorID:     "system",
		ActorType:   audit.ActorTypeSystem,
		Action:      "knowledge.reindex",
		EntityType:  &evt.EntityType,
		EntityID:    &evt.EntityID,
		Details:     details,
		Outcome:     outcome,
		CreatedAt:   time.Now(),
	})
}
