#!/usr/bin/env bash
# Manual Docker Stage 7 gate. Uses only disposable test data and containers.
set -Eeuo pipefail
mkdir -p stage7-evidence
exec > >(tee stage7-evidence/runner.log) 2>&1
echo "Stage 7 Docker gate, commit $(git rev-parse HEAD)"
docker info --format 'Docker daemon: {{.ServerVersion}}'
docker version --format 'Client {{.Client.Version}} Server {{.Server.Version}}'
test -S /var/run/docker.sock

sha=$(git rev-parse --short HEAD)
app_image="ghcr.io/farismnrr/agentic-ai-workspace:sha-$sha"
sandbox_image="ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest"
volume="aw-stage7-$(date +%s)-$$"
app="aw-stage7-app-$$"
temp=$(mktemp -d)
port=18187
cleanup() {
  status=$?
  echo "Stage 7 exit status: $status"
  docker logs --tail 100 "$app" > stage7-evidence/app.log 2>&1 || true
  docker rm -f "$app" >/dev/null 2>&1 || true
  docker volume rm "$volume" >/dev/null 2>&1 || true
  rm -rf "$temp"
}
trap cleanup EXIT

docker pull "$app_image"
docker pull "$sandbox_image"
docker image inspect "$app_image" --format 'App image ID {{.Id}}'
docker image inspect "$sandbox_image" --format 'Toolbox image ID {{.Id}}'
docker volume create "$volume" >/dev/null
docker run --rm --network none --mount "type=volume,source=$volume,target=/data" \
  --entrypoint sh busybox:1.36 -ec '
  mkdir -p /data/workspaces/a /data/workspaces/b /data/caches/a /data/caches/b /data/egress/a
  touch /data/workspaces/a/marker-a /data/workspaces/b/marker-b
  touch /data/caches/a/cache-a /data/caches/b/cache-b
  chown -R 65532:65532 /data
  '
echo 'PASS isolated named volume initialized'

STAGE7_VOLUME="$volume" STAGE7_IMAGE="$sandbox_image" \
  go test -count=1 -v -timeout 4m ./internal/platform/sandbox \
    -run '^TestStage7ActualVolumeSubpathsAreIsolated$' | tee stage7-evidence/subpath.log
if grep -q -- '--- SKIP' stage7-evidence/subpath.log; then
  echo 'Subpath test unexpectedly skipped' >&2
  exit 1
fi
echo 'PASS real Docker Engine Subpath /workspace and /cache isolation'

docker run --rm --network none --mount \
  "type=volume,source=$volume,target=/run/egress,volume-subpath=egress/a,readonly" \
  --entrypoint sh "$sandbox_image" -ec '
  if touch /run/egress/escape 2>/dev/null; then exit 1; fi
  '
echo 'PASS task egress Subpath is read-only'

docker run --rm --network none --entrypoint sh "$sandbox_image" -ec '
  command -v claude
  command -v codex
  claude --version
  codex --version
  '
echo 'PASS vendor binaries present in isolated toolbox'

SANDBOX_TEST_IMAGE="$sandbox_image" \
  go test -count=1 -v -timeout 12m ./internal/modules/coding/adapter/runner \
    -run '^Test(AGrantIsUsableOnlyFromItsOwnTasksContainer|TheContainerSeesOnlyItsOwnSocketAndCannotChangeIt|Stage7RealPublicHTTPSNeedsTaskGrant|AVendorProgramContainerGetsOnlyItsOwnTasksSocket|AVendorProgramForATaskThatIsOverDoesNotStartAndFreesItsPlace|WithoutAScopeAndForSitesTheWorkspaceSocketIsUsedAndCarriesNoGrants)$' \
  | tee stage7-evidence/socket-isolation.log
if grep -q -- '--- SKIP' stage7-evidence/socket-isolation.log; then
  echo 'Socket isolation test unexpectedly skipped' >&2
  exit 1
fi
echo 'PASS Coding Runner and vendor Lease socket isolation'

go test -race -count=1 -timeout 12m \
  -run '^Test(RevokeRacingNewTunnelsLeavesNoneOpen|ApprovalRevocationAndSweepsRacingLeaveTheRegistryEqualToTheDatabase|ConcurrentSweepsAndTaskEndNeverResurrectAGrant|ADatabaseFailureRefusesConnectionsAndTheSweepEmptiesTheRegistry|AnExistingDatabaseTakes052WithoutTouchingTheJournal|ExistingAuditMigrationGetsAppendOnlyActorAttribution)$' \
  ./internal/platform/egressproxy ./internal/modules/egressgrant/adapter/sqlite \
  ./internal/bootstrap ./internal/platform/database \
  | tee stage7-evidence/race-migrations.log
echo 'PASS focused race, fail-closed and migration tests'

docker run -d --name "$app" --network host --read-only --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --group-add "$(stat -c %g /var/run/docker.sock)" \
  --mount "type=volume,source=$volume,target=/data" \
  --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock \
  -e "PORT=$port" -e "PUBLIC_URL=http://127.0.0.1:$port" \
  -e DATA_DIR=/data -e EGRESS_GRANTS=true \
  -e "SANDBOX_IMAGE=$sandbox_image" -e "SANDBOX_DATA_VOLUME=$volume" \
  "$app_image" >/dev/null

ready() {
  for _ in $(seq 1 90); do
    if curl -fsS "http://127.0.0.1:$port/api/health" > "$temp/health.json"; then
      return 0
    fi
    sleep 2
  done
  docker logs "$app" >&2 || true
  return 1
}
ready
echo "PASS containerized app health $(cat "$temp/health.json")"
docker inspect "$app" --format '{{.HostConfig.ReadonlyRootfs}} {{.HostConfig.NetworkMode}}' \
  | grep -q '^true host$'

python3 scripts/stage7-http.py "http://127.0.0.1:$port" "$volume" setup "$temp/state.json" \
  | tee stage7-evidence/http-setup.log
docker restart "$app" >/dev/null
ready
python3 scripts/stage7-http.py "http://127.0.0.1:$port" "$volume" restart "$temp/state.json" \
  | tee stage7-evidence/http-restart.log
echo 'PASS Docker-backed subset; real marimo-pair remains a separate external gate'
