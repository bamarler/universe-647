// Package index maintains the chunk table: the derived, rebuildable search
// index over tasks/notes/tags. The Karakeep pattern — AI output (embeddings)
// only ever lands in normal DB columns; querying stays deterministic.
package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/chunk"
	"github.com/bamarler/universe-647/sophon/internal/ent/note"
	"github.com/bamarler/universe-647/sophon/internal/ent/task"
	"github.com/bamarler/universe-647/sophon/internal/llm"
)

type ref struct {
	typ string
	id  int
}

// Indexer owns a small async queue: mutations enqueue, one worker renders,
// hash-compares, embeds (only when changed), and upserts.
type Indexer struct {
	ent   *ent.Client
	llm   llm.Client
	log   *slog.Logger
	queue chan ref
}

func New(entc *ent.Client, llmc llm.Client, log *slog.Logger) *Indexer {
	return &Indexer{ent: entc, llm: llmc, log: log, queue: make(chan ref, 256)}
}

func (ix *Indexer) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-ix.queue:
				if err := ix.indexOne(ctx, r); err != nil {
					ix.log.Warn("index item", "type", r.typ, "id", r.id, "error", err)
				}
			}
		}
	}()
}

func (ix *Indexer) EnqueueTask(id int) { ix.enqueue(ref{"task", id}) }
func (ix *Indexer) EnqueueNote(id int) { ix.enqueue(ref{"note", id}) }
func (ix *Indexer) EnqueueTag(id int)  { ix.enqueue(ref{"tag", id}) }

func (ix *Indexer) enqueue(r ref) {
	select {
	case ix.queue <- r:
	default:
		ix.log.Warn("index queue full, dropping (reindex will recover)", "type", r.typ, "id", r.id)
	}
}

// Remove deletes an item's chunks (call on item delete).
func (ix *Indexer) Remove(ctx context.Context, typ string, id int) error {
	_, err := ix.ent.Chunk.Delete().
		Where(chunk.SourceTypeEQ(chunk.SourceType(typ)), chunk.SourceIDEQ(id)).
		Exec(ctx)
	return err
}

// Reindex walks everything. Unchanged content is skipped by hash, so this is
// cheap to run and is the recovery path for dropped queue items.
func (ix *Indexer) Reindex(ctx context.Context) (indexed, skipped int, err error) {
	taskIDs, err := ix.ent.Task.Query().IDs(ctx)
	if err != nil {
		return 0, 0, err
	}
	noteIDs, err := ix.ent.Note.Query().IDs(ctx)
	if err != nil {
		return 0, 0, err
	}
	tagIDs, err := ix.ent.Tag.Query().IDs(ctx)
	if err != nil {
		return 0, 0, err
	}
	refs := make([]ref, 0, len(taskIDs)+len(noteIDs)+len(tagIDs))
	for _, id := range taskIDs {
		refs = append(refs, ref{"task", id})
	}
	for _, id := range noteIDs {
		refs = append(refs, ref{"note", id})
	}
	for _, id := range tagIDs {
		refs = append(refs, ref{"tag", id})
	}
	for _, r := range refs {
		n, s, err := ix.upsertItem(ctx, r)
		if err != nil {
			ix.log.Warn("reindex item", "type", r.typ, "id", r.id, "error", err)
			continue
		}
		indexed += n
		skipped += s
	}
	return indexed, skipped, nil
}

func (ix *Indexer) indexOne(ctx context.Context, r ref) error {
	_, _, err := ix.upsertItem(ctx, r)
	return err
}

type rendered struct {
	index   int
	content string
	dueAt   *time.Time
	project string
	tags    []string
}

