# # Implementasi External Multi-Role Workspace

Task: fae3e8b731d9ae374acd8b2a70ea8ca1
Status: done
Updated: 2026-10-27T03:31:01Z

Reference material; confirm applicability before reusing.

**Scope:** Large. It adds a new data model, 8 MCP tools, a lifecycle with retries, a new tab in the web app, and changes to the sweep, requeue and recover paths. Every slice must leave `make verify` green.

**Out of scope:**
- Deleting or cleaning anything in `/workspace`, including the untracked user files, `.stage7/` and the extra worktrees.
- Changing the MCP permission model. Roles are labels only, and the coding guard's scopes stay authoritative.
- Dark theme, new ports, new runtime servers, and new libraries.
- Live testing against ChatGPT or Gemini clients. Instructions are guidance and cannot prove a client followed them.
- Changing how legacy calls without `work_id` behave. They keep the 30-minute grouping.

**Done when:**
1. `make verify` passes, and Fast CI is green on the pushed `dev` commit. Tests cover:
   - two separate works from one client
   - an idempotency-key retry returning the same answer, and a mismatch being rejected
   - a wrong-owner or wrong-project `work_id` being refused
   - stable plan-step IDs
   - research and PRD exported separately
   - requeue, recover and cancel handling works correctly
2. After an external assistant calls `begin_work`, `set_role`, `update_plan`, a file edit and `complete_work`, the task shows these tabs:
   - Plan, with its steps
   - Documents, with research and PRD
   - Team, with only the roles actually used
   - Changes, with the file edit and a working Undo
   - Evidence, which tells self-reported progress apart from server-observed file changes and real exit codes

   The screenshots at 390, 820 and 1024 px have no sideways scroll.
3. `docs/`, `agents/` and `AGENTS.md` describe the feature. Calls without `work_id` still group as before.

**Todo (build order, one per slice):**
1. **Developer: domain model and rules.** This covers the work, role, evidence and idempotency types, with unit tests. Verify: `go test` on the domain package and `make verify`.
2. **Developer: migration 055 and the SQLite work store.** It also refactors `insertEvent` for atomic save-plus-event. Verify: store tests, including migration on an existing database, and `make verify`.
3. **Developer: Begin, SetRole and State, plus rate limit.** Verify: application tests for retries, key mismatch, wrong owner, and concurrent calls.
4. **Developer: UpdatePlan, documents, progress, complete and sweep, plus service guards.** The guards are:
   - requeue refuses tasks with `Source != ""`
   - recover skips works
   - cancel stops the work
   - `session()` skips works

   Verify: tests for stable step IDs, separate research and PRD export, idle shown after 30 minutes, and truthful blocked, failed and completed states.
5. **Developer: the 8 MCP tools under scope `coding:read`.** They map to the `log_message` switch, with optional `work_id` on existing tools, plus server instructions and the bootstrap bridge. Verify: MCP adapter tests for tool switches, scope, audience, oversized input, and legacy calls unchanged.
6. **Developer: linking file changes and undo to work events.** It adds `guard_work.go`, one event per command, and undo ownership with no duplicate undo records. Verify: a test that an external edit appears in Changes with one undo record, and that Undo works once.
7. **Developer: web.** This is `ExternalTeam.svelte`, `RoleCard`, `work-roles.ts` and the header changes, using daisyUI and the existing theme. Verify: `svelte-check` and ESLint pass, plus 390, 820 and 1024 px screenshots. QA then walks external and internal tasks across all tabs, and Security checks the work-ID, owner and secret handling.
8. **DevOps: docs and delivery.** Update `docs/`, `agents/` and the `AGENTS.md` rule. Then run `make verify`, commit, push fast-forward to `origin/dev`, and wait for Fast CI. Redeploy with `make redeploy-dev` and confirm `/api/health` on 8081 only once CI is green. Verify: green CI run on the commit, and a health check response.

**Assumptions:**
- Stack: Go, Fiber and Svelte, kept as is.
- Team shows only used or declared roles.
- Idle means 30 minutes without activity.
- Undo is offered as for internal tasks.
- `work_id` is issued by the server. The caller supplies an idempotency key.
- A new detached worktree from `origin/dev` ebc4751 is created outside `/workspace`. The 6-commit gap and the unclean primary worktree are left untouched, and the push is fast-forward only.
- The Researcher's findings are cited from the team notes. I did not read the spec parts or run any code.

**Open questions:** None that block the work. The worktree handling follows the safe default above, so the person only needs to approve the plan.