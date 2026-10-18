# syntax=docker/dockerfile:1

# The toolbox the agent's commands run in: Python, Node.js, Bun, Go, Rust, Java and Bash.
# Every layer is read-only and shared by all containers started from this image, so each
# toolchain is stored once on disk however many people run commands at the same time.
# Build from the repository root: docker build -f sandbox/Dockerfile .

ARG PYTHON_VERSION=3.13

FROM ghcr.io/astral-sh/uv:0.9@sha256:538e0b39736e7feae937a65983e49d2ab75e1559d35041f9878b7b7e51de91e4 AS uv
FROM node:24-bookworm-slim@sha256:d6aa754f16b3197301076f047b5def2f02ea1dbbc2ca920407d46d7ec7f87b20 AS node
FROM golang:1.27.2-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS go
FROM rust:1-bookworm@sha256:114c7a4425406451c2866b6aafe69fe29b1b298832db1277d411ac73c82d04d6 AS rust
FROM eclipse-temurin:21-jdk-jammy@sha256:e0c60c487345d1dc9d0fc7b6f0496f3cc941e5132e09296cc17a6decc71b902b AS java
FROM maven:3.9-eclipse-temurin-21-jammy@sha256:7f8422bd81f19b9035007a75e2f82272b09af64db08848d7d4e93302466e8b6f AS maven

# Python comes from uv's standalone builds, installed once into /opt/python.
FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS python
ARG PYTHON_VERSION
COPY --from=uv /uv /uvx /usr/local/bin/
ENV UV_PYTHON_INSTALL_DIR=/opt/python UV_PYTHON_BIN_DIR=/opt/python/bin
RUN --mount=type=cache,target=/root/.cache/uv uv python install "${PYTHON_VERSION}"

# The GitHub CLI, from the official release, pinned to a version and checked against its published checksum.
FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS gh
ARG TARGETARCH
ARG GH_VERSION=2.102.0
ARG GH_SHA256_AMD64=bb766f710eef8ede859c18578c72c327597cd4c8a85b06001b1f3843c6019386
ARG GH_SHA256_ARM64=7862c86c72f43df3a2d93ddde6f473285b4e2af61b494849846827e513ef6484
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && case "${TARGETARCH}" in amd64) sum="${GH_SHA256_AMD64}" ;; arm64) sum="${GH_SHA256_ARM64}" ;; *) echo "unsupported architecture ${TARGETARCH}" >&2; exit 1 ;; esac \
    && curl -fsSL -o /tmp/gh.tar.gz "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${TARGETARCH}.tar.gz" \
    && echo "${sum}  /tmp/gh.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/gh.tar.gz -C /tmp \
    && install -m 0755 "/tmp/gh_${GH_VERSION}_linux_${TARGETARCH}/bin/gh" /usr/local/bin/gh

# Background removal: rembg (CPU) in its own virtual environment, with the two models downloaded now so that
# nothing is fetched while the agent works. Models are named explicitly because the library's default is a
# 1 GB model under a paid commercial licence; these two are Apache-2.0.
FROM python AS rembg
ARG REMBG_VERSION=2.0.85
ENV REMBG_HOME=/opt/rembg-models
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN --mount=type=cache,target=/root/.cache/uv \
    uv venv /opt/rembg --python "$(ls -d /opt/python/cpython-*/bin/python3 | head -n1)" \
    && uv pip install --python /opt/rembg/bin/python "rembg[cpu]==${REMBG_VERSION}" pillow \
    && /opt/rembg/bin/python -c "from rembg import new_session; [new_session(m) for m in ('u2netp', 'isnet-general-use')]"
RUN chmod -R a+rX /opt/rembg /opt/rembg-models

# Vector tools from npm: svgo shrinks SVG, and lucide-static is a set of ready-made icons (ISC licence) so the
# agent picks icons instead of drawing them. Installed now, so they work offline.
FROM node AS vector
ARG SVGO_VERSION=4.1.0
ARG LUCIDE_VERSION=1.59.0
RUN npm install --prefix /opt/vector --no-audit --no-fund "svgo@${SVGO_VERSION}" "lucide-static@${LUCIDE_VERSION}" \
    && chmod -R a+rX /opt/vector

