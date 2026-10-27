#!/bin/sh
# Starts the built image and proves it serves the SPA shell and the API.
set -eu

image="${1:?usage: image.test.sh <image>}"
port="${PORT:-18082}"

id="$(docker run -d --rm -p "127.0.0.1:${port}:8081" "$image")"
trap 'docker rm -f "$id" >/dev/null 2>&1 || true' EXIT

for _ in $(seq 1 30); do
  curl -fsS "http://127.0.0.1:${port}/api/health" >/dev/null 2>&1 && break
  sleep 1
done

curl -fsS "http://127.0.0.1:${port}/api/health" >/dev/null
curl -fsS "http://127.0.0.1:${port}/" | grep -qi '<div id="app"\|<html'
curl -fsS "http://127.0.0.1:${port}/.well-known/oauth-authorization-server" | grep -q '"code_challenge_methods_supported":\["S256"\]'
test "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${port}/api/providers")" = "401"
echo "image smoke test passed"
