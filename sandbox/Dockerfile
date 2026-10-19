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

# The official Nuxt UI templates (MIT, github.com/nuxt-ui-templates), each pinned to a commit, with no history and
# no dependencies installed. They are a UI reference only: the Designer reads them, the app keeps our own backend.
FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS templates
ARG TEMPLATES="landing:33723eb83d140042c4ecaa137ffdf187e1aaa34b saas:23023a19035f3008e12e8bffebcd473719981766 dashboard:c705472bd54c75eb98954d3cefa5f7402efd7f7c docs:1ed7ef403974ec3a38f9779480e2460621f73d13 chat:7e920565b33abe0c40543795e39ac7c5c2039be8 portfolio:83f6355f8710369c9353472bcf3a9091de551eee changelog:597fa840d1bd6148228ef29ecdf70d7a279c4cd9 editor:ba305c8b47267cc5bc04cddb123a64c1675ab5db calendar:11c3bab17fc5a1408b3a79707ecbb3369992960d"
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git \
    && rm -rf /var/lib/apt/lists/* \
    && for entry in ${TEMPLATES}; do \
         name="${entry%%:*}"; sha="${entry##*:}"; dir="/opt/templates/${name}"; \
         mkdir -p "${dir}" && git -C "${dir}" init -q \
         && git -C "${dir}" fetch -q --depth 1 "https://github.com/nuxt-ui-templates/${name}" "${sha}" \
         && git -C "${dir}" checkout -q FETCH_HEAD \
         && rm -rf "${dir}/.git" "${dir}/.github" "${dir}/pnpm-lock.yaml" "${dir}/renovate.json"; \
       done

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

# Documents: python-pptx, python-docx, reportlab, pypdf and openpyxl in their own virtual environment, so the Documents
# role writes PowerPoint, Word, PDF and Excel files offline. LibreOffice and poppler (installed below) render them.
FROM python AS docs
ARG PPTX_VERSION=1.0.2
ARG DOCX_VERSION=1.2.0
ARG REPORTLAB_VERSION=5.0.1
ARG PYPDF_VERSION=6.21.1
ARG OPENPYXL_VERSION=3.1.5
ARG PILLOW_VERSION=13.0.0
RUN --mount=type=cache,target=/root/.cache/uv \
    uv venv /opt/docs --python "$(ls -d /opt/python/cpython-*/bin/python3 | head -n1)" \
    && uv pip install --python /opt/docs/bin/python "python-pptx==${PPTX_VERSION}" "python-docx==${DOCX_VERSION}" \
       "reportlab==${REPORTLAB_VERSION}" "pypdf==${PYPDF_VERSION}" "openpyxl==${OPENPYXL_VERSION}" "pillow==${PILLOW_VERSION}"
RUN chmod -R a+rX /opt/docs

# Vector tools from npm: svgo shrinks SVG, and lucide-static is a set of ready-made icons (ISC licence) so the
# agent picks icons instead of drawing them. Installed now, so they work offline.
FROM node AS vector
ARG SVGO_VERSION=4.1.0
ARG LUCIDE_VERSION=1.59.0
RUN npm install --prefix /opt/vector --no-audit --no-fund "svgo@${SVGO_VERSION}" "lucide-static@${LUCIDE_VERSION}" \
    && chmod -R a+rX /opt/vector

# The vendors' own coding CLIs, unmodified, from npm and pinned. People sign in to them with their own plan, and the
# server only starts them (never the agent's commands) in a container that can reach nothing but the vendor's hosts.
FROM node AS cli
ARG CLAUDE_CODE_VERSION=2.1.303
ARG CODEX_VERSION=0.163.0
RUN npm install --prefix /opt/cli --no-audit --no-fund "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
      "@openai/codex@${CODEX_VERSION}" \
    && chmod -R a+rX /opt/cli

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
        libreoffice-writer libreoffice-impress poppler-utils qpdf fonts-dejavu-core fonts-liberation2 \
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
COPY --from=cli /opt/cli /opt/cli
COPY --from=rembg /opt/rembg /opt/rembg
COPY --from=rembg /opt/rembg-models /opt/rembg-models
COPY --from=docs /opt/docs /opt/docs
COPY sandbox/docconvert.sh /usr/local/bin/docconvert
COPY sandbox/docsite.py /usr/local/share/docsite/docsite.py
COPY sandbox/removebg.py /usr/local/bin/removebg
COPY sandbox/imagemagick-policy.xml /etc/ImageMagick-6/policy.xml
COPY sandbox/sandbox-entry.sh /usr/local/bin/sandbox-entry
COPY sandbox/capture/ /usr/local/share/capture/
COPY sandbox/nuxt-serve.ts /usr/local/share/serve/nuxt-serve.ts
COPY sandbox/starter/nuxt/ /usr/local/share/starter/nuxt/
COPY --from=templates /opt/templates /usr/local/share/templates
COPY sandbox/new-nuxt-app.sh /usr/local/bin/new-nuxt-app
RUN chmod 0755 /usr/local/bin/sandbox-entry /usr/local/bin/removebg /usr/local/bin/docconvert /usr/local/bin/new-nuxt-app \
    && ln -s /opt/vector/node_modules/.bin/svgo /usr/local/bin/svgo \
    && ln -s /opt/cli/node_modules/.bin/claude /usr/local/bin/claude \
    && ln -s /opt/cli/node_modules/.bin/codex /usr/local/bin/codex \
    && printf '#!/bin/sh\nexec /opt/docs/bin/python "$@"\n' > /usr/local/bin/docpython \
    && chmod 0755 /usr/local/bin/docpython \
    && printf '#!/bin/sh\nexec /opt/docs/bin/python /usr/local/share/docsite/docsite.py "$@"\n' > /usr/local/bin/docsite \
    && chmod 0755 /usr/local/bin/docsite \
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
      io.masihawam.toolchains="[{\"name\":\"Python\",\"version\":\"3.13\"},{\"name\":\"Node.js\",\"version\":\"24\"},{\"name\":\"Bun\",\"version\":\"1.4\"},{\"name\":\"Go\",\"version\":\"1.27\"},{\"name\":\"Rust\",\"version\":\"stable\"},{\"name\":\"Java\",\"version\":\"21\"},{\"name\":\"Maven\",\"version\":\"3.9\"},{\"name\":\"Bash\",\"version\":\"5\"},{\"name\":\"Chromium (headless, for screenshots)\",\"version\":\"155\"},{\"name\":\"libvips\",\"version\":\"8.14\"},{\"name\":\"ImageMagick\",\"version\":\"6.9\"},{\"name\":\"Background removal (removebg, offline)\",\"version\":\"rembg 2\"},{\"name\":\"Documents (docpython with python-pptx, python-docx, reportlab, pypdf, openpyxl; docconvert via LibreOffice; docsite makes a website of them; poppler)\",\"version\":\"pptx 1, docx 1, reportlab 5\"},{\"name\":\"Vector (svgo, potrace, Lucide icons in /opt/vector)\",\"version\":\"svgo 4\"}]"

USER 65532:65532
WORKDIR /workspace
