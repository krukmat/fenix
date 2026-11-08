package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

// Service tokens never authenticate as human users and cannot be used with
// pkg/auth.ParseJWT. Authentication/authorization of the service principal is
// performed separately by the future B2.A2/A3 persistent verifier.
const (
	ServiceTokenPrefix      = "fenix_s2s_v1_" // #nosec G101 -- public wire prefix, not a credential
	serviceTokenRandomBytes = 32
	serviceTokenEncodedLen  = 43 // unpadded base64url encoding of 32 bytes
	serviceTokenLength      = len(ServiceTokenPrefix) + serviceTokenEncodedLen
)

// ErrInvalidServiceToken intentionally contains no information about a bearer.
var ErrInvalidServiceToken = errors.New("invalid service token")

// ServicePrincipal is a non-human identity. Its fields are trusted only when
// loaded and verified by Fenix's own principal repository; caller-provided JSON
// must never be used to construct an authorized principal.
type ServicePrincipal struct {
	ID          string
	Consumer    string
	WorkspaceID string
}

// ServiceTokenDigest is the fixed-size SHA-256 digest of a high-entropy token.
// This is safe for randomly generated 256-bit tokens, not human passwords.
type ServiceTokenDigest [sha256.Size]byte

// GenerateServiceToken issues 256 random bits from the OS CSPRNG. The plaintext
// token must be shown to an authorized operator once and must never be logged.
// Only its digest should be persisted by B2.A2.
func GenerateServiceToken() (string, ServiceTokenDigest, error) {
	var secret [serviceTokenRandomBytes]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", ServiceTokenDigest{}, errors.New("could not generate service token")
	}
	text := ServiceTokenPrefix + base64.RawURLEncoding.EncodeToString(secret[:])
	for i := range secret {
		secret[i] = 0
	}
	return text, ServiceTokenDigest(sha256.Sum256([]byte(text))), nil
}

// ParseServiceToken enforces a strict, canonical and versioned opaque format.
// It returns only validity, not authorization or a human/tenant identity.
func ParseServiceToken(candidate string) error {
	if len(candidate) != serviceTokenLength || !strings.HasPrefix(candidate, ServiceTokenPrefix) {
		return ErrInvalidServiceToken
	}
	encoded := candidate[len(ServiceTokenPrefix):]
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(decoded) != serviceTokenRandomBytes ||
		base64.RawURLEncoding.EncodeToString(decoded) != encoded {
		return ErrInvalidServiceToken
	}
	return nil
}

// DigestServiceToken hashes a syntactically valid token without returning its
// raw secret. This function is for initial issuance/testing; verification must
// use VerifyServiceToken so equality is checked in constant time.
func DigestServiceToken(candidate string) (ServiceTokenDigest, error) {
	if err := ParseServiceToken(candidate); err != nil {
		return ServiceTokenDigest{}, err
	}
	return ServiceTokenDigest(sha256.Sum256([]byte(candidate))), nil
}

// VerifyServiceToken checks the digest using a timing-safe comparison. This
// primitive does not check principal expiry, revocation or tenant permissions;
// B2.A2/A3 must enforce those checks against authoritative Fenix state.
func VerifyServiceToken(candidate string, stored ServiceTokenDigest) bool {
	if ParseServiceToken(candidate) != nil {
		return false
	}
	actual := sha256.Sum256([]byte(candidate))
	return subtle.ConstantTimeCompare(actual[:], stored[:]) == 1
}
