# sambil parallel revisi dong

Task: 8c584e822e8dcb789ebd960e2a5f7918
Status: done
Updated: 2026-10-21T13:38:46Z

Reference material; confirm applicability before reusing.

**A. Postgres reachability:** The sandbox name resolution failure is expected: task containers have no network, even when they receive `DATABASE_URL`. The app’s connection test uses a separate, guarded network path.  
**B. Tasks:** Tasks can be pinned and have several workflow statuses, but task archiving and single-task deletion do not exist. Permanent deletion must account for both cascading database records and task-related files.

## A. Postgres blocker

### What the code does

- Task containers are created with Docker networking disabled and `NetworkMode: "none"`: [container.go](/workspace/internal/platform/sandbox/container.go:118). The toolbox entry script says containers have no network of their own: [sandbox-entry.sh](/workspace/sandbox/sandbox-entry.sh:1). **Inference:** without a network interface and resolver, a sandbox cannot resolve the database hostname; this fits the reported DNS error.
- The egress proxy is a limited exception for downloads: it accepts HTTPS `CONNECT` requests on port 443, checks the hostname against an allowlist, and only dials public addresses: [proxy.go](/workspace/internal/platform/egressproxy/proxy.go:1), [proxy.go](/workspace/internal/platform/egressproxy/proxy.go:82). The default allowlist contains package registries: [config.go](/workspace/internal/platform/config/config.go:23). A configured database host and port do not become a general route through it.
- Settings-saved database URLs are injected. The database connection service writes the URL to the project Env: [service_shared_environment.go](/workspace/internal/modules/dbconn/application/service_shared_environment.go:14). Env values are decrypted for sandbox injection: [service.go](/workspace/internal/modules/env/application/service.go:147). The task runner merges project values into the command environment: [runner.go](/workspace/internal/modules/coding/adapter/runner/runner.go:99).
- The app’s **Test connection** path is different: the Postgres probe uses `PrivateAllowed` for its dial: [postgres.go](/workspace/internal/modules/dbconn/adapter/probe/postgres.go:15), [egress.go](/workspace/internal/platform/egress/egress.go:67). That permits private and loopback addresses, subject to its exclusions.
- The project docs state that the agent receives the connection string but its sandbox cannot connect directly: [using-the-app.md](/workspace/docs/using-the-app.md:193). Compose also runs the app on the host network while child check containers are started separately: [compose.dev.yaml](/workspace/compose.dev.yaml:23).

### Assessment and options

**Assessment: by design**, not a database-settings injection bug. The URL can be present while DNS and network access are intentionally unavailable inside the sandbox. A separate issue—whether the app server itself can reach the database—would affect the connection test and provisioning, not grant sandbox access.

**Options:** Keep database operations server-side and expose only the needed operation to tasks; or request a separate network design for sandbox database access. The latter would need controlled host-and-port routing and per-workspace isolation. It conflicts with the current runtime rule limiting sandbox egress to registries and connected GitHub, so it requires an explicit architecture decision.

## B. Existing Tasks feature

### Domain, storage, and routes

- The task domain has statuses `queued`, `planning`, `working`, `needs-input`, `review`, `done`, and `failed`; it also has `PinnedAt`, but no task archive field: [task.go](/workspace/internal/modules/task/domain/task.go:5). The database status constraint has the same values: [008_tasks.sql](/workspace/internal/platform/database/migrations/008_tasks.sql:1). `pinned_at` was added separately: [048_task_pinned_at.sql](/workspace/internal/platform/database/migrations/048_task_pinned_at.sql:1).
- The application list returns the user’s tasks with a limit of 200: [service.go](/workspace/internal/modules/task/application/service.go:12), [service.go](/workspace/internal/modules/task/application/service.go:158). SQLite sorts by most recently updated and can filter by project: [store.go](/workspace/internal/modules/task/adapter/sqlite/store.go:113).
- HTTP exposes `GET /api/tasks`, task creation, pin/unpin, and `DELETE /api/tasks` for deleting **all** tasks; there is no single-task delete or archive route: [handler.go](/workspace/internal/modules/task/adapter/http/handler.go:26).
- The UI loads one shared task list from `/api/tasks`: [tasks-api.ts](/workspace/web/src/features/workspace/logic/tasks-api.ts:26), [workspace-store.ts](/workspace/web/src/features/workspace/logic/workspace-store.ts:25). The sidebar derives Recent from the first nine unpinned tasks and separately shows pinned tasks: [SidebarRecent.svelte](/workspace/web/src/features/workspace/shell/SidebarRecent.svelte:7), [SidebarPinned.svelte](/workspace/web/src/features/workspace/shell/SidebarPinned.svelte:6). A project’s Tasks tab filters that same list by project and status: [TasksTab.svelte](/workspace/web/src/features/workspace/projects/TasksTab.svelte:15).
- Project archive/delete exists separately from task archive/delete: [AboutTab.svelte](/workspace/web/src/features/workspace/projects/AboutTab.svelte:88). Projects have `archived_at`; tasks do not: [007_projects.sql](/workspace/internal/platform/database/migrations/007_projects.sql:1).

