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

secret='sk-blackbox-never-return-this-1234'
http_request POST "$jar" /api/providers \
  "$(jq -cn --arg key "$secret" --arg base "$APP_BASE/v1" \
    '{kind:"compatible",name:"Blackbox gateway",baseUrl:$base,apiKey:$key}')"
assert_status "$HTTP_STATUS" 201 'provider credential can be stored'
assert_not_contains "$HTTP_BODY_FILE" "$secret" 'provider create response redacts secret'
provider_id=$(jq -r '.id' "$HTTP_BODY_FILE")

http_request GET "$jar" /api/providers
assert_status "$HTTP_STATUS" 200 'provider settings can be read'
assert_not_contains "$HTTP_BODY_FILE" "$secret" 'provider list redacts secret'
assert_json 'any(.[]; .id == $id)' 'stored provider remains present' --arg id "$provider_id"

http_request GET "$jar" /api/account/export
assert_status "$HTTP_STATUS" 200 'account export succeeds'
archive="$APP_TMP_DIR/export.zip"
cp "$HTTP_BODY_FILE" "$archive"
unzip -p "$archive" account.json >"$APP_TMP_DIR/account.json"
assert_not_contains "$APP_TMP_DIR/account.json" "$secret" 'account export omits provider secret'

http_request POST "$jar" /api/providers \
  "$(jq -cn --arg key "$secret" '{kind:"unknown",name:"bad",apiKey:$key}')"
assert_not_contains "$HTTP_BODY_FILE" "$secret" 'validation error does not echo secret'
assert_not_contains "$APP_LOG" "$secret" 'application log does not include secret'

assertion_summary
