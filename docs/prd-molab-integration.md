# PRD — Secure MoLab / marimo-pair agent integration

**Status:** Phase 1 implemented and locally verified; live activation BLOCKED pending Docker host testing and task-scoped token delivery. **Target:** `agentic-ai-workspace` / `dev`. **Scope:** official marimo-pair skill, internal agents and vendor CLIs, sandbox egress, Compose, secrets and black-box verification.

**Implemented:** Pinned Apache-2.0 upstream skill and scripts in sandbox image, embedded Developer playbook, Codex/Claude skill discovery from private container HOME, CLI token argument rejection, TLS-only remote connections and token header passed via a private temporary file. Local HTTP mock verified session discovery, authorized SSE execution, and no inherited token in curl env/argv.

**Not implemented:** Task-scoped encrypted MoLab credential storage/injection, live MoLab authenticated integration, Docker image build on real host, dynamic grant production activation. Project Env cannot be used as a substitute: the existing command runner injects project secrets into every project command. Keep `EGRESS_GRANTS=false` until all security tests pass.

**GHCR host smoke (no local build, no MoLab token):**

Wait until the `Fast - publish dev image` GitHub Actions workflow succeeds. It publishes the matched app + sandbox pair before promoting `:dev`. From the Docker host, run:

```bash
git pull --ff-only origin dev
bash scripts/pull-ghcr-pair.sh dev
bash test/smoke/marimo-sandbox.test.sh ghcr.io/farismnrr/agentic-ai-workspace-sandbox:dev
```

Do not build release images locally. Once the GHCR pair and sandbox smoke pass, finish encrypted per-task credentials and live TLS/revoke/expiry/SSE checks before enabling Egress Grants. Rotate any notebook token previously shared in chat.

## Objective
Let an AI Code task pair with a user-provided running marimo notebook hosted on MoLab **from the task sandbox**, not from app browser, without expanding network privileges for unrelated tasks. Preserve existing app-proxy design and secure approvals.

## Current evidence
- `compose.dev.yaml` and `compose.latest.yaml` already forward `EGRESS_GRANTS` (default false) and `SANDBOX_EGRESS_HOSTS` (defaults remain unchanged).
- `internal/modules/egressgrant` already supports requests, per-task project-scoped approval/revoke/expiry and audit; request_egress may be called by eligible agent roles but cannot approve itself.
- `internal/platform/egressproxy` already supplies sandbox proxy sockets and restrictive TLS/SNI matching; socket/path isolation implemented. `EGRESS_GRANTS` rollout stays disabled by default until live marimo tests pass.
- Internal `use_skill` loads *bundled* `internal/modules/agentteam/domain/skills/<role>/<skill>/SKILL.md`. Installing an upstream skill via `npx` in one process does not automatically register the skill with internal AI Code roles.
- Official skill source: https://github.com/marimo-team/marimo-pair, `skills/marimo-pair/SKILL.md`; depends on bash/curl/jq and bundled `scripts/execute-code.sh`. Authentication prefers `MARIMO_TOKEN` env, not `--token`. `ctx.edit_cell` is used for notebook persistence, rather than direct file editing.
- The example notebook hostname is user-specific/ephemeral, so no hardcoded MoLab host or token must enter code, documentation, CI or static allowlist.

## Design
1. **App:** user chooses task + notebook URL. Existing owner/admin egress approval UI authorizes exact public HTTPS hostname on port 443, with existing TTL and task revocation; do not grant global `*.molab.run`. Notebook URL is data, not an authority to bypass network checks.
2. **Sandbox:** eligible task developer/agent runs official `marimo-pair` scripts through task-scoped socket. Preserve proxy env and network isolation. No host network, no Docker socket in sandbox.
3. **Skills:** bundle a license-compliant, pinned **audited** official marimo-pair skill and scripts in repo or implement a vetted opt-in skill distribution path. Must be discoverable by internal `use_skill` and vendor CLI agents; avoid runtime unpinned `npx skills add` from agent prompts as the primary supply chain.
4. **Token:** store notebook token in secret store with explicit project/task binding, never in chat prompts, command flags, process list, console logs or audit. Supply as `MARIMO_TOKEN` only at isolated execution boundary; rotate any previously pasted token. Never persist token in project Variables, env examples or screenshot logs.
5. **Compose/deployment:** keep current `EGRESS_GRANTS=false` as default. Only add necessary documented opt-in wiring; no `network_mode: host`, no wildcard MoLab static egress allowlist. Preserve baseline dependency registry hosts if `SANDBOX_EGRESS_HOSTS` is set.
6. **Connection:** validate HTTPS URL and exact hostname, reject SSRF/redirection/unauthorized endpoints via existing proxy. Skill must not use `curl --noproxy` or network bypass. Long-lived WebSockets must use task egress proxy.
7. **Feedback:** user-visible task event reports connection, permission waiting, authenticated/error state, and disconnect; report success only after actual authenticated notebook command.
8. **Operational safety:** activation in production requires human consent, host-specific smoke tests, rollback plan, and proved live TLS/WebSocket/revoke/expiry/task-end behavior.

## Acceptance criteria
- Internal agent role and supported Codex/Claude CLI workflows can discover and run vetted `marimo-pair` scripts from sandbox.
- Static host allowlist unchanged; unrelated task/proj cannot connect to approved notebook; agent cannot self-approve.
- A genuine MoLab notebook can be inspected (read only first), then authorized live code execution succeeds and user-visible toast is generated only with explicit user intent; token never appears in process args/logs.
- Invalid token, expired session, wrong hostname, network blocked, revoked grant, task-end and restart are safe and explainable.
- WebSocket idle > 5 min, TLS ClientHello (real curl/Python/WebSocket variants), reconnect, revoked mid-session, expiry, SNI/DNS and sandbox Subpath are tested against real notebook and actual Docker-enabled host before enabling dynamic grants.
- Black-box tests for opt-in/off-by-default, skill discovery/scripts, task E2E, permissions and secret redaction. No unnecessary white-box full CI.
- After verified changes, commit only intended scoped files, preserve other work and normal fast-forward push; clean Git. **If missing host access, live notebook/token, or ability to audit upstream scripts prevents verification, STOP and report as blocker.**

## Non-goals
- No browser UI marimo embedding, host network access, wildcard static allowlist, grant activation in production without human approval, permission escalation or new workflow tasks per delegated agent.
