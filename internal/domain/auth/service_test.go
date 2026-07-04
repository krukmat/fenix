// Task 1.6.7: TDD tests for AuthService (Register and Login business logic)
// Tests run against in-memory SQLite with real migrations.
// Traces: FR-060
package auth_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/auth"
)

// TestMain sets JWT_SECRET before any test runs.
// Task 1.6.14: pkgauth.GenerateJWT panics if JWT_SECRET is not set.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!") //nolint:errcheck
	os.Exit(m.Run())
}

// ===== REGISTER TESTS =====

// TestAuthService_Register_Success verifies that registering creates workspace, user, and returns JWT.
func TestAuthService_Register_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, err := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "alice@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Alice",
		WorkspaceName: "Acme Corp",
	})

	if err != nil {
		t.Fatalf("Register() error = %v; want nil", err)
	}

	if result.Token == "" {
		t.Error("Register() Token is empty; want JWT token")
	}

	if result.UserID == "" {
		t.Error("Register() UserID is empty; want non-empty ID")
	}

	if result.WorkspaceID == "" {
		t.Error("Register() WorkspaceID is empty; want non-empty ID")
	}
}

// TestAuthService_Register_TokenIsValid verifies that the returned token has valid claims.
func TestAuthService_Register_TokenIsValid(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "bob@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Bob",
		WorkspaceName: "Acme Corp",
	})

	// Parse and verify JWT claims
	claims, err := auth.ParseJWT(result.Token)
	if err != nil {
		t.Fatalf("Returned token is not a valid JWT: %v", err)
	}

	if claims.UserID != result.UserID {
		t.Errorf("JWT UserID = %q; want %q", claims.UserID, result.UserID)
	}

	if claims.WorkspaceID != result.WorkspaceID {
		t.Errorf("JWT WorkspaceID = %q; want %q", claims.WorkspaceID, result.WorkspaceID)
	}
}

// TestAuthService_Register_UserPersistedInDB verifies the user is stored in the database.
func TestAuthService_Register_UserPersistedInDB(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "carol@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Carol",
		WorkspaceName: "Acme Corp",
	})

	// Verify user exists in DB with correct fields
	var email, displayName, status string
	var passwordHash sql.NullString
	err := db.QueryRow(`
		SELECT email, display_name, status, password_hash
		FROM user_account WHERE id = ?
	`, result.UserID).Scan(&email, &displayName, &status, &passwordHash)

	if err != nil {
		t.Fatalf("User not found in DB after Register: %v", err)
	}

	if email != "carol@acme.com" {
		t.Errorf("email = %q; want %q", email, "carol@acme.com")
	}

	if displayName != "Carol" {
		t.Errorf("display_name = %q; want %q", displayName, "Carol")
	}

	if status != "active" {
		t.Errorf("status = %q; want %q", status, "active")
	}

	// Password should be stored as a bcrypt hash, not plaintext
	if !passwordHash.Valid || passwordHash.String == "" {
		t.Error("password_hash is NULL or empty; want bcrypt hash")
	}

	if passwordHash.String == "SecurePass123!" {
		t.Error("password_hash should not equal plaintext password")
	}
}

// TestAuthService_Register_WorkspaceCreated verifies the workspace is created.
func TestAuthService_Register_WorkspaceCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "dave@example.com",
		Password:      "SecurePass123!",
		DisplayName:   "Dave",
		WorkspaceName: "Example LLC",
	})

	var name string
	err := db.QueryRow(`SELECT name FROM workspace WHERE id = ?`, result.WorkspaceID).Scan(&name)
	if err != nil {
		t.Fatalf("Workspace not found in DB after Register: %v", err)
	}

	if name != "Example LLC" {
		t.Errorf("workspace.name = %q; want %q", name, "Example LLC")
	}
}

