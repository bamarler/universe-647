package api

import (
	"github.com/danielgtaylor/huma/v2"
)

// userHeader is set by Caddy from Authelia's forward-auth response. Sophon is
// only reachable through Caddy (no published ports), so the header is trusted;
// its absence means the request bypassed the proxy and is rejected.
const userHeader = "Remote-User"

type userKey struct{}

func authMiddleware(api huma.API, dev bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user := ctx.Header(userHeader)
		if user == "" {
			if dev {
				user = "dev"
			} else {
				_ = huma.WriteErr(api, ctx, 401, "missing identity header")
				return
			}
		}
		ctx = huma.WithValue(ctx, userKey{}, user)
		next(ctx)
	}
}
