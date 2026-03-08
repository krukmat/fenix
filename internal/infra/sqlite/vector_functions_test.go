package sqlite

import (
	"database/sql"
	"database/sql/driver"
	"math"
	"path/filepath"
	"testing"
)

func TestParseVectorJSONArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		arg   driver.Value
		ok    bool
		wantN int
	}{
		{name: "valid json", arg: "[1,2,3]", ok: true, wantN: 3},
		{name: "empty string", arg: "", ok: false},
		{name: "invalid json", arg: "not-json", ok: false},
		{name: "wrong type", arg: int64(1), ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseVectorJSONArg(tt.arg)
			if ok != tt.ok {
				t.Fatalf("parseVectorJSONArg(%v) ok=%v, want %v", tt.arg, ok, tt.ok)
			}
			if ok && len(got) != tt.wantN {
				t.Fatalf("parseVectorJSONArg(%v) len=%d, want %d", tt.arg, len(got), tt.wantN)
			}
		})
	}
}

func TestParseVectorArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []driver.Value
		ok   bool
	}{
		{name: "valid args", args: []driver.Value{"[1,0]", "[1,0]"}, ok: true},
		{name: "wrong arity", args: []driver.Value{"[1,0]"}, ok: false},
		{name: "mismatched dimensions", args: []driver.Value{"[1,0]", "[1,0,0]"}, ok: false},
		{name: "invalid left", args: []driver.Value{"x", "[1,0]"}, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, right, ok := parseVectorArgs(tt.args)
			if ok != tt.ok {
				t.Fatalf("parseVectorArgs(%v) ok=%v, want %v", tt.args, ok, tt.ok)
			}
			if ok && len(left) != len(right) {
				t.Fatalf("expected matching lengths, got %d vs %d", len(left), len(right))
			}
		})
	}
}

func TestCosineSimilarityFloat64(t *testing.T) {
	t.Parallel()

	if got := cosineSimilarityFloat64([]float64{1, 0}, []float64{1, 0}); math.Abs(got-1) > 0.001 {
		t.Fatalf("identical vectors = %f, want 1.0", got)
	}
	if got := cosineSimilarityFloat64([]float64{1, 0}, []float64{0, 1}); math.Abs(got) > 0.001 {
		t.Fatalf("orthogonal vectors = %f, want 0.0", got)
	}
	if got := cosineSimilarityFloat64([]float64{0, 0}, []float64{0, 0}); got != 0 {
		t.Fatalf("zero vectors = %f, want 0.0", got)
	}
}

func TestSQLiteCosineSimilarityFunction(t *testing.T) {
	t.Parallel()

	db := mustOpenVectorTestDB(t)
	defer db.Close()

	var identical float64
	if err := db.QueryRow(`SELECT cosine_similarity_json('[1,0,0]', '[1,0,0]')`).Scan(&identical); err != nil {
		t.Fatalf("query identical similarity: %v", err)
	}
	if math.Abs(identical-1) > 0.001 {
		t.Fatalf("identical similarity = %f, want 1.0", identical)
	}

	var malformed float64
	if err := db.QueryRow(`SELECT cosine_similarity_json('bad', '[1,0,0]')`).Scan(&malformed); err != nil {
		t.Fatalf("query malformed similarity: %v", err)
	}
	if malformed != 0 {
		t.Fatalf("malformed similarity = %f, want 0.0", malformed)
	}
}

func mustOpenVectorTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := NewDB(filepath.Join(t.TempDir(), "vector-test.sqlite"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	return db
}