// TestAuthService_Register_BootstrapsWorkspaceDefaults verifies registration creates
// the first-user role assignment and deterministic default pipelines.
func TestAuthService_Register_BootstrapsWorkspaceDefaults(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, err := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "defaults@example.com",
		Password:      "SecurePass123!",
		DisplayName:   "Default Owner",
		WorkspaceName: "Default Workspace",
	})
	if err != nil {
		t.Fatalf("Register() error = %v; want nil", err)
	}

	var roleID, roleName, roleDescription, permissions string
	err = db.QueryRow(`
		SELECT r.id, r.name, r.description, r.permissions
		FROM role r
		JOIN user_role ur ON ur.role_id = r.id
		WHERE r.workspace_id = ? AND ur.user_id = ?
	`, result.WorkspaceID, result.UserID).Scan(&roleID, &roleName, &roleDescription, &permissions)
	if err != nil {
		t.Fatalf("default role assignment not found: %v", err)
	}
	if roleID == "" {
		t.Fatal("default role id is empty")
	}
	if roleName != "workspace_owner" {
		t.Errorf("role name = %q; want workspace_owner", roleName)
	}
	if roleDescription != "Default first-user role created during workspace registration bootstrap." {
		t.Errorf("role description = %q; want default bootstrap description", roleDescription)
	}

	var parsedPermissions map[string][]string
	if err := json.Unmarshal([]byte(permissions), &parsedPermissions); err != nil {
		t.Fatalf("permissions JSON is invalid: %v", err)
	}
	assertStringSlice(t, parsedPermissions["records"], []string{"read_all"})
	assertStringSlice(t, parsedPermissions["agents"], []string{"execute"})
	// Regression guard for EXTVAL-O7-SIGNALS-403-001: without "global":["admin"],
	// policy.roleAllowsAction denies every resource="api" action (signals,
	// blackboard, eval, prompt, tool, workflow) for a freshly registered owner.
	assertStringSlice(t, parsedPermissions["global"], []string{"admin"})
	assertStringSlice(t, parsedPermissions["tools"], []string{
		"create_task",
		"update_case",
		"update_deal",
		"send_reply",
		"get_lead",
		"get_account",
		"get_deal",
		"create_knowledge_item",
		"update_knowledge_item",
		"query_metrics",
	})

	assertDefaultPipeline(t, db, result.WorkspaceID, "deal", "Sales", "Discovery")
	assertDefaultPipeline(t, db, result.WorkspaceID, "case", "Support", "Open")

	assertDefaultSupportAgent(t, db, result.WorkspaceID, parsedPermissions["tools"])
}

// TestAuthService_Register_SupportAgentToolsSubsetOfOwner asserts the bootstrap
// support-agent's allowed_tools never exceeds the workspace_owner grant, so the
// agent can never act beyond the operator who owns the workspace.
func TestAuthService_Register_SupportAgentToolsSubsetOfOwner(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	result, err := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "subset@example.com",
		Password:      "SecurePass123!",
		DisplayName:   "Subset Owner",
		WorkspaceName: "Subset Workspace",
	})
	if err != nil {
		t.Fatalf("Register() error = %v; want nil", err)
	}

	var permissions string
	if err := db.QueryRow(`
		SELECT r.permissions
		FROM role r
		JOIN user_role ur ON ur.role_id = r.id
		WHERE r.workspace_id = ? AND ur.user_id = ?
	`, result.WorkspaceID, result.UserID).Scan(&permissions); err != nil {
		t.Fatalf("owner role not found: %v", err)
	}
	var parsed map[string][]string
	if err := json.Unmarshal([]byte(permissions), &parsed); err != nil {
		t.Fatalf("permissions JSON invalid: %v", err)
	}
	ownerTools := make(map[string]struct{}, len(parsed["tools"]))
	for _, tool := range parsed["tools"] {
		ownerTools[tool] = struct{}{}
	}

	agentTools := supportAgentAllowedTools(t, db, result.WorkspaceID)
	if len(agentTools) == 0 {
		t.Fatal("support agent allowed_tools is empty; want non-empty subset")
	}
	for _, tool := range agentTools {
		if _, ok := ownerTools[tool]; !ok {
			t.Errorf("support agent tool %q not in workspace_owner grant %v", tool, parsed["tools"])
		}
	}
}

// TestAuthService_Register_DuplicateEmail verifies that duplicate email returns error.
func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	input := domainauth.RegisterInput{
		Email:         "dup@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Dup",
		WorkspaceName: "Acme Corp",
	}

	// First registration should succeed
	_, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("First Register() error = %v; want nil", err)
	}
	before := registrationBootstrapCounts(t, db)

	// Second registration with same email should fail
	_, err = svc.Register(context.Background(), input)
	if err == nil {
		t.Error("Register() with duplicate email should return error; got nil")
	}
	if !errors.Is(err, domainauth.ErrEmailAlreadyExists) {
		t.Errorf("Register() error = %v; want ErrEmailAlreadyExists", err)
	}

	after := registrationBootstrapCounts(t, db)
	if before != after {
		t.Fatalf("duplicate registration committed extra bootstrap rows: before=%+v after=%+v", before, after)
	}
}

// ===== LOGIN TESTS =====

// TestAuthService_Login_Success verifies successful login returns JWT.
func TestAuthService_Login_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	// Register first
	regResult, _ := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "eve@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Eve",
		WorkspaceName: "Acme Corp",
	})

	// Login
	loginResult, err := svc.Login(context.Background(), domainauth.LoginInput{
		Email:    "eve@acme.com",
		Password: "SecurePass123!",
	})

	if err != nil {
		t.Fatalf("Login() error = %v; want nil", err)
	}

	if loginResult.Token == "" {
		t.Error("Login() Token is empty; want JWT token")
	}

	if loginResult.UserID != regResult.UserID {
		t.Errorf("Login() UserID = %q; want %q", loginResult.UserID, regResult.UserID)
	}

	if loginResult.WorkspaceID != regResult.WorkspaceID {
		t.Errorf("Login() WorkspaceID = %q; want %q", loginResult.WorkspaceID, regResult.WorkspaceID)
	}
}

