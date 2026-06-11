# Research: The Unified Second Brain (Sophon + Tomoko)

> Compiled 2026-06-10 from five research tracks (retrieval SOTA, product landscape,
> auth/security, backend language, reference architectures) plus a full audit of this
> repo's deployed state. This document records **what was found** and **what was decided**.
> The implementation plan lives with the work; deferred ideas are listed at the end.

## Decisions (locked)

| Decision | Choice | Driver |
|---|---|---|
| Architecture | One custom app instead of federating Vikunja/Monica/etc. via MCP | The unification layer was never built; no product covers the use case (§C) |
| AI role | Retrieval-only middleman — parses commands, surfaces items as links, **never writes prose to the user** | User requirement; prior art exists and works (§C.4) |
| Database | Existing shared PostgreSQL + pgvector ≥0.8.2. No vector DB, no graph DB | §B — equivalent performance at this scale, zero new containers |
| Retrieval | Hybrid: tsvector FTS + HNSW cosine → Reciprocal Rank Fusion. No reranker v1 | §B.1 |
| Embeddings | gemini-embedding-001 via existing LiteLLM, MRL-truncated to 768 dims | §B.2 |
| Backend | Go — huma v2 over chi, ent + Atlas, pgx + pgvector-go, openai-go → LiteLLM | §D, §E |
| Frontend | Svelte 5 + Vite mobile-first PWA; future Capacitor/SwiftUI against the same API | §D.6 |
| Names / layout | Backend **sophon** (`stacks/sophon/app/`, joins the existing sophon stack), frontend **tomoko** (`stacks/tomoko/app/`, new static stack). App source lives inside its stack dir (the `stacks/core/caddy/` pattern) | User choice |
| MVP scope | Tasks + projects/contexts (typed tag system) + markdown notes. CRM next, files later | User choice |
| Auth | Keep Authelia; fix session config (§A). Pocket ID is the escape hatch, not the first move | User choice after §A findings |
| Transition | Vikunja/Monica/Open WebUI/mcpo keep running until the MVP works | User choice |

---

## A. Authelia: the logout problem and the "is it necessary" question

### A.1 Root cause of the constant logouts (it's config, not the product)

Verified directly against `stacks/core/authelia/configuration.yml` and `stacks/core/compose.yaml`:

1. **`inactivity: 10m`** — any 10-minute idle gap kills the session.
2. **`expiration: 1h` is absolute** — activity does not extend it. Without ticking
   "Remember me" (`remember_me: 1M`), a full re-login (password + TOTP) is guaranteed
   at most every hour.
