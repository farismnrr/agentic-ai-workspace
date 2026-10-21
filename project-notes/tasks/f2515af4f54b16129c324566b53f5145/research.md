# ok gw mau revisi pelan pelan, ga usah over engineering ya

Task: f2515af4f54b16129c324566b53f5145
Status: working
Updated: 2026-10-21T04:18:56Z

Reference material; confirm applicability before reusing.

**Summary:** Env variables and database connections are stored per project, but their UI is in Settings, where Env offers a project picker.  
The shared PostgreSQL flow already checks for and creates a missing project schema; there is no per-project SQLite-versus-Settings-DB choice.

## Env: where it lives and why it appears in Settings

- The Env API is mounted at `/api/projects/:id/env` in [`handler.go`](/workspace/internal/modules/env/adapter/http/handler.go:20). The `env` module owns validation and use cases in `application/service.go` and `domain/variable.go`; SQLite storage of sealed values is in [`store.go`](/workspace/internal/modules/env/adapter/sqlite/store.go:13). Migration 043 creates `project_env_vars`.
- The UI is [`EnvTab.svelte`](/workspace/web/src/features/workspace/settings/EnvTab.svelte:1), with variable forms in `settings/env/VariablesSection.svelte` and database forms in `settings/env/DatabaseSection.svelte`. It selects a project from the shared projects list and passes that project ID to both sections ([`EnvTab.svelte`](/workspace/web/src/features/workspace/settings/EnvTab.svelte:14)).
- The code shows that Env is a Settings tab, while its data and API are project-scoped. **Inference:** it is in Settings because it was grouped with configuration screens; I found no explicit rationale in the code or project notes. The current Projects detail page has Tasks, Files, Checks and About tabs ([`ProjectDetailPage.svelte`](/workspace/web/src/features/workspace/projects/ProjectDetailPage.svelte:17)).
- Checks and commands receive only the selected project’s variables through `runner.WithProjectEnv` and the CLI environment assembly; project sites use `Runtime.WithProjectEnv` ([`runner/project_env.go`](/workspace/internal/modules/coding/adapter/runner/project_env.go:10), [`taskcli_env.go`](/workspace/internal/bootstrap/taskcli_env.go:65), [`runtime/project_env.go`](/workspace/internal/modules/site/adapter/runtime/project_env.go:12)). Existing memory says sandbox containers remain offline ([`agents/implementation-notes.md`](/workspace/agents/implementation-notes.md:128)).

## Database connections and existing shared PostgreSQL support

- Project DB endpoints are in [`dbconn/adapter/http/handler.go`](/workspace/internal/modules/dbconn/adapter/http/handler.go:25). Project connection records and sealed passwords are stored by `dbconn/adapter/sqlite/store.go`; migration 044 creates `project_db_connections`, and migration 045 adds its `managed` flag. The connection form and shared-Postgres action are in `settings/env/DatabaseSection.svelte`.
- `POSTGRES_ADMIN_URL` is process configuration, loaded in [`config.go`](/workspace/internal/platform/config/config.go:107) and wired into `dbconn` in `internal/bootstrap/dbconn.go`. It is not a per-project Settings field.
- Commit **5f5f9c7** added Env storage/API/UI and manual PostgreSQL, MySQL and MongoDB connection settings and tests. Commit **ce2dfa4** added managed shared-Postgres provisioning, including schema and role creation/reconciliation, plus its Settings UI flow and migration 045. The prior task requirements explicitly scoped out database-per-project choice ([`requirements.md`](/workspace/project-notes/tasks/20c2ef530617785d0cefb7b799dc5729/requirements.md)).
- **Schema check exists:** `applySchema` checks `pg_namespace`; it creates a missing schema and verifies ownership when it already exists ([`provision/apply.go`](/workspace/internal/modules/dbconn/adapter/provision/apply.go:78)). This happens during `ProvisionShared`, which also saves the managed connection and writes `DATABASE_URL` to that project’s Env ([`service_shared.go`](/workspace/internal/modules/dbconn/application/service_shared.go:14), [`service_shared.go`](/workspace/internal/modules/dbconn/application/service_shared.go:73)).
- **Per-project database choice does not exist:** there is no project setting that chooses SQLite or the shared Settings database. Current DB UI configures a project’s manual connection or provisions shared PostgreSQL. I found no separate schema-exists check triggered by a project choice.

