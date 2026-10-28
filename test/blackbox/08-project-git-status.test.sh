#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"

app_init
app_start
owner="$APP_TMP_DIR/owner.cookies"
other="$APP_TMP_DIR/other.cookies"
touch "$owner" "$other"
http_register "$owner" 'Git owner' 'git-owner@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register owner'
http_register "$other" 'Other user' 'git-other@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register other user'
http_project_create "$owner" 'Git status project'
assert_status "$HTTP_STATUS" 201 'create project'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")

http_request GET '' "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 401 'anonymous Git status is forbidden'
http_request GET '' "/api/projects/$project_id/git/head?path=readme.txt"
assert_status "$HTTP_STATUS" 401 'anonymous Git baseline is forbidden'
http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 200 'owner reads status for non-Git project'
assert_json '.available == false and .files == []' 'non-Git project is explicit'
http_request GET "$owner" "/api/projects/$project_id/git/head?path=readme.txt"
assert_status "$HTTP_STATUS" 200 'owner reads baseline availability for non-Git project'
assert_json '.available == false and .content == ""' 'non-Git baseline is unavailable, not invented'
http_request GET "$other" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 404 'another owner cannot read Git status'
http_request GET "$other" "/api/projects/$project_id/git/head?path=readme.txt"
assert_status "$HTTP_STATUS" 404 'another owner cannot read Git baseline'

project_dir="$APP_DATA_DIR/projects/$project_id"
cd "$project_dir"
git init -q
git config user.name 'Blackbox Test'
git config user.email 'blackbox@example.test'
printf 'before\n' > 'space ü file.txt'
printf 'before\n' > 'staged.txt'
printf 'before\n' > 'both.txt'
printf '.ignored\n' > .gitignore
git add .
git commit -qm 'initial files'
printf 'edited\n' > 'space ü file.txt'
printf 'staged\n' > staged.txt
git add staged.txt
printf 'staged\n' > both.txt
git add both.txt
printf 'unstaged\n' >> both.txt
git mv 'space ü file.txt' 'renamed space ü.txt'
printf 'new\n' > untracked.txt
printf 'secret\n' > .env
printf 'ignored\n' > .ignored
cd "$REPO_ROOT"

http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 200 'owner reads repository status'
assert_json '.available and (.revision | length) > 0' 'snapshot includes repository revision'
assert_json 'any(.files[]; .path == "renamed space ü.txt" and .originalPath == "space ü file.txt")' \
  'rename preserves both paths with spaces and Unicode'
assert_json 'any(.files[]; .path == "staged.txt" and .index == "M" and .worktree == "")' \
  'staged-only change stays distinct'
assert_json 'any(.files[]; .path == "both.txt" and .index == "M" and .worktree == "M")' \
  'staged and unstaged changes stay distinct'
assert_json 'any(.files[]; .path == "untracked.txt" and .untracked)' 'untracked file is reported'
assert_json 'all(.files[]; .path != ".env" and .path != ".ignored")' 'secret and ignored paths stay hidden'
http_request GET "$owner" "/api/projects/$project_id/git/head?path=space%20%C3%BC%20file.txt"
assert_status "$HTTP_STATUS" 200 'owner reads a tracked file baseline by encoded path'
assert_json '.available and .content == "before\n" and (.content | length) < 100' 'HEAD baseline contains the committed text only'
http_request GET "$owner" "/api/projects/$project_id/git/head?path=untracked.txt"
assert_status "$HTTP_STATUS" 200 'untracked file has an empty HEAD baseline'
assert_json '.available and .content == ""' 'missing HEAD path is an explicit empty baseline'
http_request GET "$owner" "/api/projects/$project_id/git/head?path=..%2F.env"
assert_status "$HTTP_STATUS" 404 'traversal baseline path is denied'
http_request GET "$owner" "/api/projects/$project_id/git/head?path=.env"
assert_status "$HTTP_STATUS" 404 'secret baseline path is denied'

cd "$project_dir"
git add -u
git commit -qm 'save status fixture'
git branch -M main
git checkout -qb conflict-side
printf 'side\n' > both.txt
git commit -qam 'change on side'
git checkout -q main
printf 'main\n' > both.txt
git commit -qam 'change on main'
if git merge -q conflict-side; then
  printf 'Expected the fixture merge to conflict.\n' >&2
  exit 1
fi
http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 200 'conflicted repository status remains readable'
assert_json 'any(.files[]; .path == "both.txt" and .conflict)' 'merge conflict is explicit'

assertion_summary
