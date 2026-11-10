package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	b2cMaxCanonicalJSONBytes = 64 << 10
	b2cMaxJSONDepth          = 32
	b2cMaxJSONNodes          = 4096
)

var ErrServiceOperationInvalid = errors.New("invalid service operation request")

// CanonicalOperationDigest hashes normalized bounded JSON, without storing input.
// Object order and insignificant whitespace do not affect the fingerprint.
// Numeric lexical forms (1 vs 1.0) are intentionally distinct: this is not RFC 8785.
func CanonicalOperationDigest(raw []byte) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	if len(raw) == 0 || len(raw) > b2cMaxCanonicalJSONBytes || !utf8.Valid(raw) {
		return zero, ErrServiceOperationInvalid
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	nodes := 0
	value, err := decodeServiceOperationJSON(dec, 0, &nodes)
	if err != nil {
		return zero, ErrServiceOperationInvalid
	}
	if _, isObject := value.(map[string]any); !isObject {
		return zero, ErrServiceOperationInvalid
	}
	if _, err := dec.Token(); err != io.EOF {
		return zero, ErrServiceOperationInvalid
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return zero, fmt.Errorf("normalize service operation input: %w", err)
	}
	return sha256.Sum256(canonical), nil
}

func decodeServiceOperationJSON(dec *json.Decoder, depth int, nodes *int) (any, error) {
	*nodes = *nodes + 1
	if depth > b2cMaxJSONDepth || *nodes > b2cMaxJSONNodes {
		return nil, ErrServiceOperationInvalid
	}
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return token, nil
	}
	switch delim {
	case '{':
		return decodeServiceOperationObject(dec, depth, nodes)
	case '[':
		return decodeServiceOperationArray(dec, depth, nodes)
	default:
		return nil, ErrServiceOperationInvalid
	}
}

func decodeServiceOperationObject(dec *json.Decoder, depth int, nodes *int) (map[string]any, error) {
	fields := make(map[string]any)
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, ErrServiceOperationInvalid
		}
		if _, duplicate := fields[key]; duplicate {
			return nil, ErrServiceOperationInvalid
		}
		value, err := decodeServiceOperationJSON(dec, depth+1, nodes)
		if err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if token, err := dec.Token(); err != nil || token != json.Delim('}') {
		return nil, ErrServiceOperationInvalid
	}
	return fields, nil
}

func decodeServiceOperationArray(dec *json.Decoder, depth int, nodes *int) ([]any, error) {
	values := make([]any, 0)
	for dec.More() {
		value, err := decodeServiceOperationJSON(dec, depth+1, nodes)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if token, err := dec.Token(); err != nil || token != json.Delim(']') {
		return nil, ErrServiceOperationInvalid
	}
	return values, nil
}
