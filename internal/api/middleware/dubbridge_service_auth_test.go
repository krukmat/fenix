package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

type verifierStub struct {
	ident  *domainauth.ServiceIdentity
	err    error
	called int
}

func (v *verifierStub) AuthenticateServiceToken(_ context.Context, _ string, _ time.Time) (*domainauth.ServiceIdentity, error) {
	v.called++
	return v.ident, v.err
}
func defaultIdentity() *domainauth.ServiceIdentity {
	return &domainauth.ServiceIdentity{ServicePrincipal: domainauth.ServicePrincipal{ID: "id", Consumer: "dubbridge", WorkspaceID: "ws"}, DubBridgeOrgID: "org", DubBridgeProjectID: "project"}
}
func TestDubBridgeServiceAuthIdentityAndScope(t *testing.T) {
	token, _, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	verifier := &verifierStub{ident: defaultIdentity()}
	handler := DubBridgeServiceAuth(verifier, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ident, ok := VerifiedDubBridgeIdentity(r.Context())
		if !ok || ident.ID != "id" {
			t.Fatal("no trusted identity")
		}
		if !AuthorizeDubBridgeSource(ident, DubBridgeSourceScope{Product: "dubbridge", OrgID: "org", ProjectID: "project"}) {
			t.Fatal("valid scope denied")
		}
		for _, source := range []DubBridgeSourceScope{
			{Product: "different", OrgID: "org", ProjectID: "project"},
			{Product: "dubbridge", OrgID: "other", ProjectID: "project"},
			{Product: "dubbridge", OrgID: "org", ProjectID: "other"},
			{Product: "dubbridge", OrgID: "org", ProjectID: ""},
		} {
			if AuthorizeDubBridgeSource(ident, source) {
				t.Fatalf("unauthorized scope allowed: %+v", source)
			}
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	req := httptest.NewRequest(http.MethodPost, "/integrations/dubbridge/operations", strings.NewReader(s2sEnvelope(testSource("org", "project"), `{}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || verifier.called != 1 {
		t.Fatalf("status %d, called %d", rec.Code, verifier.called)
	}
	if _, ok := VerifiedDubBridgeIdentity(context.Background()); ok {
		t.Fatal("identity leaked to unrelated context")
	}
	ident := defaultIdentity()
	ident.DubBridgeProjectID = ""
	if !AuthorizeDubBridgeSource(*ident, DubBridgeSourceScope{Product: "dubbridge", OrgID: "org"}) {
		t.Fatal("org-scope denied")
	}
	if AuthorizeDubBridgeSource(*ident, DubBridgeSourceScope{Product: "dubbridge", OrgID: "org", ProjectID: "project"}) {
		t.Fatal("org-only principal was allowed project-scoped request")
	}
}
func TestDubBridgeServiceAuthRejectsHumanAndMalformed(t *testing.T) {
	token, _, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"missing": "", "jwt": "Bearer aaa.bbb.ccc", "basic": "Basic abc", "whitespace": "Bearer " + token + " ",
		"comma": "Bearer " + token + ",Bearer something", "doubleBearer": "Bearer Bearer " + token,
	} {
		t.Run(name, func(t *testing.T) {
			v := &verifierStub{ident: defaultIdentity()}
			h := DubBridgeServiceAuth(v, nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler invoked") }))
			req := httptest.NewRequest(http.MethodGet, "/integrations/dubbridge/operations/op123", nil)
			if value != "" {
				req.Header.Set("Authorization", value)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized || v.called != 0 {
				t.Fatalf("status=%d called=%d", rec.Code, v.called)
			}
			var obj map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &obj); err != nil {
				t.Fatal(err)
			}
			if obj["error_code"] != "UNAUTHENTICATED" {
				t.Fatalf("error response %s", rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), token) {
				t.Fatal("secret leaked")
			}
		})
	}
}
func TestDubBridgeServiceAuthUnavailableAndForbidden(t *testing.T) {
	token, _, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		verifier *verifierStub
		want     int
		code     string
	}{
		{"nilVerifier", nil, 503, "PROVIDER_UNAVAILABLE"},
		{"dbUnavailable", &verifierStub{err: domainauth.ErrServiceIdentityUnavailable}, 503, "PROVIDER_UNAVAILABLE"},
		{"invalid", &verifierStub{err: domainauth.ErrServiceIdentityDenied}, 401, "UNAUTHENTICATED"},
		{"missingScope", &verifierStub{ident: &domainauth.ServiceIdentity{ServicePrincipal: domainauth.ServicePrincipal{ID: "id", Consumer: "dubbridge", WorkspaceID: "ws"}}}, 403, "FORBIDDEN"},
		{"otherConsumer", &verifierStub{ident: &domainauth.ServiceIdentity{ServicePrincipal: domainauth.ServicePrincipal{ID: "id", Consumer: "other", WorkspaceID: "ws"}, DubBridgeOrgID: "org"}}, 403, "FORBIDDEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := DubBridgeServiceAuth(tt.verifier, nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("called") }))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status %d expected %d", rec.Code, tt.want)
			}
			if !strings.Contains(rec.Body.String(), tt.code) {
				t.Fatalf("code %s absent in %s", tt.code, rec.Body.String())
			}
		})
	}
}
func TestDubBridgeServiceAuthPostBodyBounded(t *testing.T) {
	token, _, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		size  int
		ctype string
		want  int
		hit   bool
	}{
		{DubBridgeS2SMaxRequestBytes, "application/json", 400, false},
		{DubBridgeS2SMaxRequestBytes + 1, "application/json", 413, false},
		{10, "text/plain", 415, false},
	} {
		called := false
		h := DubBridgeServiceAuth(&verifierStub{ident: defaultIdentity()}, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != tc.size {
				t.Fatalf("body size %d: %v", len(body), err)
			}
			w.WriteHeader(202)
		}))
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", tc.size)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", tc.ctype)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want || called != tc.hit {
			t.Fatalf("size=%d status=%d called=%t; expected %d/%t", tc.size, rec.Code, called, tc.want, tc.hit)
		}
	}
}
func TestDubBridgeServiceAuthNoHandler(t *testing.T) {
	token, _, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	h := DubBridgeServiceAuth(&verifierStub{ident: defaultIdentity()}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 503 {
		t.Fatalf("nil handler: %d", rec.Code)
	}
}
