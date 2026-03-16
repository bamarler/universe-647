# Universe 647 — Custom mcpo with pre-built monica-mcp
# Base: ghcr.io/open-webui/mcpo (Python + Node.js 22, Debian bookworm)
# Adds: Jacob-Stokes/monica-mcp built from source at pinned commit

FROM ghcr.io/open-webui/mcpo:git-788ff92

ARG MONICA_MCP_REPO=https://github.com/Jacob-Stokes/monica-mcp.git
ARG MONICA_MCP_COMMIT=788da8591ccd5603b40f78e67f61cd4d27563cc9

# Install git (for cloning) and gettext-base (for envsubst)
RUN apt-get update \
    && apt-get install -y --no-install-recommends git gettext-base \
    && rm -rf /var/lib/apt/lists/*

# Clone and build monica-mcp at pinned commit
RUN git clone "${MONICA_MCP_REPO}" /opt/monica-mcp \
    && cd /opt/monica-mcp \
    && git checkout "${MONICA_MCP_COMMIT}" \
    && rm -rf .git \
    && npm ci \
    && npm run build \
    && npm prune --production

# Add entrypoint wrapper for config templating
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /app
ENTRYPOINT ["/entrypoint.sh"]
