#!/usr/bin/env bash
# Pull ONLY verified GHCR app and sandbox images. Do not build on the host.
set -euo pipefail
channel="${1:-dev}"
case "$channel" in
  dev|latest|release) ;;
  *) echo "Usage: bash scripts/pull-ghcr-pair.sh [dev|latest|release]" >&2; exit 2 ;;
esac
image=ghcr.io/farismnrr/agentic-ai-workspace
app="$image:$channel"
sandbox="$image-sandbox:$channel"
command -v docker >/dev/null || { echo "BLOCKED: Docker CLI missing." >&2; exit 1; }
docker info >/dev/null 2>&1 || { echo "BLOCKED: Docker host unavailable." >&2; exit 1; }
docker pull "$app" || { echo "BLOCKED: GHCR app image unavailable." >&2; exit 1; }
docker pull "$sandbox" || { echo "BLOCKED: GHCR sandbox image unavailable." >&2; exit 1; }
app_rev=$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$app")
sandbox_rev=$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$sandbox")
if [[ ! "$app_rev" =~ ^[a-f0-9]{40}$ || "$app_rev" != "$sandbox_rev" ]]; then
  echo "BLOCKED: app/sandbox GHCR revision mismatch (or missing labels)." >&2
  echo "App: ${app_rev:-missing}; sandbox: ${sandbox_rev:-missing}" >&2
  echo "Wait for the GitHub Actions image publication to finish, then retry." >&2
  exit 1
fi
echo "PASS matching app + sandbox GHCR images ($channel): $app_rev"
echo "No container was built or restarted."
