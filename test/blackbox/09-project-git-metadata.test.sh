#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"

app_init
app_start
owner="$APP_TMP_DIR/owner.cookies"
touch "$owner"
http_register "$owner" 'Metadata owner' 'git-metadata@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'register metadata test owner'
http_project_create "$owner" 'Git metadata project'
assert_status "$HTTP_STATUS" 201 'create metadata test project'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")
project_dir="$APP_DATA_DIR/projects/$project_id"
outside_repo="$APP_TMP_DIR/outside-repo"
marker="$APP_TMP_DIR/filter-ran"
outside_secret='outside-object-secret-947'

mkdir "$outside_repo"
git -C "$outside_repo" init -q
git -C "$outside_repo" config user.name 'Outside Test'
git -C "$outside_repo" config user.email 'outside@example.test'
printf '%s\n' "$outside_secret" > "$outside_repo/outside-secret.txt"
git -C "$outside_repo" add .
git -C "$outside_repo" commit -qm 'outside secret object'
outside_commit=$(git -C "$outside_repo" rev-parse HEAD)

cd "$project_dir"
git init -q
git config user.name 'Metadata Test'
git config user.email 'metadata@example.test'
printf 'baseline\n' > tracked.txt
printf '*.txt filter=marker\n' > .gitattributes
git add .
git commit -qm 'initial project files'
safe_config="$APP_TMP_DIR/project-config"
cp .git/config "$safe_config"
project_head=$(cat .git/HEAD)
git gc --prune=now
cd "$REPO_ROOT"

http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 200 'packed repository status stays readable'
assert_json '.available' 'packed repository snapshot remains available'
http_request GET "$owner" "/api/projects/$project_id/git/head?path=tracked.txt"
assert_status "$HTTP_STATUS" 200 'packed repository HEAD stays readable'
assert_json '.available and .content == "baseline\n"' 'packed Git objects preserve the HEAD baseline'

for syntax in legacy quoted; do
  cp "$safe_config" "$project_dir/.git/config"
  if [[ $syntax == legacy ]]; then
    printf '\n[filter.marker]\n\tclean = "touch %s; cat"\n' "$marker" >> "$project_dir/.git/config"
  else
    printf '\n[filter "marker"]\n\tclean = "touch %s; cat"\n' "$marker" >> "$project_dir/.git/config"
  fi
  http_request GET "$owner" "/api/projects/$project_id/git/status"
  assert_status "$HTTP_STATUS" 503 "$syntax filter config is rejected"
  assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" "$syntax filter response contains no outside data"
  if [[ -e $marker ]]; then
    printf 'Git ran the %s filter marker.\n' "$syntax" >&2
    exit 1
  fi
done
printf 'PASS legacy and quoted filters did not run\n'
((ASSERTIONS_PASSED += 1))
cp "$safe_config" "$project_dir/.git/config"

printf '%s\n' "$outside_commit" > "$project_dir/.git/HEAD"
mv "$project_dir/.git/objects" "$APP_TMP_DIR/project-objects"
ln -s "$outside_repo/.git/objects" "$project_dir/.git/objects"
http_request GET "$owner" "/api/projects/$project_id/git/head?path=outside-secret.txt"
assert_status "$HTTP_STATUS" 503 'object store symlink makes HEAD unavailable'
assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'object store symlink cannot return outside content'
http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 503 'object store symlink makes status unavailable'
rm "$project_dir/.git/objects"
mv "$APP_TMP_DIR/project-objects" "$project_dir/.git/objects"
printf '%s\n' "$project_head" > "$project_dir/.git/HEAD"

