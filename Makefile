SHELL := /bin/sh

# Port the binary listens on when started with `make run`.
PORT ?= 8081

APP_NAME := agentic-ai-workspace
IMAGE := ghcr.io/farismnrr/$(APP_NAME)
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
WEB_DIR := web
WEB_EMBED_DIR := internal/platform/webassets/dist
FRONTEND_DEPS_STAMP := $(WEB_DIR)/node_modules/.package-lock.json
MAX_FILE_LINES := 200
# Build version shown by /api/health and the X-App-Version header; CI passes the commit.
VERSION ?= dev
GOLANGCI_LINT_VERSION := $(shell cat .golangci-lint-version)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: help image sandbox-image redeploy-dev e2e-sites e2e-thread e2e-projects build backend-build run verify guardrails frontend frontend-deps frontend-verify backend-test backend-lint backend-verify clean

help:
	@echo "make build    - verify, then build the single static binary ($(BIN_PATH))"
	@echo "make run      - build, then run the binary on PORT (default $(PORT))"
	@echo "make image    - build the Docker image ($(IMAGE):latest and :dev, what the compose files use)"
	@echo "make sandbox-image - build the toolbox image the agent's commands run in"
	@echo "make redeploy-dev - pull the app image published from dev and recreate the app container"
	@echo "make e2e-sites - drive a private site with a real browser (needs Docker and Playwright)"
	@echo "make e2e-thread - check a task thread and its screenshots in a real browser (needs Docker and Playwright)"
	@echo "make verify   - run every quality check (Go + frontend)"
	@echo "make clean    - remove bin/, node_modules and generated frontend output"

# Sequenced through a sub-make so `make -j` cannot embed a stale frontend build.
build: verify
	@$(MAKE) backend-build

backend-build:
	@mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o "$(BIN_PATH)" ./cmd/workspace

# Checks run as Docker containers, so a local Docker daemon is needed to run them.
run: build
	@PORT="$(PORT)" "./$(BIN_PATH)"

image:
	docker build -t $(IMAGE):latest -t $(IMAGE):dev .

# The toolbox the agent's commands run in. Use it locally with SANDBOX_IMAGE=$(IMAGE)-sandbox:latest make run.
sandbox-image:
	docker build -f sandbox/Dockerfile -t $(IMAGE)-sandbox:latest .

# Pull only the app image, recreate its container and wait until it is healthy. The sandbox image is left
# alone on purpose: only Full publishes it, so pulling it would replace the one `make sandbox-image` built.
redeploy-dev:
	docker compose -f compose.dev.yaml pull app
	docker compose -f compose.dev.yaml up -d --wait app

# Browser end-to-end check of sites; not part of verify because it needs Docker and Playwright.
e2e-sites:
	sh ./scripts/e2e-sites.sh

e2e-thread:
	sh ./scripts/e2e-thread.sh

e2e-projects:
	sh ./scripts/e2e-projects.sh

guardrails:
	MAX_FILE_LINES="$(MAX_FILE_LINES)" sh ./scripts/check-guardrails.sh

$(FRONTEND_DEPS_STAMP): $(WEB_DIR)/package-lock.json
	npm --prefix "$(WEB_DIR)" ci --no-audit --no-fund

frontend-deps: $(FRONTEND_DEPS_STAMP)

frontend: frontend-deps
	npm --prefix "$(WEB_DIR)" run build

frontend-verify: frontend-deps
	npm --prefix "$(WEB_DIR)" run check
	npm --prefix "$(WEB_DIR)" run lint
	npm --prefix "$(WEB_DIR)" run build

backend-test:
	go mod tidy -diff
	go test ./cmd/... ./internal/...

backend-lint:
	$(GOLANGCI_LINT) run ./cmd/... ./internal/...

backend-verify: backend-test backend-lint

verify: guardrails frontend-verify backend-verify

clean:
	rm -rf "$(BIN_DIR)" "$(WEB_DIR)/node_modules" "$(WEB_DIR)/.vite" "$(WEB_DIR)/.svelte-check"
	find "$(WEB_EMBED_DIR)" -mindepth 1 -maxdepth 1 ! -name placeholder.txt -exec rm -rf {} +
