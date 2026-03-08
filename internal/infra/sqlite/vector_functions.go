package sqlite

import (
	"database/sql/driver"
	"encoding/json"
	"log"
	"math"

	sqlite "modernc.org/sqlite"
)

const sqliteCosineSimilarityFunc = "cosine_similarity_json"

func init() {
	err := sqlite.RegisterFunction(sqliteCosineSimilarityFunc, &sqlite.FunctionImpl{
		NArgs:         2,
		Deterministic: true,
		Scalar: func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			left, right, ok := parseVectorArgs(args)
			if !ok {
				return float64(0), nil
			}
			return float64(cosineSimilarityFloat64(left, right)), nil
		},
	})
	if err != nil {
		log.Fatalf("register %s: %v", sqliteCosineSimilarityFunc, err)
	}
}

func parseVectorArgs(args []driver.Value) ([]float64, []float64, bool) {
	if len(args) != 2 {
		return nil, nil, false
	}

	left, ok := parseVectorJSONArg(args[0])
	if !ok {
		return nil, nil, false
	}
	right, ok := parseVectorJSONArg(args[1])
	if !ok {
		return nil, nil, false
	}
	if len(left) != len(right) || len(left) == 0 {
		return nil, nil, false
	}
	return left, right, true
}

func parseVectorJSONArg(arg driver.Value) ([]float64, bool) {
	raw, ok := arg.(string)
	if !ok || raw == "" {
		return nil, false
	}

	var vec []float64
	if err := json.Unmarshal([]byte(raw), &vec); err != nil {
		return nil, false
	}
	return vec, len(vec) > 0
}

func cosineSimilarityFloat64(a, b []float64) float64 {
	var dot float64
	var normA float64
	var normB float64

	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
