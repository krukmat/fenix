package tool

import (
	"context"
	"encoding/json"
)

// ToolExecutor defines the runtime contract for executable tools.
// Task 3.3: foundation contract used by the tool registry.
type ToolExecutor interface {
	Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error)
}
