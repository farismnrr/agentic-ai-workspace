#!/usr/bin/env bash

# MCP work helpers shared by black-box and browser lifecycle scenarios.
work_tool() {
  mcp_call_tool "$WORK_TOKEN" "$1" "$2"
}

work_begin() {
  local summary=$1
  work_tool begin_work "$(jq -cn --arg summary "$summary" \
    '{summary:$summary,role:"developer",new_task:true}')"
  assert_status "$HTTP_STATUS" 200 "begin work: $summary"
  WORK_RESULT=$(jq -c '.result.structuredContent // (.result.content[0].text | fromjson)' "$HTTP_BODY_FILE")
  WORK_ID=$(jq -r '.work_id' <<<"$WORK_RESULT")
  TASK_ID=$(jq -r '.task_id' <<<"$WORK_RESULT")
  export WORK_ID TASK_ID
}

work_log() {
  local work_id=$1 text=${2:-"The requested work is finished."}
  work_tool log_message "$(jq -cn --arg id "$work_id" --arg text "$text" \
    '{work_id:$id,role:"assistant",text:$text}')"
}

work_progress() {
  work_tool report_progress "$(jq -cn --arg id "$1" \
    '{work_id:$id,message:"Still working on the request."}')"
}

work_complete() {
  work_tool complete_work "$(jq -cn --arg id "$1" --arg summary "$2" \
    '{work_id:$id,outcome:"completed",summary:$summary}')"
}

work_submit_plan() {
  work_tool submit_plan "$(jq -cn --arg id "$1" --arg summary "$2" --argjson revision "$3" \
    '{work_id:$id,summary:$summary,base_revision:$revision,small_task:true}')"
}

work_update_plan() {
  work_tool update_plan "$(jq -cn --arg id "$1" \
    '{work_id:$id,request_id:"lifecycle-test-plan",base_revision:0,
      steps:[{title:"Finish the request",state:"active"}]}')"
}

work_task() {
  http_request GET "$1" "/api/tasks/$2"
}

work_confirm() {
  http_request POST "$1" "/api/tasks/$2/confirm" '{}'
}
