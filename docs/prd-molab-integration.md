# PRD mini — MoLab from chat

**Status:** Implemented in code; live E2E not yet tested.

User supplies the active MoLab HTTPS URL and access code together in an ordinary chat. The internal agent calls marimo_execute with Python code; the app finds both values in the same latest matching user message. No dedicated MoLab task, form, memory pairing store or manual EGRESS_GRANTS approval.

Only the exact sb-*.sb.molab.run HTTPS host is allowed through the existing public-only proxy.
The script receives the token in its transient environment, never in Docker command arguments or Project Env.
Chat messages remain in task history. MoLab owns expiry and revocation; connection errors request fresh credentials.

Tests: parser security checks, pinned marimo-pair mock HTTP smoke, Go build, frontend verify, GHCR app/sandbox publish. Live notebook toast + termination and invalid-token behavior remain to be verified on a real notebook.


## ChatGPT/Gemini connector

The AI Code MCP server also exports `marimo_execute`. Users provide the active MoLab URL and token in
ChatGPT/Gemini; the assistant sends `url`, `token`, `code` and the active `work_id` to the authenticated tool.
Start or resume one tracked work session with the enabled Developer role using `begin_work(role=developer)`,
or switch the current session using `set_role(role=developer)`. Repeated calls reuse the same task,
while each execution still uses the validated one-shot runner.

The connector requires the `coding:exec` OAuth scope and the normal `run_command` switch.
Activity records only a fixed tool title and success/failure, never the token or Python arguments.
The token exists in the chat and transient MCP request, not Docker argv, Project Env or activity logs.
A new GHCR deployment and connector refresh are required before live use.
