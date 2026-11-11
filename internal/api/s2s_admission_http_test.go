package api

import (
	"context"
	"encoding/json"
	"io"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

type fakeS2SHTTPAdmission struct {
	calls   int
	lastKey string
	lastJWT string
	body    []byte
	result  integration.S2SAdmissionReceipt
	err     error
}

func (f *fakeS2SHTTPAdmission) AdmitWithIdempotencyKey(
	_ context.Context, bearer string, raw []byte, operationID string, _ time.Time,
) (integration.S2SAdmissionReceipt, error) {
	f.calls++
	f.lastKey = operationID
	f.lastJWT = bearer
	f.body = append([]byte(nil), raw...)
	return f.result, f.err
}

const s2sTestOperationID = "operation_00000001"
const s2sTestTraceID = "trace_b4_000001"
const s2sTestEnvelope = `{"contract_version":"1","operation_id":"operation_00000001","trace_id":"trace_b4_000001"}`

func testS2SToken(t *testing.T) string {
	t.Helper()
	token, _, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func testS2SRequest(t *testing.T, token string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, DormantS2SAdmissionPath, strings.NewReader(s2sTestEnvelope))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", s2sTestOperationID)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func testS2SBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if recorder.Header().Get("Cache-Control") != "no-store" ||
		recorder.Header().Get("X-Content-Type-Options") != "nosniff" ||
		recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unsafe response headers: %v", recorder.Header())
	}
	var v map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &v); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}
	return v
}

func TestB4DormantS2SHTTPReceiptAndReplay(t *testing.T) {
	token := testS2SToken(t)
	for _, tc := range []struct {
		replayed bool
		status   int
	}{
		{false, http.StatusAccepted},
		{true, http.StatusOK},
	} {
		fake := &fakeS2SHTTPAdmission{
			result: integration.S2SAdmissionReceipt{
				OperationID: s2sTestOperationID, TraceID: s2sTestTraceID, Replayed: tc.replayed,
			},
		}
		req := testS2SRequest(t, token)
		w := httptest.NewRecorder()
		NewDormantS2SAdmissionHandler(fake).ServeHTTP(w, req)
		if w.Code != tc.status || fake.calls != 1 ||
			fake.lastKey != s2sTestOperationID || fake.lastJWT != token ||
			string(fake.body) != s2sTestEnvelope {
			t.Fatalf("unexpected HTTP admission: status=%d calls=%d key=%q", w.Code, fake.calls, fake.lastKey)
		}
		ack := testS2SBody(t, w)
		if len(ack) != 4 || ack["contract_version"] != "1" ||
			ack["operation_id"] != s2sTestOperationID ||
			ack["trace_id"] != s2sTestTraceID || ack["replayed"] != tc.replayed {
			t.Fatalf("unsafe acknowledgement: %+v", ack)
		}
	}
}

func TestB4DormantS2SHTTPRejectsBeforeInvokingAdmission(t *testing.T) {
	token := testS2SToken(t)
	for _, tc := range []struct {
		name       string
		change     func(*http.Request)
		wantStatus int
	}{
		{"GET", func(r *http.Request) { r.Method = http.MethodGet }, http.StatusForbidden},
		{"wrong path", func(r *http.Request) { r.URL.Path = "/unknown" }, http.StatusForbidden},
		{"unexpected query", func(r *http.Request) { r.URL.RawQuery = "debug=1" }, http.StatusForbidden},
		{"missing bearer", func(r *http.Request) { r.Header.Del("Authorization") }, http.StatusForbidden},
		{"human JWT", func(r *http.Request) { r.Header.Set("Authorization", "Bearer human.jwt.value") }, http.StatusForbidden},
		{"duplicate bearer", func(r *http.Request) { r.Header.Add("Authorization", "Bearer "+token) }, http.StatusForbidden},
		{"missing media type", func(r *http.Request) { r.Header.Del("Content-Type") }, http.StatusBadRequest},
		{"wrong media type", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusBadRequest},
		{"unsupported charset", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=utf-16") }, http.StatusBadRequest},
		{"compressed body", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, http.StatusBadRequest},
		{"missing key", func(r *http.Request) { r.Header.Del("Idempotency-Key") }, http.StatusBadRequest},
		{"duplicate key", func(r *http.Request) { r.Header.Add("Idempotency-Key", s2sTestOperationID) }, http.StatusBadRequest},
		{"oversized body", func(r *http.Request) {
			r.Body = http.NoBody
			r.ContentLength = maxS2SAdmissionJSONBytes + 1
		}, http.StatusBadRequest},
		{"chunked oversized body", func(r *http.Request) {
			r.Body = ioNopCloser(strings.NewReader(strings.Repeat("x", int(maxS2SAdmissionJSONBytes)+1)))
			r.ContentLength = -1
		}, http.StatusBadRequest},
		{"empty body", func(r *http.Request) { r.Body = http.NoBody; r.ContentLength = 0 }, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeS2SHTTPAdmission{}
			req := testS2SRequest(t, token)
			tc.change(req)
			w := httptest.NewRecorder()
			NewDormantS2SAdmissionHandler(fake).ServeHTTP(w, req)
			if w.Code != tc.wantStatus || fake.calls != 0 {
				t.Fatalf("invalid request reached admission: status=%d calls=%d", w.Code, fake.calls)
			}
			obj := testS2SBody(t, w)
			if len(obj) != 3 || obj["message"] != "Request could not be accepted" {
				t.Fatalf("unsafe rejection: %+v", obj)
			}
		})
	}
}

