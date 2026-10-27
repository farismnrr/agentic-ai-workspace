# # MASTER IMPLEMENTATION — Unified External Agent Workflow, R…

Task: 412f2a8ab8da57c5bab55620f9f1f4b6
Status: working
Updated: 2026-10-27T15:33:52Z

Reference material; confirm applicability before reusing.

Scope: large. It touches 8 areas across Go and Svelte, adds migrations, a new permission and a new approval-to-handoff path that must be safe and durable. It's split into 9 slices that each build alone. Nothing was changed.

**I could not show the person the form.** I have no `ask_user` tool and can't talk to them. I took your four recommended answers as defaults. The orchestrator should confirm them at "Person's yes" (see Open questions).

# PRD: Unified External Agent Workflow

## Overview
- **Problem:** External assistants (ChatGPT, Gemini, other MCP clients) can start work with no role. A saved plan leaves the task stuck on "Working on it". Nobody can approve the plan or hand it to the internal Orchestrator.
- **Goal:** Researcher → Planner → human approval → Masih Awam Orchestrator → internal team. The Orchestrator alone does the implementation.
- **Stack:** Go Fiber v3, Svelte 5, strict TypeScript, daisyUI, SQLite. This is the existing stack, kept as is. It's not a new web product, so landing, login and dashboard do not apply.

## Goals and non-goals
**Goals**
- Status is readable without color.
- Every new tracked external work has a validated role.
- Planning has its own lifecycle and is no longer a generic "working".
- Research comes before substantial plans.
- Questions and answers persist.
- One approval service, used by both the app and MCP, starts exactly one Orchestrator run.

**Non-goals**
- A second orchestration engine.
- Internal agents spawned from external role labels.
- A new task status.
- MCP Events or webhooks, and MCP Tasks or elicitation. The pinned SDK v1.8.0 is unverified for these, so state is fetched on the client's next tool call.
- Changing `Accept` (it still means "accept reviewed work").
- Claiming private thinking or streamed tokens from ChatGPT or Gemini.
- A second server, port or realtime channel.
- A dark theme.

## Assumptions (defaults for your four form questions)
1. **Approval permission:** new scope `coding:approve` and tool `approve_plan`, with a Settings switch "Approve plans". It is off by default and not tied to the `log_message` switch. `screenshot_session` is the precedent (`internal/modules/coding/domain/tools.go:146`).
2. **No new task status:** the seven stay. A separate plan state drives a clear label, such as "Ready for approval".
3. **Old clients:** `log_message` without a `work_id` stays record-only. It never enqueues anything, and its reply says to use `begin_work` with a role.
4. **Research gate:** at least one saved research note with a source, or a visible "small task" flag set by the client.

Other defaults:
- A research note must also reach a minimum length, which the Architect sets, so placeholders don't pass.
- Role changes never change permissions.
- Roles `lead` and Orchestrator are rejected for external work.
- Plan states are `draft`, `researching`, `planning`, `awaiting_clarification`, `ready_for_approval`, `approved`, `handed_off`, `rejected`, `superseded` and `cancelled`. The Architect trims this to the smallest valid machine.
- Migrations are new numbered files from `058`, never edited. `051` already exists twice, so use the next free number.

## Real code paths (checked in the repo)
- **Backend work and roles:**
  - `internal/modules/task/application/`: `works.go` (180 lines), `work_roles.go` (148), `work_plan.go` (111), `work_complete.go` (162), `work_state.go` (127), `work_documents.go` (129), `external_chat.go` (108), `researchfirst.go`, `actions.go` (150).
  - `internal/modules/task/domain/work.go`, `event.go`, `task.go`, `plan.go`.
- **Persistence and migrations:**
  - `internal/modules/task/adapter/sqlite/work_store.go`.
  - Migrations in `internal/platform/database/migrations/`; the latest is `057_linked_task_chat.sql`.
- **MCP:**
  - `internal/modules/coding/adapter/mcp/`: `tools_work.go`, `work_inputs.go`, `tools_log.go`, `guard.go`, `guard_work.go`.
  - `internal/modules/coding/domain/tools.go`, `switches.go`.
  - `internal/bootstrap/coding.go` (`codingScopes`, `codingSpace`).
- **Frontend:**
  - `web/src/features/workspace/content/status.ts`.
  - `components/StatusDot.svelte`, `StatusBadge.svelte`.
  - `shell/SidebarTaskRow.svelte` (still draws a colored dot only for needs-you or working).
  - `task/thread/TaskThread.svelte`, `EventItem.svelte`, `RoleCard.svelte`, `QuestionCard.svelte`, `PlanCard.svelte`, `HandoffCard.svelte`, `ApprovalCard.svelte`.
  - `task/panel/ExternalTeam.svelte`.
  - `settings/SettingsPage.svelte`, `SettingsNav.svelte`, `settings-nav.ts`.
  - `tasks/ArchivePage.svelte`, `extensions/ExtensionsPage.svelte`.
  - `web/src/app/router.ts`, `App.svelte`.
