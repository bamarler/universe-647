package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

const embedDims = 768

// Gateway talks to any OpenAI-compatible LLM gateway (Bifrost today; LiteLLM
// before it; a local model server later — same API shape either way).
// local model later — same API shape either way).
type Gateway struct {
	api         openai.Client
	embedModel  string
	intentModel string
}

func New(baseURL, apiKey, embedModel, intentModel string) (*Gateway, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid LLM gateway base URL %q", baseURL)
	}
	c := &Gateway{
		api: openai.NewClient(
			option.WithBaseURL(strings.TrimRight(baseURL, "/")),
			option.WithAPIKey(apiKey),
		),
		embedModel:  embedModel,
		intentModel: intentModel,
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.ping(pingCtx, baseURL, apiKey); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Gateway) ping(ctx context.Context, baseURL, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUnavailable, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: gateway returned %d", ErrUnavailable, resp.StatusCode)
	}
	return nil
}

func (c *Gateway) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	resp, err := c.api.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model:      c.embedModel,
		Input:      openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: texts},
		Dimensions: openai.Int(embedDims),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: embeddings: %s", ErrUnavailable, err.Error())
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embeddings: got %d vectors for %d inputs", len(resp.Data), len(texts))
	}
	out := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		v := make([]float32, len(d.Embedding))
		for j, f := range d.Embedding {
			v[j] = float32(f)
		}
		normalize(v) // MRL truncation requires re-normalization
		out[i] = v
	}
	return out, nil
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	n := math.Sqrt(sum)
	if n == 0 {
		return
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / n)
	}
}

const systemPrompt = `You are the intent parser for a personal task/notes system.
Translate the user's input into EXACTLY ONE tool call. Never reply with text.

Rules:
- Requests to add/remember/create something -> create_item. Extract a concise
  title; put extra detail in body_md. Pick due_at (deadline) vs defer_at
  (hide-until) carefully. Dates are ISO 8601 (YYYY-MM-DD or RFC3339).
- Questions about what exists / what to do / finding things -> search.
  Time-window questions ("this week") use due_within_days and EMPTY query.
  Content questions put the topic words in query.
- Only use project/tag names from the provided vocabulary when they clearly
  match; otherwise emit the user's words verbatim as new tag candidates.
- priority: 0 none, 1 low, 2 medium, 3 high. Default 0.`

func toolSchemas() []openai.ChatCompletionToolUnionParam {
	createParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     map[string]any{"type": "string", "enum": []string{"task", "note", "tag"}},
			"title":    map[string]any{"type": "string"},
			"body_md":  map[string]any{"type": "string"},
			"kind":     map[string]any{"type": "string", "enum": []string{"project", "context", "tag"}},
			"project":  map[string]any{"type": "string"},
			"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"due_at":   map[string]any{"type": "string"},
			"defer_at": map[string]any{"type": "string"},
			"priority": map[string]any{"type": "integer", "minimum": 0, "maximum": 3},
		},
		"required": []string{"type", "title"},
	}
	searchParams := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":           map[string]any{"type": "string"},
			"project":         map[string]any{"type": "string"},
			"tags":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"due_within_days": map[string]any{"type": "integer"},
			"status":          map[string]any{"type": "string", "enum": []string{"open", "done", "any"}},
			"types":           map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"task", "note"}}},
		},
	}
	return []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        "create_item",
			Description: openai.String("Draft a new task, note, or tag for the user to review and edit."),
			Parameters:  createParams,
		}),
		openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        "search",
			Description: openai.String("Find existing items by filters and/or a semantic query."),
			Parameters:  searchParams,
		}),
	}
}

func (c *Gateway) ParseCommand(ctx context.Context, input string, now time.Time, vocab Vocab) (*Command, error) {
	user := fmt.Sprintf(
		"Current datetime: %s (%s)\nProjects: %s\nContexts: %s\nTags: %s\n\nInput: %s",
		now.Format(time.RFC3339), now.Weekday(),
		strings.Join(vocab.Projects, ", "),
		strings.Join(vocab.Contexts, ", "),
		strings.Join(vocab.Tags, ", "),
		input,
	)
	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.intentModel,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(user),
		},
		Tools:      toolSchemas(),
		ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{OfAuto: openai.String("required")},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: chat: %s", ErrUnavailable, err.Error())
	}
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("intent parser returned no tool call")
	}
	tc := resp.Choices[0].Message.ToolCalls[0]
	args := tc.Function.Arguments

	switch tc.Function.Name {
	case "create_item":
		var raw struct {
			Type     string   `json:"type"`
			Title    string   `json:"title"`
			BodyMD   string   `json:"body_md"`
			Kind     string   `json:"kind"`
			Project  string   `json:"project"`
			Tags     []string `json:"tags"`
			DueAt    string   `json:"due_at"`
			DeferAt  string   `json:"defer_at"`
			Priority int      `json:"priority"`
		}
		if err := json.Unmarshal([]byte(args), &raw); err != nil {
			return nil, fmt.Errorf("parse create_item args: %w", err)
		}
		return &Command{Action: "create", Create: &CreateDraft{
			Type:     raw.Type,
			Title:    raw.Title,
			BodyMD:   raw.BodyMD,
			Kind:     raw.Kind,
			Project:  raw.Project,
			Tags:     raw.Tags,
			DueAt:    parseWhen(raw.DueAt, now),
			DeferAt:  parseWhen(raw.DeferAt, now),
			Priority: raw.Priority,
		}}, nil
	case "search":
		var s SearchIntent
		if err := json.Unmarshal([]byte(args), &s); err != nil {
			return nil, fmt.Errorf("parse search args: %w", err)
		}
		return &Command{Action: "search", Search: &s}, nil
	default:
		return nil, fmt.Errorf("unknown tool %q", tc.Function.Name)
	}
}

// parseWhen accepts RFC3339 or bare dates; bare due dates land end-of-day so
// "due Friday" doesn't show as overdue Friday morning.
func parseWhen(s string, now time.Time) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	if d, err := time.ParseInLocation("2006-01-02", s, now.Location()); err == nil {
		t := d.Add(23*time.Hour + 59*time.Minute)
		return &t
	}
	return nil
}
