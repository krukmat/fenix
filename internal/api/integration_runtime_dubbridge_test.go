package api

import (
    "context"
    "encoding/json"
    "errors"
    "testing"
    "time"

    "github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
    "github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

const (
    intW1TestWorkspace = "ws-int-w1"
    intW1TestOrg = "550e8400-e29b-41d4-a716-446655440000"
    intW1TestProject = "550e8400-e29b-41d4-a716-446655440001"
)

// These tests cover configuration/registration only. They do not fake a
// DubBridge server and never count as cross-product E2E evidence.
func setDubBridgeEnv(t *testing.T, rawURL, token, org, project string) {
    t.Helper()
    t.Setenv("FENIX_DUBBRIDGE_URL", rawURL)
    t.Setenv("FENIX_DUBBRIDGE_TOKEN", token)
    t.Setenv("FENIX_DUBBRIDGE_ORG_ID", org)
    t.Setenv("FENIX_DUBBRIDGE_PROJECT_ID", project)
    t.Setenv("FENIX_DUBBRIDGE_WORKSPACE_ID", intW1TestWorkspace)
}

func TestDubBridgeIntegrationDisabledByDefault(t *testing.T) {
    setDubBridgeEnv(t, "", "", "", "")
    t.Setenv("FENIX_DUBBRIDGE_WORKSPACE_ID", "")
    registry := tool.NewToolRegistry(nil)
    if err := configureDubBridgeReadOnlyFromEnv(registry, 3*time.Second); err != nil {
        t.Fatalf("disabled integration must be a no-op: %v", err)
    }
    if _, err := registry.Get(dubBridgeCapability); !errors.Is(err, tool.ErrToolExecutorNotRegistered) {
        t.Fatalf("unexpected enabled capability: %v", err)
    }
}

func TestDubBridgeIntegrationRejectsPartialAndUnsafeConfiguration(t *testing.T) {
    cases := []struct {
        name, url, token, org, project string
    }{
        {"partial", "https://dubbridge.example", "", intW1TestOrg, intW1TestProject},
        {"unsafe transport", "http://dubbridge.example", "token", intW1TestOrg, intW1TestProject},
        {"wrong project", "https://dubbridge.example", "token", intW1TestOrg, "../../"},
        {"token injection", "https://dubbridge.example", "token\nX-Role: admin", intW1TestOrg, intW1TestProject},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            setDubBridgeEnv(t, tc.url, tc.token, tc.org, tc.project)
            registry := tool.NewToolRegistry(nil)
            if err := configureDubBridgeReadOnlyFromEnv(registry, 3*time.Second); err == nil {
                t.Fatal("unsafe or partial configuration accepted")
            }
            if _, err := registry.Get(dubBridgeCapability); !errors.Is(err, tool.ErrToolExecutorNotRegistered) {
                t.Fatalf("rejected configuration registered capability: %v", err)
            }
        })
    }
}

func TestDubBridgeIntegrationRegistrationIsExplicit(t *testing.T) {
    setDubBridgeEnv(t, "https://dubbridge.example", "test-token-not-a-real-secret", intW1TestOrg, intW1TestProject)
    registry := tool.NewToolRegistry(nil)
    if err := configureDubBridgeReadOnlyFromEnv(registry, 3*time.Second); err != nil {
        t.Fatalf("capability registration failed: %v", err)
    }
    if _, err := registry.Get(dubBridgeCapability); err != nil {
        t.Fatalf("missing registered read-only executor: %v", err)
    }
    // Duplicate registration is rejected rather than silently replacing a
    // previously authorized runtime executor.
    if err := configureDubBridgeReadOnlyFromEnv(registry, 3*time.Second); !errors.Is(err, tool.ErrToolExecutorAlreadyRegistered) {
        t.Fatalf("unexpected duplicate registration result: %v", err)
    }
}

func TestDubBridgeIntegrationRejectsMissingWorkspaceBinding(t *testing.T) {
    setDubBridgeEnv(t, "https://dubbridge.example", "test-token", intW1TestOrg, intW1TestProject)
    t.Setenv("FENIX_DUBBRIDGE_WORKSPACE_ID", "")
    registry := tool.NewToolRegistry(nil)
    if err := configureDubBridgeReadOnlyFromEnv(registry, 3*time.Second); err == nil {
        t.Fatal("missing Fenix workspace binding accepted")
    }
}

func TestDubBridgeExecutorRejectsOtherWorkspaceBeforeProviderCall(t *testing.T) {
    setDubBridgeEnv(t, "https://dubbridge.example", "test-token", intW1TestOrg, intW1TestProject)
    registry := tool.NewToolRegistry(nil)
    if err := configureDubBridgeReadOnlyFromEnv(registry, 3*time.Second); err != nil {
        t.Fatalf("register read capability: %v", err)
    }
    executor, err := registry.Get(dubBridgeCapability)
    if err != nil {
        t.Fatalf("read capability missing: %v", err)
    }
    // Missing or wrong actor workspace must fail before any outbound call.
    for _, workspace := range []string{"", "another-fenix-workspace"} {
        ctx := context.Background()
        if workspace != "" {
            ctx = ctxkeys.WithValue(ctx, ctxkeys.WorkspaceID, workspace)
        }
        _, err := executor.Execute(ctx, json.RawMessage(`{"contract_version":"1","org_id":"550e8400-e29b-41d4-a716-446655440000","project_id":"550e8400-e29b-41d4-a716-446655440001"}`))
        if !errors.Is(err, errDubBridgeWorkspaceDenied) {
            t.Fatalf("workspace %q: expected denial, got %v", workspace, err)
        }
    }
}

func TestDubBridgeIntegrationRejectsUnsafeWorkspaceID(t *testing.T) {
    for _, workspace := range []string{"hello/world", "bad workspace", "../other", "line\nbreak"} {
        t.Run(workspace, func(t *testing.T) {
            setDubBridgeEnv(t, "https://dubbridge.example", "test-token", intW1TestOrg, intW1TestProject)
            t.Setenv("FENIX_DUBBRIDGE_WORKSPACE_ID", workspace)
            if err := configureDubBridgeReadOnlyFromEnv(tool.NewToolRegistry(nil), 3*time.Second); err == nil {
                t.Fatalf("unsafe workspace binding accepted")
            }
        })
    }
}
