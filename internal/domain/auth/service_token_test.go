package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestServiceTokenIssueAndVerify(t *testing.T) {
	t.Parallel()
	secret, digest, err := GenerateServiceToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(secret) != serviceTokenLength || !strings.HasPrefix(secret, ServiceTokenPrefix) {
		t.Fatal("invalid generated token shape")
	}
	if err := ParseServiceToken(secret); err != nil {
		t.Fatalf("generated token parse: %v", err)
	}
	if digest != ServiceTokenDigest(sha256.Sum256([]byte(secret))) {
		t.Fatal("issued digest mismatch")
	}
	if !VerifyServiceToken(secret, digest) {
		t.Fatal("issued token rejected")
	}
	parsedDigest, err := DigestServiceToken(secret)
	if err != nil || parsedDigest != digest {
		t.Fatal("digest round trip failed")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(secret, ServiceTokenPrefix))
	if err != nil || len(payload) != serviceTokenRandomBytes {
		t.Fatal("expected 256 random bits")
	}
	var wrong ServiceTokenDigest
	if VerifyServiceToken(secret, wrong) {
		t.Fatal("accepted zero digest")
	}
	wrong = digest
	wrong[0] ^= 0x80
	if VerifyServiceToken(secret, wrong) {
		t.Fatal("accepted different digest")
	}
}

func TestServiceTokenUnique(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for i := 0; i < 96; i++ {
		secret, _, err := GenerateServiceToken()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if seen[secret] {
			t.Fatal("duplicate random service token")
		}
		seen[secret] = true
	}
}

func TestServiceTokenRejectsMalformed(t *testing.T) {
	t.Parallel()
	valid, digest, err := GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	badEnd := valid[:len(valid)-1] + "+"
	padded := valid[:len(valid)-1] + "="
	invalidLast := valid[:len(valid)-1] + mutateUnusedBase64Bits(valid[len(valid)-1])
	cases := map[string]string{
		"empty":             "",
		"missing prefix":    strings.TrimPrefix(valid, ServiceTokenPrefix),
		"too short":         valid[:len(valid)-1],
		"extra content":     valid + "1",
		"whitespace":        " " + valid,
		"trailing space":    valid[:len(valid)-1] + " ",
		"newline":           valid[:len(valid)-1] + "\n",
		"jwt":               "aaa.bbb.ccc",
		"wrong version":     strings.Replace(valid, "v1_", "v2_", 1),
		"wrong alphabet":    badEnd,
		"padding":           padded,
		"noncanonical bits": invalidLast,
	}
	for label, raw := range cases {
		t.Run(label, func(t *testing.T) {
			if err := ParseServiceToken(raw); !errors.Is(err, ErrInvalidServiceToken) {
				t.Fatalf("parse malformed token: got %v", err)
			}
			if _, err := DigestServiceToken(raw); !errors.Is(err, ErrInvalidServiceToken) {
				t.Fatalf("digest malformed token: got %v", err)
			}
			if VerifyServiceToken(raw, digest) {
				t.Fatal("accepted malformed token")
			}
		})
	}
}

// For a 32-byte base64url payload, only the high four bits of the last
// sextet carry data. Toggle one unused low bit: decoding must fail canonicality.
func mutateUnusedBase64Bits(ch byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	i := strings.IndexByte(alphabet, ch)
	if i < 0 {
		panic("unexpected base64url character")
	}
	return string(alphabet[i^1])
}

func TestServiceTokenErrorsNeverEchoBearer(t *testing.T) {
	t.Parallel()
	secret, _, err := GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	bad := secret + "Z"
	if got := ParseServiceToken(bad); got == nil || strings.Contains(got.Error(), bad) {
		t.Fatal("error exposed secret or accepted malformed bearer")
	}
}

func TestServicePrincipalNotHuman(t *testing.T) {
	t.Parallel()
	field, ok := reflect.TypeOf(ServicePrincipal{}).FieldByName("UserID")
	if ok {
		t.Fatalf("service identity must not have human UserID: %s", field.Name)
	}
}
