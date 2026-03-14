package crm

import (
	"context"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func parseRFC3339Time(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

func parseOptionalRFC3339(value *string) *time.Time {
	if value == nil {
		return nil
	}
	t := parseRFC3339Time(*value)
	return &t
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func mapRows[T any, R any](rows []R, mapper func(R) *T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = mapper(rows[i])
	}
	return out
}

// softDeleteWithSideEffects executes the soft-delete DB call then records the
// timeline event and audit log. It is the shared skeleton for all CRM Delete methods
// that follow the pattern: soft-delete → timeline → audit.
func softDeleteWithSideEffects(
	ctx context.Context,
	q sqlcgen.Querier,
	auditSvc auditLogger,
	workspaceID, entityType, entityID, ownerID string,
	deleteAction string,
	softDelete func() error,
) error {
	if err := softDelete(); err != nil {
		return fmt.Errorf("soft delete %s: %w", entityType, err)
	}
	if timelineErr := createTimelineEvent(ctx, q, workspaceID, entityType, entityID, ownerID, timelineActionDeleted); timelineErr != nil {
		return fmt.Errorf("delete %s timeline: %w", entityType, timelineErr)
	}
	logCRMAudit(ctx, auditSvc, workspaceID, ownerID, deleteAction, entityType, entityID)
	return nil
}

// listFilteredOrPaged implements the CRM "filtered shortcut vs DB-paginated" list pattern.
// When useFiltered is true it loads all matching items in memory and applies in-process pagination;
// otherwise it delegates counting and paging to the database.
func listFilteredOrPaged[T any](
	useFiltered bool,
	loadFiltered func() ([]*T, error),
	paginate func([]*T, int, int) []*T,
	offset, limit int,
	countDB func() (int64, error),
	listDB func() ([]*T, error),
) ([]*T, int, error) {
	if useFiltered {
		filtered, err := loadFiltered()
		if err != nil {
			return nil, 0, err
		}
		return paginate(filtered, offset, limit), len(filtered), nil
	}
	total, err := countDB()
	if err != nil {
		return nil, 0, err
	}
	items, err := listDB()
	if err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

func listInputFilteredOrPaged[In any, T any](
	input *In,
	getSort func(*In) string,
	setSort func(*In, string),
	defaultSort string,
	useFiltered func(In) bool,
	loadFiltered func() ([]*T, error),
	paginate func([]*T, int, int) []*T,
	offset, limit int,
	countDB func() (int64, error),
	listDB func() ([]*T, error),
) ([]*T, int, error) {
	setSort(input, firstNonEmpty(getSort(input), defaultSort))
	return listFilteredOrPaged(useFiltered(*input), loadFiltered, paginate, offset, limit, countDB, listDB)
}

// listWorkspacePage centralizes the common CRM "count + paginated list + map"
// flow used by simple workspace-scoped entities.
func listWorkspacePage[T any, R any](
	ctx context.Context,
	workspaceID string,
	entityName string,
	limit int,
	offset int,
	count func(context.Context, string) (int64, error),
	list func(context.Context, string, int64, int64) ([]R, error),
	mapper func(R) *T,
) ([]*T, int, error) {
	total, err := count(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", entityName, err)
	}

	rows, err := list(ctx, workspaceID, int64(limit), int64(offset))
	if err != nil {
		return nil, 0, fmt.Errorf("list %s: %w", entityName, err)
	}

	return mapRows(rows, mapper), int(total), nil
}
