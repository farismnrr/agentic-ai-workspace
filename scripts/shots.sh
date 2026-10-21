#!/bin/sh
# The throwaway instance listens on all interfaces for the few seconds it runs.
set -eu

root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
binary="$root/local-build/agentic-ai-workspace"
[ -x "$binary" ] || binary="$root/bin/agentic-ai-workspace"
[ -x "$binary" ] || { echo "Needs the Developer: build the app binary first" >&2; exit 1; }

work="$(mktemp -d /tmp/workspace-shots.XXXXXX)"
port="${SHOTS_PORT:-8091}"
password="$(head -c 18 /dev/urandom | base64 | tr -d '/+=')"
server_pid=""
cleanup() {
  [ -z "$server_pid" ] || kill "$server_pid" 2>/dev/null || true
  [ -z "$server_pid" ] || wait "$server_pid" 2>/dev/null || true
  rm -rf "$work"
}
trap cleanup EXIT INT TERM

while ! python3 -c 'import socket,sys; s=socket.socket(); s.bind(("127.0.0.1",int(sys.argv[1]))); s.close()' "$port" 2>/dev/null; do
  port=$((port + 1))
  [ "$port" -le 65535 ] || { echo "No free app port found" >&2; exit 1; }
done

env -i PATH="$PATH" HOME="$HOME" DATA_DIR="$work/data" ADMIN_EMAIL=shots@example.test \
  PORT="$port" PUBLIC_URL="http://127.0.0.1:$port" "$binary" >"$work/app.log" 2>&1 &
server_pid=$!
origin="http://127.0.0.1:$port"
for _ in $(seq 1 60); do
  if curl -fsS "$origin/api/health" >/dev/null 2>&1; then break; fi
  sleep 1
done
if ! curl -fsS "$origin/api/health" >/dev/null; then
  cat "$work/app.log" >&2
  echo "Throwaway app did not answer /api/health" >&2
  exit 1
fi

curl -fsS -c "$work/cookies.txt" -H 'Content-Type: application/json' \
  -d "$(jq -cn --arg password "$password" '{name:"Screenshot Admin",email:"shots@example.test",password:$password}')" \
  "$origin/api/auth/register" >"$work/register.json"
project_json="$work/project.json"
curl -fsS -b "$work/cookies.txt" -H 'Content-Type: application/json' \
  -d '{"name":"Screenshot demo","description":"A sample project for review.","slug":"screenshot-demo"}' \
  "$origin/api/projects/" >"$project_json"
project_id="$(jq -r '.id' "$project_json")"
[ -n "$project_id" ] && [ "$project_id" != null ] || { cat "$project_json" >&2; exit 1; }

curl -fsS -b "$work/cookies.txt" -H 'Content-Type: application/json' -X PUT \
  -d '{"host":"db.example.test","port":3306,"database":"app","user":"admin","password":"demo","tls":"none"}' \
  "$origin/api/settings/database/mysql" >"$work/mysql-settings.json"

tasks_created=0
for prompt in 'Review the project overview' 'Check the main workflow' 'Prepare a release note'; do
  status="$(curl -sS -b "$work/cookies.txt" -H 'Content-Type: application/json' \
    -d "$(jq -cn --arg project "$project_id" --arg prompt "$prompt" '{projectId:$project,prompt:$prompt}')" \
    -o "$work/task.json" -w '%{http_code}' "$origin/api/tasks")"
  case "$status" in
    2*) tasks_created=$((tasks_created + 1)) ;;
    *) task_error="$(cat "$work/task.json")"; echo "Skipping remaining demo tasks: POST /api/tasks returned HTTP $status: $task_error" >&2; break ;;
  esac
done
if [ "$tasks_created" -eq 0 ]; then echo 'Tasks will show the empty state because task creation needs a configured model provider.' >&2; fi

mkdir -p "$root/local-build/shots"
if [ "$#" -gt 0 ]; then
  bun "$root/scripts/shots/capture-auth.ts" "$origin" "$work/cookies.txt" "$root/local-build/shots" "$project_id" "$@"
else
  bun "$root/scripts/shots/capture-auth.ts" "$origin" "$work/cookies.txt" "$root/local-build/shots" "$project_id"
fi
