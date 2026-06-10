// Package llm defines the narrow seam to the LiteLLM gateway. The AI is a
// retrieval-only middleman: it embeds text and parses commands into structured
// intents — it never produces prose shown to the user.
package llm

import (
	"context"
	"errors"
)

var ErrUnavailable = errors.New("llm gateway unavailable")

// Client is the only interface the rest of the app sees. Implementations:
// the LiteLLM-backed client (step 5) and Disabled (graceful degradation when
// the gateway is down — the selfserve tryInit pattern).
type Client interface {
	// Embed returns one normalized 768-dim vector per input text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Disabled is a Client that reports unavailability; the app boots and serves
// everything except semantic features.
type Disabled struct{}

func (Disabled) Embed(context.Context, []string) ([][]float32, error) {
	return nil, ErrUnavailable
}
