package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

func BenchmarkVectorSearch_SQLite(b *testing.B) {
	db := setupSearchBenchmarkDB(b)
	defer db.Close()

	stub := newStubEmbedder(8)
	wsID := createBenchmarkWorkspace(b, db)
	populateSearchBenchmarkData(b, db, wsID, stub, 100)

	svc := NewSearchService(db, stub)
	queryVec := []float32{0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.vectorSearch(context.Background(), wsID, queryVec, 10); err != nil {
			b.Fatalf("vectorSearch: %v", err)
		}
	}
}

func BenchmarkVectorSearch_InMemory(b *testing.B) {
	db := setupSearchBenchmarkDB(b)
	defer db.Close()

	stub := newStubEmbedder(8)
	wsID := createBenchmarkWorkspace(b, db)
	populateSearchBenchmarkData(b, db, wsID, stub, 100)

	queryVec := []float32{0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := vectorSearchInMemoryBenchmark(context.Background(), db, wsID, queryVec, 10); err != nil {
			b.Fatalf("vectorSearchInMemoryBenchmark: %v", err)
		}
	}
}

func setupSearchBenchmarkDB(b *testing.B) *sql.DB {
	b.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!")

	db, err := isqlite.NewDB(":memory:")
	if err != nil {
		b.Fatalf("open bench database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = isqlite.MigrateUp(db); err != nil {
		b.Fatalf("migrate bench database: %v", err)
	}
	return db
}

func createBenchmarkWorkspace(b *testing.B, db *sql.DB) string {
	b.Helper()

	id := newID()
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, plan, status, created_at, updated_at)
		VALUES (?, ?, ?, 'pro', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, "Bench Workspace", "bench-workspace-"+id[:8])
	if err != nil {
		b.Fatalf("create bench workspace: %v", err)
	}
	return id
}

func populateSearchBenchmarkData(b *testing.B, db *sql.DB, wsID string, stub *stubEmbedder, count int) {
	b.Helper()

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)

	for i := 0; i < count; i++ {
		title := fmt.Sprintf("Benchmark Doc %03d", i)
		content := fmt.Sprintf("benchmark vector retrieval document number %03d with pricing and search terms", i)
		ingestAndEmbedDocBenchmark(b, ingest, embedder, wsID, title, content)
	}
}

func ingestAndEmbedDocBenchmark(b *testing.B, ingest *IngestService, embedder *EmbedderService, wsID, title, content string) *KnowledgeItem {
	b.Helper()
	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       title,
		RawContent:  content,
	})
	if err != nil {
		b.Fatalf("ingest failed for %q: %v", title, err)
	}
	if err := embedder.EmbedChunks(context.Background(), item.ID, wsID); err != nil {
		b.Fatalf("EmbedChunks failed for %q: %v", title, err)
	}
	return item
}

func vectorSearchInMemoryBenchmark(ctx context.Context, db *sql.DB, wsID string, queryVec []float32, limit int) ([]vectorRow, error) {
	q := sqlcgen.New(db)
	rows, err := q.GetAllEmbeddedVectorsByWorkspace(ctx, wsID)
	if err != nil {
		return nil, fmt.Errorf("vectorSearch fetch: %w", err)
	}

	type scoredRow struct {
		row        sqlcgen.GetAllEmbeddedVectorsByWorkspaceRow
		similarity float32
	}

	scored := make([]scoredRow, 0, len(rows))
	for _, row := range rows {
		vec, decodeErr := decodeEmbedding(row.Embedding)
		if decodeErr != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, vec)
		scored = append(scored, scoredRow{row: row, similarity: sim})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].similarity > scored[j].similarity
	})

	results := make([]vectorRow, 0, min(limit, len(scored)))
	for i := 0; i < len(scored) && i < limit; i++ {
		results = append(results, vectorRow{
			id:              scored[i].row.ID,
			knowledgeItemID: scored[i].row.KnowledgeItemID,
			similarity:      scored[i].similarity,
		})
	}
	return results, nil
}
