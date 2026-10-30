---
name: marimo-pair
description: Pair with a running marimo/MoLab notebook using a temporary, task-scoped connection.
---

Only use this when the user wants to work with a marimo notebook. **Never request or accept a MoLab bearer token in chat.** It must go through the secure "Connect" form attached to this task.

1. To read marimo documentation, use the pinned bundled skill at `/usr/local/share/marimo-pair/SKILL.md` in the sandbox. Do not run `npx skills add` at runtime.
2. If there is no active pairing, tell the user to open this task's **MoLab notebook → Connect** form and submit the temporary notebook HTTPS URL and token. Neither is persisted to the task transcript or Project Env.
3. Request access to the notebook's *exact* hostname via `request_egress` and wait for owner/admin approval. Never use a global allowlist, direct host networking or bypass the proxy.
4. Call `marimo_execute` with only `{"code":"<Python>"}`. **Never pass a token, URL or secret in arguments.** The app obtains the per-task token from volatile memory and executes the pinned script in a one-shot isolated sandbox. The first call always inspects `marimo._code_mode`; call it again for actual task code after the inspection succeeds.
5. For durable notebook edits use `marimo._code_mode` API and never edit the live notebook source directly. Confirm success from the returned tool result, not from assumptions.
6. If pairing expires, task ends, remote notebook disconnects or permission is denied, stop and request new pairing; do not fabricate success.

Native vendor CLI subprocesses can read the vendored documentation, but do not inherit these ephemeral credentials. The `marimo_execute` operation is available only to the internal execution-enabled agent using the authenticated task tool path.
