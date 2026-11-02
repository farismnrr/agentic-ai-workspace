#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"

orchestrator_list_tools() {
	HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
	HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
	HTTP_STATUS=$(curl -sS -X POST "$APP_BASE/api/orchestrator" \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		-H 'MCP-Protocol-Version: 2025-11-25' \
		-D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}' \
		--data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}')
}

orchestrator_tool() {
	local token=$1 name=$2 arguments=$3
	HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
	HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
	HTTP_STATUS=$(curl -sS -X POST "$APP_BASE/api/orchestrator" \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		-H 'MCP-Protocol-Version: 2025-11-25' \
		-H "Authorization: Bearer $token" \
		--data "$(jq -cn --arg name "$name" --argjson arguments "$arguments" \
			'{jsonrpc:"2.0",id:1,method:"tools/call",params:{name:$name,arguments:$arguments}}')" \
		-D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}')
}

app_init
app_start
jar="$APP_TMP_DIR/alice.cookies"
touch "$jar"
http_register "$jar" Alice alice@example.test 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register account'

orchestrator_list_tools
assert_status "$HTTP_STATUS" 401 'unauthenticated Orchestrator request is rejected'
for scope in orchestrator:read orchestrator:tasks orchestrator:delegate orchestrator:files; do
	assert_contains "$HTTP_HEADERS_FILE" "$scope" "initial challenge recommends $scope"
done
assert_not_contains "$HTTP_HEADERS_FILE" 'orchestrator:approve' 'initial challenge excludes plan approval'
assert_contains "$HTTP_HEADERS_FILE" 'resource_metadata=' 'initial challenge retains resource metadata'

http_unauthenticated POST /api/coding
assert_status "$HTTP_STATUS" 401 'unauthenticated Coding request is rejected'
assert_contains "$HTTP_HEADERS_FILE" 'coding:read' 'Coding challenge keeps its original scope'
assert_not_contains "$HTTP_HEADERS_FILE" 'orchestrator:delegate' 'Coding challenge is unchanged'

http_request GET "$jar" /api/spaces
assert_status "$HTTP_STATUS" 200 'space settings are readable'
assert_json '.[] | select(.slug == "orchestrator") | .skipApprovals == false' \
	'Orchestrator approvals remain required by default'
assert_json '[.[] | select(.slug == "orchestrator") | .tools[] | select(.name == "approve_plan")][0].enabled == false' \
	'plan approval tool remains off by default'

http_request PUT "$jar" /api/spaces/orchestrator/tools/delegate_agent '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'enable delegate tool for scope check'
resource="$APP_BASE/api/orchestrator"
oauth_code_token "$jar" 'openid email orchestrator:read' "$resource"
assert_status "$HTTP_STATUS" 200 'issue read-only Orchestrator token'
token=$(jq -r '.access_token' "$HTTP_BODY_FILE")

orchestrator_tool "$token" list_projects '{}'
assert_status "$HTTP_STATUS" 200 'read-only token can invoke read tools'
assert_json 'has("result") and (.result.isError // false) == false' 'read tool succeeds'
orchestrator_tool "$token" delegate_agent \
	'{"task_id":"not-run","agent_key":"researcher","brief":"Must be refused before execution","idempotency_key":"scope-check"}'
assert_status "$HTTP_STATUS" 200 'delegate request reaches MCP scope check'
assert_json '.result.isError == true' 'read-only token cannot delegate'
assert_json '(.result._meta["mcp/www_authenticate"] // [] | join(" ")) | contains("orchestrator:delegate")' \
	'delegate denial names its required scope'

app_stop
assertion_summary
