package search

import (
	"context"
	"testing"
	"time"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/bamarler/universe-647/sophon/internal/llm"
	"github.com/bamarler/universe-647/sophon/internal/testdb"
)

// queryEmbedder returns a fixed unit vector for any query.
type queryEmbedder struct{ vec []float32 }

func (q *queryEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = q.vec
	}
	return out, nil
}

func (q *queryEmbedder) ParseCommand(context.Context, string, time.Time, llm.Vocab) (*llm.Command, error) {
	return nil, llm.ErrUnavailable
}

func unit(dims int, hot ...int) []float32 {
	v := make([]float32, dims)
	for _, h := range hot {
		v[h] = 1
	}
	// not strictly normalized for multi-hot; fine for ordering tests
	return v
}

// TestRRFFusion: the item matching BOTH branches must outrank items matching
// only one — the defining property of reciprocal rank fusion.
func TestRRFFusion(t *testing.T) {
	client, db := testdb.Open(t)
	ctx := context.Background()

	qVec := unit(768, 0)
	s := New(db, &queryEmbedder{vec: qVec})

	// A: lexical-only (contains the word; NO embedding -> absent from the
	// vector branch entirely)
	client.Chunk.Create().
		SetSourceType("task").SetSourceID(1).SetChunkIndex(0).SetChunkHash("a").
		SetContent("Task: configure the zigbee antenna").
		SaveX(ctx)
	// B: semantic-only (no shared words -> absent from the fts branch;
	// embedding exactly equals the query)
	client.Chunk.Create().
		SetSourceType("task").SetSourceID(2).SetChunkIndex(0).SetChunkHash("b").
		SetContent("Task: wireless mesh device pairing").
		SetEmbedding(pgvector.NewHalfVector(unit(768, 0))).
		SaveX(ctx)
	// C: matches both branches (close-but-second vector, lexical hit)
	cVec := unit(768, 0)
	cVec[1] = 0.1
	client.Chunk.Create().
		SetSourceType("task").SetSourceID(3).SetChunkIndex(0).SetChunkHash("c").
		SetContent("Task: zigbee pairing for the new sensor").
		SetEmbedding(pgvector.NewHalfVector(cVec)).
		SaveX(ctx)

	results, err := s.Search(ctx, Params{Query: "zigbee"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) < 3 {
		t.Fatalf("expected 3 results, got %d: %+v", len(results), results)
	}
	if results[0].SourceID != 3 {
		t.Fatalf("RRF should rank the both-branches item first, got order %+v", results)
	}
}

// TestLexicalFallback: with the LLM gateway down, search still returns
// lexical matches (graceful degradation).
func TestLexicalFallback(t *testing.T) {
	client, db := testdb.Open(t)
	ctx := context.Background()
	s := New(db, llm.Disabled{})

	client.Chunk.Create().
		SetSourceType("note").SetSourceID(7).SetChunkIndex(0).SetChunkHash("x").
		SetContent("Note: restic backup runbook for cloudflare r2").
		SaveX(ctx)

	results, err := s.Search(ctx, Params{Query: "restic backup"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].SourceID != 7 {
		t.Fatalf("expected the lexical hit, got %+v", results)
	}
}

// TestFilters: metadata filters constrain both branches.
func TestFilters(t *testing.T) {
	client, db := testdb.Open(t)
	ctx := context.Background()
	s := New(db, llm.Disabled{})

	client.Chunk.Create().
		SetSourceType("task").SetSourceID(1).SetChunkIndex(0).SetChunkHash("p1").
		SetContent("Task: write thesis outline").SetProject("school").
		SetTags([]string{"writing"}).
		SaveX(ctx)
	client.Chunk.Create().
		SetSourceType("task").SetSourceID(2).SetChunkIndex(0).SetChunkHash("p2").
		SetContent("Task: write blog post outline").SetProject("homelab").
		SaveX(ctx)

	results, err := s.Search(ctx, Params{Query: "outline", Project: "school"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].SourceID != 1 {
		t.Fatalf("project filter failed: %+v", results)
	}

	results, err = s.Search(ctx, Params{Query: "outline", Tags: []string{"writing"}})
	if err != nil {
		t.Fatalf("tag-filtered search: %v", err)
	}
	if len(results) != 1 || results[0].SourceID != 1 {
		t.Fatalf("tag filter failed: %+v", results)
	}
}
