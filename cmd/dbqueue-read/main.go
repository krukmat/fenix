// dbqueue-read calls the real DubBridge review queue with an externally issued
// DubBridge credential. It does not provision, mint or simulate credentials.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/integration/dubbridge"
)

func main() {
	org, project := os.Getenv("DUBBRIDGE_ORG_ID"), os.Getenv("DUBBRIDGE_PROJECT_ID")
	client, err := dubbridge.NewClient(dubbridge.Config{
		BaseURL: os.Getenv("DUBBRIDGE_URL"), BearerToken: os.Getenv("DUBBRIDGE_TOKEN"),
		AllowedOrgID: org, AllowedProjectID: project, Timeout: 15 * time.Second,
	})
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := client.ListReviewQueue(ctx, org, project, os.Getenv("FENIX_TRACE_ID"), os.Getenv("FENIX_EXECUTION_ID"))
	if err != nil {
		fail(err)
	}
	out := struct {
		OrgID     string         `json:"org_id"`
		ProjectID string         `json:"project_id"`
		Count     int            `json:"count"`
		States    map[string]int `json:"states"`
	}{result.OrgID, result.ProjectID, len(result.Tasks), map[string]int{}}
	for _, task := range result.Tasks {
		out.States[task.State]++
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fail(err)
	}
}
func fail(err error) {
	var provider *dubbridge.ProviderError
	if errors.As(err, &provider) {
		fmt.Fprintf(os.Stderr, "dubbridge request failed: HTTP=%d retryable=%t\n", provider.Status, provider.Retryable)
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