# Bun, the runtime and package manager apps the AI builds run on, from the official release, pinned to a version and
# checked against its published checksum.
FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS bun
ARG TARGETARCH
ARG BUN_VERSION=1.4.3
ARG BUN_SHA256_AMD64=3e730528d89775f03f87f05c06787c4df4099d0a37f3f252e7b829cb23816e18
ARG BUN_SHA256_ARM64=efa9813da5ed72423bf847f916e8d2c47c0d776add972354026a75e10da9aa21
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl unzip \
    && rm -rf /var/lib/apt/lists/* \
    && case "${TARGETARCH}" in amd64) sum="${BUN_SHA256_AMD64}"; file=bun-linux-x64 ;; arm64) sum="${BUN_SHA256_ARM64}"; file=bun-linux-aarch64 ;; *) echo "unsupported architecture ${TARGETARCH}" >&2; exit 1 ;; esac \
    && curl -fsSL -o /tmp/bun.zip "https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/${file}.zip" \
    && echo "${sum}  /tmp/bun.zip" | sha256sum -c - \
    && unzip -q /tmp/bun.zip -d /tmp \
    && install -m 0755 "/tmp/${file}/bun" /usr/local/bin/bun

FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587
ARG PYTHON_VERSION

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash build-essential ca-certificates curl git socat unzip zip jq ripgrep procps \
        chromium-headless-shell fonts-liberation fonts-noto-color-emoji \
        libvips-tools imagemagick librsvg2-bin webp libavif-bin optipng pngquant libimage-exiftool-perl potrace \
    && rm -rf /var/lib/apt/lists/*

COPY --from=uv /uv /uvx /usr/local/bin/
COPY --from=gh /usr/local/bin/gh /usr/local/bin/gh
COPY --from=bun /usr/local/bin/bun /usr/local/bin/bun
COPY --from=python /opt/python /opt/python
RUN ln -s "$(ls -d /opt/python/cpython-*/bin/python3 | head -n1)" /usr/local/bin/python3 \
    && ln -s python3 /usr/local/bin/python \
    && printf '#!/bin/sh\nexec python3 -m pip "$@"\n' > /usr/local/bin/pip \
    && chmod 0755 /usr/local/bin/pip \
    && ln -s pip /usr/local/bin/pip3

COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
    && ln -s ../lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack

COPY --from=go /usr/local/go /usr/local/go
COPY --from=rust /usr/local/cargo/bin /opt/cargo/bin
COPY --from=rust /usr/local/rustup /opt/rustup
COPY --from=java /opt/java/openjdk /opt/java/openjdk
COPY --from=maven /usr/share/maven /opt/maven

COPY --from=vector /opt/vector /opt/vector
COPY --from=rembg /opt/rembg /opt/rembg
COPY --from=rembg /opt/rembg-models /opt/rembg-models
COPY sandbox/removebg.py /usr/local/bin/removebg
COPY sandbox/imagemagick-policy.xml /etc/ImageMagick-6/policy.xml
COPY sandbox/sandbox-entry.sh /usr/local/bin/sandbox-entry
COPY sandbox/capture/ /usr/local/share/capture/
COPY sandbox/nuxt-serve.ts /usr/local/share/serve/nuxt-serve.ts
COPY sandbox/starter/nuxt/ /usr/local/share/starter/nuxt/
COPY sandbox/new-nuxt-app.sh /usr/local/bin/new-nuxt-app
RUN chmod 0755 /usr/local/bin/sandbox-entry /usr/local/bin/removebg /usr/local/bin/new-nuxt-app \
    && ln -s /opt/vector/node_modules/.bin/svgo /usr/local/bin/svgo \
    && ln -s bun /usr/local/bin/bunx

# Read-only toolchains stay put; downloads and build output go to /cache, which the server mounts per person.
ENV RUSTUP_HOME=/opt/rustup \
    JAVA_HOME=/opt/java/openjdk \
    MAVEN_HOME=/opt/maven \
    REMBG_HOME=/opt/rembg-models \
    GOFLAGS="-modcacherw -buildvcs=false" \
    PATH=/usr/local/go/bin:/opt/cargo/bin:/opt/java/openjdk/bin:/opt/maven/bin:/usr/local/bin:/usr/bin:/bin

# One JSON list the server reads to tell the agent what it can use.
LABEL org.opencontainers.image.title="agentic-ai-workspace-sandbox" \
      org.opencontainers.image.source="https://github.com/farismnrr/agentic-ai-workspace" \
      org.opencontainers.image.description="Isolated toolbox for the agent's commands" \
      io.masihawam.toolchains="[{\"name\":\"Python\",\"version\":\"3.13\"},{\"name\":\"Node.js\",\"version\":\"24\"},{\"name\":\"Bun\",\"version\":\"1.4\"},{\"name\":\"Go\",\"version\":\"1.27\"},{\"name\":\"Rust\",\"version\":\"stable\"},{\"name\":\"Java\",\"version\":\"21\"},{\"name\":\"Maven\",\"version\":\"3.9\"},{\"name\":\"Bash\",\"version\":\"5\"},{\"name\":\"Chromium (headless, for screenshots)\",\"version\":\"155\"},{\"name\":\"libvips\",\"version\":\"8.14\"},{\"name\":\"ImageMagick\",\"version\":\"6.9\"},{\"name\":\"Background removal (removebg, offline)\",\"version\":\"rembg 2\"},{\"name\":\"Vector (svgo, potrace, Lucide icons in /opt/vector)\",\"version\":\"svgo 4\"}]"

USER 65532:65532
WORKDIR /workspace
