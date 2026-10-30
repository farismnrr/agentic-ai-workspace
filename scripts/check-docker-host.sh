#!/usr/bin/env bash
# Read-only readiness check of the host Docker daemon and the existing app bridge.
# Usage: bash scripts/check-docker-host.sh [dev|latest|prod]
# Optional: COMPOSE_ENV_FILE=/path/to/.env for prod or custom deployments.
set -euo pipefail

channel="${1:-dev}"
case "$channel" in
  dev|latest|prod) ;;
  *) echo "Usage: bash scripts/check-docker-host.sh [dev|latest|prod]" >&2; exit 2 ;;
esac

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
compose="$root/compose.${channel}.yaml"
fail() { printf 'BLOCKED %s\n' "$*" >&2; exit 1; }

command -v docker >/dev/null 2>&1 || fail "Docker CLI missing on this host. Install Docker Engine and Compose, then rerun."
docker info --format '{{.ServerVersion}}' >/dev/null 2>&1 || fail "Cannot connect to Docker daemon. On the host run: docker info"
printf 'PASS Docker host daemon reachable\n'

[[ -S /var/run/docker.sock ]] || fail "Expected /var/run/docker.sock is absent. Check host Docker setup."
gid=$(stat -c %g /var/run/docker.sock)
export DOCKER_GID="$gid"
printf 'PASS Docker socket exists (group ID %s)\n' "$gid"

args=(-f "$compose")
if [[ -n "${COMPOSE_ENV_FILE:-}" ]]; then
  [[ -f "$COMPOSE_ENV_FILE" ]] || fail "COMPOSE_ENV_FILE does not exist."
  args=(--env-file "$COMPOSE_ENV_FILE" "${args[@]}")
fi
docker compose "${args[@]}" config --quiet >/dev/null 2>&1 || fail "Compose configuration cannot be resolved. Set required env vars (for prod, COMPOSE_ENV_FILE) and run: docker compose -f compose.${channel}.yaml config"
printf 'PASS Compose configuration parsed\n'

app=$(docker compose "${args[@]}" ps -q app 2>/dev/null)
[[ -n "$app" ]] || fail "App container is not running for ${channel}. Run docker compose -f compose.${channel}.yaml ps"
printf 'PASS App container running\n'

socket_source=$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/var/run/docker.sock"}}{{.Source}}{{end}}{{end}}' "$app")
[[ "$socket_source" == "/var/run/docker.sock" ]] || fail "App container does not mount the host Docker socket. Inspect its Compose deployment."
groups=$(docker inspect --format '{{json .HostConfig.GroupAdd}}' "$app")
[[ "$groups" == *\"$gid\"* ]] || fail "App container is missing the Docker socket group ID. Set DOCKER_GID and recreate only the app service."
printf 'PASS App has host Docker socket and matching supplementary group\n'

# The app is trusted and brokers all sandbox execution. It must never pass the
# Docker socket to arbitrary agent containers. This check doesn't create any.
sandbox_id=$(docker container ls --filter 'name=^/agentic-sandbox$' --format '{{.ID}}' | head -n 1)
if [[ -n "$sandbox_id" ]]; then
  agent_socket=$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/var/run/docker.sock"}}found{{end}}{{end}}' "$sandbox_id")
  [[ -z "$agent_socket" ]] || fail "Agent sandbox unexpectedly has direct Docker socket access."
  net=$(docker inspect --format '{{.HostConfig.NetworkMode}}' "$sandbox_id")
  [[ "$net" == none ]] || fail "Agent sandbox network mode is ${net}, expected none."
  printf 'PASS Agent sandbox has no Docker socket and no network\n'
else
  printf 'INFO No standing agentic-sandbox container found; isolation check requires a running sandbox\n'
fi

printf 'PASS Host-side Docker bridge looks ready. This does not execute a sandbox job.\n'