## Projects page, reuse and risks

- [`ProjectDetailPage.svelte`](/workspace/web/src/features/workspace/projects/ProjectDetailPage.svelte:21) already defines the project tabs and passes `projectId` to Checks and Tasks. [`AboutTab.svelte`](/workspace/web/src/features/workspace/projects/AboutTab.svelte:14) shows the existing pattern for project-scoped editing and saving. `ProjectsPage.svelte` is the project list; [`workspace-store.ts`](/workspace/web/src/features/workspace/logic/workspace-store.ts) supplies the shared projects list used by the detail page and Env tab.
- Existing pieces that appear reusable: Env’s `VariablesSection` and `env-api.ts` accept a project ID; `DatabaseSection` and the `dbconn` endpoints already manage project-scoped connections; `ProjectDetailPage` already has a tab structure. **Inference:** moving the Env UI into Projects can reuse these pieces, though loading and save behavior should be checked for the project-detail lifecycle.
- A database choice is a new state and behavior. The current shared provisioning action provisions a connection and writes `DATABASE_URL`; the reviewed code does not provide the requested SQLite/default selection behavior. Clarify what “keep default SQLite” means for project code that may already have a managed or manual connection.
- Existing variable values are stored against project IDs, and shared provisioning stores both a DB connection record and a `DATABASE_URL` Env value. A change should preserve those values and define how switching choices affects an existing connection or URL. Existing shared database objects are intentionally not dropped on connection removal ([`service_shared.go`](/workspace/internal/modules/dbconn/application/service_shared.go:14); [`agents/implementation-notes.md`](/workspace/agents/implementation-notes.md:167)).
- Any source change must respect the 200-line file limit and Go/frontend complexity limits ([`AGENTS.md`](/workspace/AGENTS.md): “Size and complexity limits”). User-visible behavior changes require the relevant docs update; existing descriptions are in [`docs/using-the-app.md`](/workspace/docs/using-the-app.md:119) and architecture notes in [`docs/system-design.md`](/workspace/docs/system-design.md:57). Agent memory also describes the existing Env and DB behavior ([`agents/memory-frontend.md`](/workspace/agents/memory-frontend.md:20), [`agents/memory-backend.md`](/workspace/agents/memory-backend.md:30)).

## Not found

- No explicit recorded reason for placing project-scoped Env under Settings.
- No current project-level SQLite-versus-shared-database selector.
- No project-choice-triggered schema check; schema check and creation currently happen inside shared provisioning.

## Suggestions

- The requested placement is consistent with the existing project detail tabs and project-ID-based APIs.
- Keep shared DB configuration in process Settings, and use the existing provisioning path when a project selects it; confirm the desired behavior when switching a project that already has a manual or managed DB connection.
- No web search, project commands or tests were run. No files were changed.

---

## Summary

The repository already has a `dbconn` module that seals project passwords and provisions shared PostgreSQL schemas.
Official guidance supports building connection config from validated fields, quoting SQL identifiers, and handling Fiber v3 bind errors explicitly.

## Technical references

