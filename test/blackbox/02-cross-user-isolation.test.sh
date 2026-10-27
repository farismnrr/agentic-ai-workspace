#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"

app_init
app_start
jar_a="$APP_TMP_DIR/user-a.cookies"
jar_b="$APP_TMP_DIR/user-b.cookies"
touch "$jar_a" "$jar_b"

http_register "$jar_a" 'Alice' 'alice@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'owner account registration'
http_register "$jar_b" 'Bob' 'bob@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'second account registration'

http_project_create "$jar_a" 'Alice project'
assert_status "$HTTP_STATUS" 201 'owner project creation'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")
http_project_create "$jar_b" 'Bob project'
assert_status "$HTTP_STATUS" 201 'second project creation'

http_request GET "$jar_b" "/api/projects/$project_id"
assert_status "$HTTP_STATUS" 404 'other account cannot read project'
http_request PUT "$jar_b" "/api/projects/$project_id" '{"name":"stolen"}'
assert_status "$HTTP_STATUS" 404 'other account cannot update project'
http_request DELETE "$jar_b" "/api/projects/$project_id"
assert_status "$HTTP_STATUS" 404 'other account cannot delete project'
http_request GET "$jar_b" /api/account/export
assert_status "$HTTP_STATUS" 200 'second account can export its own data'
other_export="$APP_TMP_DIR/other-export.zip"
cp "$HTTP_BODY_FILE" "$other_export"
unzip -Z1 "$other_export" >"$APP_TMP_DIR/other-export-files.txt"
assert_not_contains "$APP_TMP_DIR/other-export-files.txt" "$project_id" \
  'other account export excludes owner project'

http_request GET "$jar_a" /api/projects
assert_status "$HTTP_STATUS" 200 'owner project list remains available'
# shellcheck disable=SC2016 # The single-quoted expressions are jq syntax, not shell expansion.
assert_json 'any(.[]; .id == $id)' 'owner still has the original project' --arg id "$project_id"
http_request GET "$jar_b" /api/projects
# shellcheck disable=SC2016 # jq receives $id through --arg; the shell must not expand it.
assert_json 'all(.[]; .id != $id)' 'other account list excludes owner project' --arg id "$project_id"

http_unauthenticated GET "/run/$project_id/"
assert_status "$HTTP_STATUS" 302 'unknown site address redirects away'
assert_contains "$HTTP_HEADERS_FILE" '/not-found' 'unknown site redirects to not-found'
HTTP_STATUS=$(curl -sS -H 'Sec-Fetch-Mode: navigate' -H 'Sec-Fetch-Dest: document' \
  -o "$HTTP_BODY_FILE" -w '%{http_code}' "$APP_BASE/run/$project_id/")
assert_status "$HTTP_STATUS" 302 'forged navigation headers do not expose unknown site content'

assertion_summary
