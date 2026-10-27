#!/bin/sh
set -eu

limit="${MAX_FILE_LINES:-200}"

case "$limit" in
  ''|*[!0-9]*)
    echo "MAX_FILE_LINES must be a positive integer" >&2
    exit 1
    ;;
esac

if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  worktree_count="$(git worktree list --porcelain | awk '/^worktree / { count++ } END { print count + 0 }')"
  if [ "$worktree_count" -gt 1 ]; then
    echo "Git worktree guardrail failed: expected one worktree, found ${worktree_count}." >&2
    exit 1
  fi

  stage_directories="$(find . -maxdepth 1 -type d -name '.stage*' -print)"
  if [ -n "$stage_directories" ]; then
    echo "Git worktree guardrail failed: staging checkout directories are not allowed:" >&2
    echo "$stage_directories" >&2
    exit 1
  fi
fi

file_violations="$(
  find . -type f \
    \( -name '*.go' -o -name '*.ts' -o -name '*.svelte' -o -name '*.css' \
       -o -name '*.js' -o -name '*.mjs' -o -name '*.cjs' -o -name '*.json' \
       -o -name '*.yaml' -o -name '*.yml' -o -name '*.toml' -o -name '*.md' \
       -o -name '*.sh' -o -name '*.html' -o -name 'Makefile' -o -name 'Dockerfile' \) \
    ! -path './.git/*' \
    ! -path './web/node_modules/*' \
    ! -path './data/*' \
    ! -path './project-notes/*' \
    ! -path './internal/platform/webassets/dist/*' \
    ! -path './docs/*' \
    ! -path './agents/*' \
    ! -path './internal/modules/agentteam/domain/skills/*' \
    ! -path './README.md' \
    ! -path './AGENTS.md' \
    ! -name 'package-lock.json' \
    -exec sh -c '
      limit="$1"
      shift
      for file do
        lines=$(wc -l < "$file" | tr -d " ")
        if [ "$lines" -gt "$limit" ]; then
          printf "%s: %s lines\n" "$file" "$lines"
        fi
      done
    ' sh "$limit" {} +
)"

if [ -n "$file_violations" ]; then
  echo "Files exceed the ${limit}-line hard limit:" >&2
  echo "$file_violations" >&2
  exit 1
fi

# Docs for people stay scannable: no long single-line paragraphs or bullets. Tables and code blocks are exempt.
doc_limit="${MAX_DOC_LINE_WORDS:-60}"

doc_violations="$(
  for file in README.md docs/*.md; do
    [ -f "$file" ] || continue
    awk -v limit="$doc_limit" -v file="$file" '
      /^```/ { fence = !fence; next }
      fence || /^\|/ || /^#/ { next }
      NF > limit { printf "%s:%d: %d words\n", file, NR, NF }
    ' "$file"
  done
)"

if [ -n "$doc_violations" ]; then
  echo "Docs lines over ${doc_limit} words. Split them into bullets, or move the detail to agents/:" >&2
  echo "$doc_violations" >&2
  exit 1
fi

js_sources="$(
  find web -type f \
    \( -name '*.js' -o -name '*.jsx' -o -name '*.mjs' -o -name '*.cjs' \) \
    ! -path 'web/node_modules/*' -print
)"

if [ -n "$js_sources" ]; then
  echo "Web code and tooling must be TypeScript, not JavaScript:" >&2
  echo "$js_sources" >&2
  exit 1
fi

svelte_violations="$(
  find web/src -type f -name '*.svelte' -exec sh -c '
    for file do
      if grep -q "<script" "$file" && ! grep -q "<script lang=\"ts\">" "$file"; then
        echo "$file"
      fi
    done
  ' sh {} +
)"

if [ -n "$svelte_violations" ]; then
  echo 'Svelte script blocks must use <script lang="ts">:' >&2
  echo "$svelte_violations" >&2
  exit 1
fi

# Import direction: app -> features -> shared. Features never import other
# features, and shared never imports features or app. The router store in
# app/router is the one allowed inward import for features.
import_violations="$(
  find web/src/features web/src/shared -type f \( -name '*.ts' -o -name '*.svelte' \) | while read -r file; do
    dir=$(dirname "$file")
    own=$(echo "$file" | cut -d/ -f3-4)
    grep -oE "from ['\"]\.[^'\"]*['\"]" "$file" | sed "s/from ['\"]//; s/['\"]$//" | while read -r spec; do
      target=$(realpath -m "$dir/$spec" | sed "s|^$PWD/||")
      case "$target" in
        web/src/features/*)
          [ "$(echo "$target" | cut -d/ -f3-4)" = "$own" ] || echo "$file imports $spec" ;;
        web/src/app/router) ;;
        web/src/app/*) echo "$file imports $spec" ;;
      esac
    done
  done
)"
if [ -n "$import_violations" ]; then
  echo "Import direction violations (app -> features -> shared):" >&2
  echo "$import_violations" >&2
  exit 1
fi
