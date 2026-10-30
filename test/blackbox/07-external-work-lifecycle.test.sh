#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"
source "$REPO_ROOT/test/lib/work.sh"

app_init
export WORK_IDLE_AFTER=2s WORK_ABANDON_AFTER=6s WORK_SWEEP_EVERY=1s WORK_SWEEP_MODE=dry-run
app_start
jar="$APP_TMP_DIR/alice.cookies"
touch "$jar"
http_register "$jar" Alice alice@example.test 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register first user'
resource="$APP_BASE/api/coding"
oauth_code_token "$jar" 'openid email coding:read coding:write' "$resource"
assert_status "$HTTP_STATUS" 200 'issue MCP token'
WORK_TOKEN=$(jq -r '.access_token' "$HTTP_BODY_FILE")

work_begin reported
reported_id=$TASK_ID reported_work=$WORK_ID
work_log "$reported_work"
assert_status "$HTTP_STATUS" 200 'log result without completing work'
work_task "$jar" "$reported_id"
assert_json '.work.phase == "reported" and .work.reportedAt != null and .status != "done"' \
  'reported work stays open and not done'
work_progress "$reported_work"
assert_status "$HTTP_STATUS" 200 'later progress is accepted'
work_task "$jar" "$reported_id"
assert_json '.work.reportedAt == null' 'progress clears the report marker'
work_log "$reported_work"

http_request POST "$jar" "/api/tasks/$reported_id/confirm" '{}'
assert_status "$HTTP_STATUS" 200 'person confirms reported work'
work_task "$jar" "$reported_id"
assert_json '.status == "done" and .proof.proven == false' 'confirmation is done without proof'
http_request GET "$jar" "/api/tasks/$reported_id/events"
assert_contains "$HTTP_BODY_FILE" 'Marked complete by you after the assistant reported its result' \
  'confirmation records its receipt'
work_confirm "$jar" "$reported_id"
assert_status "$HTTP_STATUS" 200 'repeat confirmation succeeds'
work_complete "$reported_work" 'A different summary after person confirmation'
assert_status "$HTTP_STATUS" 200 'complete work after person confirmation succeeds'
assert_json '.result.structuredContent.already == true' 'complete work after confirmation is already complete'

work_begin double-confirm
double_id=$TASK_ID double_work=$WORK_ID
work_log "$double_work"
first_result="$APP_TMP_DIR/confirm-first.json"
second_result="$APP_TMP_DIR/confirm-second.json"
curl -sS -o "$first_result" -w '%{http_code}' -b "$jar" -X POST \
  "$APP_BASE/api/tasks/$double_id/confirm" -H 'Content-Type: application/json' -d '{}' \
  >"$APP_TMP_DIR/confirm-first.status" &
first_pid=$!
curl -sS -o "$second_result" -w '%{http_code}' -b "$jar" -X POST \
  "$APP_BASE/api/tasks/$double_id/confirm" -H 'Content-Type: application/json' -d '{}' \
  >"$APP_TMP_DIR/confirm-second.status" &
second_pid=$!
wait "$first_pid"
wait "$second_pid"
assert_status "$(cat "$APP_TMP_DIR/confirm-first.status")" 200 'first concurrent confirmation succeeds'
second_status=$(cat "$APP_TMP_DIR/confirm-second.status")
if [[ $second_status == 200 ]]; then
  assert_status "$second_status" 200 'second concurrent confirmation is already complete'
else
  assert_status "$second_status" 409 'stale concurrent confirmation conflicts'
fi
work_task "$jar" "$double_id"
assert_json '.status == "done" and .proof.proven == false' 'concurrent confirmations remain unproven'

work_begin running
running_id=$TASK_ID
work_confirm "$jar" "$running_id"
assert_status "$HTTP_STATUS" 409 'running work cannot be confirmed'
http_request POST "$jar" "/api/tasks/$running_id/cancel" '{}'
assert_status "$HTTP_STATUS" 204 'running work can be stopped'
work_confirm "$jar" "$running_id"
assert_status "$HTTP_STATUS" 409 'stopped work cannot be confirmed'

work_begin repeat
repeat_work=$WORK_ID
work_complete "$repeat_work" 'First summary'
assert_status "$HTTP_STATUS" 200 'complete work once'
work_complete "$repeat_work" 'Different summary'
assert_status "$HTTP_STATUS" 200 'repeat completion with a different summary'
assert_json '.result.structuredContent.already == true' 'repeat completion reports already'
work_log "$repeat_work"
assert_json '.result.isError == true' 'logging after completion returns guidance'
assert_contains "$HTTP_BODY_FILE" 'this work session has ended' 'closed work guidance is clear'

