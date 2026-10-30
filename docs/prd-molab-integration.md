# PRD — Ephemeral MoLab pairing

**Status:** Implemented in app/UI; local tests pass. **Live MoLab/Docker verification pending.**

## Goal

A person gives **one AI Code task** a temporary MoLab notebook HTTPS URL and bearer token through a private form, not through the chat transcript. The app holds credentials only in RAM. The user never has to configure a permanent Project Env secret. Pairing requires an HTTPS application public URL and enabled task-scoped egress grants.

## Delivered

- Task page provides **MoLab notebook → Connect** with a password input, only when per-task grants are active. The form submits to authenticated, same-origin POST `/api/tasks/:id/molab`. GET returns only connection metadata (no token), and DELETE revokes this pairing. Responses are non-cacheable.
- Only the signed-in owner of that task can view or alter its pairing. URL must be `https://sb-*.sb.molab.run/` with no custom path, query, userinfo or non-HTTPS port.
- The connection lives solely in app process memory (30-minute TTL), scoped to user + project + internal task, with a small global cap. External recorded work stays outside this credential flow; pair on its linked internal task instead. Process restart loses all pairings. Disconnect, replacement, task finish/failure/archive/deletion and TTL cancel active operations and clear the pairing.
- Execution-enabled **internal agents** use `marimo_execute({code})`. No token or URL belongs in tool input. The app looks up the trusted task scope, runs the pinned marimo-pair script once in the isolated sandbox with the one-shot `MARIMO_TOKEN`, and never includes the token in Project Env, the transcript or the tool result. The first tool call performs required `marimo._code_mode` inspection.
- The dedicated sandbox command does not receive project variables, Git credentials or extra project gateway/database privileges. The task's existing approved HTTPS egress proxy controls MoLab access. `EGRESS_GRANTS` stays off by default until the live security gate passes.
- Security tests cover expiry, restart, user/project/task boundaries, bad URLs, CSRF/auth, token-free HTTP responses, disable-by-default and disconnect/replacement cancelling active work.

## Still blocked / deferred

1. **GHCR Docker live smoke:** Once Fast GitHub Actions publishes the paired app and sandbox images, on the actual Docker host pull the matching images and run `bash test/smoke/marimo-sandbox.test.sh ghcr.io/farismnrr/agentic-ai-workspace-sandbox:dev`. No host-local image build.
2. **Authenticated live MoLab connection:** On a disposable staging app with Egress Grants enabled and an approved per-task hostname, enter a *new* token in the form. Verify first inspection, code execution, user-visible toast only when asked, failure/revocation, TLS, reconnect and idle more than five minutes. Never paste the token into agent chat or CI.
3. **Vendor CLI limitation:** Bundled Claude/Codex skill docs are discoverable, but vendor subprocesses do **not** inherit these RAM-only credentials. The `marimo_execute` tool works for the internal task agent only. A credential-free vendor CLI-to-app session bridge needs its own controlled authorization design before claiming vendor support.

Do not turn on unrestricted host network or mount the host Docker socket inside sandboxes. Do not claim live readiness before these gates pass. If the MoLab token previously posted in chat has not expired, rotate it.

## Acceptance

- Pair/re-pair/disconnect work through authenticated app endpoints without writing credentials to database, logs or task transcripts.
- A task cannot obtain another task's pairing; a completed task can no longer use its token. Active network calls stop on disconnect/replacement.
- Task agent can run the pinned skill only with a matching task-scoped egress grant; no agent can approve its own grant.
- Real MoLab smoke confirms authenticated inspection and code execution without exposing the token.
- GHCR publishes both app and sandbox images at the same commit; the deployment host only pulls verified images.
