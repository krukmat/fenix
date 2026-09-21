package evidence

import "strings"

const StreamPartitionWorkspaceV1 = "workspace-v1"

// RuntimeStreamID returns the frozen W5 workspace-v1 partition for durable evidence.
func RuntimeStreamID(workspaceID string) string {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return ""
	}
	return "workspace/" + workspaceID
}
