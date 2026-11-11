package integration

import (
	"crypto/sha256"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"
)

// B2.B4 uses a deliberately SMALL, deny-by-default JSON-schema subset for
// operator-provisioned admission input. This is NOT Fenix ToolRegistry's
// execution-time param/permission validator. Unsupported keywords are fatal.
const s2sMaxPolicySchemaDepth = 8

func normalizeS2SCapabilitySchema(raw []byte) ([]byte, [sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	if _, err := CanonicalOperationDigest(raw); err != nil {
		return nil, zero, ErrCapabilityConfiguration
	}
	root, err := decodeCanonicalOperationObject(raw)
	if err != nil || !validS2SSchemaNode(root, 0) || root["type"] != "object" {
		return nil, zero, ErrCapabilityConfiguration
	}
	canonical, err := json.Marshal(root)
	if err != nil {
		return nil, zero, ErrCapabilityConfiguration
	}
	return canonical, sha256.Sum256(canonical), nil
}

func validS2SSchemaNode(node map[string]any, depth int) bool {
	if depth > s2sMaxPolicySchemaDepth || len(node) == 0 {
		return false
	}
	typ, ok := node["type"].(string)
	if !ok {
		return false
	}
	switch typ {
	case "object":
		return validS2SObjectSchema(node, depth)
	case "array":
		return validS2SArraySchema(node, depth)
	case "string":
		return validS2SStringSchema(node)
	case "integer", "number", "boolean", "null":
		return len(node) == 1
	default:
		return false
	}
}

func validS2SObjectSchema(node map[string]any, depth int) bool {
	if !onlyS2SSchemaKeys(node, "type", "properties", "required", "additionalProperties") ||
		node["additionalProperties"] != false {
		return false
	}
	properties, ok := node["properties"].(map[string]any)
	if !ok || len(properties) == 0 || len(properties) > 64 {
		return false
	}
	for name, raw := range properties {
		child, isObject := raw.(map[string]any)
		if name == "" || len(name) > 128 || strings.ContainsRune(name, '\x00') ||
			!isObject || !validS2SSchemaNode(child, depth+1) {
			return false
		}
	}
	required, exists := node["required"]
	if !exists {
		return true
	}
	values, ok := required.([]any)
	if !ok || len(values) > len(properties) {
		return false
	}
	seen := make(map[string]bool, len(values))
	for _, raw := range values {
		name, isString := raw.(string)
		if !isString || seen[name] {
			return false
		}
		if _, found := properties[name]; !found {
			return false
		}
		seen[name] = true
	}
	return true
}

func validS2SArraySchema(node map[string]any, depth int) bool {
	if !onlyS2SSchemaKeys(node, "type", "items", "maxItems") {
		return false
	}
	items, ok := node["items"].(map[string]any)
	max, limited := intS2SSchemaConstraint(node["maxItems"], 256)
	return ok && limited && max > 0 && validS2SSchemaNode(items, depth+1)
}

func validS2SStringSchema(node map[string]any) bool {
	if !onlyS2SSchemaKeys(node, "type", "minLength", "maxLength") {
		return false
	}
	limit, exists := node["maxLength"]
	if !exists {
		return false
	}
	maxLength, ok := intS2SSchemaConstraint(limit, 4096)
	if !ok {
		return false
	}
	if min, present := node["minLength"]; present {
		minLength, valid := intS2SSchemaConstraint(min, 4096)
		return valid && minLength <= maxLength
	}
	return true
}

func onlyS2SSchemaKeys(node map[string]any, allowed ...string) bool {
	for name := range node {
		ok := false
		for _, valid := range allowed {
			if name == valid {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func intS2SSchemaConstraint(value any, ceiling int) (int, bool) {
	numeric, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	valueInt, err := strconv.Atoi(numeric.String())
	return valueInt, err == nil && valueInt >= 0 && valueInt <= ceiling
}

func validateS2SInputSchema(schemaRaw []byte, input map[string]any) bool {
	root, err := decodeCanonicalOperationObject(schemaRaw)
	if err != nil || !validS2SSchemaNode(root, 0) {
		return false
	}
	return matchesS2SSchemaValue(root, input, 0)
}

func matchesS2SSchemaValue(node map[string]any, value any, depth int) bool {
	if depth > s2sMaxPolicySchemaDepth {
		return false
	}
	switch node["type"] {
	case "object":
		fields, ok := value.(map[string]any)
		return ok && matchesS2SObjectSchema(node, fields, depth)
	case "array":
		items, ok := value.([]any)
		if !ok {
			return false
		}
		max, _ := intS2SSchemaConstraint(node["maxItems"], 256)
		if len(items) > max {
			return false
		}
		element := node["items"].(map[string]any)
		for _, item := range items {
			if !matchesS2SSchemaValue(element, item, depth+1) {
				return false
			}
		}
		return true
	case "string":
		str, ok := value.(string)
		if !ok || !utf8.ValidString(str) {
			return false
		}
		max, _ := intS2SSchemaConstraint(node["maxLength"], 4096)
		if utf8.RuneCountInString(str) > max {
			return false
		}
		if rawMin, exists := node["minLength"]; exists {
			min, _ := intS2SSchemaConstraint(rawMin, 4096)
			return utf8.RuneCountInString(str) >= min
		}
		return true
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		_, err := strconv.ParseInt(number.String(), 10, 64)
		return err == nil
	case "number":
		_, ok := value.(json.Number)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	}
	return false
}

func matchesS2SObjectSchema(node map[string]any, fields map[string]any, depth int) bool {
	specs := node["properties"].(map[string]any)
	if len(fields) > len(specs) {
		return false
	}
	if required, ok := node["required"].([]any); ok {
		for _, raw := range required {
			if _, present := fields[raw.(string)]; !present {
				return false
			}
		}
	}
	for name, value := range fields {
		rawSpec, known := specs[name]
		if !known || !matchesS2SSchemaValue(rawSpec.(map[string]any), value, depth+1) {
			return false
		}
	}
	return true
}
