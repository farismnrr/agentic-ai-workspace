#!/bin/sh
# Starts the whole app with the real frontend (the same test server as e2e-thread) and drives the New project
# dialog in a browser, with GitHub's answers faked in the page. Needs Go, Node 24 and Playwright:
#   npm i -g playwright && npx playwright install chromium
set -eu

image="${SANDBOX_TEST_IMAGE:-ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest}"
dir="$(mktemp -d)"
trap 'touch "$dir/done"; sleep 1; rm -rf "$dir"' EXIT

make frontend >/dev/null

THREAD_BROWSER_DIR="$dir" SANDBOX_TEST_IMAGE="$image" \
  go test ./internal/bootstrap/ -run TestServeThreadForABrowser -count=1 -timeout 8m >"$dir/server.log" 2>&1 &

for _ in $(seq 1 120); do
  [ -f "$dir/info.json" ] && break
  sleep 1
done
[ -f "$dir/info.json" ] || { cat "$dir/server.log"; echo "the test server did not start" >&2; exit 1; }

PW="$(npm root -g)/playwright" node scripts/e2e-projects.ts "$dir/info.json" "${E2E_SHOTS:-$dir}"
