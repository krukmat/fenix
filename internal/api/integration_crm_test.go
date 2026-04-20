// Phase CRM Validation — T1: integration test helper + T2: Account CRUD
// Validates that CRM entities persist to and are retrieved from real SQLite.
// Pattern: real DB (in-memory) + real router + real JWT — no mocks.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// crmTestEnv holds the shared context for a single CRM integration test.
type crmTestEnv struct {
	router      http.Handler
	token       string
	workspaceID string
	ownerID     string
}

// setupCRMIntegrationTest creates a fresh in-memory DB, applies all migrations,
// registers a user, logs in, and returns the env needed to call protected CRM endpoints.
// Each call produces an isolated DB — safe for t.Parallel() and subtests.
//
// IMPORTANT: MaxOpenConns is forced to 1 for in-memory SQLite. With pool > 1 each
// new connection opens an independent empty DB, causing "no such table" errors on
// subsequent requests that land on a different pool connection than the one that
// received the migrations.
func setupCRMIntegrationTest(t *testing.T) crmTestEnv {
	t.Helper()

	db := mustOpenAPITestDB(t)
	db.SetMaxOpenConns(1) // pin to single connection for in-memory SQLite isolation
	router := mustNewRouter(t, db)

	// Register a user + workspace.
	regBody := registerBody("crm-test@example.com", "ValidPassword1!", "CRM User", "CRM Workspace")
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", regBody)
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	router.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("setupCRMIntegrationTest: register failed: status=%d body=%s", regW.Code, regW.Body.String())
	}

	var authResp struct {
		Token       string `json:"token"`
		UserID      string `json:"userId"`
		WorkspaceID string `json:"workspaceId"`
	}
	if err := json.NewDecoder(regW.Body).Decode(&authResp); err != nil {
		t.Fatalf("setupCRMIntegrationTest: decode register response: %v", err)
	}
	if authResp.Token == "" || authResp.WorkspaceID == "" || authResp.UserID == "" {
		t.Fatalf("setupCRMIntegrationTest: incomplete auth response: %+v", authResp)
	}

	return crmTestEnv{
		router:      router,
		token:       authResp.Token,
		workspaceID: authResp.WorkspaceID,
		ownerID:     authResp.UserID,
	}
}

// doJSON performs an authenticated JSON request and returns the recorder.
func (e *crmTestEnv) doJSON(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("doJSON: marshal: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// decodeJSON decodes the recorder body into dst; fails the test on error.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(dst); err != nil {
		t.Fatalf("decodeJSON: %v (body: %s)", err, w.Body.String())
	}
}

// ─── T1 smoke test ───────────────────────────────────────────────────────────

// TestCRMIntegration_SetupHelper verifies that setupCRMIntegrationTest returns
// a valid token, workspaceID and ownerID, and that a protected route accepts the token.
func TestCRMIntegration_SetupHelper(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	if env.token == "" {
		t.Fatal("expected non-empty token")
	}
	if env.workspaceID == "" {
		t.Fatal("expected non-empty workspaceID")
	}
	if env.ownerID == "" {
		t.Fatal("expected non-empty ownerID")
	}

	// Verify the token works against a protected endpoint (accounts list).
	w := env.doJSON(t, http.MethodGet, "/api/v1/accounts", nil)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("token rejected by protected endpoint: got 401")
	}
}

// ─── T2: Account CRUD ────────────────────────────────────────────────────────

func TestCRMIntegration_Account_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// POST → create account.
	createBody := map[string]any{
		"name":        "Acme Corp",
		"domain":      "acme.com",
		"industry":    "technology",
		"sizeSegment": "mid",
		"ownerId":     env.ownerID,
	}
	w := env.doJSON(t, http.MethodPost, "/api/v1/accounts", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/accounts: got %d, want 201. body: %s", w.Code, w.Body.String())
	}

	var created struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Industry    string `json:"industry"`
		SizeSegment string `json:"sizeSegment"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id in create response")
	}
	if created.Name != "Acme Corp" {
		t.Errorf("name: got %q, want %q", created.Name, "Acme Corp")
	}

	// GET → retrieve by id.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/accounts/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/accounts/%s: got %d, want 200. body: %s", created.ID, w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Industry string `json:"industry"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("GET id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Name != "Acme Corp" {
		t.Errorf("GET name: got %q, want %q", fetched.Name, "Acme Corp")
	}
	if fetched.Industry != "technology" {
		t.Errorf("GET industry: got %q, want %q", fetched.Industry, "technology")
	}
}