- **Build the connection config:** pgx v5’s `pgx.ParseConfig` accepts PostgreSQL URI or keyword/value syntax, then exposes a config that can be modified. PostgreSQL requires percent encoding for special characters in URI parts; constructing a URL with `url.UserPassword` and `url.Values` avoids hand-escaping the password. For separately supplied fields, use `pgx.ConnConfig` fields or a keyword/value encoder rather than string concatenation. Sources: [pgx config](https://pkg.go.dev/github.com/jackc/pgx/v5), [pgconn parsing notes](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn), [PostgreSQL connection URIs](https://www.postgresql.org/docs/17/libpq-connect.html).

- **Validate SSL mode:** PostgreSQL documents `disable`, `allow`, `prefer`, `require`, `verify-ca`, and `verify-full`; `verify-full` checks both certificate trust and hostname and is recommended for security-sensitive use. Select from an allowlist rather than accepting arbitrary values. Source: [PostgreSQL SSL support](https://www.postgresql.org/docs/current/libpq-ssl.html).

- **Create and quote schemas:** PostgreSQL supports `CREATE SCHEMA IF NOT EXISTS`, but that clause alone does not verify the existing schema’s owner. Quote dynamic identifiers; pgx provides `pgx.Identifier{...}.Sanitize()`. The current provisioner already checks existence and owner and uses `pgx.Identifier` for schema, role, and database names. Sources: [CREATE SCHEMA](https://www.postgresql.org/docs/17/sql-createschema.html), [identifier syntax](https://www.postgresql.org/docs/current/sql-syntax-lexical.html), [pgx Identifier](https://pkg.go.dev/github.com/jackc/pgx/v5#Identifier).

- **Fiber v3 endpoint pattern:** The existing handler uses `c.Bind().JSON`, returns a deliberate 400 for malformed input, and registers GET separately from PUT. Fiber docs note that bind/validation errors are returned as-is; a handler should map client input errors to 4xx rather than letting them become 500s. `c.Context()` is Fiber’s fasthttp context; the docs say it returns a standard context previously set with `SetContext`, otherwise `context.Background()`. Follow neighboring handlers and their established context handling. Sources: [Fiber Bind](https://docs.gofiber.io/api/bind/), [Fiber Ctx](https://docs.gofiber.io/api/ctx/).

- **Repository patterns to copy:** `internal/modules/dbconn/adapter/sqlite/store.go` stores project connection secrets sealed through `platform/secretbox`; migrations 044–045 establish the connection table and managed flag. `internal/modules/dbconn/adapter/http/handler.go` has the project GET/PUT shape and explicit safe error responses. `internal/bootstrap/dbconn.go` wires the provisioner. Existing Settings tabs live under `web/src/features/workspace/settings/`; the new global config should follow their API and tab composition patterns. The existing project DB GET returns `domain.Metadata`, not the password.

## Quality checks

- Verify Settings GET and PUT responses never include the password, including after saving and on validation or storage errors. Check logs and error bodies for the password and full connection URL; pgconn cautions that parse-error redaction is best effort.
- Exercise passwords containing URI-reserved characters, spaces, quotes, percent signs, and Unicode. Confirm the resulting pgx config preserves the exact password, host, port, database, user, and SSL mode.
- Reject unknown SSL modes. Test the accepted values and expected TLS behavior, including certificate/hostname verification when `verify-full` is selected.
- Try hostile schema names and project identifiers containing quotes, semicolons, comments, and punctuation. Confirm dynamic identifiers use `pgx.Identifier` and cannot escape their identifier.
- For existing and missing schemas, verify provisioning preserves the existing ownership check and creates only when absent. For SQLite detach, verify only the project connection and `DATABASE_URL` are removed; no `DROP SCHEMA`, `DROP ROLE`, or other destructive SQL is issued.
- Test saved Settings config used without restart, `POSTGRES_ADMIN_URL` fallback when no config is saved, and behavior when neither source is configured.

## Not found

- I did not find a current official pgx function that encodes arbitrary keyword/value fields into a connection string; direct config-field assignment or URL construction with standard URL encoders avoids relying on a hand-built string.
- No project Settings tab in the supplied notes establishes a precise pattern for secret redaction in a server-wide settings endpoint; the project-scoped `dbconn` pattern is the closest repository precedent.

## Suggestions

Use the existing `dbconn` metadata/secretbox and handler conventions, but define a response DTO that omits the password entirely. Keep the shared config as a separate application boundary and construct its pgx config from validated fields at call time. QA should specifically verify identifier quoting and detach behavior against emitted SQL.

No files were changed and no project commands or tests were run.

---

## Summary

The reported 104 bytes is the end offset of the last pointer-containing field, not the sum of all pointer words.

`Service`’s optimal order puts `adminEmail` and `erasers` last, with `adminEmail` last.

## Findings

In [`service.go`](/workspace/internal/modules/account/application/service.go:21), the fields are `erasers []Eraser`, four interfaces (`users`, `sessions`, `hasher`, `tokens`), `now func() time.Time`, and `adminEmail string`.

The analyzer computes a type’s pointer-data length separately from its total size. In the current upstream source, interfaces count as **two pointer words**; slices, function values, and strings each count as **one pointer word**. The slice’s length and capacity words and the string’s length word contribute to total size, but not to pointer-data length. A pointer-free field contributes zero. For a struct, the analyzer tracks the end offset of each pointer-containing field and returns the last such end offset. That is why the diagnostic says “leading bytes.” [Analyzer source](https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go) · [Package documentation](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/fieldalignment)

The analyzer’s suggested order sorts by alignment first, then puts pointer-containing fields before pointer-free fields. Among pointer-containing fields it places those with fewer trailing non-pointer bytes first, then larger fields first. For this `Service`, that yields:

```go
type Service struct {
    users      UserStore
    sessions   SessionStore
    hasher     PasswordHasher
    tokens     Tokens
    now        func() time.Time
    adminEmail string
    erasers    []Eraser
}
```

The four interfaces tie: each is two words total and two pointer words. `now` is one word total and one pointer word. `adminEmail` is two words total but only its first word is pointer data, so it has one trailing non-pointer word. `erasers` is three words total and one pointer word, so it has two trailing non-pointer words. The analyzer puts the interface fields first, then `now`, then `adminEmail`, then the slice. [Analyzer ordering and size calculations](https://github.com/golang/tools/blob/master/go/analysis/passes/fieldalignment/fieldalignment.go)

That order **can satisfy the pointer-data check**: it leaves the slice’s one pointer word at the earliest possible ending position, while moving its length and capacity words after it. Putting both the string and slice last is not sufficient by itself; the order between them matters for the analyzer’s optimal value. This conclusion uses the current upstream ordering logic; the exact diagnostic depends on the analyzer version in the repository’s lint toolchain.

## Not found

The repository’s configured golangci-lint version was not established from the files read, so I can’t confirm that its bundled analyzer has precisely the same ordering logic as current upstream. The observed diagnostic itself is recorded in the archived plan, but its complete original text is truncated there.

## Suggestions

Place `erasers` last and `adminEmail` immediately before it, after the interface/function fields. The previous attempt placing the slice first would keep its pointer at a later offset because the slice’s three-word size advances the struct before subsequent pointer-containing fields.

No files were changed and no project commands or tests were run.

---

## Summary

The app shell starts with no visible loader in `index.html`. After Svelte mounts, `App.svelte` shows the global spinner while it waits for the session on protected or guest-only routes.

Tasks already belong to a user and project. There is no pin field, pin endpoint, or pin control today; the sidebar shows up to nine recently updated tasks.

## Loading trace

**Observed:** [`index.html`](/workspace/web/index.html:23) contains only the empty `#app` mount point. [`main.ts`](/workspace/web/src/main.ts:16) mounts the Svelte app. Session state starts as `loading` and `loadSession()` requests `/api/auth/me`; any failure sets it to `anonymous` ([`session.ts`](/workspace/web/src/shared/session/session.ts:11), [`session.ts`](/workspace/web/src/shared/session/session.ts:15)).

In [`App.svelte`](/workspace/web/src/app/App.svelte:47), the global spinner appears while a protected route awaits sign-in, a guest-only route awaits the session result, or an admin-only Settings route is redirecting. The `<main>` has `aria-busy="true"`, but the spinner itself has no text or status role. The source uses `grid ... place-items-center`, so it specifies a centered spinner. **I did not run the app in a browser** to confirm why it appears top-left in the reported view.

The router resolves routes synchronously and swaps page components directly; it has no navigation-wide loading state ([`router.ts`](/workspace/web/src/app/router.ts:73), [`router.ts`](/workspace/web/src/app/router.ts:88)). On a protected route after sign-in, navigation does not show the global spinner because the session is already signed in.

**Spinner/loading locations found:**

- **App/session gate:** [`App.svelte`](/workspace/web/src/app/App.svelte:47).
- **Task first load:** [`TaskScreen.svelte`](/workspace/web/src/features/workspace/task/TaskScreen.svelte:68) shows a spinner until `createThread()` receives the task; it displays a missing-task message if the stream closes before one arrives ([`thread.ts`](/workspace/web/src/features/workspace/logic/thread.ts:58)).
- **Project checks:** [`ChecksTab.svelte`](/workspace/web/src/features/workspace/projects/ChecksTab.svelte:74).
- **Project Env variables:** [`VariablesSection.svelte`](/workspace/web/src/features/workspace/projects/env/VariablesSection.svelte:89).
- **Settings data:** [`ActivityTab.svelte`](/workspace/web/src/features/workspace/settings/ActivityTab.svelte:53), [`ConnectedAppsTab.svelte`](/workspace/web/src/features/workspace/settings/ConnectedAppsTab.svelte:47), [`GitHubIntegration.svelte`](/workspace/web/src/features/workspace/settings/GitHubIntegration.svelte:59), [`ModelsTab.svelte`](/workspace/web/src/features/workspace/settings/ModelsTab.svelte:87), [`PersonaTab.svelte`](/workspace/web/src/features/workspace/settings/PersonaTab.svelte:66), [`SpacesTab.svelte`](/workspace/web/src/features/workspace/settings/SpacesTab.svelte:28), [`TeamTab.svelte`](/workspace/web/src/features/workspace/settings/TeamTab.svelte:73), [`UsageTab.svelte`](/workspace/web/src/features/workspace/settings/UsageTab.svelte:31), and [`DatabaseTab.svelte`](/workspace/web/src/features/workspace/settings/DatabaseTab.svelte:58).
- **Other in-progress work:** [`FileView.svelte`](/workspace/web/src/features/workspace/files/FileView.svelte:62), [`ChangesTab.svelte`](/workspace/web/src/features/workspace/task/panel/ChangesTab.svelte:59), and [`ProjectDocuments.svelte`](/workspace/web/src/features/workspace/task/panel/ProjectDocuments.svelte:51) and [line 63](/workspace/web/src/features/workspace/task/panel/ProjectDocuments.svelte:63). Button-level spinners appear in [`ComposerBox.svelte`](/workspace/web/src/features/workspace/components/ComposerBox.svelte:98), [`CheckModal.svelte`](/workspace/web/src/features/workspace/projects/CheckModal.svelte:84), [`DatabaseActions.svelte`](/workspace/web/src/features/workspace/projects/env/DatabaseActions.svelte:9), [`DatabaseSection.svelte`](/workspace/web/src/features/workspace/projects/env/DatabaseSection.svelte:155), and [`SubscriptionDialog.svelte`](/workspace/web/src/features/workspace/settings/SubscriptionDialog.svelte:87).
- **Progress text/dots:** [`RepoPicker.svelte`](/workspace/web/src/features/workspace/projects/RepoPicker.svelte:37) shows loading text; [`TaskThread.svelte`](/workspace/web/src/features/workspace/task/thread/TaskThread.svelte:87) shows “Waiting to start” or “Working on it” with dots. These are task-work states, not page-load loaders.

**Workspace data behavior:** `AppShell` starts a store watcher on mount ([`AppShell.svelte`](/workspace/web/src/features/workspace/shell/AppShell.svelte:34)). The watcher fetches projects, tasks, sites and extensions asynchronously ([`workspace-store.ts`](/workspace/web/src/features/workspace/logic/workspace-store.ts:19)). Most workspace pages render from initially empty stores without a loading placeholder; for example, the project list can show its empty state before that request completes ([`ProjectsPage.svelte`](/workspace/web/src/features/workspace/projects/ProjectsPage.svelte:28)). **Inference:** this can look like a briefly empty page on a slow first request.

Go serves `index.html` for SPA paths with `no-cache`; hashed assets get immutable caching. Missing `/assets/*` remain 404 ([`spa.go`](/workspace/internal/platform/httpserver/spa.go:16), [`spa.go`](/workspace/internal/platform/httpserver/spa.go:50)). The server has no loading-screen behavior; the loader is frontend-rendered.

## Task pin trace

- **Domain:** [`domain/task.go`](/workspace/internal/modules/task/domain/task.go:67) defines `Task` with `ID`, `ProjectID`, `UserID`, status and task data, but no pinned field.
- **Storage:** SQLite creates the `tasks` table in migration [`008_tasks.sql`](/workspace/internal/platform/database/migrations/008_tasks.sql:1). Later migrations add task columns; current selected columns and scanning have no pin field ([`store.go`](/workspace/internal/modules/task/adapter/sqlite/store.go:19), [`scan.go`](/workspace/internal/modules/task/adapter/sqlite/scan.go:33)). `List` filters by `user_id`, optionally by project, and orders by newest activity ([`store.go`](/workspace/internal/modules/task/adapter/sqlite/store.go:92)).
- **API:** authenticated `GET /api/tasks` is registered in [`handler.go`](/workspace/internal/modules/task/adapter/http/handler.go:29). It passes the authenticated user and optional project filter to the service ([`tasks.go`](/workspace/internal/modules/task/adapter/http/tasks.go:65)). The JSON payload contains no pin field ([`tasks.go`](/workspace/internal/modules/task/adapter/http/tasks.go:13)). There is no pin/unpin endpoint.
- **Frontend task list/store:** [`tasks-api.ts`](/workspace/web/src/features/workspace/logic/tasks-api.ts:26) loads the task list. The `Task` type has no pin field ([`task-types.ts`](/workspace/web/src/features/workspace/logic/task-types.ts:95)). `workspace-store.ts` refreshes that list and shares it with screens and the sidebar ([`workspace-store.ts`](/workspace/web/src/features/workspace/logic/workspace-store.ts:20)).
- **Sidebar today:** [`Sidebar.svelte`](/workspace/web/src/features/workspace/shell/Sidebar.svelte:32) includes Recent and Projects sections. [`SidebarRecent.svelte`](/workspace/web/src/features/workspace/shell/SidebarRecent.svelte:8) takes the first nine tasks from the newest-first store and groups them by day. Each task is a whole-link row with status dot, title and relative time ([`SidebarRecent.svelte`](/workspace/web/src/features/workspace/shell/SidebarRecent.svelte:19)). Projects separately show up to eight projects and a “needs you” count ([`SidebarProjects.svelte`](/workspace/web/src/features/workspace/shell/SidebarProjects.svelte:14)).
- **Task row/menu:** [`TaskRow.svelte`](/workspace/web/src/features/workspace/projects/TaskRow.svelte:15) renders a project task as one full-width link with status, title, summary, badge and time. Neither that row nor the sidebar recent row has a menu or pin button. **Inference:** the existing rows are possible places for a pin control, but the sidebar row’s whole-link structure would need a separate control that does not accidentally activate navigation.

**Pin scope and smallest data change — inference:** Tasks are already stored with both `user_id` and `project_id`, and normal listing is scoped to the signed-in user. A pin should therefore be per user and per task. If a task remains owned by exactly one user as it is today, a `pinned` boolean column on `tasks` is the smallest storage change; a timestamp is only needed if pin ordering or pin time matters. The change would also need to carry the field through domain, SQLite scan/list, API payload, frontend type/store, and an authenticated state-changing endpoint. The repository’s 200-line source-file limit and feature import boundaries apply ([`AGENTS.md`](/workspace/AGENTS.md), “Size and complexity limits” and “Folder structure”).

## Design references

- WAI-ARIA says `aria-busy` can mark a region that is being updated; WAI’s status-message technique recommends `role="status"` for messages that need announcement without moving focus. The current app gate uses `aria-busy` but has no status text/role. [WAI-ARIA 1.2](https://www.w3.org/TR/wai-aria/), [WCAG technique ARIA22](https://www.w3.org/WAI/WCAG22/Techniques/aria/ARIA22.html).
- daisyUI offers both a `loading` spinner (`loading-spinner`) and animated `skeleton` placeholders for component-shaped loading states. [Loading](https://daisyui.com/components/loading/), [Skeleton](https://daisyui.com/components/skeleton/).
- MDN documents `prefers-reduced-motion: reduce` for reducing or removing nonessential animation. [`app.css` currently limits its page entrance animation to `no-preference`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/%40media/prefers-reduced-motion); the daisyUI loading animation’s reduced-motion behavior was not verified here.

## Not found

- No loading UI in the HTML shell or Go SPA handler.
- No global route-transition loader.
- No task pin state, endpoint, sidebar pin list, or task-row menu.
- The source specifies a centered global spinner; the reported top-left placement was not reproduced in a browser.

## Suggestions

A loading design can address two distinct waits: session resolution before the workspace appears, and task/project data loading after the shell appears. The current local spinner patterns can be reused; daisyUI skeletons are available for layout-shaped placeholders.

For pinning, keep ownership tied to the current task owner and expose pin state through the existing task list API so the sidebar can render pinned items from the shared store. Confirm whether pinned tasks should replace or supplement the existing nine recent tasks. No code was changed and no project commands or tests were run.

---

## Summary

The reusable brand mark is already a shared Svelte component, `LogoMark.svelte`. The task store and migration runner show where the pin timestamp must flow.

Repository facts below are from files read; recommendations are marked as suggestions. No files were changed and no project commands or tests were run.

## Design references — Designer/Developer

- **Loading announcement:** WAI-ARIA defines `role="status"` as a polite live region; WAI recommends `aria-atomic="true"` when the full message should be announced. Keep “Memuat...” available as text, with the mark decorative. Sources: [WAI-ARIA status role](https://www.w3.org/TR/wai-aria/#status), [WAI technique ARIA22](https://www.w3.org/WAI/WCAG22/Techniques/aria/ARIA22.html).
- **Spinner vs skeleton:** A spinner suits the session gate, where the whole app is waiting. For projects and task lists, daisyUI’s `skeleton` is a placeholder for component-shaped content; use rows with dimensions close to the content they replace. Sources: [daisyUI Skeleton](https://daisyui.com/components/skeleton/), [daisyUI Loading](https://daisyui.com/components/loading/).
- **Avoiding layout shift:** The repo currently shows `EmptyState` when the projects array is empty ([`ProjectsPage.svelte`](/workspace/web/src/features/workspace/projects/ProjectsPage.svelte:28)) and shows an empty task state when a project has no tasks ([`TasksTab.svelte`](/workspace/web/src/features/workspace/projects/TasksTab.svelte:39)). Show skeletons until the initial fetch settles, then choose between rows and a true empty state. **Suggestion:** match skeleton row count and height to the eventual list to reduce page movement.
- **Toggle beside a link:** The sidebar task row is currently a single anchor ([`SidebarRecent.svelte`](/workspace/web/src/features/workspace/shell/SidebarRecent.svelte:21)). Put a native `<button>` as a sibling control inside the list item, outside the anchor; do not nest it in the link. Use a stable accessible name such as “Pin task” / “Unpin task”; `aria-pressed` is suitable if the name stays stable and it is presented as a toggle. [MDN: `aria-pressed`](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Attributes/aria-pressed). **Suggestion:** stop the button’s click from reaching any row-level navigation handler if the implementation uses one.
- **Reduced motion:** MDN documents `prefers-reduced-motion: reduce` for reducing or removing nonessential motion. Disable the loader animation under this preference; consider disabling skeleton shimmer while keeping static placeholders visible. [MDN: `prefers-reduced-motion`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/%40media/prefers-reduced-motion).
- **Brand asset:** Reuse [`LogoMark.svelte`](/workspace/web/src/shared/ui/LogoMark.svelte:9); it is an inline SVG using semantic `primary` and `primary-content` classes, and is already imported by [`BrandLink.svelte`](/workspace/web/src/shared/ui/BrandLink.svelte:2).

## Technical references — Developer

- **Migration runner:** [`migrate.go`](/workspace/internal/platform/database/migrate.go:19) documents migrations as append-only and verifies applied-file checksums. It sorts embedded migration filenames, then applies each unapplied migration and journal record inside a transaction ([lines 35–53]( /workspace/internal/platform/database/migrate.go:35), [115–149]( /workspace/internal/platform/database/migrate.go:115)). Add a new numbered file; don’t edit an applied migration.
- **SQLite column:** SQLite allows `ALTER TABLE ... ADD COLUMN pinned_at INTEGER` as a nullable column. A newly added column is `NULL` for existing rows; SQLite disallows adding `NOT NULL` without a non-NULL default, and disallows expression defaults such as `CURRENT_TIMESTAMP`. [SQLite `ALTER TABLE`](https://sqlite.org/lang_altertable.html#altertabaddcol).
- **Time scan:** Current task scans read integer Unix seconds for `created_at` and `updated_at`, then use `time.Unix` ([`scan.go`](/workspace/internal/modules/task/adapter/sqlite/scan.go:33)). **Suggestion:** scan nullable `pinned_at` into `sql.NullInt64`; when `Valid`, convert with `time.Unix(value, 0)` and assign a `*time.Time` or equivalent nullable domain representation. Keep the added column in sync with `taskColumns` ([`store.go`](/workspace/internal/modules/task/adapter/sqlite/store.go:19)) and scan destination order.
- **Pin operation semantics:** Current `Find` can scope by both task ID and user ID, returning `domain.ErrNotFound` for no match ([`store.go`](/workspace/internal/modules/task/adapter/sqlite/store.go:77)). **Suggestion:** make PUT set `pinned_at` only when currently `NULL` (so repeated pin does not reorder), and DELETE set it to `NULL`; scope both writes by task ID and user ID. This gives idempotent behavior and avoids disclosing another user’s task. `PinnedAt` must travel through the domain task ([`domain/task.go`](/workspace/internal/modules/task/domain/task.go:67)), SQL selection/scanning, and list response.

## Quality checks — QA/Reviewer

- Verify signed-in owner can pin and unpin; unknown IDs and another user’s task return 404 and do not change state.
- Repeat PUT and DELETE calls; confirm they succeed consistently, keep the expected state, and repeated PUT does not unexpectedly change pin order.
- Use Tab and Enter/Space to operate the pin button. Confirm focus is visible and the accessible name and pressed state match pin status.
- Click the pin button and confirm the task route does not open; separately verify the row link still opens the task.
- Delay or throttle the initial projects/tasks requests. Confirm skeletons show until the first fetch settles, including on fetch error, and “no projects” / “no tasks” does not flash before the response.

## Not found

No existing migration 048, `pinned_at` field, pin endpoint, or pin button exists in the checked files.

## Suggestions

Keep the loading status readable without animation, reserve list space with skeleton rows, and use `LogoMark.svelte` for consistent branding. Use a nullable Unix-time column and user-scoped, repeat-safe updates for pin state.