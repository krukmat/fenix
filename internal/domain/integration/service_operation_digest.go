package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
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
	if !validCanonicalOperationInput(raw) {
		return zero, ErrServiceOperationInvalid
	}
	normalized, err := decodeCanonicalOperationObject(raw)
	if err != nil {
		return zero, ErrServiceOperationInvalid
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return zero, fmt.Errorf("normalize service operation input: %w", err)
	}
	return sha256.Sum256(canonical), nil
}

func validCanonicalOperationInput(raw []byte) bool {
	return len(raw) != 0 && len(raw) <= b2cMaxCanonicalJSONBytes &&
		utf8.Valid(raw) && validJSONUnicodeEscapes(raw)
}

func decodeCanonicalOperationObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	nodes := 0
	value, err := decodeServiceOperationJSON(dec, 0, &nodes)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, ErrServiceOperationInvalid
	}
	_, err = dec.Token()
	if err != io.EOF {
		return nil, ErrServiceOperationInvalid
	}
	return object, nil
}

func decodeServiceOperationJSON(dec *json.Decoder, depth int, nodes *int) (any, error) {
	*nodes += 1
	if depth > b2cMaxJSONDepth || *nodes > b2cMaxJSONNodes {
		return nil, ErrServiceOperationInvalid
	}
	token, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("read service operation JSON token: %w", err)
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
		key, value, err := readServiceOperationField(dec, depth, nodes, fields)
		if err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if err := closeServiceOperationContainer(dec, '}'); err != nil {
		return nil, err
	}
	return fields, nil
}

func readServiceOperationField(
	dec *json.Decoder, depth int, nodes *int, fields map[string]any,
) (string, any, error) {
	token, err := dec.Token()
	if err != nil {
		return "", nil, fmt.Errorf("read service operation JSON key: %w", err)
	}
	key, ok := token.(string)
	if !ok {
		return "", nil, ErrServiceOperationInvalid
	}
	if _, duplicate := fields[key]; duplicate {
		return "", nil, ErrServiceOperationInvalid
	}
	value, err := decodeServiceOperationJSON(dec, depth+1, nodes)
	if err != nil {
		return "", nil, err
	}
	return key, value, nil
}

func closeServiceOperationContainer(dec *json.Decoder, delimiter json.Delim) error {
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("close service operation JSON container: %w", err)
	}
	if token != delimiter {
		return ErrServiceOperationInvalid
	}
	return nil
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
	if err := closeServiceOperationContainer(dec, ']'); err != nil {
		return nil, err
	}
	return values, nil
}

// validJSONUnicodeEscapes rejects escaped lone UTF-16 surrogates before
// encoding/json can normalize them into U+FFFD and alias distinct requests.
// Ordinary escaped backslashes are skipped and valid surrogate pairs survive.
func validJSONUnicodeEscapes(raw []byte) bool {
	inString := false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			inString = !inString
		case '\\':
			if !inString {
				continue
			}
			next, ok := scanJSONUnicodeEscape(raw, i)
			if !ok {
				return false
			}
			i = next
		}
	}
	return true
}

func scanJSONUnicodeEscape(raw []byte, i int) (int, bool) {
	if i+1 >= len(raw) {
		return i, false
	}
	if raw[i+1] != 'u' {
		return i + 1, true
	}
	if i+5 >= len(raw) {
		return i, false
	}
	first, ok := jsonHexCodeUnit(raw[i+2 : i+6])
	if !ok {
		return i, false
	}
	return scanJSONSurrogateContinuation(raw, i, first)
}

func scanJSONSurrogateContinuation(raw []byte, i int, first uint64) (int, bool) {
	if isLowJSONSurrogate(first) {
		return i, false
	}
	if !isHighJSONSurrogate(first) {
		return i + 5, true
	}
	return scanJSONSurrogatePair(raw, i)
}

func scanJSONSurrogatePair(raw []byte, i int) (int, bool) {
	if i+11 >= len(raw) {
		return i, false
	}
	if raw[i+6] != '\\' || raw[i+7] != 'u' {
		return i, false
	}
	second, ok := jsonHexCodeUnit(raw[i+8 : i+12])
	return i + 11, ok && isLowJSONSurrogate(second)
}

func isHighJSONSurrogate(code uint64) bool {
	return code >= 0xd800 && code <= 0xdbff
}

func isLowJSONSurrogate(code uint64) bool {
	return code >= 0xdc00 && code <= 0xdfff
}

func jsonHexCodeUnit(raw []byte) (uint64, bool) {
	value, err := strconv.ParseUint(string(raw), 16, 16)
	return value, err == nil
}
