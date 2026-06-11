package index

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamarler/universe-647/sophon/internal/llm"
	"github.com/bamarler/universe-647/sophon/internal/testdb"
)

// fakeEmbedder counts calls and returns deterministic unit vectors.
type fakeEmbedder struct {
	calls atomic.Int64
}

func (f *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	f.calls.Add(int64(len(texts)))
	out := make([][]float32, len(texts))
	for i := range texts {
		v := make([]float32, 768)
		v[0] = 1
		out[i] = v
	}
	return out, nil
}

func (f *fakeEmbedder) ParseCommand(context.Context, string, time.Time, llm.Vocab) (*llm.Command, error) {
	return nil, llm.ErrUnavailable
}

// TestHashSkip is the Cursor incremental-indexing property: unchanged content
// never re-embeds; changed content re-embeds exactly once.
func TestHashSkip(t *testing.T) {
	client, _ := testdb.Open(t)
	ctx := context.Background()
	fake := &fakeEmbedder{}
	ix := New(client, fake, slog.Default())

	proj := client.Tag.Create().SetName("homelab").SetKind("project").SaveX(ctx)
	task := client.Task.Create().SetTitle("fix caddy config").SetProjectID(proj.ID).SaveX(ctx)

	if _, _, err := ix.Reindex(ctx); err != nil {
		t.Fatalf("first reindex: %v", err)
	}
	first := fake.calls.Load()
	if first == 0 {
		t.Fatal("expected embed calls on first index")
	}

	// Unchanged: everything skips, zero new embeds.
	indexed, skipped, err := ix.Reindex(ctx)
	if err != nil {
		t.Fatalf("second reindex: %v", err)
	}
	if indexed != 0 || skipped == 0 {
		t.Fatalf("expected all-skip, got indexed=%d skipped=%d", indexed, skipped)
	}
	if got := fake.calls.Load(); got != first {
		t.Fatalf("unchanged content re-embedded: calls %d -> %d", first, got)
	}

	// Changed: exactly the changed item re-embeds.
	client.Task.UpdateOneID(task.ID).SetTitle("fix caddy OIDC config").ExecX(ctx)
	if _, _, err := ix.Reindex(ctx); err != nil {
		t.Fatalf("third reindex: %v", err)
	}
	if got := fake.calls.Load(); got != first+1 {
		t.Fatalf("expected exactly 1 new embed call, calls %d -> %d", first, got)
	}
}

// TestNoteShrinkDropsStaleChunks: a note losing content must not leave
// orphaned tail chunks behind.
func TestNoteShrinkDropsStaleChunks(t *testing.T) {
	client, db := testdb.Open(t)
	ctx := context.Background()
	ix := New(client, &fakeEmbedder{}, slog.Default())

	long := ""
	for range 200 {
		long += "antimatter propulsion lattice notes section text. "
	}
	n := client.Note.Create().SetTitle("research").SetBodyMd("# A\n" + long + "\n# B\n" + long).SaveX(ctx)
	if _, _, err := ix.Reindex(ctx); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.QueryRow("SELECT count(*) FROM chunks WHERE source_type='note'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before < 2 {
		t.Fatalf("expected multiple chunks for long note, got %d", before)
	}

	client.Note.UpdateOneID(n.ID).SetBodyMd("short now").ExecX(ctx)
	if _, _, err := ix.Reindex(ctx); err != nil {
		t.Fatal(err)
	}
	var after int
	if err := db.QueryRow("SELECT count(*) FROM chunks WHERE source_type='note'").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 1 {
		t.Fatalf("stale chunks not dropped: %d -> %d (want 1)", before, after)
	}
}