- **Size limit:** `works.go`, `work_complete.go` and `work_roles.go` are close to 200 lines. New logic goes in new files named for their job (for example `work_questions.go`, `plan_approval.go`, `handoff_outbox.go`), not appended to those.

## Todo
The Architect designs first, before slices 4 to 6, and the others can proceed. The Architect's design covers:
- Plan state machine and immutable plan-revision snapshot (`task.Plan` is mutable today).
- Questions table.
- Approval record, with unique keys on work + revision and on the idempotency key.
- Handoff outbox with a state, an attempt count and a unique link to the internal task.
- A startup sweep that re-enqueues pending handoffs. The scheduler queue is in memory and recovery only handles running tasks.
- Which task status shows during `ready_for_approval`. Suggested: `needs-input`, so the `Accept` meaning of `review` is not reused.

Every slice below follows the same pattern. It ends with `make verify` green and its own commit. The Developer writes the tests named in that slice. QA walks the result.

1. **Developer: status icons everywhere.**
   - **Outcome:** `SidebarTaskRow` shows `StatusDot` (icon plus accessible name) for every status.
   - **Plus:** check `TaskThread` for the working indicator, and make the loading dots `motion-safe`. The working indicator also hides when a persisted work summary is terminal or stale.
   - **Tests:** `status.test.ts` (one distinct icon and label per each of the 7 statuses) and a pure-logic test for the "show working indicator" rule.
   - **Verify:** the sidebar shows an icon for queued, working, needs-input, review, done and failed. Reduced motion stops the spin.

2. **Developer: mandatory initial role and legacy handling.**
   - **Outcome:** `begin_work` requires a valid `role`. Work, role and `EventRole` are created in one transaction.
   - **Rejected:** missing, unknown, disabled and Orchestrator roles, each with a clear error.
   - **Legacy:** `log_message` without `work_id` stays record-only and its reply points to `begin_work`. The reply also lists the enabled role keys.
   - **Tests:**
     - `works_begin_test.go`: missing, unknown, disabled and lead roles are rejected; the role event is written atomically with the work.
     - `works_isolation_test.go`: concurrent sessions stay isolated.
     - `tools_log_test.go`: legacy path writes no executable task.
     - `work_tools_integration_test.go`: the in-memory MCP client sees the new schema.
     - Role change does not change permissions.
   - **Also:** verify (do not enforce) that file and command tools have no role dependency, and report it.
   - **Verify:** old histories still open.

3. **Developer: Settings layout for Archive and Extensions.**
   - **Outcome:** the Settings sidebar stays on `/app/archive`, `/app/extensions` and `/app/extensions/:projectId`, with the right item highlighted.
   - **Approach:** extract the shell and `SettingsNav` frame from `SettingsPage` into one shared Settings frame, and use it from both pages. URLs do not change and the nav is not copied.
   - **Tests:** extend `settings-nav.test.ts` and a router test for the active item and the project-scoped route.
   - **Verify:** QA checks desktop, tablet and mobile (no overflow, no double sidebar), refresh, deep link and back/forward.

