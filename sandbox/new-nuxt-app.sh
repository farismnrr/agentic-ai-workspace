#!/bin/sh
# Starts a new app in the current folder from the toolbox's Nuxt starter: Nuxt 4, Nuxt UI, SQLite on Bun, TypeScript,
# login with Better Auth, a landing page, register and login pages and a dashboard shell. It never overwrites a file
# that exists, and installs the latest stable packages from the registry. Nothing is fetched from GitHub.
set -eu

src=/usr/local/share/starter/nuxt

(cd "$src" && find . -type f) | while read -r file; do
  if [ ! -e "$file" ]; then
    mkdir -p "$(dirname "$file")"
    cp "$src/$file" "$file"
  fi
done

if [ ! -e package.json ]; then
  name="$(basename "$PWD" | tr 'A-Z ' 'a-z-' | tr -cd 'a-z0-9-')"
  printf '{\n  "name": "%s",\n  "type": "module",\n  "private": true,\n  "scripts": {\n    "dev": "nuxt dev",\n    "build": "nuxt build",\n    "typecheck": "nuxt typecheck",\n    "lint": "eslint .",\n    "test": "bun test"\n  }\n}\n' "${name:-app}" > package.json
fi

bun add nuxt vue vue-router @nuxt/ui tailwindcss @iconify-json/lucide nuxt-security better-auth zod chart.js vue-chartjs \
  @fontsource-variable/public-sans
# TypeScript 7 has no lib/tsc, so vue-tsc cannot run on it: use 6, the newest that works.
bun add -d typescript@6 vue-tsc @types/bun @nuxt/eslint eslint eslint-plugin-security

if [ ! -e .env ]; then
  printf 'NUXT_AUTH_SECRET=%s\n' "$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')" > .env
fi

bun --bun run nuxt prepare
echo "The Nuxt starter is ready. Next: replace the words in app/utils/brand.ts and the pages with the PRD's."
