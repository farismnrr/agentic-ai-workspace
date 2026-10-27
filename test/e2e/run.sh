#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
flow=${1:-}
case "$flow" in
  projects|sites|thread) ;;
  *) printf 'Usage: %s {projects|sites|thread}\n' "$0" >&2; exit 2 ;;
esac

source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"

playwright_module="$REPO_ROOT/web/node_modules/playwright"
if [[ ! -d $playwright_module ]]; then
  printf 'Playwright is not installed. Run npm --prefix web ci first.\n' >&2
  exit 1
fi
shots=${E2E_SHOTS:-}

if [[ $flow == projects ]]; then
  app_init
  app_start
  cookie_jar="$APP_TMP_DIR/browser.cookies"
  touch "$cookie_jar"
  email="browser-${APP_PORT}@example.test"
  password='correct horse battery'

  http_register "$cookie_jar" 'Browser test' "$email" "$password"
  if [[ $HTTP_STATUS != 201 ]]; then
    printf 'Could not create browser test account: HTTP %s\n' "$HTTP_STATUS" >&2
    cat "$HTTP_BODY_FILE" >&2
    exit 1
  fi

  http_project_create "$cookie_jar" 'Browser test project'
  if [[ $HTTP_STATUS != 201 ]]; then
    printf 'Could not create browser test project: HTTP %s\n' "$HTTP_STATUS" >&2
    cat "$HTTP_BODY_FILE" >&2
    exit 1
  fi
  project_id=$(jq -r '.id' "$HTTP_BODY_FILE")
  info_file="$APP_TMP_DIR/info.json"
  jq -n --arg base "$APP_BASE" --arg path "/app/projects/$project_id" \
    --arg email "$email" --arg password "$password" \
    '{base:$base,path:$path,email:$email,password:$password}' >"$info_file"
  shots=${shots:-$APP_TMP_DIR}
  mkdir -p "$shots"
  spec=01-projects.spec.ts
else
  image=${SANDBOX_TEST_IMAGE:-ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest}
  dir=$(mktemp -d)
  host_pid=
  trap 'touch "$dir/done"; if [[ -n ${host_pid:-} ]]; then wait "$host_pid" || true; fi; rm -rf "$dir"' EXIT
  make -C "$REPO_ROOT" frontend >/dev/null
  if [[ $flow == sites ]]; then
    host_test=TestServeSiteForABrowser
    host_dir=SITES_BROWSER_DIR
    spec=02-sites.spec.ts
  else
    host_test=TestServeThreadForABrowser
    host_dir=THREAD_BROWSER_DIR
    spec=03-thread.spec.ts
  fi

  (cd "$REPO_ROOT" && env "$host_dir=$dir" SANDBOX_TEST_IMAGE="$image" \
    go test ./internal/bootstrap/ -run "$host_test" -count=1 -timeout 8m) \
    >"$dir/server.log" 2>&1 &
  host_pid=$!
  deadline=$((SECONDS + 120))
  while [[ ! -f "$dir/info.json" && $SECONDS -lt $deadline ]]; do
    if ! kill -0 "$host_pid" 2>/dev/null; then
      cat "$dir/server.log" >&2
      printf 'The %s fixture host exited before writing info.json.\n' "$flow" >&2
      exit 1
    fi
    sleep 1
  done
  if [[ ! -f "$dir/info.json" ]]; then
    cat "$dir/server.log" >&2
    printf 'The %s fixture host did not write info.json.\n' "$flow" >&2
    exit 1
  fi
  info_file="$dir/info.json"
  shots=${shots:-$dir}
  mkdir -p "$shots"
fi

PW="$playwright_module" node --experimental-strip-types \
  "$REPO_ROOT/test/e2e/$spec" "$info_file" "$shots"
