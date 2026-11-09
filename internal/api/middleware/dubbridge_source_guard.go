package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

// DubBridgeSourceScope is an UNTRUSTED assertion from the request envelope.
type DubBridgeSourceScope struct {
	Product   string `json:"product"`
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id,omitempty"`
}

// DubBridgeOperationScope is constructed exclusively from a Fenix-stored identity.
type DubBridgeOperationScope struct {
	PrincipalID string
	WorkspaceID string
	OrgID       string
	ProjectID   string
}

// DubBridgeStoredOperation must be loaded by the operation repository from Fenix.
// The guard compares all returned persisted ownership columns independently.
type DubBridgeStoredOperation struct {
	OperationID string
	DubBridgeOperationScope
}

var ErrDubBridgeOperationUnavailable = errors.New("scoped operation store unavailable")

// DubBridgeOperationResolver MUST query with all supplied scope predicates.
// The guard rechecks the returned persisted record to reject faulty lookups.
// B2.C has not implemented a real resolver; a nil resolver denies GET/status.
type DubBridgeOperationResolver interface {
	LookupDubBridgeOperation(context.Context, string, DubBridgeOperationScope) (DubBridgeStoredOperation, error)
}

func scopeFromFenixIdentity(ident domainauth.ServiceIdentity) DubBridgeOperationScope {
	return DubBridgeOperationScope{PrincipalID: ident.ID, WorkspaceID: ident.WorkspaceID, OrgID: ident.DubBridgeOrgID, ProjectID: ident.DubBridgeProjectID}
}

// AuthorizeDubBridgeSource checks exact organization and project binding, with
// an empty project authorized only for an org-only stored principal.
func AuthorizeDubBridgeSource(ident domainauth.ServiceIdentity, source DubBridgeSourceScope) bool {
	if !validDubBridgeIdentity(ident) || source.Product != "dubbridge" || source.OrgID == "" || source.OrgID != ident.DubBridgeOrgID {
		return false
	}
	return source.ProjectID == ident.DubBridgeProjectID
}

type dubbridgeEnvelope struct {
	ContractVersion    string               `json:"contract_version"`
	OperationID        string               `json:"operation_id"`
	CapabilityKey      string               `json:"capability_key"`
	InputSchemaVersion string               `json:"input_schema_version"`
	TraceID            string               `json:"trace_id"`
	Source             DubBridgeSourceScope `json:"source"`
	Input              json.RawMessage      `json:"input"`
}

// parseDubBridgeEnvelopeSource validates one bounded JSON object and never
// treats untrusted source fields as authoritative service principal identity.
func parseDubBridgeEnvelopeSource(payload []byte) (DubBridgeSourceScope, error) {
	fields, err := decodeDubBridgeUniqueObject(payload)
	if err != nil {
		return DubBridgeSourceScope{}, err
	}
	if _, sourceErr := decodeDubBridgeUniqueObject(fields["source"]); sourceErr != nil {
		return DubBridgeSourceScope{}, sourceErr
	}
	envelope, err := decodeDubBridgeEnvelope(payload)
	if err != nil {
		return DubBridgeSourceScope{}, err
	}
	if !validDubBridgeEnvelope(envelope) || !validDubBridgeInputObject(envelope.Input) {
		return DubBridgeSourceScope{}, errors.New("incomplete request")
	}
	if !validDubBridgeSourceShape(envelope.Source) {
		return DubBridgeSourceScope{}, errors.New("missing source")
	}
	return envelope.Source, nil
}

func validDubBridgeSourceShape(source DubBridgeSourceScope) bool {
	return source.Product != "" && source.OrgID != ""
}

func decodeDubBridgeEnvelope(payload []byte) (dubbridgeEnvelope, error) {
	var envelope dubbridgeEnvelope
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&envelope); err != nil {
		return dubbridgeEnvelope{}, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return dubbridgeEnvelope{}, errors.New("invalid trailing request")
	}
	return envelope, nil
}

// decodeDubBridgeUniqueObject rejects duplicate top-level/source keys, avoiding
// two consumers interpreting ambiguous source assertions differently.
func decodeDubBridgeUniqueObject(raw []byte) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("expected JSON object")
	}
	fields, err := collectDubBridgeUniqueKeys(dec)
	if err != nil {
		return nil, err
	}
	token, err = dec.Token()
	if err != nil || token != json.Delim('}') {
		return nil, errors.New("invalid JSON object")
	}
	var extra any
	if trailingErr := dec.Decode(&extra); trailingErr != io.EOF {
		return nil, errors.New("invalid trailing JSON")
	}
	return fields, nil
}

func collectDubBridgeUniqueKeys(dec *json.Decoder) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	for dec.More() {
		if err := appendDubBridgeUniqueKey(dec, fields); err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func appendDubBridgeUniqueKey(dec *json.Decoder, fields map[string]json.RawMessage) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	key, ok := token.(string)
	if !ok {
		return errors.New("invalid JSON key")
	}
	if _, duplicate := fields[key]; duplicate {
		return errors.New("duplicate JSON key")
	}
	var value json.RawMessage
	if decodeErr := dec.Decode(&value); decodeErr != nil {
		return decodeErr
	}
	fields[key] = value
	return nil
}

func validDubBridgeEnvelope(e dubbridgeEnvelope) bool {
	return e.ContractVersion == "1" && e.OperationID != "" && e.CapabilityKey != "" &&
		e.InputSchemaVersion != "" && e.TraceID != "" && len(e.Input) != 0
}

func validDubBridgeInputObject(raw json.RawMessage) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	return obj != nil
}

func authorizeDubBridgeStatus(r *http.Request, scope DubBridgeOperationScope, resolver DubBridgeOperationResolver) int {
	if isNilDubBridgeDependency(resolver) {
		return http.StatusForbidden
	}
	id, ok := dubbridgeStatusID(r)
	if !ok {
		return http.StatusForbidden
	}
	record, err := resolver.LookupDubBridgeOperation(r.Context(), id, scope)
	if errors.Is(err, ErrDubBridgeOperationUnavailable) {
		return http.StatusServiceUnavailable
	}
	if err != nil || !sameDubBridgeOperationScope(record, id, scope) {
		return http.StatusForbidden
	}
	return 0
}

func sameDubBridgeOperationScope(op DubBridgeStoredOperation, id string, scope DubBridgeOperationScope) bool {
	return id == op.OperationID && scope.PrincipalID == op.PrincipalID && scope.WorkspaceID == op.WorkspaceID && scope.OrgID == op.OrgID && scope.ProjectID == op.ProjectID
}

var dubbridgeOperationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,96}$`)

func dubbridgeStatusID(r *http.Request) (string, bool) {
	const prefix = "/integrations/dubbridge/operations/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	if !dubbridgeOperationIDPattern.MatchString(id) {
		return "", false
	}
	if r.URL.RawPath != "" && r.URL.EscapedPath() != r.URL.Path {
		return "", false
	}
	return id, true
}