printf '%s\n' "$outside_repo/.git/objects" > "$project_dir/.git/objects/info/alternates"
printf '%s\n' "$outside_commit" > "$project_dir/.git/HEAD"
http_request GET "$owner" "/api/projects/$project_id/git/head?path=outside-secret.txt"
assert_status "$HTTP_STATUS" 503 'object alternates make HEAD unavailable'
assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'object alternates cannot return outside content'
rm "$project_dir/.git/objects/info/alternates"
printf 'https://outside.invalid/objects\n' > "$project_dir/.git/objects/info/http-alternates"
printf '%s\n' "$outside_commit" > "$project_dir/.git/HEAD"
http_request GET "$owner" "/api/projects/$project_id/git/head?path=outside-secret.txt"
assert_status "$HTTP_STATUS" 503 'HTTP object alternates make HEAD unavailable'
assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'HTTP object alternates cannot return outside content'
rm "$project_dir/.git/objects/info/http-alternates"
printf '%s\n' "$project_head" > "$project_dir/.git/HEAD"

mv "$project_dir/.git" "$APP_TMP_DIR/project-git"
printf 'gitdir: %s\n' "$APP_TMP_DIR/project-git" > "$project_dir/.git"
http_request GET "$owner" "/api/projects/$project_id/git/head?path=outside-secret.txt"
assert_status "$HTTP_STATUS" 503 'Gitfile indirection makes HEAD unavailable'
assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'Gitfile cannot return outside content'
rm "$project_dir/.git"
mv "$APP_TMP_DIR/project-git" "$project_dir/.git"

printf '%s\n' "$outside_repo/.git" > "$project_dir/.git/commondir"
http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 503 'commondir indirection makes status unavailable'
rm "$project_dir/.git/commondir"

cp "$safe_config" "$project_dir/.git/config"
printf '\n[extensions]\n\tworktreeConfig = true\n' >> "$project_dir/.git/config"
printf '\n[filter.marker]\n\tclean = "touch %s; cat"\n' "$marker" > "$project_dir/.git/config.worktree"
http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 503 'config.worktree linked-worktree settings are rejected'
if [[ -e $marker ]]; then
  printf 'Git ran the config.worktree marker.\n' >&2
  exit 1
fi
rm "$project_dir/.git/config.worktree"
cp "$safe_config" "$project_dir/.git/config"

submodule_repo="$APP_TMP_DIR/submodule-repo"
mkdir "$submodule_repo"
git -C "$submodule_repo" init -q
git -C "$submodule_repo" config user.name 'Submodule Test'
git -C "$submodule_repo" config user.email 'submodule@example.test'
printf '*.txt filter=marker\n' > "$submodule_repo/.gitattributes"
printf 'submodule file\n' > "$submodule_repo/nested.txt"
git -C "$submodule_repo" add .
git -C "$submodule_repo" commit -qm 'submodule files'
git -C "$project_dir" -c protocol.file.allow=always submodule add -q "$submodule_repo" nested
printf '\n[filter.marker]\n\tclean = "touch %s; cat"\n' "$marker" >> \
  "$project_dir/.git/modules/nested/config"
http_request GET "$owner" "/api/projects/$project_id/git/status"
assert_status "$HTTP_STATUS" 200 'status ignores nested submodule metadata'
assert_json '.available' 'submodule status remains available'
if [[ -e $marker ]]; then
  printf 'Git read or ran a nested submodule filter.\n' >&2
  exit 1
fi
printf 'PASS nested submodule filter did not run\n'
((ASSERTIONS_PASSED += 1))

cp "$safe_config" "$APP_TMP_DIR/config-safe"
head -c 1048577 /dev/zero | tr '\0' x > "$APP_TMP_DIR/config-large"
growth_stop="$APP_TMP_DIR/config-growth-stop"
(
  while [[ ! -e $growth_stop ]]; do
    cp "$APP_TMP_DIR/config-large" "$project_dir/.git/config.next"
    mv -f "$project_dir/.git/config.next" "$project_dir/.git/config"
    cp "$APP_TMP_DIR/config-safe" "$project_dir/.git/config.next"
    mv -f "$project_dir/.git/config.next" "$project_dir/.git/config"
  done
) &
race_pid=$!
for attempt in $(seq 1 20); do
  http_request GET "$owner" "/api/projects/$project_id/git/status"
  if [[ $HTTP_STATUS != 200 && $HTTP_STATUS != 503 ]]; then
    printf 'FAIL config growth race returned HTTP %s\n' "$HTTP_STATUS"
    ((ASSERTIONS_FAILED += 1))
  else
    ((ASSERTIONS_PASSED += 1))
  fi
  assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'config growth race exposes no outside content'
