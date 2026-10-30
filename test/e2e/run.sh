#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
flow=${1:-}
case "$flow" in
  projects|sites|thread|lifecycle|git|context) ;;
  *) printf 'Usage: %s {projects|sites|thread|lifecycle|git|context}\n' "$0" >&2; exit 2 ;;
esac

source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/mcp.sh"
source "$REPO_ROOT/test/lib/work.sh"

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
elif [[ $flow == lifecycle ]]; then
  export WORK_SWEEP_MODE=off WORK_IDLE_AFTER=2s
  app_init
  app_start
  cookie_jar="$APP_TMP_DIR/browser.cookies"
  touch "$cookie_jar"
  email="lifecycle-${APP_PORT}@example.test"
  password='correct horse battery'
  http_register "$cookie_jar" 'Lifecycle test' "$email" "$password"
  assert_status "$HTTP_STATUS" 201 'register lifecycle browser user'
  oauth_code_token "$cookie_jar" 'openid email coding:read coding:write' "$APP_BASE/api/coding"
  assert_status "$HTTP_STATUS" 200 'issue lifecycle MCP token'
  WORK_TOKEN=$(jq -r '.access_token' "$HTTP_BODY_FILE")
  work_begin 'Reported lifecycle task'
  reported_id=$TASK_ID reported_work=$WORK_ID
  work_log "$reported_work" 'The reported lifecycle task is ready.'
  assert_status "$HTTP_STATUS" 200 'seed reported lifecycle task'
  work_begin 'Idle lifecycle task'
  idle_id=$TASK_ID
  sleep 2.2
  work_begin 'Running lifecycle task'
  running_id=$TASK_ID running_work=$WORK_ID
  work_begin 'Plan awaiting approval'
  approval_id=$TASK_ID approval_work=$WORK_ID
  work_update_plan "$approval_work"
  assert_status "$HTTP_STATUS" 200 'save browser approval plan'
  approval_revision=$(jq -r '.result.structuredContent.revision' "$HTTP_BODY_FILE")
  work_submit_plan "$approval_work" 'Needs your approval' "$approval_revision"
  assert_status "$HTTP_STATUS" 200 'submit browser approval plan'
  info_file="$APP_TMP_DIR/info.json"
  jq -n --arg base "$APP_BASE" --arg email "$email" --arg password "$password" \
    --arg running "$running_id" --arg runningWork "$running_work" \
    --arg reported "$reported_id" --arg reportedWork "$reported_work" --arg idle "$idle_id" \
    --arg approval "$approval_id" \
    --arg token "$WORK_TOKEN" \
    '{base:$base,email:$email,password:$password,running:$running,runningWork:$runningWork,
      reported:$reported,reportedWork:$reportedWork,idle:$idle,approval:$approval,token:$token}' >"$info_file"
  shots=${shots:-$APP_TMP_DIR}
  mkdir -p "$shots"
  spec=04-work-lifecycle.spec.ts
elif [[ $flow == context ]]; then
  make -C "$REPO_ROOT" frontend >/dev/null
  app_init
  app_start
  cookie_jar="$APP_TMP_DIR/browser.cookies"
  touch "$cookie_jar"
  email="context-${APP_PORT}@example.test"
  password='correct horse battery'
  http_register "$cookie_jar" 'Context browser test' "$email" "$password"
  assert_status "$HTTP_STATUS" 201 'register context user'
  http_project_create "$cookie_jar" 'Context meter project'
  assert_status "$HTTP_STATUS" 201 'create context project'
  project_id=$(jq -r '.id' "$HTTP_BODY_FILE")
  http_request POST "$cookie_jar" /api/tasks \
    "$(jq -cn --arg project "$project_id" '{projectId:$project,prompt:"Context meter fixture",providerId:"plan:claude",model:"default",team:false}')"
  assert_status "$HTTP_STATUS" 201 'create context task'
  task_id=$(jq -r '.id' "$HTTP_BODY_FILE")
  info_file="$APP_TMP_DIR/info.json"
  jq -n --arg base "$APP_BASE" --arg email "$email" --arg password "$password" \
    --arg project "$project_id" --arg task "$task_id" --arg db "$APP_DATA_DIR/workspace.db" \
    '{base:$base,email:$email,password:$password,project:$project,task:$task,db:$db}' >"$info_file"
  shots=${shots:-$APP_TMP_DIR}
  mkdir -p "$shots"
  spec=06-context-meter.spec.ts
elif [[ $flow == git ]]; then
  make -C "$REPO_ROOT" frontend >/dev/null
  app_init
  app_start
  cookie_jar="$APP_TMP_DIR/browser.cookies"
  touch "$cookie_jar"
  email="git-${APP_PORT}@example.test"
  password='correct horse battery'
  http_register "$cookie_jar" 'Git browser test' "$email" "$password"
  assert_status "$HTTP_STATUS" 201 'register Git browser user'
  http_project_create "$cookie_jar" 'Git decorations project'
  assert_status "$HTTP_STATUS" 201 'create Git browser project'
  project_id=$(jq -r '.id' "$HTTP_BODY_FILE")
  project_dir="$APP_DATA_DIR/projects/$project_id"
  mkdir -p "$project_dir/src"
  printf 'first line\nold line\nlast line\n' >"$project_dir/src/tracked.txt"
  git -C "$project_dir" init -q
  git -C "$project_dir" config user.name 'Browser Test'
  git -C "$project_dir" config user.email 'browser@example.test'
  git -C "$project_dir" add src/tracked.txt
  git -C "$project_dir" commit -qm 'initial file'
  http_request POST "$cookie_jar" /api/tasks \
    "$(jq -cn --arg project "$project_id" '{projectId:$project,prompt:"Git browser check",providerId:"plan:claude",model:"default",team:false}')"
  assert_status "$HTTP_STATUS" 201 'create Git browser task'
  task_id=$(jq -r '.id' "$HTTP_BODY_FILE")
  # Task startup initializes its workspace and can rewrite files; make Git edits afterwards.
  printf 'first line\nnew line\nlast line\n' >"$project_dir/src/tracked.txt"
  printf 'untracked\n' >"$project_dir/new.txt"
  git -C "$project_dir" status --short | grep -F 'M src/tracked.txt'
  info_file="$APP_TMP_DIR/info.json"
  jq -n --arg base "$APP_BASE" --arg email "$email" --arg password "$password" \
    --arg project "$project_id" --arg task "$task_id" \
    '{base:$base,email:$email,password:$password,project:$project,task:$task}' >"$info_file"
  shots=${shots:-$APP_TMP_DIR}
  mkdir -p "$shots"
  spec=05-git-decorations.spec.ts
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
    go test ./test/go/app/sites/ -run "$host_test" -count=1 -timeout 8m) \
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
