#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"
source "$REPO_ROOT/test/lib/mcp.sh"

app_init
app_start

assert_equal() {
  local actual=$1 expected=$2 label=$3
  if [[ $actual == "$expected" ]]; then
    printf 'PASS %s\n' "$label"
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL %s\n' "$label"
    ((ASSERTIONS_FAILED += 1))
  fi
}
owner="$APP_TMP_DIR/owner.cookies"
touch "$owner"
http_register "$owner" 'Build stage owner' 'build-stage@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register build-stage owner'
http_project_create "$owner" 'Build stage fixture'
assert_status "$HTTP_STATUS" 201 'create isolated build project'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")

http_request PUT "$owner" /api/spaces/coding/tools/run_command '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'enable bounded command probe'
http_request PUT "$owner" /api/spaces/coding/tools/run_build '{"enabled":true}'
assert_status "$HTTP_STATUS" 204 'enable named build tool'
oauth_code_token "$owner" 'openid email coding:read coding:exec' "$APP_BASE/api/coding"
assert_status "$HTTP_STATUS" 200 'issue execution-scoped Coding token'
WORK_TOKEN=$(jq -r '.access_token' "$HTTP_BODY_FILE")
mcp_call_tool "$WORK_TOKEN" list_projects '{}'
assert_status "$HTTP_STATUS" 200 'list projects'
project_slug=$(jq -r '.result.structuredContent.projects[] | select(.name == "Build stage fixture") | .id' "$HTTP_BODY_FILE")
mcp_call_tool "$WORK_TOKEN" begin_work "$(jq -cn --arg project "$project_slug" \
  '{project:$project,summary:"Verify isolated build staging",role:"researcher",new_task:true}')"
assert_status "$HTTP_STATUS" 200 'start tracked DevOps work'
WORK_ID=$(jq -r '.result.structuredContent.work_id' "$HTTP_BODY_FILE")
mcp_call_tool "$WORK_TOKEN" set_role "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" --arg role devops '{project:$project,work_id:$work_id,role:$role}')"
assert_status "$HTTP_STATUS" 200 'switch tracked work to DevOps'

project_dir="$APP_DATA_DIR/projects/$project_id"
mkdir -p "$project_dir/.workspace" "$project_dir/.git"
printf 'git marker\n' > "$project_dir/.git/config"
printf 'original source\n' > "$project_dir/source.txt"
printf 'protected task marker\n' > "$project_dir/.workspace/secret-marker"
mkdir -p "$project_dir/.build-output-interrupted/.build-stage"
printf 'abandoned scratch\n' > "$project_dir/.build-output-interrupted/.build-stage/leftover"
printf 'retained artifact\n' > "$project_dir/.build-output-interrupted/retained.txt"
printf 'private setting\n' > "$project_dir/.env.test"
printf 'outside private file\n' > "$APP_TMP_DIR/private-file"
ln -s "$APP_TMP_DIR/private-file" "$project_dir/outside-link"
source_hash=$(sha256sum "$project_dir/source.txt" | cut -d' ' -f1)
mkdir -p "$project_dir/src"
cat > "$project_dir/src/index.html" <<'HTML'
<!doctype html><title>Staged frontend</title><main>Built safely</main>
HTML
cat > "$project_dir/build.mjs" <<'BUILD'
import { mkdir, readFile, writeFile } from 'node:fs/promises'

const source = await readFile('src/index.html', 'utf8')
await mkdir('dist', { recursive: true })
await writeFile('dist/index.html', source.replace('Staged frontend', 'Built frontend'))
BUILD
cat > "$project_dir/.workspace/tasks.json" <<'TASKS'
{"tasks":{"stage-probe":{"description":"Build a frontend in a disposable staged copy","kind":"build","command":["sh","-c","test -f source.txt && test ! -e .env.test && test ! -e .workspace && test ! -e .git && test ! -L outside-link && test \"$PWD\" = /output/.build-stage && node build.mjs && grep -q 'Built frontend' dist/index.html && printf changed > source.txt && cp dist/index.html \"$BUILD_OUT/index.html\""]}}}
TASKS

mcp_call_tool "$WORK_TOKEN" run_build "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" \
  '{project:$project,work_id:$work_id,name:"stage-probe"}')"
assert_status "$HTTP_STATUS" 200 'run isolated named build'
assert_json '.result.structuredContent | .exit_code == 0 and (.output | contains("Generated outputs:"))' \
  'named build succeeds from its staged source copy'

assert_equal "$(sha256sum "$project_dir/source.txt" | cut -d' ' -f1)" "$source_hash" \
  'original source hash remains unchanged'
output_file=$(find "$project_dir" -maxdepth 2 -path '*/.build-output-*/index.html' -type f -print -quit)
assert_equal "$(grep -o 'Built frontend' "$output_file")" 'Built frontend' \
  'Node frontend build artifact persists in BUILD_OUT'
stage_path=$(find "$project_dir" -maxdepth 2 -path '*/.build-output-*/.build-stage' -print -quit)
assert_equal "$stage_path" "" 'staging contents and directory are cleaned'
assert_equal "$(cat "$project_dir/.build-output-interrupted/retained.txt")" 'retained artifact' \
  'recovery preserves persisted build artifacts'
assert_equal "$(test -e "$project_dir/.build-output-interrupted/.build-stage" && printf present || true)" "" \
  'recovery removes only the abandoned stage directory'
persisted_secret=$(grep -R -l 'protected task marker' "$project_dir"/.build-output-* 2>/dev/null || true)
assert_equal "$persisted_secret" "" 'protected task marker is absent from persisted artifacts'

mcp_call_tool "$WORK_TOKEN" run_command "$(jq -cn --arg project "$project_id" --arg work_id "$WORK_ID" \
  '{project:$project,work_id:$work_id,command:"printf changed > source.txt"}')"
assert_status "$HTTP_STATUS" 200 'run shell source-write boundary probe'
assert_json '.result.structuredContent | .exit_code != 0 and (.output | contains("run_build"))' \
  'read-only shell write points DevOps to named builds'
assert_equal "$(sha256sum "$project_dir/source.txt" | cut -d' ' -f1)" "$source_hash" \
  'DevOps shell cannot change source'

assertion_summary
