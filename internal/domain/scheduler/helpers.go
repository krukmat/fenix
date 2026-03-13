package scheduler

import (
	"encoding/json"
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
	return time.Parse(time.RFC3339Nano, value)
}

func normalizeJSON(raw []byte, fallback []byte) json.RawMessage {
	if len(raw) == 0 {
		return append(json.RawMessage(nil), fallback...)
	}
	return append(json.RawMessage(nil), raw...)
}

