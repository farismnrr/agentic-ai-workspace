#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"

app_init
app_start
jar="$APP_TMP_DIR/user.cookies"
touch "$jar"
http_register "$jar" 'Alice' 'alice@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'account registration'

http_project_create "$jar" 'Persistent project'
assert_status "$HTTP_STATUS" 201 'project creation'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")
http_request PUT "$jar" /api/preferences \
  '{"personaName":"Alice","about":"Long-lived setting","rules":[]}'
assert_status "$HTTP_STATUS" 200 'preference setting is saved'

http_request POST "$jar" /api/providers \
  "$(jq -cn --arg base "$APP_BASE/v1" \
    '{kind:"compatible",name:"Saved gateway",baseUrl:$base,apiKey:"sk-persist-1234"}')"
assert_status "$HTTP_STATUS" 201 'credential setting is saved'

http_request POST "$jar" /api/tasks \
  "$(jq -cn --arg project "$project_id" \
    '{projectId:$project,prompt:"@deep-review\nKeep this task mode",team:false,providerId:"plan:claude",model:"default"}')"
assert_status "$HTTP_STATUS" 201 'deep-review task is created'
task_id=$(jq -r '.id' "$HTTP_BODY_FILE")
assert_json '.mode == "deep-review"' 'task mode is returned'

app_restart
http_request GET "$jar" /api/auth/me
assert_status "$HTTP_STATUS" 200 'session remains valid after process restart'
http_request GET "$jar" /api/projects
# shellcheck disable=SC2016 # The single-quoted expression is jq syntax, not shell expansion.
assert_json 'any(.[]; .id == $id)' 'project survives restart' --arg id "$project_id"
http_request GET "$jar" /api/preferences
assert_json '.about == "Long-lived setting"' 'preference survives restart'
http_request GET "$jar" /api/providers
assert_json 'any(.[]; .name == "Saved gateway")' 'credential metadata survives restart'
assert_not_contains "$HTTP_BODY_FILE" 'sk-persist-1234' 'credential remains redacted after restart'
http_request GET "$jar" "/api/tasks/$task_id"
assert_status "$HTTP_STATUS" 200 'task survives restart'
assert_json '.mode == "deep-review"' 'task mode survives restart'

assertion_summary
