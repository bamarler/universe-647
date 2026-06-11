package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/tag"
	"github.com/bamarler/universe-647/sophon/internal/errs"
	"github.com/bamarler/universe-647/sophon/internal/llm"
	"github.com/bamarler/universe-647/sophon/internal/search"
)

type commandInput struct {
	Body struct {
		Input string `json:"input" minLength:"1" maxLength:"2000"`
	}
}

// DraftTag is a tag reference in a draft: resolved (ID set) or a new-tag
// candidate (ID zero). Model-emitted names are resolved against real rows —
// IDs from the model are never trusted (there are none to trust).
type DraftTag struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name"`
}

// CommandDraft is the pre-filled skeleton the user fully edits before saving.
type CommandDraft struct {
	Type        string     `json:"type" enum:"task,note,tag"`
	Title       string     `json:"title"`
	BodyMD      string     `json:"body_md,omitempty"`
	Kind        string     `json:"kind,omitempty" doc:"for type=tag"`
	ProjectID   *int       `json:"project_id,omitempty"`
	ProjectName string     `json:"project_name,omitempty" doc:"unresolved project candidate when project_id is absent"`
	Tags        []DraftTag `json:"tags"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	DeferAt     *time.Time `json:"defer_at,omitempty"`
	Priority    int        `json:"priority"`
}

type commandOutput struct {
	Body struct {
		Action string        `json:"action" enum:"draft,results"`
		Draft  *CommandDraft `json:"draft,omitempty"`
		Hits   []SearchHit   `json:"hits,omitempty"`
	}
}

// registerCommand is the AI middleman: one tool-calling request, two possible
// intents, model prose discarded. Output is always either a draft (user edits
// everything) or navigable hits — never generated text.
func registerCommand(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "command", Method: http.MethodPost, Path: "/api/command",
		Summary: "Parse a command-bar input into a draft or search results", Tags: []string{"command"},
	}, func(ctx context.Context, in *commandInput) (*commandOutput, error) {
		vocab, byName, err := loadVocab(ctx, deps.Ent)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}

		cmd, err := deps.LLM.ParseCommand(ctx, in.Body.Input, time.Now(), vocab)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable(
				"command parsing unavailable (LLM gateway down) — browse and search filters still work")
		}

		out := &commandOutput{}
		switch cmd.Action {
		case "create":
			out.Body.Action = "draft"
			out.Body.Draft = buildDraft(cmd.Create, byName)
		case "search":
			out.Body.Action = "results"
			s := cmd.Search
			hits, err := runSearch(ctx, deps, search.Params{
				Query:     s.Query,
				Project:   resolveName(s.Project, byName, tag.KindProject),
				Tags:      resolveNames(s.Tags, byName),
				Types:     s.Types,
				DueBefore: dueBefore(s.DueWithin),
			})
			if err != nil {
				return nil, err
			}
			out.Body.Hits = hits
		default:
			return nil, huma.Error502BadGateway("intent parser returned unknown action")
		}
		return out, nil
	})
}

// loadVocab fetches the user's tag names for the prompt and a lookup map for
// enrichment. Lowercased keys: resolution is case-insensitive exact match.
func loadVocab(ctx context.Context, db *ent.Client) (llm.Vocab, map[string]*ent.Tag, error) {
	tags, err := db.Tag.Query().Where(tag.ArchivedAtIsNil()).All(ctx)
	if err != nil {
		return llm.Vocab{}, nil, err
	}
	var v llm.Vocab
	byName := make(map[string]*ent.Tag, len(tags))
	for _, t := range tags {
		byName[strings.ToLower(t.Name)] = t
		switch t.Kind {
		case tag.KindProject:
			v.Projects = append(v.Projects, t.Name)
		case tag.KindContext:
			v.Contexts = append(v.Contexts, t.Name)
		default:
			v.Tags = append(v.Tags, t.Name)
		}
	}
	return v, byName, nil
}

func buildDraft(c *llm.CreateDraft, byName map[string]*ent.Tag) *CommandDraft {
	d := &CommandDraft{
		Type:     c.Type,
		Title:    c.Title,
		BodyMD:   c.BodyMD,
		Kind:     c.Kind,
		DueAt:    c.DueAt,
		DeferAt:  c.DeferAt,
		Priority: c.Priority,
		Tags:     []DraftTag{},
	}
	if c.Project != "" {
		if t, ok := byName[strings.ToLower(c.Project)]; ok && t.Kind == tag.KindProject {
			d.ProjectID = &t.ID
			d.ProjectName = t.Name
		} else {
			d.ProjectName = c.Project // new-project candidate; user decides
		}
	}
	for _, name := range c.Tags {
		if t, ok := byName[strings.ToLower(name)]; ok {
			d.Tags = append(d.Tags, DraftTag{ID: t.ID, Name: t.Name})
		} else {
			d.Tags = append(d.Tags, DraftTag{Name: name})
		}
	}
	return d
}

func resolveName(name string, byName map[string]*ent.Tag, kind tag.Kind) string {
	if name == "" {
		return ""
	}
	if t, ok := byName[strings.ToLower(name)]; ok && t.Kind == kind {
		return t.Name
	}
	return name // pass through; the filter simply matches nothing if bogus
}

func resolveNames(names []string, byName map[string]*ent.Tag) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if t, ok := byName[strings.ToLower(n)]; ok {
			out = append(out, t.Name)
		} else {
			out = append(out, n)
		}
	}
	return out
}
