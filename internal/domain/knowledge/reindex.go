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

const (
	reindexMaxAttempts = 3
	reindexBaseDelay   = 100 * time.Millisecond
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

var (
	errInvalidRecordChangeEvent = errors.New("invalid record change event")
	errUnsupportedReindexEntity = errors.New("unsupported entity_type for reindex")
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
	q       *sqlcgen.Queries
	bus     eventbus.EventBus
	ingest  *IngestService
	audit   *audit.AuditService
	sleepFn func(context.Context, time.Duration) error
	applyFn func(context.Context, RecordChangedEvent, *sqlcgen.KnowledgeItem) error
}

func NewReindexService(db *sql.DB, bus eventbus.EventBus, ingest *IngestService, auditSvc *audit.AuditService) *ReindexService {
	return &ReindexService{
		q:      sqlcgen.New(db),
		bus:    bus,
		ingest: ingest,
		audit:  auditSvc,
		sleepFn: func(ctx context.Context, delay time.Duration) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				return nil
			}
		},
	}
}

// Start subscribes to record.created|updated|deleted topics and handles events.
func (s *ReindexService) Start(ctx context.Context) {
	createdCh := s.forwardRecordEvents(ctx, TopicRecordCreated, ChangeTypeCreated)
	updatedCh := s.forwardRecordEvents(ctx, TopicRecordUpdated, ChangeTypeUpdated)
	deletedCh := s.forwardRecordEvents(ctx, TopicRecordDeleted, ChangeTypeDeleted)

	for {
		select {
		case <-ctx.Done():
			return
		case record := <-createdCh:
			_ = s.HandleRecordChange(ctx, record)
		case record := <-updatedCh:
			_ = s.HandleRecordChange(ctx, record)
		case record := <-deletedCh:
			_ = s.HandleRecordChange(ctx, record)
		}
	}
}

func (s *ReindexService) forwardRecordEvents(ctx context.Context, topic string, changeType ChangeType) <-chan RecordChangedEvent {
	sub := s.bus.Subscribe(topic)
	out := make(chan RecordChangedEvent)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-sub:
				record, ok := evt.Payload.(RecordChangedEvent)
				if !ok {
					continue
				}
				record.ChangeType = changeType
				out <- record
			}
		}
	}()
	return out
}

// QueueWorkspaceReindex publishes reindex update events for all linked entities.
func (s *ReindexService) QueueWorkspaceReindex(ctx context.Context, workspaceID string, entityType *string) (int, error) {
	const batchSize = 200
	offset := 0
	queued := 0

	for {
		items, err := s.listKnowledgeBatch(ctx, workspaceID, entityType, batchSize, offset)
		if err != nil {
			return 0, err
		}
		if len(items) == 0 {
			break
		}

		queued += s.publishBatchUpdateEvents(workspaceID, items)

		offset += len(items)
		if len(items) < batchSize {
			break
		}
	}

	return queued, nil
}

func (s *ReindexService) listKnowledgeBatch(ctx context.Context, workspaceID string, entityType *string, batchSize, offset int) ([]sqlcgen.KnowledgeItem, error) {
	if entityType != nil && *entityType != "" {
		return s.q.ListKnowledgeItemsByEntity(ctx, sqlcgen.ListKnowledgeItemsByEntityParams{
			WorkspaceID: workspaceID,
			EntityType:  entityType,
			Limit:       int64(batchSize),
			Offset:      int64(offset),
		})
	}

	return s.q.ListKnowledgeItemsByWorkspace(ctx, sqlcgen.ListKnowledgeItemsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(batchSize),
		Offset:      int64(offset),
	})
}

func (s *ReindexService) publishBatchUpdateEvents(workspaceID string, items []sqlcgen.KnowledgeItem) int {
	queued := 0
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
	return queued
}

// HandleRecordChange updates or soft-deletes linked knowledge items after CRM changes.
func (s *ReindexService) HandleRecordChange(ctx context.Context, evt RecordChangedEvent) error {
	if err := validateRecordChangeEvent(evt); err != nil {
		return err
	}

	item, err := s.getLinkedKnowledgeItem(ctx, evt)
	if err != nil {
		return err
	}
	if shouldSkipRecordChange(evt, item) {
		return nil
	}

	start := eventStartTime(evt)
	opErr, attemptCount, retriesExhausted := s.processRecordChange(ctx, evt, item)
	s.logReindexAudit(ctx, evt, opErr, time.Since(start), attemptCount, retriesExhausted)
	return opErr
}

func validateRecordChangeEvent(evt RecordChangedEvent) error {
	if evt.EntityType == "" || evt.EntityID == "" || evt.WorkspaceID == "" {
		return errInvalidRecordChangeEvent
	}
	return nil
}

