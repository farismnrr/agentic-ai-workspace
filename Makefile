SHELL := /bin/sh

APP_NAME := agentic-ai-workspace
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
WEB_DIR := web
WEB_EMBED_DIR := internal/web/dist

.PHONY: build frontend backend run verify clean

build: backend

frontend:
	@mkdir -p "$(WEB_EMBED_DIR)"
	@rm -rf "$(WEB_EMBED_DIR)/assets" "$(WEB_EMBED_DIR)/index.html"
	npm --prefix "$(WEB_DIR)" install --no-audit --no-fund
	npm --prefix "$(WEB_DIR)" run build

backend: frontend
	@mkdir -p "$(BIN_DIR)"
	go build -trimpath -ldflags="-s -w" -o "$(BIN_PATH)" ./cmd/workspace

run: build
	"./$(BIN_PATH)"

verify: frontend
	go test ./...
	go vet ./...

clean:
	rm -rf "$(BIN_DIR)" "$(WEB_DIR)/node_modules" "$(WEB_DIR)/.vite" "$(WEB_EMBED_DIR)/assets" "$(WEB_EMBED_DIR)/index.html"
