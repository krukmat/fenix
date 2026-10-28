// INT-W1: opt-in read-only DubBridge capability registration.
// This file is intended for Fenix's internal/api package; the local staging
// module cannot compile the whole Fenix runtime without its original deps.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	dubbridge "github.com/matiasleandrokruk/fenix/internal/infra/integration/dubbridge"
)

const dubBridgeCapability = "dubbridge.review_queue.list"

// restrictedDubBridgeExecutor marks only transient read errors retryable, leaving
// the governance runtime in control of the maximum number of attempts.
type restrictedDubBridgeExecutor struct{ wrapped *dubbridge.Executor }

func (e restrictedDubBridgeExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	out, err := e.wrapped.Execute(ctx, params)
	if err == nil {
		return out, nil
	}
	if dubbridge.RequiresRetry(err) {
		return nil, tool.NewRetryableCapabilityError(err)
	}
	return nil, err
}

func configureDubBridgeReadOnlyFromEnv(registry *tool.ToolRegistry, timeout time.Duration) error {
	const keyURL = "FENIX_DUBBRIDGE_URL"
	const keyToken = "FENIX_DUBBRIDGE_TOKEN"
	const keyOrg = "FENIX_DUBBRIDGE_ORG_ID"
	const keyProject = "FENIX_DUBBRIDGE_PROJECT_ID"
	url := strings.TrimSpace(os.Getenv(keyURL))
	token := strings.TrimSpace(os.Getenv(keyToken))
	org := strings.TrimSpace(os.Getenv(keyOrg))
	project := strings.TrimSpace(os.Getenv(keyProject))
	if url == "" && token == "" && org == "" && project == "" {
		return nil
	}
	if url == "" || token == "" || org == "" || project == "" {
		return errors.New("api: DubBridge URL, token, org and project must be configured together")
	}
	executor, err := dubbridge.NewExecutor(dubbridge.Config{
		BaseURL: url, BearerToken: token, AllowedOrgID: org, AllowedProjectID: project, Timeout: timeout,
	})
	if err != nil {
		return fmt.Errorf("api: invalid DubBridge read-only configuration: %w", err)
	}
	descriptor := tool.CapabilityDescriptor{
		Name: dubBridgeCapability, Version: "1", Operation: "list",
		SideEffectClass:  tool.SideEffectRead,
		ApprovalRequired: false,
		RetryPolicy:      tool.CapabilityRetryPolicy{MaxAttempts: 2},
		IdempotencyMode:  tool.CapabilityIdempotencyNone,
	}
	return registry.RegisterCapabilityWithDefinition(descriptor, restrictedDubBridgeExecutor{wrapped: executor}, tool.CapabilityToolDefinition{
		Description:         "Read a DubBridge project review queue under delegated organization/project authorization",
		InputSchema:         dubbridge.InputSchema,
		RequiredPermissions: []string{"tools:" + dubBridgeCapability},
	})
}
