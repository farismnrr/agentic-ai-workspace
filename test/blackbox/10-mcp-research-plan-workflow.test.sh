#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"
source "$REPO_ROOT/test/lib/work.sh"

app_init
app_start
jar="$APP_TMP_DIR/alice.cookies"
touch "$jar"
http_register "$jar" Alice alice@example.test 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register user'
resource="$APP_BASE/api/coding"
oauth_code_token "$jar" 'openid email coding:read coding:write' "$resource"
assert_status "$HTTP_STATUS" 200 'issue MCP token'
WORK_TOKEN=$(jq -r '.access_token' "$HTTP_BODY_FILE")

work_begin "MCP Research and Plan Workflow"
test_work=$WORK_ID
test_task=$TASK_ID

# 1. Test save_research failure: invalid source format
work_tool save_research "$(jq -cn --arg id "$test_work" \
  '{work_id:$id,markdown:"# Findings\nFound interesting patterns.",sources:["not-a-url"]}')"
assert_json '.result.isError == true' 'save_research rejects non-url source'
assert_contains "$HTTP_BODY_FILE" 'sources must be valid http or https URLs' 'clear validation message for bad source'

# 2. Test save_research failure: empty markdown
work_tool save_research "$(jq -cn --arg id "$test_work" \
  '{work_id:$id,markdown:"   ",sources:["https://example.com/docs"]}')"
assert_json '.result.isError == true' 'save_research rejects empty markdown'
assert_contains "$HTTP_BODY_FILE" 'markdown is required' 'clear validation message for empty markdown'

# 3. Test save_research success with substantial markdown (>400 runes) and valid source
SUBSTANTIAL_RESEARCH="# Architecture and Implementation Research

We thoroughly investigated the system design, error propagation, and validation layers of the MCP server.
The investigation revealed that previous iterations collapsed distinct validation failures into a generic
bad request error message. This prevented external assistants such as ChatGPT from identifying exactly which
input parameters required correction.

By enhancing error messaging, preserving the causal domain error, and decorating struct fields with clear
JSON Schema tags, connected assistants can self-correct malformed requests immediately and safely.
Furthermore, the research confirms that all external URL sources must strictly adhere to http or https protocols."

work_tool save_research "$(jq -cn --arg id "$test_work" --arg md "$SUBSTANTIAL_RESEARCH" \
  '{work_id:$id,markdown:$md,sources:["https://example.com/rfc"]}')"
assert_status "$HTTP_STATUS" 200 'save_research succeeds with valid inputs'
assert_json '.result.structuredContent.saved == true' 'save_research confirms saved'

# 4. Test finish_research
work_tool finish_research "$(jq -cn --arg id "$test_work" \
  '{work_id:$id,summary:"Research completed successfully"}')"
assert_status "$HTTP_STATUS" 200 'finish_research succeeds'
assert_json '.result.structuredContent.ready == true' 'finish_research marks ready'

# 5. Test save_requirements
work_tool save_requirements "$(jq -cn --arg id "$test_work" \
  '{work_id:$id,markdown:"# Requirements Document\n1. Swift-by-default mode.\n2. Deep review capability.\n3. Explicit human-authorized release."}')"
assert_status "$HTTP_STATUS" 200 'save_requirements succeeds'
assert_json '.result.structuredContent.saved == true' 'requirements confirmed saved'

# 6. Test update_plan
work_tool update_plan "$(jq -cn --arg id "$test_work" \
  '{work_id:$id,base_revision:0,steps:[{title:"Implement schema annotations",state:"active"},{title:"Add regression tests",state:"pending"}]}')"
assert_status "$HTTP_STATUS" 200 'update_plan succeeds'
plan_rev=$(jq -r '.result.structuredContent.plan_revision // .result.structuredContent.revision' "$HTTP_BODY_FILE")

# 7. Test submit_plan (full task with research + requirements)
work_tool submit_plan "$(jq -cn --arg id "$test_work" --argjson rev "$plan_rev" \
  '{work_id:$id,summary:"Comprehensive implementation plan",base_revision:$rev,small_task:false}')"
assert_status "$HTTP_STATUS" 200 'submit_plan succeeds'
assert_json '.result.structuredContent.plan_state == "ready_for_approval"' 'plan submitted for approval'

# 8. Test reading state and verifying notes were exported
work_tool get_work_state "$(jq -cn --arg id "$test_work" '{work_id:$id}')"
assert_status "$HTTP_STATUS" 200 'get_work_state succeeds'
assert_contains "$HTTP_BODY_FILE" 'Architecture and Implementation Research' 'get_work_state contains saved research preview'
assert_contains "$HTTP_BODY_FILE" 'Requirements Document' 'get_work_state contains saved requirements preview'

# 9. Verify task note files via app API
work_task "$jar" "$test_task"
assert_json '.work.planState == "ready_for_approval"' 'task reflects submitted plan state'

app_stop
assertion_summary