### Existing deletion and related data

- The current task deletion use case is bulk deletion. It cancels running tasks and removes their screenshots before deleting task rows: [erase.go](/workspace/internal/modules/task/application/erase.go:39). The SQLite delete relies on foreign-key cascades: task events, transcript, and usage reference tasks with `ON DELETE CASCADE`: [008_tasks.sql](/workspace/internal/platform/database/migrations/008_tasks.sql:21). Delegations and CLI sessions also cascade: [029_agent_team.sql](/workspace/internal/platform/database/migrations/029_agent_team.sql:31), [036_task_cli_session.sql](/workspace/internal/platform/database/migrations/036_task_cli_session.sql:3). There is no separate run table in these migrations; action/run details are carried in task events.
- Screenshots live outside SQLite under a per-task folder and have a `Remove` method: [shots.go](/workspace/internal/modules/task/adapter/shots/shots.go:32). Existing project-task deletion calls it through `stopAndForget`: [erase.go](/workspace/internal/modules/task/application/erase.go:10).
- Plans, requirements, research, and long-message input documents are exported under `project-notes/tasks/<task-id>/`; the project index is rebuilt from task records: [notefiles.go](/workspace/internal/modules/task/application/notefiles.go:40), [inputdocument.go](/workspace/internal/modules/task/application/inputdocument.go:10). The Notes port only writes files and does not delete stale ones: [files.go](/workspace/internal/modules/task/adapter/notefiles/files.go:29). **Inference:** a permanent task delete needs an explicit policy and cleanup for that task’s exported directory, then an index rebuild.
- Ordinary uploaded attachments are saved in the project’s shared `uploads/` folder and appear in project Files: [using-the-app.md](/workspace/docs/using-the-app.md:22). I found no task-to-upload ownership mapping in the task records inspected. **Inference:** deleting a task should not remove those project files without an explicit ownership rule.
- One existing bulk-delete edge case: `DeleteAll` first lists only the standard 200-task limit for cancellation and screenshot cleanup, then deletes all task rows: [erase.go](/workspace/internal/modules/task/application/erase.go:39), [service.go](/workspace/internal/modules/task/application/service.go:12). Tasks beyond that list may not get those external cleanup steps.

### Relevant tests

- Sandbox isolation: [sandbox_test.go](/workspace/internal/modules/coding/adapter/runner/sandbox_test.go:38).
- Screenshot save/read/remove: [shots_test.go](/workspace/internal/modules/task/adapter/shots/shots_test.go:13).
- Project document export and repair: [project_documents_test.go](/workspace/internal/bootstrap/project_documents_test.go:11).
- Existing application status behavior: [service_test.go](/workspace/internal/modules/task/application/service_test.go:50).

I found no single-task archive/delete tests or SQLite task-store tests in the task adapter. I did not run tests or start the app.

## Not found

- A supported sandbox route to arbitrary database hosts or ports.
- Task-level archive or permanent-delete behavior.
- A task-to-upload attachment ownership record.

## Suggestions

- Treat the DNS error as expected under current sandbox isolation. Keep database work on the app server unless the person explicitly approves a new network design.
- For task deletion, decide whether exported task documents and long-message inputs should be erased, and how project-shared uploads should be handled.
- Ensure selected tasks are cancelled and their screenshots and exported files are cleaned up; the existing bulk-delete path’s 200-task cleanup limit is relevant.

**What I did:** Read the agent notes, project index and relevant plans, docs, and implementation files; searched the repository and inspected source line numbers. No files were changed.

---

## Research report

