# # Implementasi External Multi-Role Workspace

Task: fae3e8b731d9ae374acd8b2a70ea8ca1
Status: done
Updated: 2026-10-27T03:31:01Z

Reference material; confirm applicability before reusing.

**Summary:** Existing task, event, role-catalog, MCP permission, and document-export pieces provide clear places to extend external-work tracking.  
The main gaps are 30-minute task grouping, missing role-transition semantics, and external MCP activity not carrying file-change and undo data into task events.

## Questions checked

- How are connected-app calls currently grouped and persisted?
- Which existing catalogs, events, document exporters, recorders, and UI views can be reused?
- Where do new MCP tools fit, and what security and retry boundaries apply?
- What is the current worktree and `origin/dev` state?

## Observed facts

- **External task grouping:** `external.go` uses a 30-minute quiet window and selects the latest task by user/project, then reuses it if the source matches. Otherwise it creates a task already marked `done` with an empty plan (`internal/modules/task/application/external.go:11–13, 47–74`). `Begin` and `End` temporarily switch the task to `working` while calls are in flight (`:84–119`). This grouping can merge separate work from the same client; it does not use a client-provided idempotency key.
- **Identity and permission checks:** MCP calls get user and client identity from verified token information, while the requested project is resolved for that user (`internal/modules/coding/adapter/mcp/guard.go:59–80`; `internal/bootstrap/connectedapps.go:28–83`). The common guard checks the tool switch, then token scope, then runs the tool (`guard.go:53–115`). The recorder receives the resolved project, task, and undo blob (`internal/bootstrap/bridges.go:109–124`).
- **Role source of truth:** Built-in roles are defined in `internal/modules/agentteam/domain/preset.go:12–112`; `Presets()` returns that catalog (`:124–139`). Roles have access groups and built-in role metadata (`role.go:43–82, 118–135`). This is reusable role metadata, but its access is for the internal teammate runner; MCP tool permissions are enforced separately by the coding guard.
- **Events and task data:** Existing event kinds include `EventHandoff`, `EventResearch`, `EventPlan`, and `EventPRD` (`internal/modules/task/domain/event.go:8–31`). `EventHandoff` means work passed to or returned from a teammate (`:18–19`). `PlanStep` already has stable IDs and pending/active/done/failed states (`task/domain/task.go:24–41`). Tasks store Plan, PRD, Research, status, and source (`:67–103`).
- **Internal plan/document behavior:** Internal plan updates save the plan, append an `EventPlan`, and export project notes (`task/application/agentcalls.go:70–88`; `planrecord.go:12–30`). Research and PRD have separate persistence and event paths (`teamplan.go:46–82, 91–115`). The exporter writes current plan/PRD/research and distinct research/PRD event files into `project-notes/` (`notefiles.go:77–135`).
- **Team and Changes views:** `teamJobs` opens a job only for an `assigned` handoff, counts events carrying that teammate’s name, and closes on a later handoff (`web/src/features/workspace/logic/team.ts:37–59`). Team UI displays those derived jobs (`task/panel/TeamTab.svelte:14–46`). Therefore, a role change by one external assistant should not be represented as a teammate handoff. The Changes tab consumes file-change data from task events and offers Undo when metadata is present (`ChangesTab.svelte:23–62`).
- **External mutation recording gap:** MCP file tools produce an encoded undo description (`coding/adapter/mcp/tools_write.go:54–70`), and the MCP guard stores it in the activity recorder entry (`guard_thread.go:33–46`). The bridge wraps project plus change data for the activity log’s undo path (`bootstrap/bridges.go:114–145`). But task-thread notes currently carry only kind, text, technical detail, and optional run information (`bootstrap/threads.go:59–77`; `coding/adapter/mcp/ports.go:59–74`). Unlike internal task action events, which include `Files` and `Undo` (`task/application/reporting.go:19–34`), external task events do not currently get those fields.
- **MCP setup hook:** The shared server is created in `internal/bootstrap/coding.go:41–49`. `NewServer` sets `ServerOptions.Instructions` and registers tool groups (`coding/adapter/mcp/server.go:74–96`). New tool handlers can follow the typed `sdk.AddTool` registration pattern and be placed in the MCP adapter; task application behavior is currently passed through `Options.Tasks` as `TaskLog` (`server.go:25–32`; `ports.go:67–75`).
- **Current thread persistence:** `AppendEvent` writes event body and undo bytes into `task_events` (`task/adapter/sqlite/events.go:16–46`). Event append and task field saves are separate operations in the observed interfaces; atomic save-plus-event behavior is not provided by these methods.

## Hook map and gaps

The likely extension points are the MCP tool registrations and server instructions in `coding/adapter/mcp/`, the task-facing application service behind the MCP adapter, and dependency wiring in `bootstrap/coding.go` / `bootstrap/threads.go`. The existing `TaskLog` bridge can continue recording ordinary legacy calls.

