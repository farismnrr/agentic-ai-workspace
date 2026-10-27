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
assert_status "$HTTP_STATUS" 201 'first account registration'
assert_json '.email == "alice@example.test"' 'registration returns account profile'

http_register "$jar_b" 'Bob' 'bob@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'second account registration'

http_request GET '' /api/projects
assert_status "$HTTP_STATUS" 401 'protected API rejects anonymous access'

http_request POST "$jar_a" /api/auth/login \
  '{"email":"alice@example.test","password":"incorrect password"}'
assert_status "$HTTP_STATUS" 401 'bad password is rejected'

http_request GET "$jar_a" /api/auth/me
assert_status "$HTTP_STATUS" 200 'registered session is valid'
http_request POST "$jar_a" /api/auth/logout
assert_status "$HTTP_STATUS" 204 'sign out succeeds'
http_request GET "$jar_a" /api/auth/me
assert_status "$HTTP_STATUS" 401 'signed-out cookie no longer authenticates'

assertion_summary