// TestAuthService_Login_TokenIsValid verifies the login token has valid JWT claims.
func TestAuthService_Login_TokenIsValid(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	svc.Register(context.Background(), domainauth.RegisterInput{ //nolint:errcheck
		Email: "frank@acme.com", Password: "SecurePass123!", DisplayName: "Frank", WorkspaceName: "Acme Corp",
	})

	result, _ := svc.Login(context.Background(), domainauth.LoginInput{
		Email: "frank@acme.com", Password: "SecurePass123!",
	})

	claims, err := auth.ParseJWT(result.Token)
	if err != nil {
		t.Fatalf("Login() token is not valid JWT: %v", err)
	}

	if claims.UserID == "" || claims.WorkspaceID == "" {
		t.Error("Login() JWT claims missing UserID or WorkspaceID")
	}
}

// TestAuthService_Login_WrongPassword verifies that wrong password returns error.
func TestAuthService_Login_WrongPassword(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	svc.Register(context.Background(), domainauth.RegisterInput{ //nolint:errcheck
		Email: "grace@acme.com", Password: "SecurePass123!", DisplayName: "Grace", WorkspaceName: "Acme Corp",
	})

	_, err := svc.Login(context.Background(), domainauth.LoginInput{
		Email:    "grace@acme.com",
		Password: "WrongPassword!",
	})

	if err == nil {
		t.Error("Login() with wrong password should return error; got nil")
	}
}

// TestAuthService_Login_NonExistentEmail verifies that unknown email returns error.
func TestAuthService_Login_NonExistentEmail(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	_, err := svc.Login(context.Background(), domainauth.LoginInput{
		Email:    "nobody@acme.com",
		Password: "SomePassword!",
	})

	if err == nil {
		t.Error("Login() with non-existent email should return error; got nil")
	}
}

// TestAuthService_Login_ErrorMessageGeneric verifies error message doesn't reveal whether email exists.
func TestAuthService_Login_ErrorMessageGeneric(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	svc := domainauth.NewAuthService(db)

	svc.Register(context.Background(), domainauth.RegisterInput{ //nolint:errcheck
		Email: "hank@acme.com", Password: "SecurePass123!", DisplayName: "Hank", WorkspaceName: "Acme Corp",
	})

	// Wrong password — should say "invalid credentials", not "password incorrect"
	_, errWrongPw := svc.Login(context.Background(), domainauth.LoginInput{
		Email: "hank@acme.com", Password: "WrongPassword!",
	})

	// Non-existent email — should give the same generic error
	_, errNoUser := svc.Login(context.Background(), domainauth.LoginInput{
		Email: "nosuchuser@acme.com", Password: "SecurePass123!",
	})

	// Both should return the same error type (ErrInvalidCredentials)
	if errWrongPw == nil || errNoUser == nil {
		t.Fatal("Both login attempts should fail")
	}

	if errWrongPw.Error() != errNoUser.Error() {
		t.Errorf("Error messages should be identical for security: got %q vs %q",
			errWrongPw.Error(), errNoUser.Error())
	}
}

func TestAuthService_NewAuthServiceWithAudit_Works(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	auditSvc := audit.NewAuditService(db)
	svc := domainauth.NewAuthServiceWithAudit(db, auditSvc)

	result, err := svc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "audit-constructor@acme.com",
		Password:      "SecurePass123!",
		DisplayName:   "Audit Constructor",
		WorkspaceName: "Audit Workspace",
	})
	if err != nil {
		t.Fatalf("Register() error = %v; want nil", err)
	}
	if result.Token == "" || result.UserID == "" || result.WorkspaceID == "" {
		t.Fatalf("expected auth result fields to be populated, got %#v", result)
	}
}

// ===== TEST HELPERS =====

// mustOpenDB opens an in-memory SQLite DB with all migrations applied.
func mustOpenDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("sqlite.NewDB error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp error = %v", err)
	}

	return db
}

type bootstrapCounts struct {
	workspaces       int
	users            int
	roles            int
	userRoles        int
	pipelines        int
	stages           int
	agentDefinitions int
}

