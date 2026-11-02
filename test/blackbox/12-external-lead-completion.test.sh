#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"

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

orchestrator_list_tools() {
	HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
	HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
	HTTP_STATUS=$(curl -sS -X POST "$APP_BASE/api/orchestrator" \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		-H 'MCP-Protocol-Version: 2025-11-25' \
		-H "Authorization: Bearer $token" \
		--data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' \
		-D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}')
}

app_init
app_start
jar="$APP_TMP_DIR/alice.cookies"
touch "$jar"
http_register "$jar" Alice alice@example.test 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register user'
http_project_create "$jar" 'External lead completion'
assert_status "$HTTP_STATUS" 201 'create project'
project=$(jq -r '.id' "$HTTP_BODY_FILE")

for tool in create_task complete_task stop_task save_task_requirements update_task_plan submit_plan; do
	http_request PUT "$jar" "/api/spaces/orchestrator/tools/$tool" '{"enabled":true}'
	assert_status "$HTTP_STATUS" 204 "enable Orchestrator $tool"
done
oauth_code_token "$jar" 'openid email orchestrator:read orchestrator:tasks' "$APP_BASE/api/orchestrator"
assert_status "$HTTP_STATUS" 200 'issue Orchestrator token'
token=$(jq -r '.access_token' "$HTTP_BODY_FILE")
orchestrator_list_tools
assert_status "$HTTP_STATUS" 200 'read Orchestrator tool catalog'
assert_json '[.result.tools[] | select(.name == "complete_task")][0].description | contains("Call once after all workers finish")' \
	'catalog says to finish once after workers complete'

orchestrator_tool "$token" create_task "$(jq -cn --arg project "$project" \
	'{project:$project,summary:"Explicit completion",idempotency_key:"completion-001"}')"
task=$(jq -r '.result.structuredContent.ID' "$HTTP_BODY_FILE")
orchestrator_tool "$token" complete_task "$(jq -cn --arg id "$task" '{task_id:$id,summary:"   "}')"
assert_json '.result.isError == true' 'empty final summary is rejected'
orchestrator_tool "$token" complete_task "$(jq -cn --arg id "$task" \
	'{task_id:$id,summary:"PARTIAL: scoped work is complete; verification is not claimed."}')"
assert_json '.result.structuredContent.status == "done"' 'explicit completion returns done'
http_request GET "$jar" "/api/tasks/$task"
assert_json '.status == "done" and .summary == "PARTIAL: scoped work is complete; verification is not claimed." and .proof == null' \
	'external report is saved as done without proof'
http_request GET "$jar" "/api/tasks/$task/events"
assert_json '[.[] | select(.technical == "External lead report; not independently verified.")] | length == 1' \
	'completion records one unverified report event'
event_count=$(jq '[.[] | select(.technical == "External lead report; not independently verified.")] | length' "$HTTP_BODY_FILE")
orchestrator_tool "$token" complete_task "$(jq -cn --arg id "$task" \
	'{task_id:$id,summary:"PARTIAL: scoped work is complete; verification is not claimed."}')"
assert_json '.result.structuredContent.status == "done"' 'same-summary retry remains successful'
http_request GET "$jar" "/api/tasks/$task/events"
assert_json '[.[] | select(.technical == "External lead report; not independently verified.")] | length == $count' \
	'retry does not append a duplicate event' --argjson count "$event_count"

orchestrator_tool "$token" create_task "$(jq -cn --arg project "$project" \
	'{project:$project,summary:"Owner approval gate",idempotency_key:"completion-pending-001"}')"
pending=$(jq -r '.result.structuredContent.ID' "$HTTP_BODY_FILE")
orchestrator_tool "$token" save_task_requirements "$(jq -cn --arg id "$pending" \
	'{task_id:$id,markdown:"# PRD\nWait for owner approval."}')"
orchestrator_tool "$token" update_task_plan "$(jq -cn --arg id "$pending" \
	'{task_id:$id,steps:[{id:"approval",title:"Wait for approval",state:"pending"}]}')"
orchestrator_tool "$token" submit_plan "$(jq -cn --arg id "$pending" '{task_id:$id}')"
assert_json '.result.structuredContent.status == "pending"' 'plan approval is pending'
orchestrator_tool "$token" complete_task "$(jq -cn --arg id "$pending" \
	'{task_id:$id,summary:"Must wait for the owner."}')"
assert_json '.result.isError == true' 'pending owner approval blocks completion'
http_request GET "$jar" "/api/tasks/$pending"
assert_json '.status == "needs-input"' 'blocked completion preserves needs-input'

orchestrator_tool "$token" create_task "$(jq -cn --arg project "$project" \
	'{project:$project,summary:"Manual stop",idempotency_key:"completion-stop-001"}')"
stopped=$(jq -r '.result.structuredContent.ID' "$HTTP_BODY_FILE")
orchestrator_tool "$token" stop_task "$(jq -cn --arg id "$stopped" '{task_id:$id}')"
assert_json '.result.structuredContent.status == "stopped"' 'manual stop has its distinct result'
orchestrator_tool "$token" stop_task "$(jq -cn --arg id "$stopped" '{task_id:$id}')"
assert_json '.result.structuredContent.status == "stopped"' 'manual stop retry remains successful'
http_request GET "$jar" "/api/tasks/$stopped"
assert_json '.status == "done"' 'manual stop remains terminal done'
http_request GET "$jar" "/api/tasks/$stopped/events"
assert_contains "$HTTP_BODY_FILE" 'Stopped by the external lead' 'manual stop keeps its distinct event'
assert_json '[.[] | select(.text == "Stopped by the external lead")] | length == 1' \
	'manual stop retry does not duplicate its event'

app_stop
assertion_summary
