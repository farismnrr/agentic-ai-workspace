SHELL := /bin/sh

APP_NAME := agentic-ai-workspace
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
WEB_DIR := web
WEB_EMBED_DIR := internal/platform/webassets/dist
GO_PACKAGES := ./cmd/... ./internal/...
MAX_FILE_LINES := 200
GOLANGCI_LINT_VERSION := v2.14.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: build run verify guardrails frontend frontend-deps frontend-verify backend-verify clean

build: verify
	@mkdir -p "$(BIN_DIR)"
	go build -trimpath -ldflags="-s -w" -o "$(BIN_PATH)" ./cmd/workspace

run: build
	"./$(BIN_PATH)"

guardrails:
	MAX_FILE_LINES="$(MAX_FILE_LINES)" sh ./scripts/check-guardrails.sh

frontend-deps:
	npm --prefix "$(WEB_DIR)" install --no-audit --no-fund

frontend: frontend-deps
	npm --prefix "$(WEB_DIR)" run build

frontend-verify: frontend-deps
	npm --prefix "$(WEB_DIR)" run check
	npm --prefix "$(WEB_DIR)" run lint
	npm --prefix "$(WEB_DIR)" run build

backend-verify: frontend-verify
	go mod tidy -diff
	go test $(GO_PACKAGES)
	$(GOLANGCI_LINT) run $(GO_PACKAGES)

verify: guardrails backend-verify

clean:
	rm -rf "$(BIN_DIR)" "$(WEB_DIR)/node_modules" "$(WEB_DIR)/.vite" "$(WEB_DIR)/.svelte-check"
	find "$(WEB_EMBED_DIR)" -mindepth 1 -maxdepth 1 ! -name placeholder.txt -exec rm -rf {} +
