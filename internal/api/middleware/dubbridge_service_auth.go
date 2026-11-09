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
type dubbridgeAuthorizedScopeContextKey struct{}

// DubBridgeTokenAuthenticator must authenticate against current Fenix SQLite
// principal, credential, revocation, expiry and workspace binding state.
type DubBridgeTokenAuthenticator interface {
	AuthenticateServiceToken(context.Context, string, time.Time) (*domainauth.ServiceIdentity, error)
}

// VerifiedDubBridgeIdentity is populated only after mandatory source authorization.
// It is separate from all human JWT context keys.
func VerifiedDubBridgeIdentity(ctx context.Context) (domainauth.ServiceIdentity, bool) {
	ident, ok := ctx.Value(dubbridgeIdentityContextKey{}).(domainauth.ServiceIdentity)
	return ident, ok
}

// AuthorizedDubBridgeScope is the verified Fenix-owned authorization scope, never
// the caller's untrusted source payload or operation ID.
func AuthorizedDubBridgeScope(ctx context.Context) (DubBridgeOperationScope, bool) {
	scope, ok := ctx.Value(dubbridgeAuthorizedScopeContextKey{}).(DubBridgeOperationScope)
	return scope, ok
}

// DubBridgeServiceAuth composes the only supported isolated S2S ingress.
// Both authentication and authorization must succeed before next is called.
// The resolver may be nil: POST can be authorized, but GET/status fails closed.
// No route is registered here or in the production human API router.
func DubBridgeServiceAuth(verifier DubBridgeTokenAuthenticator, resolver DubBridgeOperationResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isNilDubBridgeDependency(verifier) || isNilDubBridgeDependency(next) {
			writeDubBridgeS2SError(w, http.StatusServiceUnavailable)
			return
		}
		ident, status := authenticatedDubBridgeIdentity(r, verifier)
		if status != 0 {
			writeDubBridgeS2SError(w, status)
			return
		}
		scope := scopeFromFenixIdentity(ident)
		if status = authorizeDubBridgeRequest(w, r, ident, scope, resolver); status != 0 {
			writeDubBridgeS2SError(w, status)
			return
		}
		ctx := context.WithValue(r.Context(), dubbridgeIdentityContextKey{}, ident)
		ctx = context.WithValue(ctx, dubbridgeAuthorizedScopeContextKey{}, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authenticatedDubBridgeIdentity(r *http.Request, verifier DubBridgeTokenAuthenticator) (domainauth.ServiceIdentity, int) {
	if !supportedDubBridgeMethod(r.Method) {
		return domainauth.ServiceIdentity{}, http.StatusMethodNotAllowed
	}
	bearer, ok := dubbridgeBearer(r)
	if !ok {
		return domainauth.ServiceIdentity{}, http.StatusUnauthorized
	}
	ident, err := verifier.AuthenticateServiceToken(r.Context(), bearer, time.Now().UTC())
	if errors.Is(err, domainauth.ErrServiceIdentityUnavailable) {
		return domainauth.ServiceIdentity{}, http.StatusServiceUnavailable
	}
	if err != nil || ident == nil {
		return domainauth.ServiceIdentity{}, http.StatusUnauthorized
	}
	if !validDubBridgeIdentity(*ident) {
		return domainauth.ServiceIdentity{}, http.StatusForbidden
	}
	return *ident, 0
}

func supportedDubBridgeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodGet:
		return true
	default:
		return false
	}
}

func isNilDubBridgeDependency(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func dubbridgeBearer(r *http.Request) (string, bool) {
	const bearerPrefix = "Bearer "
	if len(r.Header.Values(authorizationHeader)) != 1 {
		return "", false
	}
	raw := r.Header.Get(authorizationHeader)
	if !strings.HasPrefix(raw, bearerPrefix) || strings.Count(raw, bearerPrefix) != 1 {
		return "", false
	}
	bearer := strings.TrimPrefix(raw, bearerPrefix)
	return bearer, domainauth.ParseServiceToken(bearer) == nil
}

func validDubBridgeIdentity(ident domainauth.ServiceIdentity) bool {
	return ident.Consumer == "dubbridge" && ident.ID != "" && ident.WorkspaceID != "" && ident.DubBridgeOrgID != ""
}

func authorizeDubBridgeRequest(w http.ResponseWriter, r *http.Request, ident domainauth.ServiceIdentity, scope DubBridgeOperationScope, resolver DubBridgeOperationResolver) int {
	if r.Method == http.MethodGet {
		return authorizeDubBridgeStatus(r, scope, resolver)
	}
	if r.Header.Get(contentTypeHeader) != jsonMediaType {
		return http.StatusUnsupportedMediaType
	}
	if r.Body == nil {
		return http.StatusBadRequest
	}
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, DubBridgeS2SMaxRequestBytes))
	_ = r.Body.Close()
	if err != nil {
		return http.StatusRequestEntityTooLarge
	}
	source, err := parseDubBridgeEnvelopeSource(payload)
	if err != nil {
		return http.StatusBadRequest
	}
	if !AuthorizeDubBridgeSource(ident, source) {
		return http.StatusForbidden
	}
	r.Body = io.NopCloser(bytes.NewReader(payload))
	r.ContentLength = int64(len(payload))
	return 0
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
	w.Header().Set(contentTypeHeader, jsonMediaType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(`{"contract_version":"1","error_code":"` + code + `","message":"Request rejected"}`)); err != nil {
		return
	}
}