3. **Sessions are in-memory.** `storage.postgres` holds TOTP secrets and regulation
   data, *not* sessions; with no `session.redis` configured, Authelia uses its
   in-memory provider, so **every container restart wipes all sessions** including
   remember-me ones (authelia/authelia discussions #6953, #6911).
4. **Probable OOM-kill loop**: `mem_limit: 128m` while argon2id is configured with
   `memory: 65536` KiB — every password verification allocates **64MiB, half the
   container's RAM**. Memory pressure → OOM kill → `restart: unless-stopped` → all
   sessions gone (see 3). This plausibly explains "logged out at random ~30 min
   intervals even while active." Diagnostic:
   `docker inspect authelia --format '{{.RestartCount}} {{.State.OOMKilled}}'`.
5. OIDC `refresh_token: 90m` prevents Vikunja/n8n sessions from silently refreshing
   past ~90 minutes.

**Fix applied (gives ~monthly logins):** `expiration: 12h`, `inactivity: 2h`,
remember_me kept at 1M (tick the box), persistent sessions via a dedicated
`authelia-redis` sidecar (`--appendonly yes`, own internal network per the
monica_net/nextcloud_redis_net pattern), authelia `mem_limit` 128m → 256m (argon2id
kept at full strength), OIDC `refresh_token` → 720h.

### A.2 Is Tailscale + Caddy enough without Authelia?

Threat-model breakdown for this exact setup (no public ports, Tailnet Lock + device
approval on, CrowdSec, internal Docker networks):

- **Covered by Tailscale already:** network eavesdropping, public exposure,
  coordination-server compromise (Tailnet Lock means a compromised Tailscale control
  plane cannot inject nodes), unapproved devices.
- **NOT covered without an auth layer:** a **stolen unlocked device** on the tailnet,
  or a **malicious app/process on a tailnet device** — either gets unchallenged access
  to everything behind Caddy. Tailscale auth is also binary: ACLs can't distinguish
  n8n from Homepage because everything shares Caddy's port 443.
- **Why it matters here specifically:** n8n can execute arbitrary code via workflows
  and stores Google API credentials; AdGuard controls DNS for the whole network
  (hijack → silent MITM of every device). These two surfaces deserve a stronger wall
  than personal apps.
- Community consensus: "Tailscale-only" is *defensible* for single-user labs, but the
  near-universal advice is that the right response to painful SSO is fixing the SSO
  config, not removing the layer.

**Verdict:** keep Authelia, fix the config (A.1). If friction persists after a month,
the right replacement is **Pocket ID** (passkey-only OIDC — Face ID/Touch ID instead
of TOTP, no hardware key purchase needed, documented Vikunja integration), paired with
oauth2-proxy/caddy-security for forward-auth. Dropping auth entirely is not
recommended while n8n and AdGuard sit behind the same door as everything else.

A middle option for later: Authelia access-control rules keyed on the Tailscale CGNAT
range (`100.64.0.0/10`) → `one_factor` for personal apps, `two_factor` only for
n8n/AdGuard. ⚠️ Requires Caddy `trusted_proxies` plumbing so Authelia sees real client
IPs (all connections currently appear to come from the proxy) — verify with debug logs
before relying on it.

### A.3 Flagged discrepancy: Caddy port binding

Docs and CLAUDE.md state Caddy binds `127.0.0.1:443`, but `stacks/core/compose.yaml`
actually binds `0.0.0.0:443` and `0.0.0.0:80` — Caddy is reachable from the entire
LAN, not just localhost/tailnet. Likely necessary because Tailscale runs
`network_mode: host` (tailnet traffic arrives on a host interface), but the documented
posture overstates reality. **Decision deferred**: either bind to the Tailscale
interface IP or correct the docs.

Key sources: [Authelia session docs](https://www.authelia.com/configuration/session/introduction/) ·
[Authelia access control](https://www.authelia.com/configuration/security/access-control/) ·
[sessions lost on restart #6953](https://github.com/authelia/authelia/discussions/6953) ·
[Tailnet Lock](https://tailscale.com/kb/1226/tailnet-lock/) ·
[Tailnet Lock white paper](https://tailscale.com/kb/1230/tailnet-lock-whitepaper/) ·
[Tailscale reserved IPs](https://tailscale.com/docs/reference/reserved-ip-addresses) ·
[Pocket ID](https://github.com/pocket-id/pocket-id) ·
[Pocket ID + Vikunja](https://pocket-id.org/docs/client-examples/vikunja) ·
[Caddy + Pocket ID forward auth](https://msfjarvis.dev/posts/setting-up-forward-auth-with-caddy-and-pocket-id/) ·
[Tinyauth](https://tinyauth.app/docs/about/) ·
[Lobsters: securing self-hosted services](https://lobste.rs/s/rmenr4/how_do_you_secure_access_your_self_hosted)

---

## B. Retrieval & database design (state of the art, June 2026)

### B.1 What actually matters at <100k chunks

The production-standard pipeline is **BM25/full-text + dense vectors in parallel →
Reciprocal Rank Fusion → (optional) rerank**:

- **Hybrid search is the single highest-leverage choice.** Lexical search catches
  exact tokens (names, project codes — common in a tasks/CRM corpus); vectors catch
  paraphrase. RRF (`score = Σ 1/(60+rank)`) fuses by rank, sidestepping score
  normalization, and consistently beats either method alone. Notably, even Cursor
  frames embeddings as a complement to lexical search (+12.5% agent accuracy over
  grep alone).
- **Cosine similarity: still standard, and a non-question** — modern embedding models
  emit normalized vectors, so cosine and dot product produce identical rankings.
- **Skip at this scale (with upgrade paths):**
  - *Rerankers* — on a small clean corpus first-stage retrieval already puts answers
    near rank 1; a local cross-encoder costs ~1GB RAM + ~350ms on CPU. If precision
    ever disappoints, add Cohere/Voyage rerank through LiteLLM's `/rerank` endpoint
    (zero local RAM).
  - *ColBERT/late-interaction* — 10–100x storage for no gain on short text.
  - *Binary/int8 quantization* — gains start ~1M vectors; binary degrades <1024 dims.
    pgvector's `halfvec` (fp16) is the simpler, safer storage saving.
  - *pgvectorscale / DiskANN* — built for tens of millions of vectors.
- **Useful now:** Matryoshka (MRL) truncation — Gemini embeddings truncate
  3072 → 768 dims with minimal loss, cutting vector storage/index memory ~4x.

### B.2 Embedding model

**gemini-embedding-001 via the existing LiteLLM proxy** (`output_dimensionality=768`,
re-normalize after truncation): top-tier MTEB among APIs, already in this stack's
routing, batch pricing $0.075/M tokens → embedding the entire corpus costs pennies;
cost is irrelevant at personal scale, pick on quality and integration. Offline
fallback if ever needed: EmbeddingGemma-308M via Ollama (768-dim MRL family, <200MB
on CPU) — but never mix two models' vectors in one column.

### B.3 Postgres is sufficient — no new datastore

- pgvector 0.8.x: HNSW, `halfvec`, **iterative index scans**
  (`hnsw.iterative_scan = relaxed_order`) which fixed filtered vector search, parallel
  builds. **Pin ≥0.8.2** (CVE-2026-3172, parallel HNSW build overflow). The official
  `postgres` image does not include it — the image swaps to `pgvector/pgvector:pg16`
  (same major, drop-in on the existing data dir).
- Benchmarks repeatedly show Qdrant vs pgvector is a wash at small scale (p50 within
  1ms, recall ~equal). A dedicated store = another container, backup target, and RAM
  bill for zero observable benefit. 100k × 768-dim halfvec ≈ 150MB data + ~300MB index.
- Lexical side: start with native `tsvector` + GIN (zero new extensions; RRF consumes
  ranks, which dampens tsvector's lack of corpus-level IDF). Upgrade path if ranking
  ever disappoints: ParadeDB `pg_search` or Timescale `pg_textsearch` (true BM25).

### B.4 Graph databases are overkill here

GraphRAG-class systems (GraphRAG, LightRAG, HippoRAG) exist to **extract** entity
graphs from unstructured text. This system's relations are already foreign keys
(task→project, item→tag, later contact→note). The ICLR'26 GraphRAG-Bench results:
plain vector RAG hits 83% evidence recall on factual queries; graph methods only pull
ahead on genuine multi-hop synthesis — and practitioner guidance is unanimous: add
graph retrieval only when query logs show vector+metadata failing. Personal-scale
traversal ("notes linked to people linked to this company") is a recursive CTE over an
`edges` table; **Apache AGE** (openCypher inside Postgres) is the escape hatch if real
multi-hop needs ever appear. The "second brain graph DB" trend is mostly a UI
metaphor, not a retrieval requirement.

### B.5 Chunking & indexing strategy

- **Short structured items (tasks, later contacts): one item = one chunk, never split
  — enrich instead.** Embed a rendered template
  (`"Task: <title> | Project: <p> | Tags: <t> | Due: <d> | Notes: …"`); a 7-word title
  alone embeds poorly.
- **Markdown notes:** heading-aware splits, ~300–500 tokens, 10–15% overlap, with doc
  title + heading path prepended to each chunk.
- **Steal from Cursor (the plumbing, not the vector store):**
  content-hash-keyed embedding cache — `chunk_hash = sha256(rendered_text)` with a
  unique key per source; on re-index, skip unchanged hashes so re-embedding is cheap
  and idempotent (their Merkle tree, collapsed to one column at this scale). Plus
  structure-aware chunking (their AST splitting ≈ our heading-aware splitting).
- **Metadata filtering is the superpower:** one unified `chunk` table carrying
  source_type/source_id/project/tags/dates next to the vector, with B-tree/GIN
  indexes; pgvector 0.8 iterative scans make filtered ANN correct.
- **Tags get no embeddings.** They are exact-match symbols (text[] + GIN + lexical).
  Only a project/tag's *description text* is embedded (so "the project about home
  automation" resolves semantically).

Key sources: [ParadeDB hybrid search manual](https://www.paradedb.com/blog/hybrid-search-in-postgresql-the-missing-manual) ·
[Hybrid search reference 2026](https://www.digitalapplied.com/blog/hybrid-search-bm25-vector-reranking-reference-2026) ·
[pgvector 0.8.0](https://www.postgresql.org/about/news/pgvector-080-released-2952/) ·
[pgvector 0.8.2 / CVE-2026-3172](https://www.postgresql.org/about/news/pgvector-082-released-3245/) ·
[Qdrant vs pgvector](https://medium.com/@TheWake/qdrant-vs-pgvector-theyre-the-same-speed-5ac6b7361d9d) ·
[pg_textsearch](https://github.com/timescale/pg_textsearch) ·
[Gemini embedding pricing/dims](https://tokenmix.ai/blog/gemini-embedding-001-dimensions-pricing-guide-2026) ·
[EmbeddingGemma](https://developers.googleblog.com/en/introducing-embeddinggemma/) ·
[Cursor: secure codebase indexing](https://cursor.com/blog/secure-codebase-indexing) ·
[Cursor → Turbopuffer case study](https://turbopuffer.com/customers/cursor) ·
[GraphRAG-Bench (ICLR'26)](https://arxiv.org/html/2506.05690v3) ·
[Graph RAG practitioner's guide 2026](https://medium.com/graph-praxis/graph-rag-in-2026-a-practitioners-guide-to-what-actually-works-dca4962e7517) ·
[Apache AGE](https://github.com/apache/age) ·
[Chunking strategies](https://www.tigerdata.com/blog/which-rag-chunking-and-formatting-strategy-is-best) ·
[Embedding quantization](https://huggingface.co/blog/embedding-quantization) ·
[Rerankers leaderboard](https://agentset.ai/rerankers) ·
[LiteLLM /embeddings](https://docs.litellm.ai/docs/embedding/supported_embedding) · [LiteLLM /rerank](https://docs.litellm.ai/docs/rerank)

---

## C. Product landscape: what exists, what to borrow

### C.1 The gap is real

No self-hosted product unifies tasks + CRM + notes + files under one retrieval layer.
Every open-source project covers one vertical (Vikunja tasks, Monica contacts,
Karakeep bookmarks, Memos notes); every unified product (Tana, Mem.ai, Capacities,
Saner.ai) is cloud-only. No open-source supertag implementation exists. Retrieval-only
AI is rare — nearly all OSS AI-PKM defaults to RAG chat. The genuinely novel surface
is small: a unified entity/tag schema, the hybrid retrieval layer, and the
browser+command-bar UI — all over infrastructure already running here.

### C.2 Patterns borrowed

| Source | Pattern |
|---|---|
| **Karakeep** (AGPL, active) | "AI organizes, never answers": async worker → LLM writes tags/fields into normal DB columns → search stays deterministic |
| **Tana** (cloud, commercial) | **Supertags** — a tag is a *type* carrying optional schema. `#project`, `#context`, later `#person` unify tasks/CRM/notes under one tag system |
| **Todoist/Things/OmniFocus** (converged GTD) | Contexts are dead, multi-tags won; **two date semantics (due vs defer)** — lose defer and you lose the GTD tickler; saved filters are the only "views" feature. Todoist's API is the best-documented schema reference |
| **Obsidian Smart Connections** | The retrieval-only UX in the wild: command bar returns ranked links to your own items; click to navigate; no generation |
| **Memos** (MIT, Go+Postgres, very active) | Proof of shape: single ~20MB binary, ~50MB RAM. MIT → code can be lifted directly |
| **Khoj** (AGPL) | Study (don't deploy — ~4GB RAM): Postgres+pgvector ingestion/chunking pipeline at exactly this scale |
| Agent-memory consensus | Entities (tasks/tags/contacts) belong in the DB; long-form documents stay as files with text+embeddings indexed; the index is disposable/rebuildable |

### C.3 Cautionary findings

- **Monica**: v4 last release May 2024; the v5 rewrite has been beta since 2023;
  700+ open issues, two maintainers. Treat as a data source to migrate *from* —
  vindicates replacing it rather than extending it.
- **Logseq** in prolonged limbo (DB-version rewrite still beta); **Focalboard**
  effectively unmaintained; **Reor** drifting; **AnyType** not-OSI-licensed and needs
  MongoDB+Redis+S3. The single-vertical OSS field is shakier than it looks — another
  argument for owning the thin custom layer over a stable Postgres.
- **Vikunja** itself is fine software (its labels/projects/filters schema ≈ 80% of the
  MVP task model and informed the schema design) — it's the *federation* approach that
  was abandoned, not a deficiency in Vikunja.

Key sources: [Karakeep](https://github.com/karakeep-app/karakeep) ·
[Karakeep architecture](https://docs.karakeep.app/development/architecture/) ·
[Memos](https://github.com/usememos/memos) ·
[Khoj](https://github.com/khoj-ai/khoj) ·
[Smart Connections](https://github.com/brianpetro/obsidian-smart-connections) ·
[Vikunja task model](https://deepwiki.com/go-vikunja/vikunja/4.1-task-model) ·
[Todoist API](https://developer.todoist.com/api/v1/) ·
[OmniFocus 3 tags](https://learnomnifocus.com/forty-ways-to-use-omnifocus-3-tags/) ·
[Monica status](https://github.com/monicahq/monica/issues/6626) ·
[Tana review](https://www.saner.ai/blogs/tana-reviews) ·
[Anytype self-hosting/licensing](https://tech.anytype.io/how-to/self-hosting) ·
[Markdown-as-memory critique](https://limitededitionjonathan.substack.com/p/stop-calling-it-memory-the-problem)

---

## D. Backend language: Go vs Rust vs TypeScript

### D.1 The framing that settles it

This app's "AI" is HTTP calls to an OpenAI-compatible LiteLLM endpoint (embeddings +
tool-calling chat completions) plus JSON-schema validation; the hardest logic (hybrid
search with RRF) lives **in SQL**. Language "AI capability" is therefore mostly a red
herring — and a future switch to a local model is still an OpenAI-compatible HTTP API
via Ollama/LiteLLM, changing nothing architecturally.

### D.2 Go (chosen)

- Web: huma v2 (typed handlers → auto OpenAPI 3.1 + validation) over chi
  (100% net/http-compatible).
- LLM: official **openai-go** against any compatible base URL; structured outputs via
  JSON-schema generation from Go structs. (Avoid langchaingo — seeking maintainers;
  official SDK + a thin own layer is the durable choice in every language.)
- Proof of shape: Memos and Vikunja — Go+Postgres single binaries, ~50MB RAM, ~20–40MB
  images. Expected footprint here: 30–70MB RAM, well under the 150MB target.
- Operationally the best long-term story: one static binary, no runtime upgrades,
  minimal dependency churn, distroless images.

### D.3 Rust (declined)

Mature stack exists (axum + sqlx + pgvector-rust + rig.rs), idles 20–40MB lighter, and
refactors best — but the app is I/O-bound (Postgres ms + LLM hundreds-of-ms dominate),
so Rust's advantages are unobservable at single-user scale, while compile times and
async-Rust's learning curve tax exactly what a solo project needs: iteration speed.
Its one genuine edge — in-process local inference (fastembed-rs/candle) — is
neutralized because local models would run in Ollama/LiteLLM as a service anyway.
"The answer to a question this project isn't asking."

### D.4 TypeScript everywhere (runner-up)

Bun + Hono + Drizzle is legitimately lean, and the Vercel AI SDK is the best AI
library in any language. One-language velocity is real. Declined on: heaviest runtime
footprint of the three, largest dependency/supply-chain churn, and the ecosystem's
defaults pull toward heavy (Karakeep needs ~2GB; Memos needs 50MB — that contrast in
miniature).

### D.5 DB layer: ent + Atlas (the SQLAlchemy/Alembic flow)

User preference: minimize hand-authored SQL; loved the SQLAlchemy+Alembic flow
(change model code → autogenerated reviewable migration). Mapping:

- **ent + Atlas = that flow in Go**: schema as Go code, typed query builder,
  `atlas migrate diff` autogenerates versioned `.sql` migrations from schema changes.
- **sqlc** (the GenerateNU-adjacent alternative) is compile-time-typed, *not* runtime
  string interpretation — but it has no migration autogeneration and every CRUD query
  is hand-authored SQL. Better fit for SQL-first teams; declined for this one.
- Hand-written SQL is confined to: ~3 one-time DDL migrations (CREATE EXTENSION
  vector; the `vector(768)` column; HNSW + tsvector GENERATED column + GIN indexes —
  DDL no ORM DSL expresses, SQLAlchemy included) and the hybrid_search query via
  ent's raw escape hatch.

### D.6 Frontend & the mobile path

Svelte 5 (compiler, runes/signals — markedly smaller bundles and less boilerplate than
React; chosen partly because the user wants to learn it). The mobile path is
framework-independent: PWA now → optional Capacitor wrap (works identically for Svelte
or React) → likely SwiftUI app later. React's "easy React Native migration" is
overstated (zero UI code carries over). The Go API is the contract — huma's OpenAPI
output generates the TS client now and a Swift client later, which makes the web
frontend deliberately disposable.

Key sources: [Encore: Go frameworks](https://encore.dev/articles/best-go-backend-frameworks) ·
[huma](https://github.com/danielgtaylor/huma) ·
[brandur: sqlc/pgx in production](https://brandur.org/sqlc) ·
[GORM/sqlx/pgx compared](https://dasroot.net/posts/2025/12/go-database-patterns-gorm-sqlx-pgx-compared/) ·
[openai-go](https://github.com/openai/openai-go) ·
[pgvector-go](https://github.com/pgvector/pgvector-go) ·
[ent](https://entgo.io/) · [Atlas versioned migrations](https://atlasgo.io/versioned/intro) ·
[Memos footprint](https://www.bytesizego.com/blog/memos-the-self-hosted-note-taking-app-written-in-go) ·
[SPA from Go embed](https://hackandsla.sh/posts/2021-11-06-serve-spa-from-go/) ·
[JetBrains: Rust vs Go](https://blog.jetbrains.com/rust/2025/06/12/rust-vs-go/) ·
[Rust learning curve](https://corrode.dev/blog/flattening-rusts-learning-curve/) ·
[rig.rs](https://rig.rs/) · [fastembed-rs](https://crates.io/crates/fastembed) ·
[Bun vs Node 2026](https://tech-insider.org/bun-vs-nodejs-2026/) ·
[Vercel AI SDK](https://vercel.com/docs/ai-sdk) ·
[tiktoken-go](https://github.com/tiktoken-go/tokenizer)

---

## E. Reference architectures: the GenerateNU repos

Analyzed [toggo](https://github.com/GenerateNU/toggo),
[skillspark](https://github.com/GenerateNU/skillspark), and
[selfserve](https://github.com/GenerateNU/selfserve) (clones were placed under
`/tmp/gen/` during research; re-clone if needed).

### E.1 The house style (consistent across all three)

Monorepo (`backend/` + clients + infra, Makefile/justfile, path-filtered CI); Fiber as
the HTTP engine (skillspark layers **huma v2** on top — the only OpenAPI-first one);
layered `internal/` (cmd → config → app wiring → routes → handlers → repositories →
models) with **interfaces at the repository seam** and **manual constructor DI** in one
`InitApp` function; **pgx + hand-written SQL** (nobody uses GORM/sqlc/ent); plain
timestamped SQL migrations; central `errs` package + one framework-level error
handler; envconfig structs; slog; graceful shutdown; two-stage alpine Docker;
golangci-lint CI.

### E.2 Adopted for sophon

- The layering/DI style as-is (it's genuinely industry-standard and zero-magic).
- **selfserve's `tryInit*` graceful degradation** — boot even when optional infra
  (LiteLLM) is down; perfect for a homelab.
- **toggo's `errs` two-file design** — sentinel/API errors + pgconn error-code →
  domain-error mapping (23505→ErrDuplicate, 23503→ErrForeignKey), mapped to huma
  errors.
- **toggo's FTS query builder** — tsquery sanitization + `:*` prefix matching +
  `ts_rank`, plus a dedicated FTS-index migration (the lexical half of hybrid search,
  already written).
- **skillspark's huma usage** — explicit `huma.Register` operations, free
  `/docs` + `/openapi.json`, the **`cmd/genapi` trick** (boot the router with stub
  deps to dump `openapi.yaml` at build time → generate the TS client), and the
  middleware-ordering pattern (public routes registered before `UseMiddleware(auth)`).
- **selfserve's aiflows shape** — prompt builders as pure functions, typed structured
  output, **DB-enrichment of LLM output** (resolve model-emitted names against real
  rows; never trust model-emitted IDs), flows depending on narrow lookup interfaces.
- **toggo's testkit** — fluent integration tests against a real dockerized Postgres
  (mocks can't validate FTS/pgvector SQL). CI: gofmt, golangci-lint, `go mod tidy`
  drift check.
- selfserve's keyset/cursor pagination for list endpoints.

### E.3 Deliberately not adopted

- **Temporal** (background jobs → goroutines + Postgres job rows), **OpenSearch**
  (→ Postgres FTS+pgvector; selfserve's swappable-search interface seam is kept,
  the engine is not), **Redis pub/sub, SQS, Stripe, Clerk/Supabase, Doppler, Codecov**
  — team/SaaS-scale scaffolding irrelevant at n=1.
- skillspark's one-file-per-query embed.FS SQL loading + hand-written `Scan` —
  exactly the boilerplate a typed layer eliminates (ent here).
- Interface+mock for every repository — with a real test DB, most mocks disappear;
  interfaces kept only where substitution is wanted (LLM client, search).

---

## F. Recommended architecture (as decided)

```
iPhone/laptop → Tailscale → Caddy → Authelia (fixed sessions)
                              │
                tomoko.{DOMAIN}            (Caddy splits one subdomain, same-origin)
                ├── /api/* ──→ sophon (Go, stacks/sophon/app — joins LiteLLM + dormant
                │              Ollama in the sophon stack: "the brain")
                └── /*     ──→ tomoko (static Svelte 5 PWA, stacks/tomoko/app)

sophon ──→ PostgreSQL (`sophon` db: ent schema + chunk table w/ halfvec(768) + tsvector)
       ──→ LiteLLM ──→ gemini-embedding-001 (embeddings)
                  ──→ Gemini 2.5 Flash "tool-caller" (intent parsing only)
```

- **`POST /api/command`** — one tool-calling request, exactly two tools:
  `create_item` (returns a draft the user fully edits) and `search`
  (filters + optional semantic query → ranked links). Model text output is discarded.
- **Indexing worker** — on create/update: render enriched chunk → sha256 → skip if
  unchanged → embed via LiteLLM → upsert. `reindex` endpoint rebuilds everything.
- **Hybrid query** — fts top-20 ∥ HNSW top-20 (both metadata-filtered) → RRF → top 10
  deep links. Time-window queries ("this week") are pure filter SQL, no vectors.
- **Tomoko UI** — three surfaces only: tree browser (projects/contexts as folders),
  persistent command bar (navigates, never chats), item editor (every field editable).

## G. Gateway swap: LiteLLM → Bifrost (June 2026 addendum)

LiteLLM was retired after v1.88 measured **~956MiB idle** on the server. The
footprint is structural: ~500MB/worker per LiteLLM's own performance roadmap
(~200MB of it just Prisma imports), a bundled Next.js admin UI in the image,
eagerly-imported provider SDKs, plus open unbounded-memory-growth issues
(LiteLLM ships an official memory-troubleshooting page). The replacement is
**Bifrost** (maximhq/bifrost — Go, Apache-2.0, single container, ~tens of MB):

- OpenAI-compatible inbound at `/openai[/v1]/...`; openai-go works unchanged.
- Verified in source: OpenAI `dimensions` → Gemini `outputDimensionality`
  (embeddings) and `tool_choice: "required"` → Gemini `ANY` mode.
- Model aliases per provider key (`tool-caller` → gemini-3.1-flash-lite,
  `smart` → claude-haiku-4-5) in `stacks/sophon/bifrost/config.json`.
- Inbound auth in OSS: governance virtual keys (`BIFROST_VK`) with
  `enforce_auth_on_inference`; dashboard credentials separate.
- Caveat: fast release train with prior translation-layer regressions — pin
  exact image tags and smoke-test tool calls + embeddings after every bump.

**"No gateway at all" was considered and declined**: Google's native
OpenAI-compat endpoint is beta, silently ignores unknown params (`dimensions`
on embeddings undocumented), and a gateway keeps n8n + future consumers on the
same one-line-config abstraction where only the model name changes. Sophon's
config is now vendor-neutral (`LLM_BASE_URL`/`LLM_API_KEY`).

## Deferred / next steps (not in the current build)

1. **Cleanup** once the MVP is in daily use: remove Vikunja, Monica+MariaDB, mcpo,
   Open WebUI (+ Caddy routes, the Vikunja OIDC client, Homepage/Uptime-Kuma entries).
2. CRM extension: `person` tag kind + interactions; migrate Monica data.
3. File ingestion beyond markdown; Nextcloud unaffected meanwhile.
4. n8n workflows targeting sophon's API through Caddy (morning briefing, email
   task extraction, Canvas sync).
5. Optional retrieval upgrades, in order of likely value: rerank via the gateway
   (Bifrost has a native `/v1/rerank` route) → BM25 extension
   (pg_search/pg_textsearch) → Apache AGE if real multi-hop queries appear in logs.
6. Authelia tailnet `one_factor` rule (needs Caddy `trusted_proxies` work);
   Pocket ID migration if auth friction persists after the session fix.
7. Resolve the Caddy `0.0.0.0:443` vs documented `127.0.0.1:443` discrepancy (§A.3).
7a. Restructure the Authelia mount: `configuration.yml` as an individual `:ro` file
   mount; move `users_database.yml` (the only file Authelia writes — password
   changes) to `/srv/u647/authelia/`. The root-running container chowns the
   rw-mounted `./authelia` dir today, which breaks `git pull` on the server
   (unlink needs dir write permission) until a manual `chown` restores ownership.
8. Update gitignored `agent-docs/` to reflect the pivot.
9. Capacitor wrap or SwiftUI client generated from sophon's OpenAPI spec.