func registrationBootstrapCounts(t *testing.T, db *sql.DB) bootstrapCounts {
	t.Helper()
	return bootstrapCounts{
		workspaces:       countRows(t, db, "workspace"),
		users:            countRows(t, db, "user_account"),
		roles:            countRows(t, db, "role"),
		userRoles:        countRows(t, db, "user_role"),
		pipelines:        countRows(t, db, "pipeline"),
		stages:           countRows(t, db, "pipeline_stage"),
		agentDefinitions: countRows(t, db, "agent_definition"),
	}
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(countQueryForTable(t, table)).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func countQueryForTable(t *testing.T, table string) string {
	t.Helper()
	switch table {
	case "workspace":
		return `SELECT COUNT(*) FROM workspace`
	case "user_account":
		return `SELECT COUNT(*) FROM user_account`
	case "role":
		return `SELECT COUNT(*) FROM role`
	case "user_role":
		return `SELECT COUNT(*) FROM user_role`
	case "pipeline":
		return `SELECT COUNT(*) FROM pipeline`
	case "pipeline_stage":
		return `SELECT COUNT(*) FROM pipeline_stage`
	case "agent_definition":
		return `SELECT COUNT(*) FROM agent_definition`
	default:
		t.Fatalf("unsupported count table %q", table)
		return ""
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length = %d; want %d. got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q; want %q. got=%v", i, got[i], want[i], got)
		}
	}
}

func assertDefaultPipeline(t *testing.T, db *sql.DB, workspaceID, entityType, pipelineName, stageName string) {
	t.Helper()

	var pipelineID, gotPipelineName string
	err := db.QueryRow(`
		SELECT id, name
		FROM pipeline
		WHERE workspace_id = ? AND entity_type = ?
	`, workspaceID, entityType).Scan(&pipelineID, &gotPipelineName)
	if err != nil {
		t.Fatalf("default %s pipeline not found: %v", entityType, err)
	}
	if gotPipelineName != pipelineName {
		t.Fatalf("default %s pipeline name = %q; want %q", entityType, gotPipelineName, pipelineName)
	}

	var gotStageName string
	var gotPosition int
	err = db.QueryRow(`
		SELECT name, position
		FROM pipeline_stage
		WHERE pipeline_id = ?
	`, pipelineID).Scan(&gotStageName, &gotPosition)
	if err != nil {
		t.Fatalf("default %s stage not found: %v", entityType, err)
	}
	if gotStageName != stageName {
		t.Fatalf("default %s stage name = %q; want %q", entityType, gotStageName, stageName)
	}
	if gotPosition != 1 {
		t.Fatalf("default %s stage position = %d; want 1", entityType, gotPosition)
	}
}

// supportAgentAllowedTools reads the bootstrap support-agent's allowed_tools.
func supportAgentAllowedTools(t *testing.T, db *sql.DB, workspaceID string) []string {
	t.Helper()

	var allowedTools string
	err := db.QueryRow(`
		SELECT allowed_tools
		FROM agent_definition
		WHERE workspace_id = ? AND agent_type = 'support'
	`, workspaceID).Scan(&allowedTools)
	if err != nil {
		t.Fatalf("default support agent not found: %v", err)
	}
	var tools []string
	if err := json.Unmarshal([]byte(allowedTools), &tools); err != nil {
		t.Fatalf("support agent allowed_tools JSON invalid: %v", err)
	}
	return tools
}

// assertDefaultSupportAgent verifies the bootstrap support-agent row exists with
// the expected identity, an active status, a UUID id (not a literal), and an
// allowed_tools set that is a subset of the owner tool grant.
func assertDefaultSupportAgent(t *testing.T, db *sql.DB, workspaceID string, ownerTools []string) {
	t.Helper()

	var id, name, agentType, status, limits string
	err := db.QueryRow(`
		SELECT id, name, agent_type, status, limits
		FROM agent_definition
		WHERE workspace_id = ? AND agent_type = 'support'
	`, workspaceID).Scan(&id, &name, &agentType, &status, &limits)
	if err != nil {
		t.Fatalf("default support agent not found: %v", err)
	}
	if id == "" || id == "support-agent" {
		t.Fatalf("support agent id = %q; want a bootstrap-generated UUID, not empty or the literal", id)
	}
	if name != "Support Agent" {
		t.Errorf("support agent name = %q; want Support Agent", name)
	}
	if status != "active" {
		t.Errorf("support agent status = %q; want active", status)
	}
	if limits == "" || limits == "{}" {
		t.Errorf("support agent limits = %q; want a non-empty cost ceiling", limits)
	}

	ownerSet := make(map[string]struct{}, len(ownerTools))
	for _, tool := range ownerTools {
		ownerSet[tool] = struct{}{}
	}
	agentTools := supportAgentAllowedTools(t, db, workspaceID)
	if len(agentTools) == 0 {
		t.Fatal("support agent allowed_tools is empty; want non-empty subset")
	}
	for _, tool := range agentTools {
		if _, ok := ownerSet[tool]; !ok {
			t.Errorf("support agent tool %q not in owner grant %v", tool, ownerTools)
		}
	}
}