func eventStartTime(evt RecordChangedEvent) time.Time {
	if evt.OccurredAt.IsZero() {
		return time.Now()
	}
	return evt.OccurredAt
}

func shouldSkipRecordChange(evt RecordChangedEvent, item *sqlcgen.KnowledgeItem) bool {
	return item == nil && evt.ChangeType != ChangeTypeCreated
}

func (s *ReindexService) processRecordChange(
	ctx context.Context,
	evt RecordChangedEvent,
	item *sqlcgen.KnowledgeItem,
) (error, int, bool) {
	for attempt := 1; attempt <= reindexMaxAttempts; attempt++ {
		opErr := s.applyChange(ctx, evt, item)
		if opErr == nil {
			return nil, attempt, false
		}
		if !shouldRetryReindex(opErr) || attempt == reindexMaxAttempts {
			return opErr, attempt, shouldRetryReindex(opErr)
		}
		if waitErr := s.sleepFn(ctx, retryDelay(attempt)); waitErr != nil {
			return waitErr, attempt, false
		}
	}
	return nil, 0, false
}

func (s *ReindexService) getLinkedKnowledgeItem(ctx context.Context, evt RecordChangedEvent) (*sqlcgen.KnowledgeItem, error) {
	entityType := evt.EntityType
	entityID := evt.EntityID

	item, err := s.q.GetKnowledgeItemByEntity(ctx, sqlcgen.GetKnowledgeItemByEntityParams{
		WorkspaceID: evt.WorkspaceID,
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}

func (s *ReindexService) applyChange(ctx context.Context, evt RecordChangedEvent, item *sqlcgen.KnowledgeItem) error {
	if s.applyFn != nil {
		return s.applyFn(ctx, evt, item)
	}
	if evt.ChangeType == ChangeTypeDeleted {
		if item == nil {
			return nil
		}
		return s.handleDelete(ctx, *item)
	}
	return s.handleUpsert(ctx, evt, item)
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

func (s *ReindexService) handleUpsert(ctx context.Context, evt RecordChangedEvent, item *sqlcgen.KnowledgeItem) error {
	title, rawContent, sourceType, buildErr := s.buildKnowledgePayloadFromEntity(ctx, evt)
	if buildErr != nil {
		return buildErr
	}

	if item != nil {
		if delErr := s.q.DeleteVecEmbeddingsByKnowledgeItem(ctx, sqlcgen.DeleteVecEmbeddingsByKnowledgeItemParams{
			KnowledgeItemID: item.ID,
			WorkspaceID:     item.WorkspaceID,
		}); delErr != nil {
			return delErr
		}
	}

	_, ingestErr := s.ingest.Ingest(ctx, CreateKnowledgeItemInput{
		WorkspaceID: evt.WorkspaceID,
		SourceType:  sourceType,
		Title:       title,
		RawContent:  rawContent,
		EntityType:  &evt.EntityType,
		EntityID:    &evt.EntityID,
	})
	return ingestErr // Task 3.8: fixed unused variable (was returning undefined err)
}

func (s *ReindexService) buildKnowledgePayloadFromEntity(ctx context.Context, evt RecordChangedEvent) (string, string, SourceType, error) {
	switch evt.EntityType {
	case EntityTypeCaseTicket:
		return s.buildCasePayload(ctx, evt)
	case EntityTypeAccount:
		return s.buildAccountPayload(ctx, evt)
	default:
		return "", "", "", fmt.Errorf("%w: %s", errUnsupportedReindexEntity, evt.EntityType)
	}
}

func (s *ReindexService) buildCasePayload(ctx context.Context, evt RecordChangedEvent) (string, string, SourceType, error) {
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
}

func (s *ReindexService) buildAccountPayload(ctx context.Context, evt RecordChangedEvent) (string, string, SourceType, error) {
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
}

func (s *ReindexService) logReindexAudit(ctx context.Context, evt RecordChangedEvent, opErr error, latency time.Duration, attemptCount int, retriesExhausted bool) {
	if s.audit == nil {
		return
	}

	details, _ := json.Marshal(map[string]any{
		"change_type":       evt.ChangeType,
		"latency_ms":        latency.Milliseconds(),
		"attempt_count":     attemptCount,
		"retries_exhausted": retriesExhausted,
		"entity_type":       evt.EntityType,
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

func shouldRetryReindex(err error) bool {
	return err != nil &&
		!errors.Is(err, errInvalidRecordChangeEvent) &&
		!errors.Is(err, errUnsupportedReindexEntity)
}

func retryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return reindexBaseDelay
	}
	return reindexBaseDelay * time.Duration(1<<(attempt-1))
}
