#!/bin/sh
# Starts the whole app with the real frontend and a running private site (a real container), then
# drives it with a signed-out browser. Needs Docker, Go, Node 24 and Playwright:
#   npm i -g playwright && npx playwright install chromium
# SANDBOX_TEST_IMAGE picks the toolbox image; screenshots go to E2E_SHOTS (default: a temp folder).
set -eu

image="${SANDBOX_TEST_IMAGE:-ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest}"
dir="$(mktemp -d)"
trap 'touch "$dir/done"; sleep 1; rm -rf "$dir"' EXIT

make frontend >/dev/null

SITES_BROWSER_DIR="$dir" SANDBOX_TEST_IMAGE="$image" \
  go test ./internal/bootstrap/ -run TestServeSiteForABrowser -count=1 -timeout 8m >"$dir/server.log" 2>&1 &

for _ in $(seq 1 90); do
  [ -f "$dir/info.json" ] && break
  sleep 1
done
[ -f "$dir/info.json" ] || { cat "$dir/server.log"; echo "the test server did not start" >&2; exit 1; }

shots="${E2E_SHOTS:-$dir}"
mkdir -p "$shots"
PW="$(npm root -g)/playwright" node scripts/e2e-sites.ts "$dir/info.json" "$shots"
