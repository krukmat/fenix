package evidence

import "testing"

func TestRuntimeStreamIDFreezesWorkspaceV1Partition(t *testing.T) {
	t.Parallel()

	if StreamPartitionWorkspaceV1 != "workspace-v1" {
		t.Fatalf("partition policy = %q", StreamPartitionWorkspaceV1)
	}
	if got := RuntimeStreamID(" ws-1 "); got != "workspace/ws-1" {
		t.Fatalf("RuntimeStreamID = %q", got)
	}
	if got := RuntimeStreamID("   "); got != "" {
		t.Fatalf("empty workspace stream = %q", got)
	}
}
