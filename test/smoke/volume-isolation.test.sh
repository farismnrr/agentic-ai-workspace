#!/usr/bin/env bash
# shellcheck source=test/lib/assert.sh
# shellcheck source=test/lib/http.sh
# shellcheck source=test/lib/grants.sh
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_root"
source "$repo_root/test/lib/assert.sh"
source "$repo_root/test/lib/http.sh"
source "$repo_root/test/lib/grants.sh"

logs_dir=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/volume-isolation.XXXXXX")
work_dir=$(mktemp -d)
exec > >(tee "$logs_dir/runner.log") 2>&1

sha=$(git -C "$repo_root" rev-parse --short HEAD)
app_image="ghcr.io/farismnrr/agentic-ai-workspace:sha-$sha"
sandbox_image=${VOLUME_ISOLATION_IMAGE:-${SANDBOX_TEST_IMAGE:-ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest}}
volume="agentic-volume-isolation-$(date +%s)-$$"
app="agentic-volume-isolation-app-$$"
port=18187
APP_BASE="http://127.0.0.1:$port"
# shellcheck disable=SC2034 # Read by sourced test/lib/grants.sh.
APP_TMP_DIR=$work_dir
jar="$work_dir/grant-check.cookies"
state_file="$work_dir/grant-state.json"
touch "$jar"

cleanup() {
  local status=$?
  trap - EXIT
  docker logs --tail 100 "$app" > "$logs_dir/app.log" 2>&1 || true
  docker rm -f "$app" >/dev/null 2>&1 || true
  docker volume rm "$volume" >/dev/null 2>&1 || true
  rm -rf -- "$work_dir"
  printf 'Docker verification logs: %s\n' "$logs_dir"
  exit "$status"
}
trap cleanup EXIT

echo "Docker volume and egress grant checks, commit $(git -C "$repo_root" rev-parse HEAD)"
docker info --format 'Docker daemon: {{.ServerVersion}}'
docker version --format 'Client {{.Client.Version}} Server {{.Server.Version}}'
test -S /var/run/docker.sock
docker pull "$app_image"
docker pull "$sandbox_image"
docker image inspect "$app_image" --format 'App image ID {{.Id}}'
docker image inspect "$sandbox_image" --format 'Toolbox image ID {{.Id}}'
docker volume create "$volume" >/dev/null
docker run --rm --network none --mount "type=volume,source=$volume,target=/data" \
  --entrypoint sh busybox:1.36 -ec '
  mkdir -p /data/workspaces/a /data/workspaces/b /data/caches/a /data/caches/b \
    /data/egress/a /data/egress/b
  touch /data/workspaces/a/marker-a /data/workspaces/b/marker-b
  touch /data/caches/a/cache-a /data/caches/b/cache-b
  touch /data/egress/a/task-socket /data/egress/b/task-socket
  chown -R 65532:65532 /data
  '
echo 'PASS disposable owner volume initialized'

VOLUME_ISOLATION_VOLUME="$volume" VOLUME_ISOLATION_IMAGE="$sandbox_image" \
  go test -count=1 -v -timeout 4m ./internal/platform/sandbox \
    -run '^TestOwnerVolumeSubpathsAreIsolated$' | tee "$logs_dir/subpath.log"
if grep -q -- '--- SKIP' "$logs_dir/subpath.log"; then
  echo 'Subpath test unexpectedly skipped' >&2
  exit 1
fi
echo 'PASS actual Docker volume Subpath workspace and cache isolation'

docker run --rm --network none --mount \
  "type=volume,source=$volume,target=/run/egress,volume-subpath=egress/a,readonly" \
  --entrypoint sh "$sandbox_image" -ec '
  test -e /run/egress/task-socket
  if touch /run/egress/escape 2>/dev/null; then exit 1; fi
  '
echo 'PASS task egress socket Subpath is read-only'
for program in claude codex; do
  socket_scope=a
  [[ $program != codex ]] || socket_scope=b
  docker run --rm --network none --mount \
    "type=volume,source=$volume,target=/run/egress,volume-subpath=egress/$socket_scope,readonly" \
    --entrypoint sh "$sandbox_image" -ec "
      test -e /run/egress/task-socket
      command -v $program
      $program --version
    "
done
echo 'PASS vendor programs run with a task-scoped egress mount'

SANDBOX_TEST_IMAGE="$sandbox_image" go test -count=1 -v -timeout 12m \
  ./internal/modules/coding/adapter/runner \
  -run '^Test(OnlyTheWorkspaceIsWritable|TaskHasNoNetworkAndNoPrivileges)$' \
  | tee "$logs_dir/sandbox-boundaries.log"

go test -race -count=1 -timeout 12m ./internal/platform/egressproxy \
  -run '^Test(RevokeRacingNewTunnelsLeavesNoneOpen|ADNSAnswerThatChangesAfterTheCheckChangesNothing)$' \
  | tee "$logs_dir/grant-race.log"
go test -race -count=1 -timeout 12m ./internal/modules/task/adapter/sqlite \
  -run '^TestConcurrentApprovalCreatesOneDecisionAndHandoff$' \
  | tee "$logs_dir/approval-race.log"
go test -race -count=1 -timeout 12m ./internal/bootstrap \
  -run '^Test(ApprovalReachesTheProxyAndTaskEndClosesIt|RestartRebuildsFromTheDatabaseOnly|ADatabaseFailureRefusesConnectionsAndTheSweepEmptiesTheRegistry|EachTaskSocketServesOnlyItsOwnTasksGrantsThroughTheRealProxy)$' \
  | tee "$logs_dir/grant-http-regressions.log"
go test -race -count=1 -timeout 12m ./internal/platform/database \
  -run '^Test(JournalRecordsChecksums|EditedMigrationIsRejected|RemovedMigrationIsRejected)$' \
  | tee "$logs_dir/migrations.log"
echo 'PASS focused grant race, fail-closed and migration checks'

docker run -d --name "$app" --network host --read-only --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --group-add "$(stat -c %g /var/run/docker.sock)" \
  --mount "type=volume,source=$volume,target=/data" \
  --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock \
  -e "PORT=$port" -e "PUBLIC_URL=$APP_BASE" -e DATA_DIR=/data -e EGRESS_GRANTS=true \
  -e "SANDBOX_IMAGE=$sandbox_image" -e "SANDBOX_DATA_VOLUME=$volume" \
  "$app_image" >/dev/null

ready() {
  local http_ready=0 healthy=0
  for _ in $(seq 1 90); do
    if curl -fsS "$APP_BASE/api/health" > "$work_dir/health.json"; then
      http_ready=1
      if [[ $(docker inspect "$app" --format '{{.State.Health.Status}}') == healthy ]]; then
        healthy=1
        break
      fi
    fi
    sleep 2
  done
  if ((http_ready == 0 || healthy == 0)); then
    docker logs "$app" >&2 || true
    return 1
  fi
  printf 'PASS container health: %s\n' "$(cat "$work_dir/health.json")"
}

ready
docker inspect "$app" --format '{{.HostConfig.ReadonlyRootfs}} {{.HostConfig.NetworkMode}}' \
  | grep -q '^true host$'
echo 'PASS app container is read-only and uses host networking'
grant_setup "$jar" "$state_file"
docker restart "$app" >/dev/null
ready
grant_verify_restart "$jar" "$state_file"
assertion_summary
echo 'PASS approved grant restored and remained revocable after restart'
