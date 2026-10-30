#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
pid=''
cleanup() {
  [[ -z "$pid" ]] || { kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; }
  rm -rf -- "$work"
}
trap cleanup EXIT

python3 "$repo/test/smoke/marimo-mock-server.py" "$work/port" >"$work/server.log" 2>&1 &
pid=$!
for i in $(seq 1 40); do
  [[ ! -f "$work/port" ]] || break
  sleep .1
done
[[ -s "$work/port" ]] || { cat "$work/server.log"; exit 1; }
mkdir -p "$work/bin"
curl_bin=$(command -v curl)
cat > "$work/bin/curl" <<'EOF'
#!/bin/sh
# Child must not inherit the bearer credential or see it in argv.
if env | grep -q '^MARIMO_TOKEN='; then
  echo 'LEAK: curl inherited MARIMO_TOKEN' >&2
  exit 70
fi
for arg do
  case "$arg" in *synthetic-marimo-token*)
    echo 'LEAK: curl argv includes token' >&2
    exit 71 ;;
  esac
done
exec "$REAL_CURL" "$@"
EOF
chmod 755 "$work/bin/curl"
url="http://127.0.0.1:$(cat "$work/port")"
result=$(REAL_CURL="$curl_bin" PATH="$work/bin:$PATH" MARIMO_TOKEN=synthetic-marimo-token \
  bash "$repo/sandbox/marimo-pair/scripts/execute-code.sh" --url "$url" -c '1 + 1')
[[ "$result" == 'MOCK_OK' ]] || { echo "Unexpected response: $result"; exit 1; }
echo 'PASS mock marimo sessions+SSE auth; no credential in curl args or env'

if bash "$repo/sandbox/marimo-pair/scripts/execute-code.sh" --url 'http://remote.example.org/' -c '1 + 1' > "$work/out" 2>&1; then
  echo 'FAIL remote plaintext HTTP permitted'; exit 1
fi
grep -q 'must use HTTPS' "$work/out"
if bash "$repo/sandbox/marimo-pair/scripts/execute-code.sh" --url "$url" --token synthetic-marimo-token -c '1 + 1' > "$work/out" 2>&1; then
  echo 'FAIL --token permitted'; exit 1
fi
grep -q 'disabled' "$work/out"
echo 'PASS insecure remote URL and token argument rejected'

if REAL_CURL="$curl_bin" PATH="$work/bin:$PATH" MARIMO_TOKEN=wrong-synthetic-code-123 \
  bash "$repo/sandbox/marimo-pair/scripts/execute-code.sh" --url "$url" -c '1 + 1' >"$work/out" 2>&1; then
  echo 'FAIL: invalid session code was accepted'; exit 1
fi
grep -q '401' "$work/out"
echo 'PASS revoked/invalid session code returns HTTP 401'

if bash "$repo/sandbox/marimo-pair/scripts/execute-code.sh" --url 'http://127.0.0.1:1/' -c '1 + 1' >"$work/out" 2>&1; then
  echo 'FAIL: terminated notebook appeared available'; exit 1
fi
grep -q 'Failed to connect' "$work/out"
echo 'PASS terminated notebook returns connection error'
