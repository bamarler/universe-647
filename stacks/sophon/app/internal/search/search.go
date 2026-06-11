// Package search runs the hybrid retrieval pipeline over the chunk table:
// lexical (tsvector) and semantic (HNSW cosine) branches in parallel SQL,
// fused with Reciprocal Rank Fusion. The one query ent can't express.
package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/bamarler/universe-647/sophon/internal/llm"
)

const (
	branchLimit = 20 // per-branch candidates before fusion
	rrfK        = 60 // standard RRF constant
)

type Params struct {
	Query     string
	Project   string   // project tag name (chunks denormalize it)
	Tags      []string // all must be present
	Types     []string // task | note | tag
	DueBefore *time.Time
}

type Result struct {
	SourceType string  `json:"source_type"`
	SourceID   int     `json:"source_id"`
	Snippet    string  `json:"snippet"`
	Project    string  `json:"project,omitempty"`
	Score      float64 `json:"score"`
}

type Searcher struct {
	db  *sql.DB
	llm llm.Client
}

func New(db *sql.DB, llmc llm.Client) *Searcher {
	return &Searcher{db: db, llm: llmc}
}

// Search embeds the query (when the gateway is up) and fuses both branches;
// with the gateway down it degrades to lexical-only. An empty query with
// filters becomes a plain filtered scan.
func (s *Searcher) Search(ctx context.Context, p Params) ([]Result, error) {
	var emb *pgvector.HalfVector
	if strings.TrimSpace(p.Query) != "" {
		if vecs, err := s.llm.Embed(ctx, []string{p.Query}); err == nil && len(vecs) == 1 {
			hv := pgvector.NewHalfVector(vecs[0])
			emb = &hv
		}
	}

	where, args := buildFilters(p)

	var query string
	switch {
	case strings.TrimSpace(p.Query) == "":
		query = fmt.Sprintf(`
			SELECT source_type, source_id, left(content, 240), coalesce(project, ''), 1.0
			FROM chunks %s
			ORDER BY item_due_at ASC NULLS LAST, updated_at DESC
			LIMIT 50`, where)

	case emb == nil:
		args = append(args, p.Query)
		qArg := fmt.Sprintf("$%d", len(args))
		query = fmt.Sprintf(`
			SELECT source_type, source_id, left(content, 240), coalesce(project, ''),
			       ts_rank_cd(fts, websearch_to_tsquery('english', %[1]s))::float8
			FROM chunks %[2]s AND fts @@ websearch_to_tsquery('english', %[1]s)
			ORDER BY 5 DESC
			LIMIT 10`, qArg, where)

	default:
		// Full hybrid: both branches share the filter set, RRF fuses ranks.
		args = append(args, p.Query)
		qArg := fmt.Sprintf("$%d", len(args))
		args = append(args, *emb)
		eArg := fmt.Sprintf("$%d", len(args))
		query = fmt.Sprintf(`
			WITH fts AS (
				SELECT id, ROW_NUMBER() OVER (
					ORDER BY ts_rank_cd(fts, websearch_to_tsquery('english', %[1]s)) DESC
				) AS rnk
				FROM chunks %[3]s AND fts @@ websearch_to_tsquery('english', %[1]s)
				LIMIT %[4]d
			),
			vec AS (
				SELECT id, ROW_NUMBER() OVER (ORDER BY embedding <=> %[2]s::halfvec(768)) AS rnk
				FROM chunks %[3]s AND embedding IS NOT NULL
				ORDER BY embedding <=> %[2]s::halfvec(768)
				LIMIT %[4]d
			)
			SELECT c.source_type, c.source_id, left(c.content, 240), coalesce(c.project, ''),
			       (coalesce(1.0/(%[5]d + f.rnk), 0) + coalesce(1.0/(%[5]d + v.rnk), 0))::float8 AS score
			FROM fts f
			FULL OUTER JOIN vec v USING (id)
			JOIN chunks c ON c.id = coalesce(f.id, v.id)
			ORDER BY score DESC
			LIMIT 10`, qArg, eArg, where, branchLimit, rrfK)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}
	defer rows.Close()

	var out []Result
	seen := map[string]bool{}
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.SourceType, &r.SourceID, &r.Snippet, &r.Project, &r.Score); err != nil {
			return nil, err
		}
		// Multiple chunks of one note may both rank — collapse to the item.
		key := r.SourceType + ":" + fmt.Sprint(r.SourceID)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out, rows.Err()
}

// buildFilters renders the shared WHERE clause. Always returns a clause
// starting with WHERE so branches can append "AND ...".
func buildFilters(p Params) (string, []any) {
	conds := []string{"TRUE"}
	var args []any
	add := func(c string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(c, len(args)))
	}
	if p.Project != "" {
		add("project = $%d", p.Project)
	}
	for _, t := range p.Tags {
		add("tags @> to_jsonb(ARRAY[$%d::text])", t)
	}
	if len(p.Types) > 0 {
		add("source_type = ANY($%d)", typeArray(p.Types))
	}
	if p.DueBefore != nil {
		add("item_due_at <= $%d", *p.DueBefore)
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func typeArray(types []string) any {
	// pq-style array literal via pgx stdlib: pass as []string is supported
	// by pgx for text[]; source_type is varchar so cast happens server-side.
	return types
}
