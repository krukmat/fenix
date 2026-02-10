// Task 1.3.8: API error definitions
package api

import "errors"

var (
	// ErrMissingWorkspaceID is returned when workspace_id is missing from context
	ErrMissingWorkspaceID = errors.New("missing workspace_id in context")
)
