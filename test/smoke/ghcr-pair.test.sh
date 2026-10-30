#!/usr/bin/env bash
# Black-box check: pull gate prevents mismatched GHCR pairs before compose runs.
set -euo pipefail
repo=$(cd "$(dirname "$0")/../.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf -- "$tmp"' EXIT
cat > "$tmp/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "$*" >> "$DOCKER_CALLS"
case "$1 ${2:-}" in
  'info ') exit 0 ;;
  pull\ *) exit 0 ;;
  'image inspect')
    if [[ "$*" == *-sandbox:* && "${FAKE_MISMATCH:-0}" = 1 ]]; then
      printf 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n'
    else
      printf 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n'
    fi ;;
  *) echo 'Unexpected Docker command' >&2; exit 9 ;;
esac
EOF
chmod +x "$tmp/docker"
export DOCKER_CALLS="$tmp/calls"
PATH="$tmp:$PATH" bash "$repo/scripts/pull-ghcr-pair.sh" dev > "$tmp/out"
grep -q 'PASS matching app + sandbox' "$tmp/out"
test "$(grep -c '^pull ' "$tmp/calls")" -eq 2
! grep -Eq '^build |^compose ' "$tmp/calls"
echo 'PASS matched GHCR app/sandbox pull-only'
: > "$tmp/calls"
if FAKE_MISMATCH=1 PATH="$tmp:$PATH" bash "$repo/scripts/pull-ghcr-pair.sh" dev > "$tmp/out" 2>&1; then
  echo 'FAIL: mismatched image labels were accepted' >&2
  exit 1
fi
grep -q '^BLOCKED:' "$tmp/out"
! grep -Eq '^build |^compose ' "$tmp/calls"
echo 'PASS mismatched revisions rejected without deployment'
