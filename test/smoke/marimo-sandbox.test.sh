#!/usr/bin/env bash
# Read-only container smoke: the pinned marimo-pair skill must be installed and
# discoverable by both vendor CLI conventions without Docker-in-Docker.
set -euo pipefail
image="${1:-ghcr.io/farismnrr/agentic-ai-workspace-sandbox:latest}"
docker info >/dev/null
docker image inspect "$image" >/dev/null
docker run --rm --network none --read-only --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --tmpfs /tmp:rw,exec,nosuid,nodev,size=64m \
  --user 65532:65532 --env HOME=/tmp \
  --entrypoint /usr/local/bin/sandbox-entry "$image" sh -ec '
    root=/usr/local/share/marimo-pair
    test -f "$root/SKILL.md"
    test -f "$root/LICENSE"
    test -f "$root/scripts/execute-code.sh"
    test -L /tmp/.agents/skills/marimo-pair
    test -L /tmp/.claude/skills/marimo-pair
    test "$(readlink /tmp/.agents/skills/marimo-pair)" = "$root"
    test "$(readlink /tmp/.claude/skills/marimo-pair)" = "$root"
    bash -n "$root/scripts/execute-code.sh"
    bash -n "$root/scripts/discover-servers.sh"
    test ! -S /var/run/docker.sock
    echo "PASS bundled marimo-pair and CLI discovery; no Docker socket"
  '
