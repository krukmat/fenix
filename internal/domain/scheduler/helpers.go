package scheduler

import (
	"encoding/json"
	"fmt"
	"time"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse RFC3339 time: %w", err)
	}
	return parsed, nil
}

func normalizeJSON(raw []byte, fallback []byte) json.RawMessage {
	if len(raw) == 0 {
		return append(json.RawMessage(nil), fallback...)
	}
	return append(json.RawMessage(nil), raw...)
}
