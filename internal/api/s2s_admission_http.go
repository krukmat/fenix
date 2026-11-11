package api

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

// DormantS2SAdmissionPath is the proposed DubBridge POST admission path.
// The normal NewRouter / NewRouterWithRuntime MUST NOT register it until
// B2.B4 operator policy, T4 security and Go/Rust E2E gates are approved.
const DormantS2SAdmissionPath = "/integrations/dubbridge/operations"

const maxS2SAdmissionJSONBytes int64 = 64 * 1024

// S2SHTTPAdmission is the smallest in-process boundary needed by the HTTP
// handler. The authenticated principal, scope, capability and operation
// reservation are all evaluated inside Fenix's admission service.
type S2SHTTPAdmission interface {
	AdmitWithIdempotencyKey(
		context.Context, string, []byte, string, time.Time,
	) (integration.S2SAdmissionReceipt, error)
}

type dormantS2SAdmissionHandler struct {
	admission S2SHTTPAdmission
}

// NewDormantS2SAdmissionHandler constructs a *detached* handler.
// No production router invokes this function or mounts the S2S path.
// Mounting it without separately reviewed rate limits, operator credentials
// and audited capability configuration is NOT an approved activation.
func NewDormantS2SAdmissionHandler(admission S2SHTTPAdmission) http.Handler {
	return &dormantS2SAdmissionHandler{admission: admission}
}

func (h *dormantS2SAdmissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.URL.Path != DormantS2SAdmissionPath || r.Method != http.MethodPost || r.URL.RawQuery != "" {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionDenied)
		return
	}
	bearer, ok := singleCanonicalS2SBearer(r.Header.Values("Authorization"))
	if !ok {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionDenied)
		return
	}
	if h == nil || h.admission == nil {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionDenied)
		return
	}
	if !validS2SAdmissionContentType(r.Header.Get("Content-Type")) ||
		!validS2SAdmissionContentEncoding(r.Header.Get("Content-Encoding")) {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionInvalid)
		return
	}
	operationKeys := r.Header.Values("Idempotency-Key")
	if len(operationKeys) != 1 {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionInvalid)
		return
	}
	if r.ContentLength > maxS2SAdmissionJSONBytes || r.Body == nil {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionInvalid)
		return
	}
	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxS2SAdmissionJSONBytes))
	if err != nil || len(body) == 0 {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionInvalid)
		return
	}

	// The caller cannot supply an authenticated identity or a trusted digest.
	// The service first authenticates the persisted Fenix principal, then
	// compares the header with B1's validated operation_id before SQLite.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	receipt, err := h.admission.AdmitWithIdempotencyKey(
		ctx, bearer, body, operationKeys[0], time.Now().UTC(),
	)
	if err != nil {
		writeS2SAdmissionFailure(w, err)
		return
	}
	status, ack, err := integration.ProjectS2SAdmissionWireAck(receipt)
	if err != nil {
		writeS2SAdmissionFailure(w, integration.ErrS2SAdmissionUnavailable)
		return
	}
	writeS2SAdmissionJSON(w, status, ack)
}

func singleCanonicalS2SBearer(values []string) (string, bool) {
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if domainauth.ParseServiceToken(token) != nil {
		return "", false
	}
	return token, true
}

func validS2SAdmissionContentType(value string) bool {
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil || mediaType != "application/json" {
		return false
	}
	if len(params) == 0 {
		return true
	}
	return len(params) == 1 && strings.EqualFold(params["charset"], "utf-8")
}

func validS2SAdmissionContentEncoding(value string) bool {
	return value == "" || strings.EqualFold(value, "identity")
}

func writeS2SAdmissionFailure(w http.ResponseWriter, err error) {
	status, response := integration.ProjectS2SAdmissionWireFailure(err)
	writeS2SAdmissionJSON(w, status, response)
}

func writeS2SAdmissionJSON(w http.ResponseWriter, status int, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		status = http.StatusServiceUnavailable
		raw = []byte(`{"contract_version":"1","error_code":"PROVIDER_UNAVAILABLE","message":"Request could not be accepted"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(raw, '\n'))
}
