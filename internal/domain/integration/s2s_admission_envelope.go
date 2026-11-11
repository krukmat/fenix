package integration

import (
	"crypto/sha256"
	"errors"
	"regexp"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
)

// The B1 request envelope requires canonical hyphenated UUIDs at its S2S
// boundary. C2's internal SQLite fixtures deliberately use opaque scope IDs;
// this lexical check belongs only to the B2.E external admission preflight.
var s2sEnvelopeUUIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

var ErrS2SAdmissionInvalid = errors.New("invalid service admission request")
var ErrS2SAdmissionDenied = errors.New("service admission unavailable")
var ErrS2SAdmissionConflict = errors.New("service admission conflict")
var ErrS2SAdmissionUnavailable = errors.New("service admission storage unavailable")

// parsedS2SEnvelope is never constructed from untrusted body scope: the
// verified principal supplied by the credential store is authoritative.
type parsedS2SEnvelope struct {
	request ServiceOperationRequest
	input   map[string]any
}

// parseVerifiedS2SEnvelope validates the entire bounded B1 v1 envelope.
// The request hash is always computed inside Fenix; neither a supplied digest
// nor an untrusted workspace/organization selector can override it.
func parseVerifiedS2SEnvelope(raw []byte, identity *domainauth.ServiceIdentity) (parsedS2SEnvelope, error) {
	if identity == nil || identity.Consumer != "dubbridge" || !validOperationScope(scopeFromS2SIdentity(identity)) {
		return parsedS2SEnvelope{}, ErrS2SAdmissionDenied
	}
	digest, err := CanonicalOperationDigest(raw)
	if err != nil {
		return parsedS2SEnvelope{}, ErrS2SAdmissionInvalid
	}
	root, err := decodeCanonicalOperationObject(raw)
	if err != nil {
		return parsedS2SEnvelope{}, ErrS2SAdmissionInvalid
	}
	return buildS2SEnvelope(root, identity, digest)
}

func scopeFromS2SIdentity(identity *domainauth.ServiceIdentity) ServiceOperationScope {
	return ServiceOperationScope{
		PrincipalID: identity.ID,
		WorkspaceID: identity.WorkspaceID,
		OrgID:       identity.DubBridgeOrgID,
		ProjectID:   identity.DubBridgeProjectID,
	}
}

func buildS2SEnvelope(
	root map[string]any, identity *domainauth.ServiceIdentity, digest [sha256.Size]byte,
) (parsedS2SEnvelope, error) {
	if !validS2SEnvelopeFields(root) {
		return parsedS2SEnvelope{}, ErrS2SAdmissionInvalid
	}
	source, ok := root["source"].(map[string]any)
	if !ok || !validS2SSourceFields(source) {
		return parsedS2SEnvelope{}, ErrS2SAdmissionInvalid
	}
	if !s2sSourceMatchesIdentity(source, identity) {
		return parsedS2SEnvelope{}, ErrS2SAdmissionDenied
	}
	input := root["input"].(map[string]any)
	return parsedS2SEnvelope{
		request: ServiceOperationRequest{
			Scope:         scopeFromS2SIdentity(identity),
			OperationID:   root["operation_id"].(string),
			CapabilityKey: root["capability_key"].(string),
			SchemaVersion: root["input_schema_version"].(string),
			TraceID:       root["trace_id"].(string),
			RequestSHA256: digest,
		},
		input: input,
	}, nil
}

func validS2SEnvelopeFields(root map[string]any) bool {
	if len(root) != 7 || root["contract_version"] != "1" {
		return false
	}
	input, ok := root["input"].(map[string]any)
	return ok && len(input) <= 64 && validS2SEnvelopeIdentifiers(root)
}

func validS2SEnvelopeIdentifiers(root map[string]any) bool {
	return validS2SOperationID(root["operation_id"]) &&
		validS2STraceID(root["trace_id"]) &&
		validS2SCapabilityKey(root["capability_key"]) &&
		validS2SSchemaVersion(root["input_schema_version"])
}

func validS2SOperationID(v any) bool {
	id, ok := v.(string)
	return ok && b2cOperationIDPattern.MatchString(id)
}

func validS2STraceID(v any) bool {
	trace, ok := v.(string)
	return ok && b2cTraceIDPattern.MatchString(trace)
}

func validS2SCapabilityKey(v any) bool {
	key, ok := v.(string)
	return ok && len(key) <= 96 && capabilityKeyPattern.MatchString(key)
}

func validS2SSchemaVersion(v any) bool {
	version, ok := v.(string)
	return ok && schemaVersionPattern.MatchString(version)
}

func validS2SSourceFields(source map[string]any) bool {
	if len(source) < 2 || len(source) > 3 || source["product"] != "dubbridge" {
		return false
	}
	org, ok := source["org_id"].(string)
	if !ok || !s2sEnvelopeUUIDPattern.MatchString(org) {
		return false
	}
	if len(source) == 2 {
		return true
	}
	project, ok := source["project_id"].(string)
	return ok && s2sEnvelopeUUIDPattern.MatchString(project)
}

func s2sSourceMatchesIdentity(source map[string]any, identity *domainauth.ServiceIdentity) bool {
	if source["org_id"] != identity.DubBridgeOrgID {
		return false
	}
	project, hasProject := source["project_id"]
	if identity.DubBridgeProjectID == "" {
		return !hasProject
	}
	return hasProject && project == identity.DubBridgeProjectID
}
