package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LiteLLM talks to the OpenAI-compatible gateway. Step 5 fills in the real
// /embeddings and /chat/completions calls; the constructor only validates
// reachability so app.Init can decide between live and Disabled.
type LiteLLM struct {
	baseURL    string
	apiKey     string
	embedModel string
	http       *http.Client
}

func New(baseURL, apiKey, embedModel string) (*LiteLLM, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid LiteLLM base URL %q", baseURL)
	}
	c := &LiteLLM{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		embedModel: embedModel,
		http:       &http.Client{Timeout: 30 * time.Second},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.ping(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *LiteLLM) ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUnavailable, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: gateway returned %d", ErrUnavailable, resp.StatusCode)
	}
	return nil
}

// Embed is implemented in step 5 (indexing worker + hybrid search).
func (c *LiteLLM) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("embeddings not yet implemented")
}
