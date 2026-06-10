// genapi boots the router with stub dependencies purely to dump the OpenAPI
// spec (the skillspark trick) — the committed spec generates the TypeScript
// client for tomoko and, later, a Swift client.
package main

import (
	"fmt"
	"os"

	"github.com/bamarler/universe-647/sophon/internal/api"
	"github.com/bamarler/universe-647/sophon/internal/llm"
)

func main() {
	_, humaAPI := api.New(api.Deps{
		Ent: nil, // handlers are registered but never invoked
		LLM: llm.Disabled{},
		Dev: true,
	})
	out, err := humaAPI.OpenAPI().YAML()
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate openapi:", err)
		os.Exit(1)
	}
	os.Stdout.Write(out)
}
