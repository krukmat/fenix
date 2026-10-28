package dubbridge

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestRealDubBridge only uses an actual configured DubBridge API and JWT.
// It has no mock HTTP server, stub issuer, or synthetic response.
func TestRealDubBridge(t *testing.T) {
	env := []string{"DUBBRIDGE_URL", "DUBBRIDGE_TOKEN", "DUBBRIDGE_ORG_ID", "DUBBRIDGE_PROJECT_ID", "FENIX_TRACE_ID", "FENIX_EXECUTION_ID"}
	for _, name := range env {
		if os.Getenv(name) == "" {
			if os.Getenv("INTW1_REQUIRE_REAL") == "1" {
				t.Fatalf("required real DubBridge integration not configured: %s", name)
			}
			t.Skip("real E2E not configured: " + name)
		}
	}
	c, err := NewClient(Config{BaseURL: os.Getenv("DUBBRIDGE_URL"), BearerToken: os.Getenv("DUBBRIDGE_TOKEN"), AllowedOrgID: os.Getenv("DUBBRIDGE_ORG_ID"), AllowedProjectID: os.Getenv("DUBBRIDGE_PROJECT_ID")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	queue, err := c.ListReviewQueue(ctx, os.Getenv("DUBBRIDGE_ORG_ID"), os.Getenv("DUBBRIDGE_PROJECT_ID"), os.Getenv("FENIX_TRACE_ID"), os.Getenv("FENIX_EXECUTION_ID"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Real DubBridge GET passed: task_count=%d", len(queue.Tasks))
}
