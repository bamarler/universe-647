// Package llm is the narrow seam to the LiteLLM gateway. The AI is a
// retrieval-only middleman: it embeds text and parses commands into
// structured intents — it never produces prose shown to the user.
package llm

import (
	"context"
	"errors"
	"time"
)

var ErrUnavailable = errors.New("llm gateway unavailable")

// Command is the entire universe of things the model may do: draft an item
// for the user to edit, or search. Any prose it produces is discarded.
type Command struct {
	Action string        `json:"action"` // "create" | "search"
	Create *CreateDraft  `json:"create,omitempty"`
	Search *SearchIntent `json:"search,omitempty"`
}

// CreateDraft is a pre-filled skeleton. Names (project, tags) are the model's
// raw strings — the API layer resolves them against real rows and never
// trusts model-emitted IDs.
type CreateDraft struct {
	Type     string     `json:"type"` // task | note | tag
	Title    string     `json:"title"`
	BodyMD   string     `json:"body_md,omitempty"`
	Kind     string     `json:"kind,omitempty"` // for type=tag: project|context|tag
	Project  string     `json:"project,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
	DueAt    *time.Time `json:"due_at,omitempty"`
	DeferAt  *time.Time `json:"defer_at,omitempty"`
	Priority int        `json:"priority,omitempty"`
}

// SearchIntent maps to the hybrid search: structured filters plus an optional
// semantic query. Pure time-window questions arrive with an empty query.
type SearchIntent struct {
	Query     string   `json:"query,omitempty"`
	Project   string   `json:"project,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	DueWithin *int     `json:"due_within_days,omitempty"`
	Status    string   `json:"status,omitempty"` // open | done | any
	Types     []string `json:"types,omitempty"`  // task | note
}

// Vocab gives the model the user's real tag/project names so it emits exact
// strings; resolution still happens against the DB afterwards.
type Vocab struct {
	Projects []string
	Contexts []string
	Tags     []string
}

type Client interface {
	// Embed returns one L2-normalized 768-dim vector per input text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// ParseCommand turns free text into exactly one Command.
	ParseCommand(ctx context.Context, input string, now time.Time, vocab Vocab) (*Command, error)
}

// Disabled keeps the app fully functional minus semantic features when the
// gateway is down (the tryInit pattern).
type Disabled struct{}

func (Disabled) Embed(context.Context, []string) ([][]float32, error) {
	return nil, ErrUnavailable
}

func (Disabled) ParseCommand(context.Context, string, time.Time, Vocab) (*Command, error) {
	return nil, ErrUnavailable
}
