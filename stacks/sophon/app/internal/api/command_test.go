package api

import (
	"testing"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/tag"
	"github.com/bamarler/universe-647/sophon/internal/llm"
)

// TestBuildDraftEnrichment: model-emitted names resolve against real rows;
// unknown names become new-tag candidates with no ID — model output is never
// trusted as an identifier.
func TestBuildDraftEnrichment(t *testing.T) {
	byName := map[string]*ent.Tag{
		"homelab": {ID: 10, Name: "homelab", Kind: tag.KindProject},
		"errands": {ID: 20, Name: "errands", Kind: tag.KindContext},
	}

	d := buildDraft(&llm.CreateDraft{
		Type:    "task",
		Title:   "renew passport",
		Project: "HomeLab", // case-insensitive resolution
		Tags:    []string{"Errands", "travel"},
	}, byName)

	if d.ProjectID == nil || *d.ProjectID != 10 || d.ProjectName != "homelab" {
		t.Fatalf("project not resolved: %+v", d)
	}
	if len(d.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %+v", d.Tags)
	}
	if d.Tags[0].ID != 20 || d.Tags[0].Name != "errands" {
		t.Fatalf("known tag not resolved: %+v", d.Tags[0])
	}
	if d.Tags[1].ID != 0 || d.Tags[1].Name != "travel" {
		t.Fatalf("unknown tag should be a candidate with no ID: %+v", d.Tags[1])
	}
}

// TestBuildDraftWrongKind: a context name in the project slot must not
// resolve to a project.
func TestBuildDraftWrongKind(t *testing.T) {
	byName := map[string]*ent.Tag{
		"errands": {ID: 20, Name: "errands", Kind: tag.KindContext},
	}
	d := buildDraft(&llm.CreateDraft{Type: "task", Title: "x", Project: "errands"}, byName)
	if d.ProjectID != nil {
		t.Fatalf("context resolved as project: %+v", d)
	}
	if d.ProjectName != "errands" {
		t.Fatalf("candidate name lost: %+v", d)
	}
}
