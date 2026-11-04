#!/usr/bin/env bash
set -euo pipefail

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
http_request PUT "$owner" /api/spaces/coding/tools/run_task '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'enable QA verification tool'
http_request PUT "$owner" /api/spaces/coding/tools/run_build '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'enable named DevOps build tool'
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
git remote add local-fixture remote.git
git init --bare -q "$project_dir/remote.git"
git push -q local-fixture HEAD:refs/heads/main
printf 'Researcher commit probe\n' >> README.md
# A minimal source check and a build writing only into the isolated /output
# mount. Both must survive an oversized pre-existing (ignored) project file.
mkdir -p .workspace
cat > go.mod <<'GOMOD'
module example.test/oversized

go 1.20
GOMOD
cat > parity_test.go <<'GOTEST'
package oversized
import "testing"
func TestSmoke(t *testing.T) {}
GOTEST
cat > .workspace/tasks.json <<'TASKS'
{"tasks":{"smoke-build":{"description":"Bounded isolated build probe","kind":"build","command":["sh","-c","printf safe-output > $BUILD_OUT/probe.txt"]}}}
TASKS
cd "$REPO_ROOT"

git_tool() {
  local input
  input=$(jq -cn --arg project "$project_slug" --arg work_id "$WORK_ID" --args \
    '{project:$project,work_id:$work_id,args:$ARGS.positional}' -- "$@")
  mcp_call_tool "$WORK_TOKEN" git "$input"
}

# Regression: a repository may be larger than the normal command quota
# because of generated/ignored assets. Simulate >2 GiB with a sparse file
# without consuming 2 GiB of physical storage.
oversized_fixture="$project_dir/.oversized-git-fixture"
printf '%s\n' '.oversized-git-fixture' >> "$project_dir/.git/info/exclude"
truncate -s $(( (2 << 30) + 4096 )) "$oversized_fixture"

git_tool status --short --branch
assert_status "$HTTP_STATUS" 200 'oversized project still reaches server Git status'
assert_json '.result.structuredContent.exit_code == 0' 'quota-safe Git status succeeds over 2 GiB'

# These were previously rejected because the quota exception covered only a
# handful of Git verbs, despite all checks reading the existing checkout.
git_tool diff --check
assert_status "$HTTP_STATUS" 200 'oversized project reaches safe Git diff'
assert_json '.result.structuredContent.exit_code == 0' 'quota-safe Git diff --check succeeds'

git_tool branch -vv
assert_status "$HTTP_STATUS" 200 'oversized project reaches Git branch inspection'
assert_json '.result.structuredContent.exit_code == 0' 'quota-safe Git branch -vv succeeds'

git_tool log -1 --oneline
assert_status "$HTTP_STATUS" 200 'oversized project reaches Git log inspection'
assert_json '.result.structuredContent.exit_code == 0' 'quota-safe Git log -1 succeeds'

git_tool ls-files
assert_status "$HTTP_STATUS" 200 'oversized project reaches Git tracked file inspection'
assert_json '.result.structuredContent.exit_code == 0' 'quota-safe Git ls-files succeeds'

git_tool push origin HEAD
assert_status "$HTTP_STATUS" 200 'oversized project still reaches Git push authorization gate'
assert_json '.result.structuredContent | .exit_code == -1 and (.output | contains("CONNECT_GITHUB_REQUIRED")) and (.output | contains("size limit") | not)' \
  'quota-safe Git push reaches GitHub connection policy before quota refusal'

# QA verification uses a read-only OS source mount. The project can be larger
# than the execution quota before the check begins.
mcp_call_tool "$WORK_TOKEN" set_role "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" --arg role qa '{project:$project,work_id:$work_id,role:$role}')"
assert_status "$HTTP_STATUS" 200 'switch to QA for oversized immutable check'
mcp_call_tool "$WORK_TOKEN" verify "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" '{project:$project,work_id:$work_id,args:["go","test","./..."]}')"
assert_status "$HTTP_STATUS" 200 'run QA verification on oversized source'
assert_json '.result.structuredContent | .exit_code == 0 and (.output | contains("workspace is over its size limit") | not)' \
  'QA Go test succeeds despite old oversized read-only source'