func TestCRMIntegration_Account_Update(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// Create.
	w := env.doJSON(t, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name":    "Old Name",
		"ownerId": env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	// PUT → update name.
	w2 := env.doJSON(t, http.MethodPut, "/api/v1/accounts/"+created.ID, map[string]any{
		"name": "New Name",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("PUT: got %d. body: %s", w2.Code, w2.Body.String())
	}

	// GET → verify update persisted.
	w3 := env.doJSON(t, http.MethodGet, "/api/v1/accounts/"+created.ID, nil)
	var fetched struct {
		Name string `json:"name"`
	}
	decodeJSON(t, w3, &fetched)
	if fetched.Name != "New Name" {
		t.Errorf("after update: name = %q, want %q", fetched.Name, "New Name")
	}
}

func TestCRMIntegration_Account_ListContainsCreated(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// Create two accounts.
	for _, name := range []string{"Alpha Inc", "Beta Ltd"} {
		w := env.doJSON(t, http.MethodPost, "/api/v1/accounts", map[string]any{
			"name":    name,
			"ownerId": env.ownerID,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("create %q: got %d. body: %s", name, w.Code, w.Body.String())
		}
	}

	// GET list.
	w := env.doJSON(t, http.MethodGet, "/api/v1/accounts", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: got %d. body: %s", w.Code, w.Body.String())
	}

	var list struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w, &list)
	if list.Meta.Total < 2 {
		t.Errorf("list meta.total: got %d, want >= 2", list.Meta.Total)
	}

	found := make(map[string]bool)
	for _, a := range list.Data {
		found[a.Name] = true
	}
	for _, name := range []string{"Alpha Inc", "Beta Ltd"} {
		if !found[name] {
			t.Errorf("account %q not found in list", name)
		}
	}
}

func TestCRMIntegration_Account_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// Create.
	w := env.doJSON(t, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name":    "To Be Deleted",
		"ownerId": env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	// DELETE.
	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/accounts/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	// GET after delete → 404 (soft-delete must filter).
	w3 := env.doJSON(t, http.MethodGet, "/api/v1/accounts/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}

	// List → deleted account must not appear.
	w4 := env.doJSON(t, http.MethodGet, "/api/v1/accounts", nil)
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, w4, &list)
	for _, a := range list.Data {
		if a.ID == created.ID {
			t.Errorf("deleted account %q still appears in list", created.ID)
		}
	}
}

// ─── T3: Contact CRUD ────────────────────────────────────────────────────────

// createTestAccount is a helper that creates an account and returns its ID.
func createTestAccount(t *testing.T, env *crmTestEnv, name string) string {
	t.Helper()
	w := env.doJSON(t, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name":    name,
		"ownerId": env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("createTestAccount %q: got %d. body: %s", name, w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &resp)
	return resp.ID
}

func TestCRMIntegration_Contact_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Acme Corp")

	// POST → create contact.
	w := env.doJSON(t, http.MethodPost, "/api/v1/contacts", map[string]any{
		"accountId": accountID,
		"firstName": "Jane",
		"lastName":  "Doe",
		"email":     "jane@acme.com",
		"title":     "Engineer",
		"ownerId":   env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/contacts: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID        string `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		AccountID string `json:"accountId"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id in create response")
	}
	if created.FirstName != "Jane" || created.LastName != "Doe" {
		t.Errorf("name: got %q %q, want Jane Doe", created.FirstName, created.LastName)
	}
	if created.AccountID != accountID {
		t.Errorf("accountId: got %q, want %q", created.AccountID, accountID)
	}

	// GET → retrieve by id.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/contacts/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/contacts/%s: got %d, want 200. body: %s", created.ID, w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID        string `json:"id"`
		FirstName string `json:"firstName"`
		AccountID string `json:"accountId"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("GET id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.FirstName != "Jane" {
		t.Errorf("GET firstName: got %q, want Jane", fetched.FirstName)
	}
	if fetched.AccountID != accountID {
		t.Errorf("GET accountId: got %q, want %q", fetched.AccountID, accountID)
	}
}

func TestCRMIntegration_Contact_Update(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Beta Ltd")

	w := env.doJSON(t, http.MethodPost, "/api/v1/contacts", map[string]any{
		"accountId": accountID,
		"firstName": "John",
		"lastName":  "Smith",
		"ownerId":   env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	// PUT → update title.
	w2 := env.doJSON(t, http.MethodPut, "/api/v1/contacts/"+created.ID, map[string]any{
		"title": "Senior Engineer",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("PUT: got %d. body: %s", w2.Code, w2.Body.String())
	}

	// GET → verify title persisted.
	w3 := env.doJSON(t, http.MethodGet, "/api/v1/contacts/"+created.ID, nil)
	var fetched struct {
		Title *string `json:"title"`
	}
	decodeJSON(t, w3, &fetched)
	if fetched.Title == nil || *fetched.Title != "Senior Engineer" {
		t.Errorf("after update: title = %v, want 'Senior Engineer'", fetched.Title)
	}
}

func TestCRMIntegration_Contact_ListByAccount(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountA := createTestAccount(t, &env, "Account A")
	accountB := createTestAccount(t, &env, "Account B")

	// Create 2 contacts on accountA, 1 on accountB.
	for _, name := range []string{"Alice", "Bob"} {
		w := env.doJSON(t, http.MethodPost, "/api/v1/contacts", map[string]any{
			"accountId": accountA,
			"firstName": name,
			"lastName":  "Test",
			"ownerId":   env.ownerID,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("create contact %q: got %d. body: %s", name, w.Code, w.Body.String())
		}
	}
	w := env.doJSON(t, http.MethodPost, "/api/v1/contacts", map[string]any{
		"accountId": accountB,
		"firstName": "Charlie",
		"lastName":  "Test",
		"ownerId":   env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create contact Charlie: got %d. body: %s", w.Code, w.Body.String())
	}

	// GET /api/v1/accounts/{account_id}/contacts → only accountA's contacts.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/accounts/"+accountA+"/contacts", nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("list by account: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var list struct {
		Data []struct {
			FirstName string `json:"firstName"`
			AccountID string `json:"accountId"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w2, &list)
	if list.Meta.Total != 2 {
		t.Errorf("listByAccount total: got %d, want 2", list.Meta.Total)
	}
	for _, c := range list.Data {
		if c.AccountID != accountA {
			t.Errorf("contact from wrong account in list: got accountId %q, want %q", c.AccountID, accountA)
		}
	}
}

func TestCRMIntegration_Contact_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Delete Corp")

	w := env.doJSON(t, http.MethodPost, "/api/v1/contacts", map[string]any{
		"accountId": accountID,
		"firstName": "To",
		"lastName":  "Delete",
		"ownerId":   env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	// DELETE.
	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/contacts/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	// GET after delete → 404.
	w3 := env.doJSON(t, http.MethodGet, "/api/v1/contacts/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}

func TestCRMIntegration_Contact_InvalidAccount_Returns400(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// Contact with non-existent account_id → should fail (FK constraint).
	w := env.doJSON(t, http.MethodPost, "/api/v1/contacts", map[string]any{
		"accountId": "non-existent-account-id",
		"firstName": "Ghost",
		"lastName":  "User",
		"ownerId":   env.ownerID,
	})
	if w.Code == http.StatusCreated {
		t.Errorf("expected failure for non-existent accountId, got 201")
	}
}

// ─── T4: Pipeline + Stages ───────────────────────────────────────────────────

// createTestPipeline creates a pipeline and returns its ID.
func createTestPipeline(t *testing.T, env *crmTestEnv, name string) string {
	t.Helper()
	w := env.doJSON(t, http.MethodPost, "/api/v1/pipelines", map[string]any{
		"name":       name,
		"entityType": "deal",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("createTestPipeline %q: got %d. body: %s", name, w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &resp)
	return resp.ID
}

// createTestStage creates a stage on a pipeline and returns its ID.
func createTestStage(t *testing.T, env *crmTestEnv, pipelineID, name string, position int64) string {
	t.Helper()
	w := env.doJSON(t, http.MethodPost, "/api/v1/pipelines/"+pipelineID+"/stages", map[string]any{
		"name":     name,
		"position": position,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("createTestStage %q: got %d. body: %s", name, w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &resp)
	return resp.ID
}

func TestCRMIntegration_Pipeline_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// POST → create pipeline.
	w := env.doJSON(t, http.MethodPost, "/api/v1/pipelines", map[string]any{
		"name":       "Sales Pipeline",
		"entityType": "deal",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/pipelines: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		EntityType string `json:"entityType"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id in create response")
	}
	if created.Name != "Sales Pipeline" {
		t.Errorf("name: got %q, want %q", created.Name, "Sales Pipeline")
	}

	// GET → retrieve by id.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/pipelines/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/pipelines/%s: got %d, want 200. body: %s", created.ID, w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		EntityType string `json:"entityType"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("GET id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.EntityType != "deal" {
		t.Errorf("GET entityType: got %q, want deal", fetched.EntityType)
	}
}

func TestCRMIntegration_Pipeline_List(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	for _, name := range []string{"Pipeline A", "Pipeline B"} {
		createTestPipeline(t, &env, name)
	}

	w := env.doJSON(t, http.MethodGet, "/api/v1/pipelines", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/pipelines: got %d. body: %s", w.Code, w.Body.String())
	}
	var list struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w, &list)
	if list.Meta.Total < 2 {
		t.Errorf("list meta.total: got %d, want >= 2", list.Meta.Total)
	}
}

func TestCRMIntegration_Stage_CreateAndList(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	pipelineID := createTestPipeline(t, &env, "Support Pipeline")

	// POST stage.
	w := env.doJSON(t, http.MethodPost, "/api/v1/pipelines/"+pipelineID+"/stages", map[string]any{
		"name":     "Prospecting",
		"position": 1,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST stage: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var stage struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		PipelineID string `json:"pipelineId"`
	}
	decodeJSON(t, w, &stage)
	if stage.ID == "" {
		t.Fatal("expected non-empty stage id")
	}
	if stage.Name != "Prospecting" {
		t.Errorf("stage name: got %q, want Prospecting", stage.Name)
	}

	// Create a second stage.
	env.doJSON(t, http.MethodPost, "/api/v1/pipelines/"+pipelineID+"/stages", map[string]any{
		"name":     "Qualified",
		"position": 2,
	})

	// GET stages list.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/pipelines/"+pipelineID+"/stages", nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET stages: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w2, &list)
	if list.Meta.Total != 2 {
		t.Errorf("stages total: got %d, want 2", list.Meta.Total)
	}
}

func TestCRMIntegration_Stage_Update(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	pipelineID := createTestPipeline(t, &env, "Update Pipeline")
	stageID := createTestStage(t, &env, pipelineID, "Draft", 1)

	// PUT → update name.
	w := env.doJSON(t, http.MethodPut, "/api/v1/pipelines/stages/"+stageID, map[string]any{
		"name":     "Published",
		"position": 1,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT stage: got %d, want 200. body: %s", w.Code, w.Body.String())
	}
	var updated struct {
		Name string `json:"name"`
	}
	decodeJSON(t, w, &updated)
	if updated.Name != "Published" {
		t.Errorf("stage name after update: got %q, want Published", updated.Name)
	}
}

func TestCRMIntegration_Stage_Delete(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	pipelineID := createTestPipeline(t, &env, "Delete Pipeline")
	stageID := createTestStage(t, &env, pipelineID, "ToRemove", 1)

	w := env.doJSON(t, http.MethodDelete, "/api/v1/pipelines/stages/"+stageID, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE stage: got %d, want 204. body: %s", w.Code, w.Body.String())
	}

	// After delete, list should be empty.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/pipelines/"+pipelineID+"/stages", nil)
	var list struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w2, &list)
	if list.Meta.Total != 0 {
		t.Errorf("stages after delete: got %d, want 0", list.Meta.Total)
	}
}

// ─── T5: Deal CRUD ───────────────────────────────────────────────────────────

func TestCRMIntegration_Deal_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Deal Corp")
	pipelineID := createTestPipeline(t, &env, "Sales")
	stageID := createTestStage(t, &env, pipelineID, "Prospecting", 1)

	amount := 9999.99
	w := env.doJSON(t, http.MethodPost, "/api/v1/deals", map[string]any{
		"accountId":  accountID,
		"pipelineId": pipelineID,
		"stageId":    stageID,
		"ownerId":    env.ownerID,
		"title":      "Big Deal",
		"amount":     amount,
		"currency":   "USD",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/deals: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID         string   `json:"id"`
		Title      string   `json:"title"`
		Amount     *float64 `json:"amount"`
		AccountID  string   `json:"accountId"`
		PipelineID string   `json:"pipelineId"`
		StageID    string   `json:"stageId"`
		Status     string   `json:"status"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Title != "Big Deal" {
		t.Errorf("title: got %q, want Big Deal", created.Title)
	}
	if created.Amount == nil || *created.Amount != amount {
		t.Errorf("amount: got %v, want %v", created.Amount, amount)
	}

	// GET → same data persisted.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/deals/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/deals/%s: got %d. body: %s", created.ID, w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		AccountID  string `json:"accountId"`
		PipelineID string `json:"pipelineId"`
		StageID    string `json:"stageId"`
		Status     string `json:"status"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.AccountID != accountID {
		t.Errorf("accountId: got %q, want %q", fetched.AccountID, accountID)
	}
	if fetched.StageID != stageID {
		t.Errorf("stageId: got %q, want %q", fetched.StageID, stageID)
	}
	if fetched.Status != "open" {
		t.Errorf("default status: got %q, want open", fetched.Status)
	}
}

func TestCRMIntegration_Deal_UpdateStageAndStatus(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Stage Corp")
	pipelineID := createTestPipeline(t, &env, "Pipeline")
	stage1 := createTestStage(t, &env, pipelineID, "Stage 1", 1)
	stage2 := createTestStage(t, &env, pipelineID, "Stage 2", 2)

	w := env.doJSON(t, http.MethodPost, "/api/v1/deals", map[string]any{
		"accountId":  accountID,
		"pipelineId": pipelineID,
		"stageId":    stage1,
		"ownerId":    env.ownerID,
		"title":      "Movable Deal",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	// PUT → move to stage2 + won.
	w2 := env.doJSON(t, http.MethodPut, "/api/v1/deals/"+created.ID, map[string]any{
		"accountId":  accountID,
		"pipelineId": pipelineID,
		"stageId":    stage2,
		"ownerId":    env.ownerID,
		"title":      "Movable Deal",
		"status":     "won",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("PUT: got %d. body: %s", w2.Code, w2.Body.String())
	}

	// GET → verify persisted.
	w3 := env.doJSON(t, http.MethodGet, "/api/v1/deals/"+created.ID, nil)
	var fetched struct {
		StageID string `json:"stageId"`
		Status  string `json:"status"`
	}
	decodeJSON(t, w3, &fetched)
	if fetched.StageID != stage2 {
		t.Errorf("stageId after update: got %q, want %q", fetched.StageID, stage2)
	}
	if fetched.Status != "won" {
		t.Errorf("status after update: got %q, want won", fetched.Status)
	}
}

func TestCRMIntegration_Deal_NullableContactID(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "No Contact Corp")
	pipelineID := createTestPipeline(t, &env, "Pipeline")
	stageID := createTestStage(t, &env, pipelineID, "Open", 1)

	// Deal without contactId — optional field must not cause error.
	w := env.doJSON(t, http.MethodPost, "/api/v1/deals", map[string]any{
		"accountId":  accountID,
		"pipelineId": pipelineID,
		"stageId":    stageID,
		"ownerId":    env.ownerID,
		"title":      "No Contact Deal",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("deal without contactId: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
}

func TestCRMIntegration_Deal_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Delete Deal Corp")
	pipelineID := createTestPipeline(t, &env, "Pipeline")
	stageID := createTestStage(t, &env, pipelineID, "Open", 1)

	w := env.doJSON(t, http.MethodPost, "/api/v1/deals", map[string]any{
		"accountId":  accountID,
		"pipelineId": pipelineID,
		"stageId":    stageID,
		"ownerId":    env.ownerID,
		"title":      "To Delete",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/deals/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/deals/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}

// ─── T6: Case CRUD ───────────────────────────────────────────────────────────

func TestCRMIntegration_Case_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	w := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId":  env.ownerID,
		"subject":  "Login broken",
		"priority": "high",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/cases: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID       string `json:"id"`
		Subject  string `json:"subject"`
		Priority string `json:"priority"`
		Status   string `json:"status"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Subject != "Login broken" {
		t.Errorf("subject: got %q, want Login broken", created.Subject)
	}
	if created.Status != "open" {
		t.Errorf("default status: got %q, want open", created.Status)
	}

	// GET → same data persisted.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/cases/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID       string `json:"id"`
		Subject  string `json:"subject"`
		Priority string `json:"priority"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Priority != "high" {
		t.Errorf("priority: got %q, want high", fetched.Priority)
	}
}

func TestCRMIntegration_Case_StatusTransitions(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	w := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Transitions test",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	for _, status := range []string{"in_progress", "resolved", "closed"} {
		w2 := env.doJSON(t, http.MethodPut, "/api/v1/cases/"+created.ID, map[string]any{
			"ownerId": env.ownerID,
			"subject": "Transitions test",
			"status":  status,
		})
		if w2.Code != http.StatusOK {
			t.Fatalf("PUT status=%q: got %d. body: %s", status, w2.Code, w2.Body.String())
		}
		w3 := env.doJSON(t, http.MethodGet, "/api/v1/cases/"+created.ID, nil)
		var fetched struct {
			Status string `json:"status"`
		}
		decodeJSON(t, w3, &fetched)
		if fetched.Status != status {
			t.Errorf("after PUT status=%q: got %q", status, fetched.Status)
		}
	}
}

func TestCRMIntegration_Case_StandaloneNoAccountContact(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// Case without accountId/contactId — both optional.
	w := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Standalone case",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("standalone case: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
}

func TestCRMIntegration_Case_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	w := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "To delete",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/cases/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/cases/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}

// ─── T7: Lead CRUD ───────────────────────────────────────────────────────────

func TestCRMIntegration_Lead_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	w := env.doJSON(t, http.MethodPost, "/api/v1/leads", map[string]any{
		"ownerId": env.ownerID,
		"source":  "website",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/leads: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Source string `json:"source"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Status != "new" {
		t.Errorf("default status: got %q, want new", created.Status)
	}

	w2 := env.doJSON(t, http.MethodGet, "/api/v1/leads/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID     string `json:"id"`
		Source string `json:"source"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Source != "website" {
		t.Errorf("source: got %q, want website", fetched.Source)
	}
}

func TestCRMIntegration_Lead_StatusTransitions(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	w := env.doJSON(t, http.MethodPost, "/api/v1/leads", map[string]any{
		"ownerId": env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	for _, status := range []string{"contacted", "qualified"} {
		w2 := env.doJSON(t, http.MethodPut, "/api/v1/leads/"+created.ID, map[string]any{
			"ownerId": env.ownerID,
			"status":  status,
		})
		if w2.Code != http.StatusOK {
			t.Fatalf("PUT status=%q: got %d. body: %s", status, w2.Code, w2.Body.String())
		}
		w3 := env.doJSON(t, http.MethodGet, "/api/v1/leads/"+created.ID, nil)
		var fetched struct {
			Status string `json:"status"`
		}
		decodeJSON(t, w3, &fetched)
		if fetched.Status != status {
			t.Errorf("after PUT status=%q: got %q", status, fetched.Status)
		}
	}
}

func TestCRMIntegration_Lead_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	w := env.doJSON(t, http.MethodPost, "/api/v1/leads", map[string]any{
		"ownerId": env.ownerID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/leads/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/leads/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}

// ─── T8: Activity CRUD + polymorphism ────────────────────────────────────────

func TestCRMIntegration_Activity_CreateOnAccount(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Activity Corp")

	w := env.doJSON(t, http.MethodPost, "/api/v1/activities", map[string]any{
		"activityType": "task",
		"entityType":   "account",
		"entityId":     accountID,
		"ownerId":      env.ownerID,
		"subject":      "Follow up call",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/activities: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID         string `json:"id"`
		EntityType string `json:"entityType"`
		EntityID   string `json:"entityId"`
		Status     string `json:"status"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.EntityType != "account" {
		t.Errorf("entityType: got %q, want account", created.EntityType)
	}
	if created.EntityID != accountID {
		t.Errorf("entityId: got %q, want %q", created.EntityID, accountID)
	}
	if created.Status != "pending" {
		t.Errorf("default status: got %q, want pending", created.Status)
	}

	// GET → persisted.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/activities/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Subject != "Follow up call" {
		t.Errorf("subject: got %q, want Follow up call", fetched.Subject)
	}
}

func TestCRMIntegration_Activity_CreateOnCase(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	// Create a case first.
	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Case for activity",
	})
	if wc.Code != http.StatusCreated {
		t.Fatalf("create case: got %d. body: %s", wc.Code, wc.Body.String())
	}
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	w := env.doJSON(t, http.MethodPost, "/api/v1/activities", map[string]any{
		"activityType": "call",
		"entityType":   "case",
		"entityId":     caseResp.ID,
		"ownerId":      env.ownerID,
		"subject":      "Support call",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST activity on case: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		EntityType string `json:"entityType"`
		EntityID   string `json:"entityId"`
	}
	decodeJSON(t, w, &created)
	if created.EntityType != "case" {
		t.Errorf("entityType: got %q, want case", created.EntityType)
	}
	if created.EntityID != caseResp.ID {
		t.Errorf("entityId: got %q, want %q", created.EntityID, caseResp.ID)
	}
}

func TestCRMIntegration_Activity_UpdateStatus(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Status Corp")

	w := env.doJSON(t, http.MethodPost, "/api/v1/activities", map[string]any{
		"activityType": "task",
		"entityType":   "account",
		"entityId":     accountID,
		"ownerId":      env.ownerID,
		"subject":      "Task to complete",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodPut, "/api/v1/activities/"+created.ID, map[string]any{
		"activityType": "task",
		"entityType":   "account",
		"entityId":     accountID,
		"ownerId":      env.ownerID,
		"subject":      "Task to complete",
		"status":       "completed",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("PUT: got %d. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/activities/"+created.ID, nil)
	var fetched struct {
		Status string `json:"status"`
	}
	decodeJSON(t, w3, &fetched)
	if fetched.Status != "completed" {
		t.Errorf("status after update: got %q, want completed", fetched.Status)
	}
}

func TestCRMIntegration_Activity_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Delete Activity Corp")

	w := env.doJSON(t, http.MethodPost, "/api/v1/activities", map[string]any{
		"activityType": "task",
		"entityType":   "account",
		"entityId":     accountID,
		"ownerId":      env.ownerID,
		"subject":      "To delete",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/activities/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/activities/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}

// ─── T9: Note CRUD + polymorphism ────────────────────────────────────────────

func TestCRMIntegration_Note_CreateOnAccount(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Note Corp")

	w := env.doJSON(t, http.MethodPost, "/api/v1/notes", map[string]any{
		"entityType": "account",
		"entityId":   accountID,
		"authorId":   env.ownerID,
		"content":    "Important note",
		"isInternal": false,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/notes: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID         string `json:"id"`
		EntityType string `json:"entityType"`
		EntityID   string `json:"entityId"`
		IsInternal bool   `json:"isInternal"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.EntityType != "account" {
		t.Errorf("entityType: got %q, want account", created.EntityType)
	}
	if created.IsInternal != false {
		t.Errorf("isInternal: got %v, want false", created.IsInternal)
	}

	// GET → persisted.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/notes/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Content != "Important note" {
		t.Errorf("content: got %q, want Important note", fetched.Content)
	}
}

func TestCRMIntegration_Note_InternalFlagOnCase(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Case for note",
	})
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	w := env.doJSON(t, http.MethodPost, "/api/v1/notes", map[string]any{
		"entityType": "case",
		"entityId":   caseResp.ID,
		"authorId":   env.ownerID,
		"content":    "Internal only",
		"isInternal": true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST internal note: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		IsInternal bool `json:"isInternal"`
	}
	decodeJSON(t, w, &created)
	if !created.IsInternal {
		t.Errorf("isInternal: got false, want true")
	}
}

func TestCRMIntegration_Note_SoftDelete(t *testing.T) {
	env := setupCRMIntegrationTest(t)
	accountID := createTestAccount(t, &env, "Delete Note Corp")

	w := env.doJSON(t, http.MethodPost, "/api/v1/notes", map[string]any{
		"entityType": "account",
		"entityId":   accountID,
		"authorId":   env.ownerID,
		"content":    "To delete",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/notes/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/notes/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}

// ─── T10: Timeline auto-generated ────────────────────────────────────────────
// Timeline wiring confirmed only on Case (internal/domain/crm/case.go).
// Account wiring is absent — logged as T12 bug finding.

func TestCRMIntegration_Timeline_CaseCreatesEvent(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Timeline case",
	})
	if wc.Code != http.StatusCreated {
		t.Fatalf("create case: got %d. body: %s", wc.Code, wc.Body.String())
	}
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	// GET timeline by entity → created event must exist.
	w := env.doJSON(t, http.MethodGet, "/api/v1/timeline/case/"+caseResp.ID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET timeline: got %d. body: %s", w.Code, w.Body.String())
	}
	var list struct {
		Data []struct {
			EventType string `json:"eventType"`
			EntityID  string `json:"entityId"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w, &list)
	if list.Meta.Total < 1 {
		t.Fatalf("timeline total: got %d, want >= 1", list.Meta.Total)
	}
	if list.Data[0].EventType != "created" {
		t.Errorf("first event type: got %q, want created", list.Data[0].EventType)
	}
	if list.Data[0].EntityID != caseResp.ID {
		t.Errorf("entityId: got %q, want %q", list.Data[0].EntityID, caseResp.ID)
	}
}

func TestCRMIntegration_Timeline_CaseUpdateCreatesEvent(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Update timeline case",
	})
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	// Update the case → should produce an updated event.
	env.doJSON(t, http.MethodPut, "/api/v1/cases/"+caseResp.ID, map[string]any{
		"ownerId": env.ownerID,
		"subject": "Updated subject",
		"status":  "in_progress",
	})

	w := env.doJSON(t, http.MethodGet, "/api/v1/timeline/case/"+caseResp.ID, nil)
	var list struct {
		Data []struct {
			EventType string `json:"eventType"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w, &list)
	if list.Meta.Total < 2 {
		t.Fatalf("timeline total after update: got %d, want >= 2", list.Meta.Total)
	}
	found := false
	for _, e := range list.Data {
		if e.EventType == "updated" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no updated event found in timeline")
	}
}

func TestCRMIntegration_Timeline_CaseDeleteCreatesEvent(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Delete timeline case",
	})
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	env.doJSON(t, http.MethodDelete, "/api/v1/cases/"+caseResp.ID, nil)

	w := env.doJSON(t, http.MethodGet, "/api/v1/timeline/case/"+caseResp.ID, nil)
	var list struct {
		Data []struct {
			EventType string `json:"eventType"`
		} `json:"data"`
	}
	decodeJSON(t, w, &list)
	found := false
	for _, e := range list.Data {
		if e.EventType == "deleted" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no deleted event found in timeline after case delete")
	}
}

// ─── T11: Attachment ─────────────────────────────────────────────────────────

func TestCRMIntegration_Attachment_CreateAndGet(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Case for attachment",
	})
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	w := env.doJSON(t, http.MethodPost, "/api/v1/attachments", map[string]any{
		"entityType":  "case",
		"entityId":    caseResp.ID,
		"uploaderId":  env.ownerID,
		"filename":    "report.pdf",
		"contentType": "application/pdf",
		"storagePath": "./data/attachments/report.pdf",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/attachments: got %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"contentType"`
		StoragePath string `json:"storagePath"`
	}
	decodeJSON(t, w, &created)
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Filename != "report.pdf" {
		t.Errorf("filename: got %q, want report.pdf", created.Filename)
	}
	if created.ContentType != "application/pdf" {
		t.Errorf("contentType: got %q, want application/pdf", created.ContentType)
	}
	if created.StoragePath != "./data/attachments/report.pdf" {
		t.Errorf("storagePath: got %q", created.StoragePath)
	}

	// GET → persisted.
	w2 := env.doJSON(t, http.MethodGet, "/api/v1/attachments/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET: got %d. body: %s", w2.Code, w2.Body.String())
	}
	var fetched struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
	}
	decodeJSON(t, w2, &fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Filename != "report.pdf" {
		t.Errorf("filename: got %q, want report.pdf", fetched.Filename)
	}
}

func TestCRMIntegration_Attachment_ListByEntity(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Case for attachments list",
	})
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	for _, name := range []string{"file1.pdf", "file2.pdf"} {
		w := env.doJSON(t, http.MethodPost, "/api/v1/attachments", map[string]any{
			"entityType":  "case",
			"entityId":    caseResp.ID,
			"uploaderId":  env.ownerID,
			"filename":    name,
			"storagePath": "./data/" + name,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("create %q: got %d. body: %s", name, w.Code, w.Body.String())
		}
	}

	w := env.doJSON(t, http.MethodGet, "/api/v1/attachments?entityType=case&entityId="+caseResp.ID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: got %d. body: %s", w.Code, w.Body.String())
	}
	var list struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	decodeJSON(t, w, &list)
	if list.Meta.Total < 2 {
		t.Errorf("list total: got %d, want >= 2", list.Meta.Total)
	}
}

func TestCRMIntegration_Attachment_Delete(t *testing.T) {
	env := setupCRMIntegrationTest(t)

	wc := env.doJSON(t, http.MethodPost, "/api/v1/cases", map[string]any{
		"ownerId": env.ownerID,
		"subject": "Case for delete attachment",
	})
	var caseResp struct {
		ID string `json:"id"`
	}
	decodeJSON(t, wc, &caseResp)

	w := env.doJSON(t, http.MethodPost, "/api/v1/attachments", map[string]any{
		"entityType":  "case",
		"entityId":    caseResp.ID,
		"uploaderId":  env.ownerID,
		"filename":    "to_delete.pdf",
		"storagePath": "./data/to_delete.pdf",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d. body: %s", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, w, &created)

	w2 := env.doJSON(t, http.MethodDelete, "/api/v1/attachments/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204. body: %s", w2.Code, w2.Body.String())
	}

	w3 := env.doJSON(t, http.MethodGet, "/api/v1/attachments/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after delete: got %d, want 404", w3.Code)
	}
}