func (ix *Indexer) upsertItem(ctx context.Context, r ref) (indexed, skipped int, err error) {
	chunks, err := ix.render(ctx, r)
	if ent.IsNotFound(err) {
		// Item vanished between enqueue and processing — drop its chunks.
		return 0, 0, ix.Remove(ctx, r.typ, r.id)
	}
	if err != nil {
		return 0, 0, err
	}

	// Existing hashes by chunk_index — unchanged content never re-embeds
	// (the Cursor incremental pattern).
	existing, err := ix.ent.Chunk.Query().
		Where(chunk.SourceTypeEQ(chunk.SourceType(r.typ)), chunk.SourceIDEQ(r.id)).
		All(ctx)
	if err != nil {
		return 0, 0, err
	}
	byIndex := make(map[int]*ent.Chunk, len(existing))
	for _, c := range existing {
		byIndex[c.ChunkIndex] = c
	}

	var toEmbed []rendered
	for _, rc := range chunks {
		if old, ok := byIndex[rc.index]; ok && old.ChunkHash == hash(rc.content) {
			skipped++
			continue
		}
		toEmbed = append(toEmbed, rc)
	}

	// Drop stale tail chunks (note shrank).
	for idx := range byIndex {
		if idx >= len(chunks) {
			_, _ = ix.ent.Chunk.Delete().
				Where(chunk.SourceTypeEQ(chunk.SourceType(r.typ)),
					chunk.SourceIDEQ(r.id), chunk.ChunkIndexEQ(idx)).
				Exec(ctx)
		}
	}

	if len(toEmbed) == 0 {
		return 0, skipped, nil
	}

	texts := make([]string, len(toEmbed))
	for i, rc := range toEmbed {
		texts[i] = rc.content
	}
	vecs, embErr := ix.llm.Embed(ctx, texts)
	// Gateway down → store chunks without vectors; lexical search still works
	// and the next reindex backfills embeddings (hash unchanged but embedding
	// NULL is treated as changed below... so re-render path re-embeds).
	if embErr != nil {
		ix.log.Warn("embedding unavailable, indexing lexical-only", "error", embErr)
		vecs = nil
	}

	for i, rc := range toEmbed {
		create := ix.ent.Chunk.Create().
			SetSourceType(chunk.SourceType(r.typ)).
			SetSourceID(r.id).
			SetChunkIndex(rc.index).
			SetChunkHash(hash(rc.content)).
			SetContent(rc.content).
			SetProject(rc.project).
			SetTags(rc.tags).
			SetUpdatedAt(time.Now())
		if rc.dueAt != nil {
			create.SetItemDueAt(*rc.dueAt)
		}
		if vecs != nil {
			create.SetEmbedding(pgvector.NewHalfVector(vecs[i]))
		}
		err := create.
			OnConflictColumns(chunk.FieldSourceType, chunk.FieldSourceID, chunk.FieldChunkIndex).
			UpdateNewValues().
			Exec(ctx)
		if err != nil {
			return indexed, skipped, fmt.Errorf("upsert chunk: %w", err)
		}
		indexed++
	}
	return indexed, skipped, nil
}

// render produces the enriched chunk text(s) for an item. Short structured
// items are one chunk with context serialized in (bare titles embed poorly);
// notes split by heading into ~1600-char windows.
func (ix *Indexer) render(ctx context.Context, r ref) ([]rendered, error) {
	switch r.typ {
	case "task":
		t, err := ix.ent.Task.Query().
			Where(task.IDEQ(r.id)).
			WithProject().WithTags().
			Only(ctx)
		if err != nil {
			return nil, err
		}
		var sb strings.Builder
		sb.WriteString("Task: " + t.Title)
		project := ""
		if t.Edges.Project != nil {
			project = t.Edges.Project.Name
			sb.WriteString(" | Project: " + project)
		}
		names := tagNames(t.Edges.Tags)
		if len(names) > 0 {
			sb.WriteString(" | Tags: " + strings.Join(names, ", "))
		}
		if t.DueAt != nil {
			sb.WriteString(" | Due: " + t.DueAt.Format("2006-01-02"))
		}
		if t.Status == "done" {
			sb.WriteString(" | Status: done")
		}
		if t.BodyMd != "" {
			sb.WriteString(" | Notes: " + t.BodyMd)
		}
		return []rendered{{index: 0, content: sb.String(), dueAt: t.DueAt, project: project, tags: names}}, nil

	case "note":
		n, err := ix.ent.Note.Query().Where(note.IDEQ(r.id)).WithTags().Only(ctx)
		if err != nil {
			return nil, err
		}
		names := tagNames(n.Edges.Tags)
		parts := splitMarkdown(n.Title, n.BodyMd)
		out := make([]rendered, len(parts))
		for i, p := range parts {
			out[i] = rendered{index: i, content: p, tags: names}
		}
		return out, nil

	case "tag":
		tg, err := ix.ent.Tag.Get(ctx, r.id)
		if err != nil {
			return nil, err
		}
		// Tags are exact-match symbols; only their description text embeds.
		if tg.Description == "" {
			return nil, nil
		}
		content := fmt.Sprintf("%s (%s): %s", tg.Name, tg.Kind, tg.Description)
		return []rendered{{index: 0, content: content, project: tg.Name}}, nil
	}
	return nil, fmt.Errorf("unknown source type %q", r.typ)
}

func tagNames(tags []*ent.Tag) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = t.Name
	}
	return out
}

func hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
