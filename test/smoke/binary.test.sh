#!/usr/bin/env bash
# Proves the built binary serves the SPA shell and API.
set -euo pipefail

bin="${1:?usage: binary.test.sh <binary>}"
REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/app.sh"
port=${PORT:-$(app_free_port)}

data="$(mktemp -d)"
PORT="$port" DATA_DIR="$data" "$bin" &
pid=$!
cleanup() {
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
  fi
  wait "$pid" 2>/dev/null || true
  rm -rf -- "$data"
}
trap cleanup EXIT

for _ in $(seq 1 30); do
  curl -fsS "http://127.0.0.1:${port}/api/health" >/dev/null 2>&1 && break
  sleep 1
done

curl -fsS "http://127.0.0.1:${port}/api/health" >/dev/null
curl -fsS "http://127.0.0.1:${port}/" | grep -qi '<div id="app"\|<html'
curl -fsS "http://127.0.0.1:${port}/.well-known/oauth-authorization-server" | grep -q '"code_challenge_methods_supported":\["S256"\]'
test "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${port}/api/providers")" = "401"
echo "smoke test passed"
