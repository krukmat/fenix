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
