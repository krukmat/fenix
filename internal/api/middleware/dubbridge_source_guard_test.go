package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

const s2sTestToken = "fenix_s2s_v1_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
const testOperationID = "operation_123456789"

type s2sVerifierStub struct {
	identity *domainauth.ServiceIdentity
	err      error
	calls    int
}

func (v *s2sVerifierStub) AuthenticateServiceToken(_ context.Context, _ string, _ time.Time) (*domainauth.ServiceIdentity, error) {
	v.calls++
	return v.identity, v.err
}
func s2sTestIdentity() *domainauth.ServiceIdentity {
	return &domainauth.ServiceIdentity{ServicePrincipal: domainauth.ServicePrincipal{ID: "principal-1", Consumer: "dubbridge", WorkspaceID: "ws-1"}, DubBridgeOrgID: "org-1", DubBridgeProjectID: "project-1"}
}
func s2sEnvelope(source string, input string) string {
	return fmt.Sprintf(`{"contract_version":"1","operation_id":"%s","capability_key":"integration.diagnostic","input_schema_version":"1","trace_id":"trace_12345678","source":%s,"input":%s}`, testOperationID, source, input)
}
func testSource(org, project string) string {
	return fmt.Sprintf(`{"product":"dubbridge","org_id":%q,"project_id":%q}`, org, project)
}
func invokeS2S(method, path, body string, verifier DubBridgeTokenAuthenticator, resolver DubBridgeOperationResolver, handler http.Handler) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+s2sTestToken)
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	DubBridgeServiceAuth(verifier, resolver, handler).ServeHTTP(w, r)
	return w
}
func TestMandatoryDubBridgePOSTAuthorization(t *testing.T) {
	cases := []struct {
		name, source   string
		status, called int
	}{
		{"exact", testSource("org-1", "project-1"), 200, 1},
		{"differentOrg", testSource("org-2", "project-1"), 403, 0},
		{"differentProject", testSource("org-1", "project-2"), 403, 0},
		{"projectOmitted", `{"product":"dubbridge","org_id":"org-1"}`, 403, 0},
		{"differentProduct", `{"product":"other","org_id":"org-1","project_id":"project-1"}`, 403, 0},
		{"missingOrg", `{"product":"dubbridge","project_id":"project-1"}`, 400, 0},
		{"callerWorkspaceForged", `{"product":"dubbridge","org_id":"org-1","project_id":"project-1","workspace_id":"elsewhere"}`, 400, 0},
		{"missingSource", `null`, 400, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hits := 0
			verifiedScope := false
			verifier := &s2sVerifierStub{identity: s2sTestIdentity()}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				ident, ok := VerifiedDubBridgeIdentity(r.Context())
				scope, scoped := AuthorizedDubBridgeScope(r.Context())
				verifiedScope = ok && scoped && ident.WorkspaceID == "ws-1" && scope.PrincipalID == "principal-1" && scope.ProjectID == "project-1"
				w.WriteHeader(200)
			})
			w := invokeS2S("POST", "/integrations/dubbridge/operations", s2sEnvelope(tc.source, `{}`), verifier, nil, handler)
			if w.Code != tc.status || hits != tc.called {
				t.Fatalf("status=%d hits=%d expected %d/%d", w.Code, hits, tc.status, tc.called)
			}
			if hits == 1 && !verifiedScope {
				t.Fatal("authorized private scope not populated")
			}
			if hits == 0 && (strings.Contains(w.Body.String(), "org-1") || strings.Contains(w.Body.String(), "project-1")) {
				t.Fatal("scope disclosed")
			}
		})
	}
}
func TestMandatoryDubBridgeOrgOnly(t *testing.T) {
	ident := s2sTestIdentity()
	ident.DubBridgeProjectID = ""
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(200) })
	w := invokeS2S("POST", "/integrations/dubbridge/operations", s2sEnvelope(`{"product":"dubbridge","org_id":"org-1"}`, `{}`), &s2sVerifierStub{identity: ident}, nil, handler)
	if w.Code != 200 || calls != 1 {
		t.Fatalf("org-only: %d %d", w.Code, calls)
	}
	w = invokeS2S("POST", "/integrations/dubbridge/operations", s2sEnvelope(testSource("org-1", "project-1"), `{}`), &s2sVerifierStub{identity: ident}, nil, handler)
	if w.Code != 403 || calls != 1 {
		t.Fatalf("project escalation: %d %d", w.Code, calls)
	}
}

