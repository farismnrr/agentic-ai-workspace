#!/bin/sh
# Starts the built binary and proves it serves the SPA shell and the API.
set -eu

bin="${1:?usage: smoke-test.sh <binary>}"
port="${PORT:-18081}"

PORT="$port" "$bin" &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' EXIT

for _ in $(seq 1 30); do
  curl -fsS "http://127.0.0.1:${port}/api/health" >/dev/null 2>&1 && break
  sleep 1
done

curl -fsS "http://127.0.0.1:${port}/api/health" >/dev/null
curl -fsS "http://127.0.0.1:${port}/" | grep -qi '<div id="app"\|<html'
echo "smoke test passed"
