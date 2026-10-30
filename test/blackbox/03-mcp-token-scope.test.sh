#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"

app_init
app_start
jar="$APP_TMP_DIR/user.cookies"
touch "$jar"
http_register "$jar" 'Alice' 'alice@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'account registration'
resource="$APP_BASE/api/coding"

oauth_code_token "$jar" 'openid' "$resource"
assert_status "$HTTP_STATUS" 200 'narrow OAuth token issued'
narrow_token=$(jq -r '.access_token' "$HTTP_BODY_FILE")
mcp_list_tools "$narrow_token"
assert_status "$HTTP_STATUS" 403 'missing MCP scope is rejected'
assert_contains "$HTTP_HEADERS_FILE" 'insufficient_scope' 'scope denial names insufficient scope'

oauth_code_token "$jar" 'openid email coding:read' "$resource"
assert_status "$HTTP_STATUS" 200 'read-scoped OAuth token issued'
token=$(jq -r '.access_token' "$HTTP_BODY_FILE")
mcp_list_tools "$token"
assert_status "$HTTP_STATUS" 200 'scoped token can list MCP tools'
assert_contains "$HTTP_BODY_FILE" 'list_files' 'MCP response contains the coding tools'

http_request PUT "$jar" /api/spaces/coding/tools/write_file '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'write tool is enabled for the scope check'
mcp_call_tool "$token" write_file '{"path":"scope-check.txt","content":"not written"}'
assert_status "$HTTP_STATUS" 200 'write tool request reaches MCP'
assert_json '.result.isError == true' 'tool rejects a missing per-tool scope'
assert_json '(.result._meta["mcp/www_authenticate"] // [] | join(" ")) | contains("insufficient_scope")' \
  'tool denial includes insufficient-scope challenge'

oauth_code_token "$jar" 'openid email coding:read coding:write' "$resource"
assert_status "$HTTP_STATUS" 200 'write-scoped OAuth token issued'
write_token=$(jq -r '.access_token' "$HTTP_BODY_FILE")
refresh_token=$(jq -r '.refresh_token' "$HTTP_BODY_FILE")
mcp_call_tool "$write_token" begin_work '{"role":"developer","summary":"Verify write-scoped MCP access"}'
assert_status "$HTTP_STATUS" 200 'write-scoped client can start tracked work'
assert_json '(.result.structuredContent.work_id // "") | length > 0' 'tracked work has an id'
work_id=$(jq -r '.result.structuredContent.work_id' "$HTTP_BODY_FILE")
mcp_call_tool "$write_token" write_file "$(jq -cn --arg work_id "$work_id" \
  '{work_id:$work_id,path:"scope-check.txt",content:"written"}')"
assert_status "$HTTP_STATUS" 200 'write-scoped tool request reaches MCP'
assert_json 'has("result") and (.result.isError // false) == false' 'tool call succeeds with its scope'

http_form /oauth/token --data-urlencode 'grant_type=refresh_token' \
  --data-urlencode "refresh_token=$refresh_token" --data-urlencode "client_id=$OAUTH_CLIENT_ID" \
  --data-urlencode "resource=$resource"
assert_status "$HTTP_STATUS" 200 'refresh token rotates successfully'
rotated_access=$(jq -r '.access_token' "$HTTP_BODY_FILE")
rotated_refresh=$(jq -r '.refresh_token' "$HTTP_BODY_FILE")
mcp_list_tools "$rotated_access"
assert_status "$HTTP_STATUS" 200 'rotated access token works before reuse'

http_form /oauth/token --data-urlencode 'grant_type=refresh_token' \
  --data-urlencode "refresh_token=$refresh_token" --data-urlencode "client_id=$OAUTH_CLIENT_ID" \
  --data-urlencode "resource=$resource"
assert_status "$HTTP_STATUS" 400 'reusing a refresh token is refused'
mcp_list_tools "$rotated_access"
assert_status "$HTTP_STATUS" 401 'refresh-token reuse revokes the rotated access token'

http_form /oauth/token --data-urlencode 'grant_type=refresh_token' \
  --data-urlencode "refresh_token=$rotated_refresh" --data-urlencode "client_id=$OAUTH_CLIENT_ID" \
  --data-urlencode "resource=$resource"
assert_status "$HTTP_STATUS" 400 'refresh-token reuse revokes the rotated refresh token'

http_form /oauth/token \
  --data-urlencode 'grant_type=authorization_code' \
  --data-urlencode "code=$OAUTH_CODE" --data-urlencode "client_id=$OAUTH_CLIENT_ID" \
  --data-urlencode 'redirect_uri=https://chatgpt.com/connector_platform_oauth_redirect' \
  --data-urlencode 'code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk' \
  --data-urlencode "resource=$resource"
assert_status "$HTTP_STATUS" 400 'authorization code cannot be reused'

oauth_code_token "$jar" 'coding:read' "$resource" "$APP_BASE/api/other"
assert_status "$HTTP_STATUS" 400 'authorization code cannot be exchanged for another resource'

oauth_code_token "$jar" 'coding:read' "$resource" "$resource" 'wrong-pkce-verifier'
assert_status "$HTTP_STATUS" 400 'authorization code rejects a different PKCE verifier'

grants_file="$APP_TMP_DIR/grants.json"
http_request GET "$jar" /api/oauth/grants
assert_status "$HTTP_STATUS" 200 'OAuth grants can be listed'
cp "$HTTP_BODY_FILE" "$grants_file"
while IFS= read -r grant_id; do
  http_request DELETE "$jar" "/api/oauth/grants/$grant_id"
  assert_status "$HTTP_STATUS" 204 'OAuth grant can be revoked'
done < <(jq -r '.[].id' "$grants_file")
mcp_list_tools "$token"
assert_status "$HTTP_STATUS" 401 'revoked token is rejected immediately'
mcp_call_tool "$token" list_files '{}'
assert_status "$HTTP_STATUS" 401 'revoked token cannot call read tools'

assertion_summary
