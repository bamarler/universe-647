FROM caddy:2.11.1-builder AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/cloudflare@v0.2.4 \
    --with github.com/hslatman/caddy-crowdsec-bouncer

FROM caddy:2.11.1
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