done
touch "$growth_stop"
wait "$race_pid"
cp "$safe_config" "$project_dir/.git/config"
printf 'PASS config growth race stayed within the cap\n'
((ASSERTIONS_PASSED += 1))

printf '\n[filter.marker]\n\tclean = "touch %s; cat"\n' "$marker" > "$APP_TMP_DIR/config-attack"
race_stop="$APP_TMP_DIR/config-race-stop"
(
  while [[ ! -e $race_stop ]]; do
    cp "$APP_TMP_DIR/config-attack" "$project_dir/.git/config.next"
    mv -f "$project_dir/.git/config.next" "$project_dir/.git/config"
    cp "$APP_TMP_DIR/config-safe" "$project_dir/.git/config.next"
    mv -f "$project_dir/.git/config.next" "$project_dir/.git/config"
  done
) &
race_pid=$!
for attempt in $(seq 1 30); do
  http_request GET "$owner" "/api/projects/$project_id/git/status"
  if [[ $HTTP_STATUS != 200 && $HTTP_STATUS != 503 ]]; then
    printf 'FAIL config replacement race returned HTTP %s\n' "$HTTP_STATUS"
    ((ASSERTIONS_FAILED += 1))
  else
    ((ASSERTIONS_PASSED += 1))
  fi
  assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'config replacement race exposes no outside content'
done
touch "$race_stop"
wait "$race_pid"
cp "$safe_config" "$project_dir/.git/config"
if [[ -e $marker ]]; then
  printf 'Git ran a filter during config replacement.\n' >&2
  exit 1
fi
printf 'PASS config replacement race did not run a filter\n'
((ASSERTIONS_PASSED += 1))

mv "$project_dir/.git/objects" "$APP_TMP_DIR/objects-live"
printf '%s\n' "$outside_commit" > "$project_dir/.git/HEAD"
object_race_stop="$APP_TMP_DIR/object-race-stop"
(
  while [[ ! -e $object_race_stop ]]; do
    if mv "$APP_TMP_DIR/objects-live" "$project_dir/.git/objects" 2>/dev/null; then
      mv "$project_dir/.git/objects" "$APP_TMP_DIR/objects-live"
      ln -s "$outside_repo/.git/objects" "$project_dir/.git/objects"
      sleep 0.002
      rm "$project_dir/.git/objects"
      mv "$APP_TMP_DIR/objects-live" "$project_dir/.git/objects"
    fi
  done
) &
race_pid=$!
for attempt in $(seq 1 30); do
  http_request GET "$owner" "/api/projects/$project_id/git/head?path=outside-secret.txt"
  if [[ $HTTP_STATUS != 200 && $HTTP_STATUS != 503 ]]; then
    printf 'FAIL object replacement race returned HTTP %s\n' "$HTTP_STATUS"
    ((ASSERTIONS_FAILED += 1))
  else
    ((ASSERTIONS_PASSED += 1))
  fi
  assert_not_contains "$HTTP_BODY_FILE" "$outside_secret" 'object replacement race exposes no outside content'
done
touch "$object_race_stop"
wait "$race_pid"
if [[ -L $project_dir/.git/objects ]]; then
  rm "$project_dir/.git/objects"
fi
if [[ -d $APP_TMP_DIR/objects-live ]]; then
  mv "$APP_TMP_DIR/objects-live" "$project_dir/.git/objects"
fi
printf '%s\n' "$project_head" > "$project_dir/.git/HEAD"
printf 'PASS object store replacement race exposed no outside content\n'
((ASSERTIONS_PASSED += 1))

assertion_summary
