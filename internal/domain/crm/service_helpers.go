package crm

import (
	"time"
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

func mapRows[T any, R any](rows []R, mapper func(R) *T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = mapper(rows[i])
	}
	return out
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