# DevOps named build writes only into a fresh /output mount. Existing ignored
# project data must not consume this new output's own size cap.
mcp_call_tool "$WORK_TOKEN" set_role "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" --arg role devops '{project:$project,work_id:$work_id,role:$role}')"
assert_status "$HTTP_STATUS" 200 'switch to DevOps for isolated output build'
mcp_call_tool "$WORK_TOKEN" run_build "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" '{project:$project,work_id:$work_id,name:"smoke-build"}')"
assert_status "$HTTP_STATUS" 200 'run DevOps build on oversized source'
assert_json '.result.structuredContent | .exit_code == 0 and (.output | contains("Generated outputs:"))' \
  'isolated build succeeds with output cap independent of old project size'

mcp_call_tool "$WORK_TOKEN" set_role "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" --arg role researcher '{project:$project,work_id:$work_id,role:$role}')"
assert_status "$HTTP_STATUS" 200 'return to Researcher for quota mutation denial'

git_tool add README.md
assert_status "$HTTP_STATUS" 200 'oversized project still rejects Git index mutation'
assert_json '.result.structuredContent | .exit_code == -1 and (.output | contains("workspace is over its size limit"))' \
  'workspace-expanding Git operations remain quota-limited'

rm -- "$oversized_fixture"

git_tool status --short
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git status'
assert_json '.result.structuredContent.exit_code == 0' 'Researcher Git status succeeds'

# The disposable fixture is not connected to the owner's GitHub account.
# This policy gate is expected and does not verify a successful fetch write.
git_tool fetch local-fixture main
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git fetch'
assert_json '.result.structuredContent | .exit_code == -1 and (.output | contains("CONNECT_GITHUB_REQUIRED")) and (.output | contains("Read-only file system") | not)' \
  'Researcher fetch is stopped by the GitHub connection policy, not a filesystem error'

git_tool add README.md
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git add'
assert_json '.result.structuredContent.exit_code == 0' 'Researcher Git add succeeds'

git_tool commit -m 'Researcher Git permission probe'
assert_status "$HTTP_STATUS" 200 'Researcher can invoke native Git commit'
assert_json '.result.structuredContent.exit_code == 0' 'Researcher Git commit succeeds'

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
  assert_json '.result.structuredContent.exit_code == 0' "$role Git status succeeds"
  git_tool fetch local-fixture main
  assert_status "$HTTP_STATUS" 200 "$role can invoke native Git fetch"
  assert_json '.result.structuredContent | .exit_code == -1 and (.output | contains("CONNECT_GITHUB_REQUIRED")) and (.output | contains("Read-only file system") | not)' \
    "$role fetch is stopped by the GitHub connection policy, not a filesystem error"

  printf '%s native Git write probe\n' "$role" > "$project_dir/$role-git-probe.txt"
  git_tool add "$role-git-probe.txt"
  assert_status "$HTTP_STATUS" 200 "$role native Git add is available"
  assert_json '.result.structuredContent.exit_code == 0' "$role native Git add succeeds"
  git_tool commit -m "$role native Git write probe"
  assert_status "$HTTP_STATUS" 200 "$role native Git commit is available"
  assert_json '.result.structuredContent.exit_code == 0' "$role native Git commit succeeds"

  git_tool push origin HEAD
  assert_status "$HTTP_STATUS" 200 "$role push reaches native Git policy"
  assert_json '.result.structuredContent.output | contains("CONNECT_GITHUB_REQUIRED")' \
    "$role push has the same connected-GitHub restriction"

  if [[ "$role" == "developer" ]]; then
    mcp_call_tool "$WORK_TOKEN" run_command \
      "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" \
        '{project:$project,work_id:$work_id,command:"pwd"}')"
    assert_json '.result.isError == true' "$role still cannot invoke general shell commands"
  else
    mcp_call_tool "$WORK_TOKEN" run_command \
      "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" \
        '{project:$project,work_id:$work_id,command:"pwd"}')"
    assert_json '.result.structuredContent.exit_code == 0' "$role can invoke general shell commands"
  fi
done

assertion_summary
