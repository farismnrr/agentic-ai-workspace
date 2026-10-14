# syntax=docker/dockerfile:1

# 1. Compile the Svelte SPA (Node.js is a build-time dependency only).
FROM node:24-bookworm-slim@sha256:d6aa754f16b3197301076f047b5def2f02ea1dbbc2ca920407d46d7ec7f87b20 AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN mkdir -p ../internal/platform/webassets/dist && npm run build

# 2. Build the single static Go binary with the SPA embedded.
FROM golang:1.27.2-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=web /src/internal/platform/webassets/dist internal/platform/webassets/dist
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/agentic-ai-workspace ./cmd/workspace \
    && mkdir /out/data

# 3. The runtime image holds the binary and nothing else.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
LABEL org.opencontainers.image.title="agentic-ai-workspace" \
      org.opencontainers.image.source="https://github.com/farismnrr/agentic-ai-workspace" \
      org.opencontainers.image.description="Proof-first AI work for people who are not developers"
COPY --from=server /out/agentic-ai-workspace /agentic-ai-workspace
COPY --from=server --chown=65532:65532 /out/data /data
ENV PORT=8081 DATA_DIR=/data
VOLUME /data
EXPOSE 8081
USER 65532:65532
ENTRYPOINT ["/agentic-ai-workspace"]
