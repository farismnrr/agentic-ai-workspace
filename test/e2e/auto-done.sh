#!/usr/bin/env bash
set -euo pipefail
REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/mcp.sh"
source "$REPO_ROOT/test/lib/work.sh"
export WORK_SWEEP_MODE=on WORK_IDLE_AFTER=2s WORK_ABANDON_AFTER=4s WORK_SWEEP_EVERY=1s
app_init
app_start
jar="$APP_TMP_DIR/user.cookies"
touch "$jar"
http_register "$jar" 'Auto-done blackbox' "auto-${APP_PORT}@example.test" 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register'
oauth_code_token "$jar" 'openid email coding:read coding:write' "$APP_BASE/api/coding"
assert_status "$HTTP_STATUS" 200 'OAuth'
WORK_TOKEN=$(jq -r '.access_token' "$HTTP_BODY_FILE")
work_begin 'No progress timeout'
idle_task=$TASK_ID
work_begin 'Progress timeout'
progress_task=$TASK_ID progress_work=$WORK_ID
work_progress "$progress_work"
assert_status "$HTTP_STATUS" 200 'record progress'
work_begin 'Awaiting human approval'
approval_task=$TASK_ID approval_work=$WORK_ID
work_update_plan "$approval_work"
revision=$(jq -r '.result.structuredContent.revision' "$HTTP_BODY_FILE")
work_submit_plan "$approval_work" 'Human approval required' "$revision"
assert_status "$HTTP_STATUS" 200 'submit approval'
wait_done() {
  local task=$1 expected=$2 deadline=$((SECONDS+15))
  while (( SECONDS < deadline )); do
    work_task "$jar" "$task"
    if jq -e --arg reason auto_completed_unverified '.status == "done" and .work.completionReason == $reason and .proof.proven == false' "$HTTP_BODY_FILE" >/dev/null; then
      echo "PASS auto-done $expected: persisted reason and unverified proof"
      return 0
    fi
    sleep 0.5
  done
  echo "FAIL auto-done $expected" >&2
  cat "$HTTP_BODY_FILE" >&2
  return 1
}
wait_done "$idle_task" idle
wait_done "$progress_task" progress
work_task "$jar" "$approval_task"
if jq -e '.status != "done" and .work.planState == "ready_for_approval"' "$HTTP_BODY_FILE" >/dev/null; then
  echo 'PASS approval excluded from reconciler'
else
  echo 'FAIL approval auto-completed' >&2; cat "$HTTP_BODY_FILE" >&2; exit 1
fi

work_task "$jar" "$idle_task"
project=$(jq -r '.projectId' "$HTTP_BODY_FILE")
jq -n --arg base "$APP_BASE" --arg email "auto-${APP_PORT}@example.test" --arg password 'correct horse battery' --arg task "$idle_task" --arg project "$project" '{base:$base,email:$email,password:$password,task:$task,project:$project}' > "$APP_TMP_DIR/browser.json"
PW="$REPO_ROOT/web/node_modules/playwright" node --experimental-strip-types "$REPO_ROOT/test/e2e/auto-done-browser.spec.ts" "$APP_TMP_DIR/browser.json"
