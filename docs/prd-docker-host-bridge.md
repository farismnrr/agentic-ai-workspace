# PRD mini — Safe Docker Host Bridge (No Docker-in-Docker)
**Status:** Approved direction by owner; implementation limited to read-only readiness + reuse current mediated command runner. **Repo:** agentic-ai-workspace, dev. **Why:** distinguish an unavailable coding sandbox daemon from an actual deployment host failure; agents must not get root-equivalent Docker credentials.

## Ground truth
- Compose dev/latest/prod ALREADY mounts host `/var/run/docker.sock` into the **trusted app** and passes `DOCKER_GID`.
- `internal/platform/sandbox/docker.go` already talks to the host daemon using the Moby Go Engine API; `internal/bootstrap/sandbox.go` creates an engine and `run_command` runs inside unprivileged, network-none sandbox containers launched by the app.
- No Docker-in-Docker required. Sandbox intentionally has no Docker socket, host network or privileged flag.
- The remote coding-agent sandbox used by ChatGPT is an *independent environment*; it cannot gain the actual Docker host just by changing deployed Compose. Docker-enabled tests must be executed on the actual host.

## Minimal product change (v1)
1. Add a small read-only `docker_host_status` capability for internal exec-enabled agent roles. The **app**, not sandbox, pings its configured Docker endpoint with a short timeout through the existing Go SDK. Return only reachable/unreachable and a short operator action; never expose raw socket, arbitrary Docker endpoints, inspect other users' containers or change host resources.
2. Document the existing path: `Agent → approved app tool → app (Docker SDK) → host daemon → isolated sandbox`. Keep `run_command` / `run_task` as the only way for an agent to execute untrusted code; do not proxy arbitrary docker CLI/Engine API calls.
3. Supply `scripts/check-docker-host.sh`, a **read-only** operator command to verify daemon, compose socket mount/group and sandbox image. The user runs it on their Docker-enabled host when requested.
4. Compose files retain existing mount/group settings and no added access for sandboxes; clarify no-DinD boundaries in comments/docs. No Docker TCP port, `network_mode: host` for sandbox, privileged agent, user added to docker group, or other root-equivalent permissions.

## Acceptance
- Exec-enabled internal agent can request non-mutating Docker host status; roles without exec do not receive the tool. No model-provided parameters are accepted.
- Status fails quickly with a readable cause if daemon/socket inaccessible, not a false `ready` claim.
- Host status probe has no side effects and doesn't pull/build/start/remove images or containers.
- Existing `run_command` still launches sandbox via app's host Docker SDK, with `NetworkMode=none` and no Docker socket mount. Group permission and host setup verified via operator script.
- `go test` targeted compilation, script `bash -n`, and frontend unaffected; Docker host live test remains pending until the operator runs a command on actual host.
- No automatic rollback/host modifications. A future ability to run arbitrary host Docker commands would require a separate, human-approved least-privilege operation protocol.

## Host operator handoff (only when needed)
Run from repo root on the Docker-enabled host:
```bash
bash scripts/check-docker-host.sh dev
```
If it reports permission/mount issues, return sanitized output; the agent provides targeted human-run remediation commands. No secrets or Docker socket grants to sandbox.
