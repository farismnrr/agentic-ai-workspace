# Orchestrator Space (external AI)

Orchestrator Space lets ChatGPT, Claude, or another connected AI app lead work in Masih Awam and delegate directly to your **internal Agent Team workers**.

It is a separate connector. Existing Coding Space, for both internal and external use, stays unchanged.

## Connect

1. Open **Settings → Spaces → Orchestrator**.
2. Turn on only the tools you want to allow. Actions that create tasks, run agents, or write files are off by default.
3. In your external AI app, add a remote MCP connector at `<CONNECTOR_URL>/api/orchestrator`.
4. Sign in and approve the OAuth scopes you want the client to use.

A Coding Space token cannot access Orchestrator Space, and an Orchestrator token cannot access Coding Space.

## How it works

1. The external AI calls `list_projects` and `list_agents` to choose an existing project and available workers.
2. It creates **one primary task** for the issue, then writes a plan or research as needed.
3. It calls `delegate_agent` to assign work directly to an internal Researcher, Planner, Developer, QA, or another enabled worker.
4. Internal workers run with their configured models, tools, sandbox, and role restrictions. The external AI remains the lead.
5. The AI reads `get_delegation` and the task activity before deciding which worker runs next.

A worker delegation adds activity to the same task, not a new internal-led task. Research and PRD documents appear on the task. By default a person approves a submitted PRD in the task UI. The owner can separately enable **Approve plans** in Settings → Spaces → Orchestrator, then reconnect the client to grant the additional `orchestrator:approve` OAuth scope. Only then can an external lead call `approve_plan` with the pending `event_id`, `task_id` and unique `idempotency_key`; the server checks that the approval belongs to the same owner, connected client and latest PRD/plan revision. A stale or already answered event cannot approve a different plan. Coding Space approval permissions do not grant this capability. Security requires the owner-enabled deep-review mode.

Workers may run in parallel when their roles allow it, but file-changing workers are serialized to prevent conflicting edits.

## Files

For text or code, the AI can call `save_file` with the project, primary task ID, relative path, and content. File writes cannot bypass pending plan approval or overwrite an existing file without the expected hash.

For binary files, the connector offers a resumable sequence: `start_upload`, `upload_chunk`, `upload_status`, and `finish_upload`. The upload is checked against its SHA-256, project quota, and file-type restrictions. Maximum binary size is 512 MiB; each binary chunk is at most 512 KiB before Base64 encoding.

Binary transfer requires the connected AI app to provide the actual bytes to the tool. Some apps cannot access their generated attachments this way; direct binary transfer may then be unavailable. A separate browser-upload handoff is not yet provided.

## Important limits

- This Space cannot create projects.
- No internal orchestrator takes over the task: internal workers report directly to the external lead.
- Mutating tools and delegation must be explicitly enabled.
- Workers needing Docker may not run until a Docker-enabled host is available.
- Remote app behavior depends on the client and the configured internal models.
- File operation and job activity are visible in the project's existing UI.

The new Space is independent of the existing Coding connector. For Coding Space instructions, see [Connecting AI apps](connector.md).