SQLite supports adding nullable columns with `ALTER TABLE ... ADD COLUMN`; without constraints that require checking existing rows, the operation does not rewrite table contents. Partial indexes can keep only active rows, but SQLite must be able to match the query’s `WHERE` terms to the index predicate. (Official docs: [ALTER TABLE](https://www.sqlite.org/lang_altertable.html), [Partial Indexes](https://www.sqlite.org/partialindex.html).)

This repo uses Go 1.27.2, Fiber v3.5.0, `modernc.org/sqlite` v1.59.0, Svelte 5.57.1, and daisyUI 5.7.47 (`go.mod`, `web/package.json`). Current online daisyUI docs show 5.7.50, so the installed version is slightly behind; confirm the needed classes against the pinned package. No files were changed.

### Technical references

- **Repo behavior:** Migrations are append-only and checksum-verified (`internal/platform/database/migrate.go`; `agents/memory-backend.md`). The tasks table has indexes on user and project activity, while the project archive migration provides a local pattern for nullable `archived_at` and a partial index (`008_tasks.sql`, `017_project_source.sql`). A new migration should be appended.
- **SQLite batches:** The documented default variable limit is 32,766 in SQLite versions since 3.32.0; older versions default to 999, and runtime limits can be lowered. The repo’s SQLite driver docs describe its SQLite build, but not a guaranteed runtime parameter limit. Use bounded chunks rather than assuming 32,766 parameters ([SQLite limits](https://www.sqlite.org/limits.html), [modernc.org/sqlite](https://modernc.org/sqlite)).
- **Fiber batch bodies:** Fiber v3 uses `c.Bind().Body(&request)`; `Body` needs an appropriate `Content-Type`. `DELETE` bodies are not prohibited by Fiber, but HTTP does not define their semantics and some clients or intermediaries may reject them. For batch ids, a `POST` action endpoint avoids that ambiguity ([Fiber Bind](https://docs.gofiber.io/api/bind/), [HTTP DELETE](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Methods/DELETE)).
- **Svelte selection state:** Svelte 5 runes support reactive local state via `$state`; arrays and objects are deeply reactive. A `Set` of selected ids is a reasonable implementation choice, but the docs’ ordinary state examples do not establish reactive `Set` behavior. Verify the chosen collection’s reactivity in Svelte’s `$state` guidance before relying on in-place mutation ([Svelte `$state`](https://svelte.dev/docs/svelte/$state)).
- **daisyUI:** The project already has a native `<dialog>` confirmation component, `ConfirmModal.svelte`; reuse it. daisyUI documents `checkbox` and `modal`, while its toast is a positioning wrapper for stacked content, not a complete notification system. Existing app notification patterns should determine how toast messages are announced ([checkbox](https://daisyui.com/components/checkbox/), [modal](https://daisyui.com/components/modal/), [toast](https://daisyui.com/components/toast/)).
- **Filesystem cleanup:** `os.RemoveAll` accepts a path, so joining an unchecked id to a base path risks traversal. `os.Root` operations are traversal-resistant; `Root.RemoveAll` was added in Go 1.25, compatible with this repo’s Go 1.27.2 target. Existing screenshot and note-file adapters already use `os.OpenRoot` (`shots.go`, `notefiles/files.go`). Prefer validating ids and removing relative to a root ([Go traversal-resistant file APIs](https://go.dev/blog/osroot), [Go 1.25 release notes](https://go.dev/doc/go1.25)).

### Quality checks

- Test archive/restore/delete for owned, unknown, and already-transitioned ids; retries should be safe. For partial failures, verify per-id results and ensure a successful id is not repeated as failed.
- Test batches above 200 and above the chosen SQL chunk size; verify every task’s rows and external files are handled. Include malformed and traversal-shaped ids, and confirm shared uploads remain untouched.
- Verify running tasks are stopped before cleanup and repeated delete requests do not fail merely because rows or files are already gone.
- For multi-select controls, check keyboard operation, visible focus, associated labels, “select all shown” mixed state, and selection clearing when rows disappear. For confirmation dialogs, check focus placement/return, Escape behavior, clear task count, and the running-task warning. Announce success and per-id failures with text, not color alone. These checks align with the repo’s [design system](https://www.sqlite.org/partialindex.html) accessibility rules; use WCAG 2.2 AA as the QA target.

### Not found

- An official guarantee that the repo’s compiled SQLite runtime has a particular host-parameter ceiling.
- A complete daisyUI toast accessibility contract; daisyUI documents layout/positioning, so the app must provide suitable live-region semantics.

### Suggestions

Use a nullable timestamp column, matching query predicates to the active-row partial index. Keep batch operations bounded and make idempotency explicit per id. Prefer `POST` for batch actions and root-scoped filesystem cleanup. Reuse `ConfirmModal.svelte` and theme-semantic daisyUI controls; test keyboard/focus and live announcements alongside the functional cases.