#!/bin/sh
set -eu

limit="${MAX_FILE_LINES:-200}"

case "$limit" in
  ''|*[!0-9]*)
    echo "MAX_FILE_LINES must be a positive integer" >&2
    exit 1
    ;;
esac

file_violations="$(
  find . -type f \
    \( -name '*.go' -o -name '*.ts' -o -name '*.svelte' -o -name '*.css' \
       -o -name '*.js' -o -name '*.mjs' -o -name '*.cjs' -o -name '*.json' \
       -o -name '*.yaml' -o -name '*.yml' -o -name '*.toml' -o -name '*.md' \
       -o -name '*.sh' -o -name '*.html' -o -name 'Makefile' \) \
    ! -path './.git/*' \
    ! -path './web/node_modules/*' \
    ! -path './internal/platform/webassets/dist/*' \
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
