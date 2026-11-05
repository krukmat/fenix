package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

const DubBridgeS2SMaxRequestBytes = 64 << 10

type dubbridgeIdentityContextKey struct{}

// DubBridgeTokenAuthenticator validates Fenix-owned service tokens. Its
// implementation must authenticate against live SQLite revocation/expiry state.
type DubBridgeTokenAuthenticator interface {
	AuthenticateServiceToken(context.Context, string, time.Time) (*domainauth.ServiceIdentity, error)
}

// VerifiedDubBridgeIdentity returns only a principal authenticated by the
// dedicated S2S middleware. It never reads human UserID JWT context claims.
func VerifiedDubBridgeIdentity(ctx context.Context) (domainauth.ServiceIdentity, bool) {
	ident, ok := ctx.Value(dubbridgeIdentityContextKey{}).(domainauth.ServiceIdentity)
	return ident, ok
}

// DubBridgeSourceScope is untrusted wire context. Domain authorization in
// DubBridge and this Fenix-side trusted-binding comparison are BOTH required.
type DubBridgeSourceScope struct{ Product, OrgID, ProjectID string }

// AuthorizeDubBridgeSource enforces exact org/project binding from Fenix SQLite.
// A principal with no project binding can only access org-scoped requests.
func AuthorizeDubBridgeSource(ident domainauth.ServiceIdentity, source DubBridgeSourceScope) bool {
	if ident.Consumer != "dubbridge" || ident.ID == "" || ident.WorkspaceID == "" ||
		ident.DubBridgeOrgID == "" || source.Product != "dubbridge" ||
		source.OrgID == "" || source.OrgID != ident.DubBridgeOrgID {
		return false
	}
	return source.ProjectID == ident.DubBridgeProjectID
}

// DubBridgeServiceAuth is the isolated S2S boundary. It does NOT register a
// route and MUST NOT be installed inside the existing human /api/v1 subtree.
// Business capabilities remain unregistered and denied in B2.B.
func DubBridgeServiceAuth(verifier DubBridgeTokenAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if verifier == nil || (reflect.ValueOf(verifier).Kind() == reflect.Ptr && reflect.ValueOf(verifier).IsNil()) || next == nil {
			writeDubBridgeS2SError(w, http.StatusServiceUnavailable)
			return
		}
		if len(r.Header.Values("Authorization")) != 1 {
			writeDubBridgeS2SError(w, http.StatusUnauthorized)
			return
		}
		raw := r.Header.Get("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") || strings.Count(raw, "Bearer ") != 1 {
			writeDubBridgeS2SError(w, http.StatusUnauthorized)
			return
		}
		bearer := strings.TrimPrefix(raw, "Bearer ")
		if domainauth.ParseServiceToken(bearer) != nil {
			writeDubBridgeS2SError(w, http.StatusUnauthorized)
			return
		}
		ident, err := verifier.AuthenticateServiceToken(r.Context(), bearer, time.Now().UTC())
		if err != nil || ident == nil {
			if errors.Is(err, domainauth.ErrServiceIdentityUnavailable) {
				writeDubBridgeS2SError(w, http.StatusServiceUnavailable)
			} else {
				writeDubBridgeS2SError(w, http.StatusUnauthorized)
			}
			return
		}
		if ident.Consumer != "dubbridge" || ident.ID == "" || ident.WorkspaceID == "" || ident.DubBridgeOrgID == "" {
			writeDubBridgeS2SError(w, http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" {
				writeDubBridgeS2SError(w, http.StatusUnsupportedMediaType)
				return
			}
			if r.Body == nil {
				writeDubBridgeS2SError(w, http.StatusBadRequest)
				return
			}
			payload, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, DubBridgeS2SMaxRequestBytes))
			_ = r.Body.Close()
			if readErr != nil {
				writeDubBridgeS2SError(w, http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(payload))
			r.ContentLength = int64(len(payload))
		}
		ctx := context.WithValue(r.Context(), dubbridgeIdentityContextKey{}, *ident)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeDubBridgeS2SError(w http.ResponseWriter, status int) {
	code := "INVALID_REQUEST"
	switch status {
	case http.StatusUnauthorized:
		code = "UNAUTHENTICATED"
	case http.StatusForbidden:
		code = "FORBIDDEN"
	case http.StatusServiceUnavailable:
		code = "PROVIDER_UNAVAILABLE"
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"contract_version":"1","error_code":"` + code + `","message":"Request rejected"}`))
}