// Keep the request test fixtures readable without depending on an HTTP server.
func ioNopCloser(reader *strings.Reader) io.ReadCloser { return io.NopCloser(reader) }

func TestB4DormantS2SHTTPClassifiesAndRedactsErrors(t *testing.T) {
	token := testS2SToken(t)
	for _, tc := range []struct {
		err    error
		code   string
		status int
	}{
		{integration.ErrS2SAdmissionInvalid, "INVALID_REQUEST", 400},
		{integration.ErrS2SAdmissionDenied, "REQUEST_DENIED", 403},
		{integration.ErrS2SAdmissionConflict, "OPERATION_CONFLICT", 409},
		{integration.ErrS2SAdmissionUnavailable, "PROVIDER_UNAVAILABLE", 503},
		{errors.New("SQL principal_id token=supersecret"), "PROVIDER_UNAVAILABLE", 503},
	} {
		fake := &fakeS2SHTTPAdmission{err: tc.err}
		w := httptest.NewRecorder()
		NewDormantS2SAdmissionHandler(fake).ServeHTTP(w, testS2SRequest(t, token))
		v := testS2SBody(t, w)
		if w.Code != tc.status || fake.calls != 1 || len(v) != 3 ||
			v["contract_version"] != "1" || v["error_code"] != tc.code ||
			v["message"] != "Request could not be accepted" ||
			strings.Contains(w.Body.String(), "principal_id") {
			t.Fatalf("unsafe error projection: status=%d body=%s", w.Code, w.Body.String())
		}
	}
}

func TestB4DormantS2SHTTPRejectsInvalidReceipt(t *testing.T) {
	fake := &fakeS2SHTTPAdmission{
		result: integration.S2SAdmissionReceipt{OperationID: "bad", TraceID: s2sTestTraceID},
	}
	w := httptest.NewRecorder()
	NewDormantS2SAdmissionHandler(fake).ServeHTTP(w, testS2SRequest(t, testS2SToken(t)))
	v := testS2SBody(t, w)
	if w.Code != http.StatusServiceUnavailable || v["error_code"] != "PROVIDER_UNAVAILABLE" {
		t.Fatalf("invalid receipt treated as success: %d %+v", w.Code, v)
	}
}

func TestB4DormantS2SHTTPIsAbsentFromNormalRouter(t *testing.T) {
	db := mustOpenAPITestDB(t)
	router := mustNewRouterWithConfig(db, testCfg())
	req := httptest.NewRequest(http.MethodPost, DormantS2SAdmissionPath, strings.NewReader(s2sTestEnvelope))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusOK || w.Code == http.StatusAccepted ||
		w.Header().Get("Content-Type") == "application/json" &&
			strings.Contains(w.Body.String(), "operation_id") {
		t.Fatalf("S2S route unexpectedly exposed: %d %s", w.Code, w.Body.String())
	}
}
