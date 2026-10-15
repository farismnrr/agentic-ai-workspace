#!/bin/sh
# Hotfix deploy: swaps the binary of the running app container without rebuilding the image or
# removing the container. The binary lives in a host folder mounted read-only at /opt/app (a folder,
# not a single file, so an atomic rename is seen inside), so a swap is rename + `docker restart`.
# If the new build does not answer with its version, the previous binary is put back.
set -eu

bin="${1:?usage: deploy-binary.sh <binary> <version> <image>}"
version="${2:?usage: deploy-binary.sh <binary> <version> <image>}"
image="${3:?usage: deploy-binary.sh <binary> <version> <image>}"
dir="${DEPLOY_DIR:-/var/jenkins_home/deploy/agentic-ai-workspace}"
name="${CONTAINER:-agentic-ai-workspace-prod}"
port="${HOST_PORT:-8080}"

# The published port lives on the Docker host: the daemon's host for tcp://, else this machine.
host=127.0.0.1
case "${DOCKER_HOST:-}" in tcp://*) host="${DOCKER_HOST#tcp://}" && host="${host%:*}" ;; esac
url="http://${host}:${port}"

running_version() {
  curl -fsS "$url/api/health" 2>/dev/null | sed -n 's/.*"version":"\([^"]*\)".*/\1/p'
}

wait_for() {
  for _ in $(seq 1 30); do
    [ "$(running_version)" = "$1" ] && return 0
    sleep 1
  done
  return 1
}

install_binary() {
  cp "$1" "$dir/agentic-ai-workspace.new"
  chmod 755 "$dir/agentic-ai-workspace.new"
  mv -f "$dir/agentic-ai-workspace.new" "$dir/agentic-ai-workspace"
}

mkdir -p "$dir"

if ! docker container inspect "$name" >/dev/null 2>&1; then
  echo "first deploy: creating $name from $image"
  install_binary "$bin"
  docker run -d --name "$name" --restart unless-stopped -p "$port:8080" -e PORT=8080 \
    -v "$dir:/opt/app:ro" -v "$name-data:/data" --entrypoint /opt/app/agentic-ai-workspace "$image"
  wait_for "$version" || { echo "new container did not report $version" >&2; docker logs --tail 50 "$name" >&2; exit 1; }
  curl -sSi "$url/"  | grep -i '^x-app-version'
  exit 0
fi

echo "before swap:"
curl -sSi "$url/" | grep -i '^x-app-version' || true
curl -sS "$url/api/health" || true
echo

cp "$dir/agentic-ai-workspace" "$dir/agentic-ai-workspace.prev"
old="$(running_version || true)"
install_binary "$bin"

start="$(date +%s)"
docker restart -t 5 "$name" >/dev/null
if wait_for "$version"; then
  echo "after swap (back in $(( $(date +%s) - start ))s, no image rebuild):"
  curl -sSi "$url/" | grep -i '^x-app-version'
  curl -sS "$url/api/health"
  echo
  exit 0
fi

echo "new binary did not report $version; rolling back to ${old:-the previous binary}" >&2
docker logs --tail 50 "$name" >&2 || true
mv -f "$dir/agentic-ai-workspace.prev" "$dir/agentic-ai-workspace"
docker restart -t 5 "$name" >/dev/null
wait_for "$old" && echo "rolled back to $old" >&2
exit 1
