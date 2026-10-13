SHELL := /bin/sh

# Port the binary listens on when started with `make run`.
PORT ?= 8080

APP_NAME := agentic-ai-workspace
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
WEB_DIR := web
WEB_EMBED_DIR := internal/platform/webassets/dist
FRONTEND_DEPS_STAMP := $(WEB_DIR)/node_modules/.package-lock.json
MAX_FILE_LINES := 200
GOLANGCI_LINT_VERSION := $(shell cat .golangci-lint-version)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: help build backend-build run verify guardrails frontend frontend-deps frontend-verify backend-test backend-lint backend-verify clean

help:
	@echo "make build    - verify, then build the single static binary ($(BIN_PATH))"
	@echo "make run      - build, then run the binary on PORT (default $(PORT))"
	@echo "make verify   - run every quality check (Go + frontend)"
	@echo "make clean    - remove bin/, node_modules and generated frontend output"

# Sequenced through a sub-make so `make -j` cannot embed a stale frontend build.
build: verify
	@$(MAKE) backend-build

backend-build:
	@mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$(BIN_PATH)" ./cmd/workspace

run: build
	PORT="$(PORT)" "./$(BIN_PATH)"

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
