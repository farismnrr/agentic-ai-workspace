# # Fix Task Bubble Lifecycle, Live Streaming, and External Ta…

Task: a4c3f733b96d6e0396ac72e2c102a4eb
Status: done
Updated: 2026-10-27T08:12:55Z

Reference material; confirm applicability before reusing.

**Summary:** The composer permits external-task follow-ups, but the backend rejects them; explicit MCP work sessions also turn chat messages into action events.  
**Summary:** SSE reconnects and periodic checks can recover appended events, but cannot refresh an event that changed while keeping its sequence.

## Confirmed findings

- **A — External chat conflicts with the backend.** [TaskScreen.svelte](/workspace/web/src/features/workspace/task/TaskScreen.svelte:39) enables the composer for `review`, `done`, and `failed` without checking `source`. [task-actions.ts](/workspace/web/src/features/workspace/logic/task-actions.ts:53) always sends through `sendMessage`. [service.go](/workspace/internal/modules/task/application/service.go:117) rejects any task with a nonempty `Source` as `ErrWrongState`. The HTTP handler passes that result through at [tasks.go](/workspace/internal/modules/task/adapter/http/tasks.go:128). **Confirmed root cause:** UI and backend have different follow-up contracts.

- **B — Updated historical events are invisible to the thread reconciler.** [thread.ts](/workspace/web/src/features/workspace/logic/thread.ts:48) only adds events whose sequence exceeds the last seen sequence. [work_calls.go](/workspace/internal/modules/task/adapter/sqlite/work_calls.go:119) updates the event body while preserving its ID and sequence. [stream.go](/workspace/internal/modules/task/adapter/http/stream.go:85) asks for events after the cursor; its 15-second heartbeat repeats that same query. **Confirmed root cause:** the client and stream only reconcile by append cursor, not event identity. A dropped hub notification is recoverable for a *new* sequence on the next heartbeat, but an in-place update remains outside the cursor query.

- **C — Explicit work sessions lose chat bubble semantics.** `log_message` creates user/assistant notes at [tools_log.go](/workspace/internal/modules/coding/adapter/mcp/tools_log.go:30). Legacy recording maps those to request/message events in [threads.go](/workspace/internal/bootstrap/threads.go:63). In explicit `work_id` sessions, [guard_thread.go](/workspace/internal/modules/coding/adapter/mcp/guard_thread.go:105) routes to `workThread`; [workbridge_calls.go](/workspace/internal/bootstrap/workbridge_calls.go:36) converts every call note to `EventAction`; [work_calls.go](/workspace/internal/modules/task/application/work_calls.go:29) also forces the event kind to action. [blocks.ts](/workspace/web/src/features/workspace/task/thread/blocks.ts:5) renders consecutive actions as grouped blocks. **Confirmed root cause:** explicit work-session messages lose user/assistant event types before rendering; the legacy path preserves them.

- **D — Persistence success can be overstated, and a running action can remain stale.** `logMessage` returns `Saved: true` at [tools_log.go](/workspace/internal/modules/coding/adapter/mcp/tools_log.go:49), before the guard records the note. The guard intentionally ignores thread-log failures at [guard_thread.go](/workspace/internal/modules/coding/adapter/mcp/guard_thread.go:103). Explicit work recording may only append `recorded:false` to internal activity detail after failure at [guard_work.go](/workspace/internal/modules/coding/adapter/mcp/guard_work.go:75); finishing an existing call likewise ignores a failed update at [guard_work.go](/workspace/internal/modules/coding/adapter/mcp/guard_work.go:65). **Confirmed root cause:** MCP success is not conditional on thread persistence. If the finish update fails, the stored start event can remain `running`.

## Existing flow and design facts

- Internal task creation stores the task, appends its request, then enqueues it in [service.go](/workspace/internal/modules/task/application/service.go:52). `Scheduler.Enqueue` deduplicates already queued task IDs at [scheduler.go](/workspace/internal/modules/task/application/scheduler.go:68). External `begin_work` instead creates a `working` task, work row, and request event together; SQLite commits them atomically in [works.go](/workspace/internal/modules/task/application/works.go:132) and [work_store.go](/workspace/internal/modules/task/adapter/sqlite/work_store.go:17), then notifies task and user subscribers at [works.go](/workspace/internal/modules/task/application/works.go:162). Agent enqueueing is not part of that path.

- `Observe` wraps task and event writes so successful create/status/event changes notify subscribers at [observe.go](/workspace/internal/modules/task/application/observe.go:19). The hub broadcasts to task and user topics but drops updates when a subscriber’s buffer is full at [hub.go](/workspace/internal/modules/task/application/hub.go:69). Task SSE sends the initial state, uses `Last-Event-ID`/`after` as a sequence cursor, and checks again on a 15-second heartbeat at [stream.go](/workspace/internal/modules/task/adapter/http/stream.go:15).

