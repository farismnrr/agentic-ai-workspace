#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"

app_init
app_start
owner="$APP_TMP_DIR/owner.cookies"
touch "$owner"
http_register "$owner" 'Git parity owner' 'git-parity@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register owner'
http_project_create "$owner" 'Researcher Git parity'
assert_status "$HTTP_STATUS" 201 'create isolated Git project'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")

http_request PUT "$owner" /api/spaces/coding/tools/run_command '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'enable the shared Git tool switch'
oauth_code_token "$owner" 'openid email coding:read coding:exec' "$APP_BASE/api/coding"
assert_status "$HTTP_STATUS" 200 'issue execution-scoped Coding token'
WORK_TOKEN=$(jq -r '.access_token' "$HTTP_BODY_FILE")

mcp_call_tool "$WORK_TOKEN" list_projects '{}'
assert_status "$HTTP_STATUS" 200 'list the connected app projects'
project_slug=$(jq -r '.result.structuredContent.projects[] | select(.name == "Researcher Git parity") | .id' \
  "$HTTP_BODY_FILE")

mcp_call_tool "$WORK_TOKEN" begin_work "$(jq -cn --arg project "$project_slug" \
  '{project:$project,summary:"Verify bounded Git permissions",role:"researcher",new_task:true}')"
assert_status "$HTTP_STATUS" 200 'start one tracked Researcher work session'
WORK_ID=$(jq -r '.result.structuredContent.work_id' "$HTTP_BODY_FILE")

project_dir="$APP_DATA_DIR/projects/$project_id"
cd "$project_dir"
git init -q
git config user.name 'Blackbox Test'
git config user.email 'blackbox@example.test'
printf 'tracked fixture\n' > README.md
git add README.md
git commit -qm 'seed isolated Git fixture'
printf 'Researcher commit probe\n' >> README.md
cd "$REPO_ROOT"

git_tool() {
  local input
  input=$(jq -cn --arg project "$project_slug" --arg work_id "$WORK_ID" --args \
    '{project:$project,work_id:$work_id,args:$ARGS.positional}' -- "$@")
  mcp_call_tool "$WORK_TOKEN" git "$input"
}

git_tool status --short
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git status'
assert_json '.result.structuredContent.exitCode == 0' 'Researcher Git status succeeds'

git_tool add README.md
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git add'
assert_json '.result.structuredContent.exitCode == 0' 'Researcher Git add succeeds'

git_tool commit -m 'Researcher Git permission probe'
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git commit'
assert_json '.result.structuredContent.exitCode == 0' 'Researcher Git commit succeeds'

git_tool push origin HEAD
assert_status "$HTTP_STATUS" 200 'Researcher push reaches native Git policy'
assert_json '.result.structuredContent.output | contains("CONNECT_GITHUB_REQUIRED")' \
  'Researcher push preserves the connected-GitHub requirement without contacting a remote'

mcp_call_tool "$WORK_TOKEN" run_command \
  "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" \
    '{project:$project,work_id:$work_id,command:"pwd"}')"
assert_json '.result.isError == true' 'Researcher still cannot invoke general shell commands'

for role in developer devops; do
  mcp_call_tool "$WORK_TOKEN" set_role "$(jq -cn --arg project "$project_id" \
    --arg work_id "$WORK_ID" --arg role "$role" \
    '{project:$project,work_id:$work_id,role:$role}')"
  assert_status "$HTTP_STATUS" 200 "switch tracked work to $role"
  git_tool status --short
  assert_status "$HTTP_STATUS" 200 "$role can invoke native Git status"
  assert_json '.result.structuredContent.exitCode == 0' "$role Git status succeeds"
  git_tool push origin HEAD
  assert_status "$HTTP_STATUS" 200 "$role push reaches native Git policy"
  assert_json '.result.structuredContent.output | contains("CONNECT_GITHUB_REQUIRED")' \
    "$role push has the same connected-GitHub restriction"

  if [[ "$role" == "devops" ]]; then
    printf 'DevOps native Git write probe\n' > "$project_dir/devops-git-probe.txt"
    git_tool add devops-git-probe.txt
    assert_status "$HTTP_STATUS" 200 'DevOps native Git add is available'
    assert_json '.result.structuredContent.exitCode == 0' 'DevOps native Git add can write to the workspace'
    git_tool commit -m 'DevOps native Git write probe'
    assert_status "$HTTP_STATUS" 200 'DevOps native Git commit is available'
    assert_json '.result.structuredContent.exitCode == 0' 'DevOps native Git commit can write to the workspace'
  fi

  mcp_call_tool "$WORK_TOKEN" run_command \
    "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" \
      '{project:$project,work_id:$work_id,command:"pwd"}')"
  assert_json '.result.isError == true' "$role still cannot invoke general shell commands"
done