The implementation will need a distinct durable work/session identifier instead of relying on the current source-plus-30-minute lookup. New role changes need an event whose meaning is “one external assistant changed its working role,” not “a teammate received work.” Role labels must remain descriptive: the existing MCP permission gate should remain authoritative.

To make Changes and evidence reflect external work, the task-event path will need the resolved file paths, actual run result, and reversible metadata, coordinated with the existing activity-log undo path to avoid duplicate or inconsistent undo records. Research and PRD can reuse task fields, event types, and project-note export. Plan persistence can reuse `Task.Plan` and `SavePlan`, but the observed implementation does not provide stable-ID reconciliation or idempotent retries for external plan updates.

## Technical references

- **MCP protocol:** The current [MCP Tools specification](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) defines typed tool schemas, tool calls, structured outputs, and advises deterministic tool ordering. The [versioning specification](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning) describes the 2026 stateless protocol and explicitly identifies server-minted handles passed as ordinary tool arguments for cross-call state.
- **Go SDK:** The repository pins `github.com/modelcontextprotocol/go-sdk v1.8.0` (`go.mod:12`) and Go `1.27.2` (`go.mod:3`). The official [SDK compatibility table](https://github.com/modelcontextprotocol/go-sdk) says v1.7.0+ supports protocol `2026-07-28` and earlier listed revisions. Its [quick start](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/quick_start.md) shows typed `AddTool` handlers. Current SDK release notes say v1.8.0 was released September 14, 2026.
- **Instructions are guidance:** The official [MCP server-instructions article](https://blog.modelcontextprotocol.io/posts/2025-11-03-using-server-instructions/) describes instructions as a user manual for server tools, not a way to change model behavior. Therefore instructions can explain the workflow but cannot prove a client used it.
- **OAuth and security:** The [MCP authorization spec](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization) requires the MCP resource server to validate tokens specifically intended for itself, use bearer headers, and enforce least-privilege scopes. [RFC 9700](https://datatracker.ietf.org/doc/rfc9700/) recommends privilege restriction and audience restriction, among other protections. The SDK’s listed [security advisories](https://github.com/modelcontextprotocol/go-sdk/security/advisories) include high-severity issues fixed by v1.4.1 or earlier; the pinned v1.8.0 is above those patched-version cutoffs. This is a version comparison, not a vulnerability scan.
- **Idempotency:** MCP tool calls do not establish application-level idempotency. AWS’s [retry-safety guidance](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) recommends a caller-provided request identifier, returning a semantically equivalent response on retry, rejecting reuse with changed parameters, and atomically persisting the identifier with mutations. That pattern fits `begin_work` and mutation-style updates better than time-window grouping.

## Quality checks

Acceptance checks should cover separate works from the same client, multiple concurrent works and projects, reconnect/restart, idempotency-key retries and mismatches, stable plan-step IDs, separate Research/PRD export, and truthful blocked/failed/completed states.

Security checks should cover invalid work IDs, wrong owner or project, token audience and scope, tool switches, role changes that do not expand permissions, oversized inputs, concurrent updates, and secret handling. Evidence checks should distinguish self-reported progress from server-observed file changes and actual command exit codes.

UI checks should verify Plan, Documents, Team, Changes, Files, and timeline for both external and internal tasks. Repository frontend guidance identifies 390, 820, and 1024 px as useful responsive widths (`agents/memory-frontend.md:19`); this research did not run UI or browser checks.

## Worktree situation

Read-only git inspection found:

- Primary `/workspace` is on `dev` at `fff7a2b`; it reports `behind 6` relative to `origin/dev`.
- `git ls-remote` reports `origin/dev` at `ebc4751b2cf8ff213d6480492d9ad09ece6109fc`.
- Additional detached worktrees are `/workspace/.stage7` at `ebc4751` and `/tmp/agentic-stage7.4XDEQd` at `fff7a2b` (listed as prunable).
- The primary worktree has untracked `.stage7/`, `masih-awam-reference.zip`, several `project-notes/tasks/` directories, and `reference-bundle/`.

This does not match the attached spec’s note to have only a clean primary worktree. I made no changes and did not clean or remove any worktree or untracked content.

## Not found

I did not verify live ChatGPT or Gemini client behavior, including whether either reliably follows server instructions or preserves the proposed work ID across reconnects. No tests, vulnerability scan, or live MCP calls were run.

## Suggestions

Keep the feature on the existing task and MCP infrastructure. Make `work_id` explicit and server-issued, keep role as attribution only, and preserve legacy calls without the new ID. Before implementation, the Planner/Architect should account for the observed unclean worktree and six-commit remote gap without deleting or rewriting user data.

---

**Summary:** The repository uses typed MCP registration and a pure-Go SQLite driver; the PRD’s proposed work ID fits the current MCP model for carrying state across calls.  
**Read-only research:** No files changed, and I ran no tests or application checks.

## Technical references

### MCP typed tools and structured output

- **Observed:** `go.mod` pins `github.com/modelcontextprotocol/go-sdk v1.8.0`. The project’s existing registration hook is `internal/modules/coding/adapter/mcp/server.go`; the earlier research notes identify typed `sdk.AddTool` handlers as the existing pattern.
- **Official guidance:** The [Go SDK server guide](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/server.md) says `mcp.AddTool` can infer input and output schemas from Go types, validate input and output, and populate both `StructuredContent` and `Content`. The [v1.8.0 release notes](https://github.com/modelcontextprotocol/go-sdk/releases) say this release supports protocol revision `2026-07-28` and earlier supported revisions. Confirm behavior against the pinned version when implementing.
- **Work IDs:** The MCP [protocol change notes](https://modelcontextprotocol.io/specification/draft/changelog) describe server-minted handles passed as ordinary tool arguments for cross-call state. This supports the PRD’s explicit `work_id` approach; it does not establish whether a particular client preserves the ID.

### SQLite transactions and optimistic locking

- **Observed:** `go.mod` pins `modernc.org/sqlite v1.59.0`; `internal/platform/database/database.go` registers that driver. Existing SQLite adapters use `RowsAffected()` to detect whether conditional updates changed a row, for example `internal/modules/egressgrant/adapter/sqlite/decide.go`.
- **Official guidance:** The driver is a [CGo-free SQLite driver](https://pkg.go.dev/modernc.org/sqlite). Go’s [transaction guide](https://go.dev/doc/database/execute-transactions) says to group related writes in `sql.Tx`, committing them together or rolling them back, and to use `Tx` methods throughout the transaction.
- **Concurrency:** SQLite documents that `BEGIN IMMEDIATE` acquires the write transaction at its start; it can fail with `SQLITE_BUSY` if another writer is active. See [SQLite transactions](https://www.sqlite.org/lang_transaction.html) and [isolation](https://sqlite.org/isolation.html). The project’s exact transaction mode and busy handling were not established here.
- **Optimistic locking:** The cited SQLite docs describe transaction and update behavior, not a built-in optimistic-lock feature. A common application pattern is a revision column with a conditional update (`WHERE id = ? AND revision = ?`), incrementing the revision and treating zero affected rows as a conflict. This is a design suggestion, not an existing project convention. Put the work mutation, event append, and idempotency record in one transaction if they must succeed or fail together.

### Detached worktrees and secret redaction

- **Worktree guidance:** Git’s [worktree documentation](https://git-scm.com/docs/git-worktree.html) describes `git worktree add --detach <path> <commit-ish>` for a linked worktree with detached `HEAD`. For the PRD’s target, the corresponding guidance is `git worktree add --detach <outside-path> origin/dev` after confirming `origin/dev` is present. This creates a linked worktree sharing repository metadata; it does not make that worktree inherently read-only. No worktree command was run.
- **Redaction:** OWASP’s [Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) says access tokens, passwords, database connection strings, encryption keys, and other secrets should generally be removed, masked, sanitized, hashed, or encrypted before logging. In this project, `internal/modules/egressgrant/adapter/sqlite/audit_test.go` already checks that a sentinel secret is absent from audit entries. For external-work events, test that secret-like values in tool inputs and errors do not leak into task events, exports, or logs.

## Quality checks

These checks follow the approved PRD and existing permission boundaries; they are recommendations for QA and Security.

- **Idempotency:** Test retrying the same key and same request returns the same answer without duplicate work, plan steps, or events; reusing a key with changed parameters is rejected. Race concurrent retries and verify the key, mutation, and answer remain consistent. The prior project research links this retry pattern to [AWS guidance on safe API retries](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/).
- **OAuth scope and audience:** Verify accepted tokens are intended for this resource and have the required scope; reject wrong-audience/resource and insufficient-scope tokens. The [MCP authorization specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization) describes bearer-token and resource validation. Existing app checks are in `internal/modules/coding/adapter/mcp/guard.go`; role labels must not grant permissions beyond those checks, per the PRD.
- **Work ID access (IDOR):** Try missing, malformed, unknown, wrong-owner, and wrong-project IDs across every work operation. Confirm failures do not reveal another user’s work or mutate it. The PRD explicitly requires wrong-owner and wrong-project refusal.
- **Responsive UI:** At the PRD’s 390, 820, and 1024 px widths, check the new tab and task views for horizontal overflow; verify both external and internal task tabs. These widths and the no-sideways-scroll acceptance point are in `project-notes/tasks/fae3e8b731d9ae374acd8b2a70ea8ca1/prd/36.md`.

## Not found

I did not verify the exact SQLite DSN or transaction settings used at runtime, run a vulnerability scan, run UI/browser checks, or test live MCP clients. The current browser client’s handling of a work ID across reconnects is not established by these references.

## Suggestions

Use the pinned SDK’s typed `AddTool` behavior and include an output schema for machine-readable results. Keep work ownership checks and existing scope checks authoritative. Decide on a transaction and version-conflict strategy in the SQLite store, then test simultaneous updates and atomic event persistence. Treat instructions to clients as guidance, not proof that a client followed the lifecycle.