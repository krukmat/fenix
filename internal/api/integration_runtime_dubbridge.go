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

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	dubbridge "github.com/matiasleandrokruk/fenix/internal/infra/integration/dubbridge"
)

const dubBridgeCapability = "dubbridge.review_queue.list"

var errDubBridgeWorkspaceDenied = errors.New("api: DubBridge workspace is not authorized")

// restrictedDubBridgeExecutor marks only transient read errors retryable, leaving
// the governance runtime in control of the maximum number of attempts.
type restrictedDubBridgeExecutor struct {
	wrapped *dubbridge.Executor
	workspaceID string
}

func (e restrictedDubBridgeExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	// The provider bearer is globally configured; enforce the Fenix workspace
	// boundary *before* the network call even when another workspace is
	// mistakenly granted the same tool permission.
	workspaceID, _ := ctx.Value(ctxkeys.WorkspaceID).(string)
	if workspaceID == "" || workspaceID != e.workspaceID {
		return nil, errDubBridgeWorkspaceDenied
	}
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
	const keyWorkspace = "FENIX_DUBBRIDGE_WORKSPACE_ID"
	url := strings.TrimSpace(os.Getenv(keyURL))
	token := strings.TrimSpace(os.Getenv(keyToken))
	org := strings.TrimSpace(os.Getenv(keyOrg))
	project := strings.TrimSpace(os.Getenv(keyProject))
	workspace := strings.TrimSpace(os.Getenv(keyWorkspace))
	if url == "" && token == "" && org == "" && project == "" && workspace == "" {
		return nil
	}
	if url == "" || token == "" || org == "" || project == "" || workspace == "" {
		return errors.New("api: DubBridge URL, token, org, project and Fenix workspace must be configured together")
	}
	if len(workspace) > 128 || !validDubBridgeWorkspaceID(workspace) {
		return errors.New("api: invalid DubBridge Fenix workspace binding")
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
	return registry.RegisterCapabilityWithDefinition(descriptor, restrictedDubBridgeExecutor{wrapped: executor, workspaceID: workspace}, tool.CapabilityToolDefinition{
		Description:         "Read a DubBridge project review queue under delegated organization/project authorization",
		InputSchema:         dubbridge.InputSchema,
		RequiredPermissions: []string{"tools:" + dubBridgeCapability},
	})
}

func validDubBridgeWorkspaceID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