type s2sResolverStub struct {
	record   DubBridgeStoredOperation
	err      error
	calls    int
	captured DubBridgeOperationScope
}

func (s *s2sResolverStub) LookupDubBridgeOperation(_ context.Context, _ string, scope DubBridgeOperationScope) (DubBridgeStoredOperation, error) {
	s.calls++
	s.captured = scope
	return s.record, s.err
}
func persistedOp() DubBridgeStoredOperation {
	return DubBridgeStoredOperation{OperationID: testOperationID, DubBridgeOperationScope: scopeFromFenixIdentity(*s2sTestIdentity())}
}
func TestMandatoryDubBridgeGETStatus(t *testing.T) {
	cases := []struct {
		name         string
		resolver     *s2sResolverStub
		status, hits int
	}{
		{"nilStore", nil, 403, 0},
		{"exactPersisted", &s2sResolverStub{record: persistedOp()}, 200, 1},
		{"missing", &s2sResolverStub{err: errors.New("not found")}, 403, 0},
		{"storeUnavailable", &s2sResolverStub{err: ErrDubBridgeOperationUnavailable}, 503, 0},
		{"wrongPrincipal", &s2sResolverStub{record: func() DubBridgeStoredOperation { x := persistedOp(); x.PrincipalID = "another"; return x }()}, 403, 0},
		{"wrongWorkspace", &s2sResolverStub{record: func() DubBridgeStoredOperation { x := persistedOp(); x.WorkspaceID = "other"; return x }()}, 403, 0},
		{"wrongOrg", &s2sResolverStub{record: func() DubBridgeStoredOperation { x := persistedOp(); x.OrgID = "other"; return x }()}, 403, 0},
		{"wrongProject", &s2sResolverStub{record: func() DubBridgeStoredOperation { x := persistedOp(); x.ProjectID = "other"; return x }()}, 403, 0},
		{"wrongOperation", &s2sResolverStub{record: func() DubBridgeStoredOperation { x := persistedOp(); x.OperationID = "other"; return x }()}, 403, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(200) })
			w := invokeS2S("GET", "/integrations/dubbridge/operations/"+testOperationID, "", &s2sVerifierStub{identity: s2sTestIdentity()}, tc.resolver, handler)
			if w.Code != tc.status || calls != tc.hits {
				t.Fatalf("status %d calls %d want %d/%d", w.Code, calls, tc.status, tc.hits)
			}
			if tc.resolver != nil && (tc.resolver.calls != 1 || tc.resolver.captured.WorkspaceID != "ws-1") {
				t.Fatalf("unscoped lookup: %#v", tc.resolver)
			}
			if calls == 0 && strings.Contains(w.Body.String(), "ws-1") {
				t.Fatal("tenant leak")
			}
		})
	}
}
func TestMandatoryDubBridgeGETRejectsIDOnly(t *testing.T) {
	resolver := &s2sResolverStub{record: persistedOp()}
	calls := 0
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ })
	for _, path := range []string{"/integrations/dubbridge/operations/short", "/integrations/dubbridge/operations/operation_123456789/extra", "/integrations/dubbridge/operations/operation%2F123456789"} {
		w := invokeS2S("GET", path, "", &s2sVerifierStub{identity: s2sTestIdentity()}, resolver, handler)
		if w.Code != 403 || calls != 0 {
			t.Fatalf("path=%s code=%d calls=%d", path, w.Code, calls)
		}
	}
	if resolver.calls != 0 {
		t.Fatalf("unsafe resolver reached %d", resolver.calls)
	}
}
func TestMandatoryDubBridgeMalformedPOST(t *testing.T) {
	inputs := []string{
		`invalid`,
		`{"contract_version":"1","operation_id":"operation_123456789","capability_key":"integration.diagnostic","input_schema_version":"1","trace_id":"trace_12345678","source":{"product":"dubbridge","org_id":"org-evil","org_id":"org-1","project_id":"project-1"},"input":{}}`,
		`{"contract_version":"1","operation_id":"operation_123456789","capability_key":"integration.diagnostic","input_schema_version":"1","trace_id":"trace_12345678","source":{"product":"dubbridge","org_id":"org-1","project_id":"project-1"},"source":{"product":"dubbridge","org_id":"org-1","project_id":"project-1"},"input":{}}`,
		s2sEnvelope(testSource("org-1", "project-1"), `{}`) + ` {}`,
		s2sEnvelope(testSource("org-1", "project-1"), `[]`),
		s2sEnvelope(testSource("org-1", "project-1"), `null`),
		`{"contract_version":"2","operation_id":"op","capability_key":"cap.x","input_schema_version":"1","trace_id":"trace123","source":{"product":"dubbridge","org_id":"org-1","project_id":"project-1"},"input":{}}`,
	}
	for _, body := range inputs {
		calls := 0
		h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ })
		w := invokeS2S("POST", "/integrations/dubbridge/operations", body, &s2sVerifierStub{identity: s2sTestIdentity()}, nil, h)
		if w.Code != 400 || calls != 0 {
			t.Fatalf("status=%d calls=%d for %.60s", w.Code, calls, body)
		}
	}
}
func TestMandatoryDubBridgeBodyLimit(t *testing.T) {
	// Envelope function inserts the closing braces; create a validated request of exact size.
	makeBody := func(size int) string {
		base := s2sEnvelope(testSource("org-1", "project-1"), `{"padding":""}`)
		closing := strings.LastIndex(base, `"}`)
		if closing < 0 {
			t.Fatal("invalid fixture")
		}
		if size < len(base) {
			t.Fatal("fixture too large")
		}
		return base[:closing] + strings.Repeat("x", size-len(base)) + base[closing:]
	}
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var obj map[string]any
		if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(200)
	})
	for _, tc := range []struct{ size, status, hits int }{{DubBridgeS2SMaxRequestBytes, 200, 1}, {DubBridgeS2SMaxRequestBytes + 1, 413, 0}} {
		before := calls
		w := invokeS2S("POST", "/integrations/dubbridge/operations", makeBody(tc.size), &s2sVerifierStub{identity: s2sTestIdentity()}, nil, handler)
		if w.Code != tc.status || calls-before != tc.hits {
			t.Fatalf("size %d: code %d calls %d", tc.size, w.Code, calls)
		}
	}
}
func TestMandatoryDubBridgeAuthFailures(t *testing.T) {
	cases := []struct {
		name     string
		verifier DubBridgeTokenAuthenticator
		status   int
	}{
		{"nilVerifier", nil, 503},
		{"typedNilVerifier", (*s2sVerifierStub)(nil), 503},
		{"unavailable", &s2sVerifierStub{err: domainauth.ErrServiceIdentityUnavailable}, 503},
		{"denied", &s2sVerifierStub{err: domainauth.ErrServiceIdentityDenied}, 401},
		{"missingScope", &s2sVerifierStub{identity: &domainauth.ServiceIdentity{ServicePrincipal: domainauth.ServicePrincipal{ID: "id", Consumer: "dubbridge"}}}, 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hits := 0
			handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ })
			w := invokeS2S("POST", "/integrations/dubbridge/operations", s2sEnvelope(testSource("org-1", "project-1"), `{}`), tc.verifier, nil, handler)
			if w.Code != tc.status || hits != 0 {
				t.Fatalf("code %d hits %d", w.Code, hits)
			}
		})
	}
}
func TestMandatoryDubBridgeBearerAndMethod(t *testing.T) {
	for _, raw := range []string{"Bearer aaa.bbb.ccc", "Basic abc", "Bearer " + s2sTestToken + " ", "Bearer Bearer " + s2sTestToken} {
		verifier := &s2sVerifierStub{identity: s2sTestIdentity()}
		hits := 0
		handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ })
		r := httptest.NewRequest("POST", "/integrations/dubbridge/operations", strings.NewReader(s2sEnvelope(testSource("org-1", "project-1"), `{}`)))
		r.Header.Set("Authorization", raw)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		DubBridgeServiceAuth(verifier, nil, handler).ServeHTTP(w, r)
		if w.Code != 401 || verifier.calls != 0 || hits != 0 {
			t.Fatalf("raw %s: code %d verifier %d hits %d", raw, w.Code, verifier.calls, hits)
		}
	}
	w := invokeS2S("DELETE", "/integrations/dubbridge/operations/"+testOperationID, "", &s2sVerifierStub{identity: s2sTestIdentity()}, nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler invoked") }))
	if w.Code != 405 {
		t.Fatalf("unsupported method %d", w.Code)
	}
}
