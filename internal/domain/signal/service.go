package signal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrInvalidSignalInput   = errors.New("invalid signal input")
	ErrSignalDismissInvalid = errors.New("invalid signal dismiss operation")
)

const (
	TopicSignalCreated   = "signal.created"
	TopicSignalDismissed = "signal.dismissed"

	entityTypeContact = "contact"
	entityTypeLead    = "lead"
	entityTypeDeal    = "deal"
	entityTypeCase    = "case"
)

type CreatedEventPayload struct {
	SignalID     string    `json:"signalId"`
	WorkspaceID  string    `json:"workspaceId"`
	EntityType   string    `json:"entityType"`
	EntityID     string    `json:"entityId"`
	SignalType   string    `json:"signalType"`
	Confidence   float64   `json:"confidence"`
	SourceType   string    `json:"sourceType"`
	SourceID     string    `json:"sourceId"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DismissedEventPayload struct {
	SignalID     string     `json:"signalId"`
	WorkspaceID  string     `json:"workspaceId"`
	EntityType   string     `json:"entityType"`
	EntityID     string     `json:"entityId"`
	SignalType   string     `json:"signalType"`
	Status       Status     `json:"status"`
	DismissedBy  string     `json:"dismissedBy"`
	DismissedAt  *time.Time `json:"dismissedAt,omitempty"`
}

type CreateSignalInput struct {
	WorkspaceID string
	EntityType  string
	EntityID    string
	SignalType  string
	Confidence  float64
	EvidenceIDs []string
	SourceType  string
	SourceID    string
	Metadata    map[string]any
	ExpiresAt   *time.Time
}

type Service struct {
	db   *sql.DB
	repo *Repository
	bus  eventbus.EventBus
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db:   db,
		repo: NewRepository(db),
	}
}

func NewServiceWithRepository(db *sql.DB, repo *Repository) *Service {
	return &Service{db: db, repo: repo}
}

func NewServiceWithBus(db *sql.DB, repo *Repository, bus eventbus.EventBus) *Service {
	return &Service{db: db, repo: repo, bus: bus}
}

func (s *Service) Create(ctx context.Context, input CreateSignalInput) (*Signal, error) {
	if err := validateCreateSignalInput(input); err != nil {
		return nil, err
	}
	if err := ensureSignalEntityExists(ctx, s.db, input.WorkspaceID, input.EntityType, input.EntityID); err != nil {
		return nil, invalidSignalInput("entity reference is invalid", err)
	}

	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		return nil, invalidSignalInput("metadata must be valid json", err)
	}

	created, err := s.repo.Create(ctx, CreateInput{
		ID:          uuid.NewV7().String(),
		WorkspaceID: input.WorkspaceID,
		EntityType:  normalizeEntityType(input.EntityType),
		EntityID:    strings.TrimSpace(input.EntityID),
		SignalType:  strings.TrimSpace(input.SignalType),
		Confidence:  input.Confidence,
		EvidenceIDs: trimNonEmptyStrings(input.EvidenceIDs),
		SourceType:  strings.TrimSpace(input.SourceType),
		SourceID:    strings.TrimSpace(input.SourceID),
		Metadata:    metadata,
		Status:      StatusActive,
		ExpiresAt:   input.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	s.publishCreated(created)
	return created, nil
}

func (s *Service) List(ctx context.Context, workspaceID string, filters Filters) ([]*Signal, error) {
	return s.repo.List(ctx, workspaceID, filters)
}

func (s *Service) GetByEntity(ctx context.Context, workspaceID, entityType, entityID string) ([]*Signal, error) {
	return s.repo.GetByEntity(ctx, workspaceID, normalizeEntityType(entityType), strings.TrimSpace(entityID))
}

func (s *Service) Dismiss(ctx context.Context, workspaceID, signalID, actorID string) error {
	if strings.TrimSpace(actorID) == "" {
		return invalidSignalInput("actor_id is required", nil)
	}

	existing, err := s.repo.GetByID(ctx, workspaceID, signalID)
	if err != nil {
		return err
	}
	if existing.Status != StatusActive {
		return ErrSignalDismissInvalid
	}

	dismissed, err := s.repo.Dismiss(ctx, workspaceID, signalID, actorID)
	if err != nil {
		return err
	}
	s.publishDismissed(dismissed)
	return err
}

func (s *Service) publishCreated(created *Signal) {
	if s.bus == nil || created == nil {
		return
	}
	s.bus.Publish(TopicSignalCreated, CreatedEventPayload{
		SignalID:    created.ID,
		WorkspaceID: created.WorkspaceID,
		EntityType:  created.EntityType,
		EntityID:    created.EntityID,
		SignalType:  created.SignalType,
		Confidence:  created.Confidence,
		SourceType:  created.SourceType,
		SourceID:    created.SourceID,
		Status:      created.Status,
		CreatedAt:   created.CreatedAt,
	})
}

func (s *Service) publishDismissed(dismissed *Signal) {
	if s.bus == nil || dismissed == nil || dismissed.DismissedBy == nil {
		return
	}
	s.bus.Publish(TopicSignalDismissed, DismissedEventPayload{
		SignalID:    dismissed.ID,
		WorkspaceID: dismissed.WorkspaceID,
		EntityType:  dismissed.EntityType,
		EntityID:    dismissed.EntityID,
		SignalType:  dismissed.SignalType,
		Status:      dismissed.Status,
		DismissedBy: *dismissed.DismissedBy,
		DismissedAt: dismissed.DismissedAt,
	})
}

func validateCreateSignalInput(input CreateSignalInput) error {
	if err := validateSignalIDFields(input); err != nil {
		return err
	}
	return validateSignalConstraints(input)
}

func validateSignalIDFields(input CreateSignalInput) error {
	if strings.TrimSpace(input.WorkspaceID) == "" {
		return invalidSignalInput("workspace_id is required", nil)
	}
	if strings.TrimSpace(input.EntityType) == "" {
		return invalidSignalInput("entity_type is required", nil)
	}
	if strings.TrimSpace(input.EntityID) == "" {
		return invalidSignalInput("entity_id is required", nil)
	}
	if strings.TrimSpace(input.SignalType) == "" {
		return invalidSignalInput("signal_type is required", nil)
	}
	if strings.TrimSpace(input.SourceType) == "" {
		return invalidSignalInput("source_type is required", nil)
	}
	if strings.TrimSpace(input.SourceID) == "" {
		return invalidSignalInput("source_id is required", nil)
	}
	return nil
}

func validateSignalConstraints(input CreateSignalInput) error {
	if len(trimNonEmptyStrings(input.EvidenceIDs)) == 0 {
		return invalidSignalInput("evidence_ids requires at least one value", nil)
	}
	if input.Confidence < 0.0 || input.Confidence > 1.0 {
		return invalidSignalInput("confidence must be in range [0.0, 1.0]", nil)
	}
	switch normalizeEntityType(input.EntityType) {
	case entityTypeContact, entityTypeLead, entityTypeDeal, entityTypeCase:
		return nil
	default:
		return invalidSignalInput("entity_type is invalid", nil)
	}
}

func ensureSignalEntityExists(ctx context.Context, db *sql.DB, workspaceID, entityType, entityID string) error {
	var query string
	switch normalizeEntityType(entityType) {
	case entityTypeContact:
		query = `SELECT 1 FROM contact WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1`
	case entityTypeLead:
		query = `SELECT 1 FROM lead WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1`
	case entityTypeDeal:
		query = `SELECT 1 FROM deal WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1`
	case entityTypeCase:
		query = `SELECT 1 FROM case_ticket WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1`
	default:
		return errors.New("unsupported entity type")
	}

	var exists int
	if err := db.QueryRowContext(ctx, query, strings.TrimSpace(entityID), workspaceID).Scan(&exists); err != nil {
		return err
	}
	return nil
}

func normalizeEntityType(entityType string) string {
	return strings.TrimSpace(strings.ToLower(entityType))
}

func trimNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func invalidSignalInput(reason string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalidSignalInput, reason)
	}
	return fmt.Errorf("%w: %s: %w", ErrInvalidSignalInput, reason, err)
}
