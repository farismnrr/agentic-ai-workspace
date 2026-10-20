#!/bin/sh
# Offline regression tests for imported Git repositories and ship status.
set -eu
root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git init -q --bare "$tmp/remote.git"
mkdir "$tmp/source"
(
  cd "$tmp/source"
  git init -q -b main
  git config user.name Test
  git config user.email test@example.com
  printf 'hello\n' > README.md
  git add README.md
  git commit -qm initial
  git remote add origin "$tmp/remote.git"
  git push -q origin main
)
# Bare repositories created on a branch other than main need their HEAD updated.
git --git-dir="$tmp/remote.git" symbolic-ref HEAD refs/heads/main
git clone -q --depth 1 "file://$tmp/remote.git" "$tmp/imported2"
(
  cd "$tmp/imported2"
  PROJECT_DIR="$tmp/imported2" sh "$root/sandbox/ship.sh" 'Test imported GitHub workflow' > "$tmp/out"
  grep -q 'pushed: yes (origin/dev at ' "$tmp/out"
  test "$(git rev-parse HEAD)" = "$(git ls-remote origin refs/heads/dev | cut -f1)"
  test "$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}')" = origin/dev
  PROJECT_DIR="$tmp/imported2" sh "$root/sandbox/ship.sh" "Check repeat" > "$tmp/out"
  test "$(git config --get-all remote.origin.fetch | grep -Fc "+refs/heads/dev:refs/remotes/origin/dev")" = 1
)
mkdir "$tmp/local"
(
  cd "$tmp/local"
  PROJECT_DIR="$tmp/local" sh "$root/sandbox/ship.sh" 'Test no remote' > "$tmp/out2"
  grep -q 'pushed: no (this repository has no Git remotes' "$tmp/out2"
)
echo 'ship regression tests passed'