work_begin approval
approval_id=$TASK_ID approval_work=$WORK_ID
work_update_plan "$approval_work"
assert_status "$HTTP_STATUS" 200 'save plan before asking for approval'
approval_revision=$(jq -r '.result.structuredContent.revision' "$HTTP_BODY_FILE")
work_submit_plan "$approval_work" 'Needs your approval' "$approval_revision"
assert_status "$HTTP_STATUS" 200 'submit a plan for person approval'
if [[ $HTTP_STATUS == 200 ]]; then
  work_complete "$approval_work" 'premature completion'
  assert_json '.result.isError == true' 'completion while approval is pending is rejected'
  assert_contains "$HTTP_BODY_FILE" 'awaits the person' 'approval rejection gives clear guidance'
fi
work_task "$jar" "$approval_id"
assert_json '.work.state == "open" and .work.planState == "ready_for_approval"' \
  'rejected completion keeps approval work open'

work_begin abandoned-report
reported_sweep_id=$TASK_ID reported_sweep_work=$WORK_ID
work_log "$reported_sweep_work"
assert_status "$HTTP_STATUS" 200 'report work for sweep'

work_begin activity-at-sweep-boundary
boundary_id=$TASK_ID boundary_work=$WORK_ID
sleep 5
work_progress "$boundary_work"
assert_status "$HTTP_STATUS" 200 'progress just before the abandon window is accepted'
sleep 2
work_task "$jar" "$boundary_id"
assert_json '.work.state == "open" and .status != "failed"' \
  'recent activity keeps work open after the earlier sweep cutoff'
assert_not_contains "$APP_LOG" "would abandon $boundary_work" \
  'recent activity keeps work out of the stale sweep candidates'

work_begin other-owner
owner_task_id=$TASK_ID
other_jar="$APP_TMP_DIR/other.cookies"
touch "$other_jar"
http_register "$other_jar" Bob bob@example.test 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register second user'
work_confirm "$other_jar" "$owner_task_id"
if [[ $HTTP_STATUS == 403 || $HTTP_STATUS == 404 ]]; then
  ((ASSERTIONS_PASSED += 1))
else
  ((ASSERTIONS_FAILED += 1))
fi

sleep 7
assert_contains "$APP_LOG" "would auto-complete $reported_sweep_work reported" 'dry-run logs reported work'
assert_contains "$APP_LOG" "would auto-complete $WORK_ID unreported" 'dry-run logs unreported work'
work_task "$jar" "$owner_task_id"
assert_json '.work.state == "open" and .status != "failed"' 'dry-run leaves unreported work open'
work_task "$jar" "$reported_sweep_id"
assert_json '.work.state == "open" and .status != "review"' 'dry-run leaves reported work open'
work_task "$jar" "$reported_id"
assert_json '.status == "done"' 'dry-run does not change confirmed work'
work_task "$jar" "$approval_id"
assert_json '.work.state == "open" and .status != "done"' 'dry-run leaves waiting work unchanged'
app_restart
work_task "$jar" "$owner_task_id"
assert_json '.work.state == "open" and .status != "failed"' 'restart does not fail open work'
work_task "$jar" "$approval_id"
assert_json '.work.state == "open" and .status != "failed"' 'restart keeps open work open'
app_stop
export WORK_SWEEP_MODE=on
app_start
sleep 1
work_task "$jar" "$owner_task_id"
assert_json '.status == "done" and .work.state == "completed" and .work.completionReason == "auto_completed_unverified" and .proof.proven == false' 'unreported inactive work becomes done without proof'
http_request GET "$jar" "/api/tasks/$owner_task_id/events"
assert_contains "$HTTP_BODY_FILE" 'Auto-completed (inactive)' 'inactive work records an unverified completion'
work_task "$jar" "$reported_sweep_id"
assert_json '.status == "done" and .work.state == "completed" and .work.completionReason == "auto_completed_unverified" and .proof.proven == false' 'reported inactive work is done without proof'
work_confirm "$jar" "$reported_sweep_id"
assert_status "$HTTP_STATUS" 409 'auto-completed work cannot be confirmed again'
work_task "$jar" "$approval_id"
assert_json '.work.state == "open" and .status != "failed"' 'work waiting for approval stays open'

assertion_summary