- Existing schema stores tasks and ordered events (`seq`, stable `id`) in migration [008_tasks.sql](/workspace/internal/platform/database/migrations/008_tasks.sql:1). Migration [055_external_works.sql](/workspace/internal/platform/database/migrations/055_external_works.sql:1) adds `tasks.work_id`, `external_works`, and idempotent operation rows. No task-to-task orchestrator link was found in the inspected schema.

- **Design option, inferred from the current schema:** add a nullable internal-task reference to the external task in a new migration, with uniqueness to prevent multiple linked conversations. Resolve and validate ownership and same-project scope in the task application service. A separate typed link table is another option, but appears more machinery for the one-to-one relationship described in the request. Do not enqueue or mutate the external task’s lifecycle when creating or following the linked internal task.

## Design / technical references

- **Svelte:** the existing thread renderer uses keyed each blocks in its components; keep keys stable and unique when replacing event objects. The current docs call out duplicate and non-idempotent keys: [Svelte runtime errors](https://svelte.dev/docs/svelte/runtime-errors).
- **Fiber v3:** the current stream uses `SendStreamWriter`. Fiber documents flushing the buffered writer for prompt delivery and checking flush errors for disconnects: [Fiber `Ctx` streaming docs](https://docs.gofiber.io/api/ctx/), [Fiber v3 changes](https://docs.gofiber.io/next/whats_new/).
- **SSE:** browsers reconnect and send `Last-Event-ID`, but that only resumes from an event ID; it does not provide server-side replay or reconcile mutated historical records. The project’s cursor is a sequence number, while event updates retain that sequence: [WHATWG EventSource standard](https://html.spec.whatwg.org/dev/server-sent-events.html).
- **MCP:** progress is optional and request-scoped; clients must supply a `progressToken`, and receivers may omit progress updates. It does not imply arbitrary model-token streaming: [MCP Progress spec, 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/basic/utilities/progress). The project currently uses explicit `begin_work`/progress/report tools rather than MCP token streaming. MCP lifecycle also defines initialization, negotiated capabilities, operation, shutdown, and request timeouts: [MCP Lifecycle spec](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle).

## Existing test infrastructure

- Go tests are run by `make backend-test` (`go test ./cmd/... ./internal/...`); this is part of `make verify` in [Makefile](/workspace/Makefile:84). There are 83 task-module test files and 6 MCP adapter test files. Relevant tests include [work_calls_test.go](/workspace/internal/modules/task/adapter/sqlite/work_calls_test.go) and MCP work-recording integration tests.
- Frontend uses Vitest via `npm test`; the package script is `vitest run` in [web/package.json](/workspace/web/package.json:10). No `thread.ts`-specific test was found.
- Playwright scripts exist through `make e2e-thread`, but the current [e2e-thread.ts](/workspace/scripts/e2e-thread.ts:1) covers thread rendering and screenshots, not the requested streaming/reconnect scenarios. Browser checks require the existing app and Playwright setup; none were run for this research task.

## Preflight

- One worktree: `/workspace`; branch `dev`; HEAD `751c98556e43cb4770b95f91acdd9d384c22da33`.
- Existing untracked files include `masih-awam-reference.zip`, `reference-bundle/`, and numerous `project-notes/tasks/` files, including this task’s inputs and plan. No tracked modifications were shown.
- `git fetch` failed with `error: cannot open '.git/FETCH_HEAD': Read-only file system`. The local `dev...origin/dev` comparison is `0 0`, but it reflects the existing local remote-tracking ref and was not refreshed. No files were changed.

## Not found

- A server-rendered or accepted-confirmation response that guarantees the external `log_message` event is durable before MCP reports `saved`.
- A frontend test for thread event reconciliation or an existing browser test for SSE reconnection.
- Token-level ChatGPT/Gemini streaming support in the inspected MCP recording flow.

## Suggestions

- Plan reconciliation around stable event IDs/upserts, with a bounded full-event refresh or versioned update mechanism to recover missed mutations; retain sequence ordering for display.
- Preserve request/message kinds across the explicit work-call adapter, and make the persistence claim reflect whether the task event was actually committed.
- Treat external task follow-up as creation or reuse of a separate linked internal conversation. Keep external execution and its permissions independent.
- Fetch and rerun the remote comparison before implementation if a writable checkout becomes available; the current fetch did not refresh `origin/dev`.

No tests or application checks were run; only read-only repository inspection, Git preflight, and official documentation research were performed.

---

**Summary:** The project’s schema and handlers fit a database-enforced link plus explicit event reconciliation; a unique nullable column permits multiple unlinked rows.  
**Summary:** QA should exercise reconnect and in-place event updates, concurrent retries, forged requests, and cross-project access.

## Technical references

### SQLite: nullable unique link and concurrent get-or-create

**Seen in the project:** [008_tasks.sql](/workspace/internal/platform/database/migrations/008_tasks.sql) stores task IDs and ordered events; [055_external_works.sql](/workspace/internal/platform/database/migrations/055_external_works.sql) uses `UNIQUE` and a partial unique index for idempotency. The project pins `modernc.org/sqlite v1.59.0` in [go.mod](/workspace/go.mod); its driver is CGo-free, per the [driver documentation](https://modernc.org/sqlite/).

**Official guidance:** SQLite treats `NULL` values as distinct for uniqueness, so a unique nullable link permits many `NULL` rows but only one row per non-NULL link value. SQLite supports `INSERT ... ON CONFLICT ... DO NOTHING`; it allows only one simultaneous write transaction, and `BEGIN IMMEDIATE` acquires the write transaction up front but can return `SQLITE_BUSY`. [SQLite UNIQUE constraints](https://www.sqlite.org/lang_createtable.html), [UPSERT](https://www.sqlite.org/lang_upsert.html), [transactions](https://www.sqlite.org/lang_transaction.html)

**Application:** Let the unique constraint arbitrate concurrent creators. Insert with the targeted conflict behavior, then select the persisted link within the transaction; keep linked-task creation and its initial records atomic. Handle or retry transient busy errors according to the project’s database policy. This is implementation guidance inferred from SQLite’s constraints and locking model.

### Svelte 5: keyed each and replace-by-id state

Svelte’s keyed each blocks use a unique, stable key to preserve item identity as arrays change. Duplicate or unstable keys are runtime errors. `$state` arrays support reactive updates. For event reconciliation, upsert by stable event ID, then sort by sequence; retain that ID as the each-block key. [Svelte keyed each documentation](https://v4.svelte.dev/docs/logic-blocks), [Svelte runtime errors](https://svelte.dev/docs/svelte/runtime-errors), [Svelte deep state](https://svelte.dev/tutorial/svelte/deep-state)

### Fiber v3: SSE flush and disconnect

Fiber’s `SendStreamWriter` supplies a buffered writer. Flush after each SSE frame that must arrive promptly; a flush error signals a disconnected client. Fiber’s v3 notes describe stream setup and headers, but application code still owns replay and notification behavior. [Fiber `Ctx` streaming docs](https://docs.gofiber.io/api/ctx/), [Fiber v3 changes](https://docs.gofiber.io/next/whats_new/)

**Seen in the project:** [stream.go](/workspace/internal/modules/task/adapter/http/stream.go) uses `Last-Event-ID`/`after`, a heartbeat, and `SendStreamWriter`. The prior research found that its sequence cursor cannot retrieve an event whose body changed without a new sequence. A version marker or equivalent changed-event query is needed to make changed historical events discoverable.

### EventSource reconnect

Browsers reconnect after a dropped connection and send `Last-Event-ID` when available. They expose `open`, `error`, and `readyState`; `error` alone does not mean reconnection has permanently stopped. A server-provided `retry` field can set the retry delay; HTTP 204 tells the browser to stop reconnecting. Reconnection does not replay application data by itself: the server must use the cursor to return missed or changed events. [WHATWG EventSource standard](https://html.spec.whatwg.org/dev/server-sent-events.html)

## Quality checks

The PRD calls for tests covering event updates, linked-task idempotency, project boundaries, archived-task rejection, CSRF, persistence failure, and ensuring external updates never enqueue work.

- **SSE reconciliation:** verify append and same-ID update delivery; reconnect using `Last-Event-ID`; recover a missed notification on heartbeat; ensure client upserts by ID, sorts by sequence, avoids duplicates, ignores late responses after task switching, and closes obsolete streams.
- **Idempotency:** race concurrent first messages through separate requests/connections; assert one linked internal task and one initial conversation, with retries reusing it. Include busy/error and failed-transaction cases.
- **CSRF:** reject missing, invalid, and wrong-session tokens on the state-changing chat route; accept a valid session token. The project’s egress-grant route already applies same-origin and CSRF middleware in [handler.go](/workspace/internal/modules/egressgrant/adapter/http/handler.go), with token validation in [csrf.go](/workspace/internal/modules/egressgrant/adapter/http/csrf.go). Confirm which protection the new route uses.
- **Cross-project isolation:** attempt access with a different project/task ID and assert no linked task, events, or data are exposed or changed. OWASP recommends establishing tenant context early and testing authorization boundaries. [OWASP Multi-Tenant Security](https://cheatsheetseries.owasp.org/cheatsheets/Multi_Tenant_Security_Cheat_Sheet.html), [Authorization Testing Automation](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Testing_Automation_Cheat_Sheet.html)
- **Browser flows:** cover internal streaming, external work updates, external-task chat, disconnect/reconnect, and refresh during activity at desktop, tablet, and mobile sizes, recording console errors and screenshots as the PRD specifies.

## Not found

The precise CSRF middleware expected for the new chat route is not settled in the project notes. The browser reconnect behavior and multi-connection idempotency are not yet verified by tests.

## Suggestions

Keep the database uniqueness constraint as the final concurrency guard and test it with genuinely concurrent requests. Treat event updates as versioned changes that can be fetched on reconnect and heartbeat. QA should verify authorization at the route and persistence boundaries, not only in the UI.

**Work done:** Read the indexed task plan, requirements, research, PRD, and relevant migrations and code references; searched current official docs. No files were changed and no tests or application checks were run. Git fetch failed because `.git/FETCH_HEAD` is read-only; existing untracked project notes and reference files were left untouched.