// Task 1.3 TD-1 fix + Task 1.6: Shared context keys for API layer.
// Extracted to a leaf package to avoid import cycles between api and api/handlers.
package ctxkeys

import "context"

// Key is the unexported named type for all API context keys.
// Using a named type avoids collisions with string keys from other packages
// at runtime (context.Value compares both type and value).
type Key string

const (
	// WorkspaceID is the context key for the active workspace.
	// Injected by AuthMiddleware (Task 1.6) from JWT claims, read by all handlers.
	WorkspaceID Key = "workspace_id"

	// UserID is the context key for the authenticated user.
	// Injected by AuthMiddleware (Task 1.6) from JWT claims, read by handlers that need actor identity.
	UserID Key = "user_id"
)

// WithValue adds a ctxkeys.Key value to the context.
// Task 1.6.10: Helper used by AuthMiddleware to inject claims using typed keys.
func WithValue(ctx context.Context, key Key, value string) context.Context {
	return context.WithValue(ctx, key, value)
}
