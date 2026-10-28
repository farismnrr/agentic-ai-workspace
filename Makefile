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
FLOW ?= all
SANDBOX_TEST_IMAGE ?= ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest
GOLANGCI_LINT_VERSION := $(shell cat .golangci-lint-version)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: help image sandbox-image redeploy-dev e2e build backend-build run verify verify-full guardrails shellcheck frontend frontend-deps frontend-verify backend-test backend-lint backend-verify blackbox smoke docker-boundary volume-isolation whitebox-check clean

help:
	@echo "make build    - verify, then build the single static binary ($(BIN_PATH))"
	@echo "make run      - build, then run the binary on PORT (default $(PORT))"
	@echo "make image    - build the Docker image ($(IMAGE):latest and :dev, what the compose files use)"
	@echo "make sandbox-image - build the toolbox image the agent's commands run in"
	@echo "make redeploy-dev - pull the app image published from dev and recreate the app container"
	@echo "make smoke    - smoke-test the built binary"
	@echo "make e2e      - run browser flows (or FLOW=projects, sites or thread)"
	@echo "make verify-full - run verify, black-box, smoke, browser and isolation checks"
	@echo "make whitebox-check - check the retained Go test registry"
	@echo "make verify   - run every quality check (Go + frontend)"
	@echo "make blackbox - run the built-binary black-box scenarios"
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

# Pull the app and sandbox images published from dev and recreate the app container.
redeploy-dev:
	docker pull $(IMAGE)-sandbox:latest
	docker compose -f compose.dev.yaml pull
	docker compose -f compose.dev.yaml up -d --wait --remove-orphans app

smoke:
	@./test/run.sh smoke

e2e:
	@set -eu; \
	if [ "$(FLOW)" = all ]; then flows='projects sites thread'; else flows='$(FLOW)'; fi; \
	for flow in $$flows; do ./test/run.sh e2e "$$flow"; done

whitebox-check:
	@./test/run.sh whitebox-check

guardrails:
	MAX_FILE_LINES="$(MAX_FILE_LINES)" sh ./scripts/check-guardrails.sh

shellcheck:
	@if command -v shellcheck >/dev/null 2>&1; then \
		find scripts test -type f -name '*.sh' -exec shellcheck -x {} +; \
	elif [ "$${CI:-}" = true ]; then \
		echo "shellcheck is required in CI but was not found." >&2; exit 1; \
	else \
		echo "shellcheck not found; skipping local shell checks."; \
	fi

$(FRONTEND_DEPS_STAMP): $(WEB_DIR)/package-lock.json
	npm --prefix "$(WEB_DIR)" ci --ignore-scripts --no-audit --no-fund

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

verify: guardrails shellcheck frontend-verify backend-verify
	@$(MAKE) backend-build
	@APP_BIN="$(abspath $(BIN_PATH))" $(MAKE) blackbox smoke

blackbox:
	@./test/run.sh blackbox "$(SCENARIO)"

docker-boundary:
	@if [ -z "$(SANDBOX_TEST_IMAGE)" ]; then \
		if [ "$${CI:-}" = true ]; then echo "SANDBOX_TEST_IMAGE is required in CI." >&2; exit 1; fi; \
		echo "SANDBOX_TEST_IMAGE is unset; skipping Docker boundary checks."; \
	elif ! command -v docker >/dev/null 2>&1; then \
		if [ "$${CI:-}" = true ]; then echo "Docker is required for boundary checks in CI." >&2; exit 1; fi; \
		echo "Docker not found; skipping Docker boundary checks."; \
	elif ! docker info >/dev/null 2>&1; then \
		if [ "$${CI:-}" = true ]; then echo "Docker daemon is required for boundary checks in CI." >&2; exit 1; fi; \
		echo "Docker daemon unavailable; skipping Docker boundary checks."; \
	else \
		sh test/smoke/sandbox.test.sh "$(SANDBOX_TEST_IMAGE)"; \
	fi

volume-isolation:
	@if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then \
		if [ "$${CI:-}" = true ]; then echo "Docker is required for volume isolation checks in CI." >&2; exit 1; fi; \
		echo "Docker unavailable; skipping volume isolation check."; exit 0; \
	fi; \
	VOLUME_ISOLATION_IMAGE="$${VOLUME_ISOLATION_IMAGE:-$(SANDBOX_TEST_IMAGE)}" \
	SANDBOX_TEST_IMAGE="$(SANDBOX_TEST_IMAGE)" \
		bash test/smoke/volume-isolation.test.sh

verify-full: verify
	@APP_BIN="$(abspath $(BIN_PATH))" SANDBOX_TEST_IMAGE="$(SANDBOX_TEST_IMAGE)" $(MAKE) e2e
	@SANDBOX_TEST_IMAGE="$(SANDBOX_TEST_IMAGE)" $(MAKE) docker-boundary
	@$(MAKE) volume-isolation
	go test -race -count=1 ./internal/platform/egressproxy ./internal/modules/task/adapter/sqlite ./internal/platform/database

clean:
	rm -rf "$(BIN_DIR)" "$(WEB_DIR)/node_modules" "$(WEB_DIR)/.vite" "$(WEB_DIR)/.svelte-check"
	find "$(WEB_EMBED_DIR)" -mindepth 1 -maxdepth 1 ! -name placeholder.txt -exec rm -rf {} +