4. **Developer: plan lifecycle and notifications** (after the Architect's design).
   - **Outcome:** migration `058` and domain plan state. `update_plan` only saves. A new `submit_plan` moves to `ready_for_approval`. The thread shows "Ready for approval", not "Working on it".
   - **Notifications:** task creation, role change, research, plan revision and state change each call the hub.
   - **Tests:**
     - State-machine unit test for valid and invalid transitions.
     - Plan saved while open stays "planning".
     - `submit_plan` makes the state ready, then a new revision supersedes it.
     - Blocked and resumed.
     - Missing completion leaves the work stale, not done.
     - Duplicate or late events do not regress state in the `thread.ts` merge test.
     - A hub drop is recovered by SSE heartbeat re-read.
   - **Verify:** a plan is saved in a test client and the UI label changes without refresh.

5. **Developer: research gate, questions and answers.**
   - **Outcome:**
     - Migration `059` and new tools `finish_research`, `ask_question` and `answer_question` (user, in the app).
     - `get_work_state` returns open and answered questions.
     - `submit_plan` is refused without a sourced research note, unless the client set `small_task` (shown in the UI as "Small task").
     - The question card appears with status `needs-input`.
   - **Tests:**
     - Gate matrix (note and source, note only, none, `small_task`).
     - Appended research rounds without duplicates on retry.
     - Question ID, owner and project binding.
     - Answered questions are not asked again.
     - Cross-user and cross-project answers are rejected.
     - Concurrent answers: one wins.
     - Reconnect keeps pending questions.
   - **Verify:** QA answers a question in the app and reads it via `get_work_state`.

6. **Developer (backend): approval service, scope and durable handoff** (after the Architect).
   - **Outcome:** migration `060`.
     - **Scope and tool:** scope `coding:approve` is added to `codingScopes`, and `approve_plan` to `tools.go` and `switches.go`. It is off by default.
     - **Approval record:** it binds user, project, task, work, plan revision, source (`app` or `mcp`) and idempotency key. A stale or superseded revision and an unverified user are rejected.
     - **Handoff:** one transaction writes the approval and an outbox row. A dispatcher creates the internal Orchestrator task through the existing `ChatOnExternal` path, with research, requirements, answers and the plan as untrusted context. A startup sweep retries pending rows.
     - **`Accept`:** it is untouched.
   - **Tests:**
     - Scope and switch: a `coding:read` token cannot approve, and a disabled switch cannot approve.
     - Authorization, wrong project and wrong user.
     - Stale revision.
     - Repeat approval returns the same result.
     - Two concurrent approvals produce one internal task.
     - Timeout then retry.
     - Dispatch failure then recovery.
     - Restart with a pending outbox row.
     - A plan edit after approval does not alter the approved snapshot.
     - External `complete_work` cannot mark the internal task done.
     - `external_report` trust boundary is kept.
     - Migration applies on a copy of an existing database.

7. **Developer (UI and MCP): approve and revise in the app and via MCP.**
   - **Outcome:**
     - A "Ready for approval" card shows the exact revision with Approve and Request changes buttons, calling the same service.
     - A "Handed off" card links to the internal task, and the internal task links back.
     - `approve_plan` and `get_work_state` expose decision and link. The Settings tools list shows "Approve plans".
     - Request changes marks the revision superseded and back to planning.
   - **Tests:** HTTP handler test for the app path (session and CSRF), MCP integration test for the tool path, a test that both paths give the same handoff reference, and frontend logic tests for card state.
   - **Verify:** QA runs scenarios A to F from the request, including an approval from both sides at once.

8. **Developer: timeline provenance.**
   - **Outcome:** the event timeline labels who did what. The labels are "Performed by ChatGPT/Gemini" (assistant-reported), "Recorded by Masih Awam", "Approved by you" and "Executed by the Orchestrator".
   - **Rules:** planner activity never shows "Proven to work". Only server-observed evidence shows "verified".
   - **Tests:** a provenance-label logic test per event type. The `EvidenceBadge` rule is checked (claimed, observed, ran).
   - **Verify:** QA walks the whole external flow in the app.

9. **Developer: docs and agent memory.**
   - **Outcome:** people docs updated in short, scannable lines (no line over about 45 words): `docs/connector.md`, `docs/system-design.md`, `docs/using-the-app.md`, `docs/design-system.md`.
   - **Content:** the new tools and permission, and the honest platform limits (no unsolicited pushes to ChatGPT or Gemini, state is read on the next call, MCP Events and elicitation not used).
   - **Also:** `AGENTS.md` and the right `agents/` files get the durable rules (role required, `coding:approve` separate, outbox handoff, no new task status).
   - **Check:** run `scripts/check-guardrails.sh`.

## Done when
1. `make verify` passes after the last slice.
2. Fast CI is green on the pushed commit, then `make redeploy-dev` is run and `/api/health` on port 8081 answers.
3. QA reproduces scenarios A, B, C, D and F with one approved revision, giving exactly one Orchestrator task. Archive and Extensions keep the Settings sidebar, and statuses are understandable without color.

## Delivery rules (for every slice and the end)
- **Per slice:** `make verify`, then commit with explicit file paths only (`git add <paths>`, never `-A` or `-a`).
- **Never stage:**
  - The six modified files under `project-notes/tasks/a4c3f733b96d6e0396ac72e2c102a4eb/`.
  - The server-maintained `project-notes/tasks/412f2a8ab8da57c5bab55620f9f1f4b6/` files.
- **Before each commit:** check `git status` and `git diff --cached --stat` for those paths.
- **Push and CI:** push to `dev` normally, with no force, and wait for the Fast run on that commit. Fix it if red.
- **Redeploy:** run `make redeploy-dev` once, after slice 9 is green. Do not run a plain `docker compose pull` or pull the sandbox image.
- **Docs-only slice:** slice 9 changes no Go or web files if done last alone, so Fast skips it. Push it and stop there.
- **Developers:** read `agents/git-safety.md`, `agents/memory-backend.md` and `agents/memory-frontend.md` first. The Researcher's notes cover the web research the repo rules require before the first code edit.

## Open questions
- **Not asked yet:** the person has not seen the form, so the four defaults above are my assumptions. The orchestrator should ask before building, in plain words, with the recommended option first:
  1. Add a separate, off-by-default "Approve plans" permission for ChatGPT/Gemini? (Recommended: yes.)
  2. Add a new task status for planning? (Recommended: no, use a separate plan label.)
  3. Keep old clients that record without a role as record-only? (Recommended: yes.)
  4. Require at least one sourced research note before a plan can be finalized, with a visible "small task" exception? (Recommended: yes.)
- **No other blockers.** Whether ChatGPT or Gemini can call `approve_plan` through their own OAuth is unverified. The server side is built and tested in-memory only, and docs will say so.