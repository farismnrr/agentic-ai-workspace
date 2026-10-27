#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
source "$REPO_ROOT/test/lib/assert.sh"
source "$REPO_ROOT/test/lib/app.sh"
source "$REPO_ROOT/test/lib/http.sh"

app_init
app_start
jar="$APP_TMP_DIR/user.cookies"
touch "$jar"
http_register "$jar" 'Alice' 'alice@example.test' 'correct horse battery'
assert_status "$HTTP_STATUS" 201 'account registration'
http_project_create "$jar" 'Clean shutdown project'
assert_status "$HTTP_STATUS" 201 'project creation'
project_id=$(jq -r '.id' "$HTTP_BODY_FILE")

second_log="$APP_TMP_DIR/second.log"
PORT="$APP_PORT" DATA_DIR="$APP_DATA_DIR" PUBLIC_URL="$APP_BASE" \
  "$APP_BIN" >"$second_log" 2>&1 &
second_pid=$!
sleep 1
if kill -0 "$second_pid" 2>/dev/null; then
  kill "$second_pid" 2>/dev/null || true
  wait "$second_pid" 2>/dev/null || true
  printf 'FAIL second process refused to bind the occupied listener\n'
  ((ASSERTIONS_FAILED += 1))
else
  if wait "$second_pid" 2>/dev/null; then
    second_exit=0
  else
    second_exit=$?
  fi
  if [[ $second_exit != 0 ]] && grep -Eqi 'address already in use|another instance already holds the lock' "$second_log"; then
    printf 'PASS second process reported the occupied port or instance lock\n'
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL second process did not report the occupied port or instance lock\n'
    cat "$second_log"
    ((ASSERTIONS_FAILED += 1))
  fi
fi

http_request GET "$jar" /api/health
assert_status "$HTTP_STATUS" 200 'first process stays healthy'
old_pid=$APP_PID
children=$(ps -o pid= --ppid "$old_pid" 2>/dev/null | awk '{$1=$1}; NF {print $1}')
app_stop
if kill -0 "$old_pid" 2>/dev/null; then
  printf 'FAIL app process stopped cleanly\n'
  ((ASSERTIONS_FAILED += 1))
else
  printf 'PASS app process stopped cleanly\n'
  ((ASSERTIONS_PASSED += 1))
fi
remaining_children=''
for child in $children; do
  if kill -0 "$child" 2>/dev/null; then
    remaining_children+="$child "
  fi
done
if [[ -z $remaining_children ]]; then
  printf 'PASS stopped process has no child processes\n'
  ((ASSERTIONS_PASSED += 1))
else
  printf 'FAIL stopped process left child processes: %s\n' "$remaining_children"
  ((ASSERTIONS_FAILED += 1))
fi

app_start
http_request GET "$jar" /api/projects
# shellcheck disable=SC2016 # The single-quoted expression is jq syntax, not shell expansion.
assert_json 'any(.[]; .id == $id)' 'data survives clean shutdown and restart' --arg id "$project_id"

assertion_summary
