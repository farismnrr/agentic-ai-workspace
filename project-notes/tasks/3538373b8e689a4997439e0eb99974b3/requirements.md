# P0 — Investigate External Assistant Lifecycle Desynchronizat…

Task: 3538373b8e689a4997439e0eb99974b3
Status: working
Updated: 2026-10-28T04:27:50Z

Reference material; confirm applicability before reusing.

Scope: **medium to large** (one migration, one new background job, one new endpoint, a UI status rework, plus tests and docs). I'm treating it as large and splitting it into small slices. Nothing here is confirmed against a real incident, so the fix is built to work whichever root cause turns out to be real.

Out of scope:
- Any change to migrations 001–060.
- Marking work "done" or "proven" only because it went quiet.
- Changing `recovery.go`'s skip of external work, or wrapping the host assistant itself.
- A dark theme, new colours, or new routes or ports.
- Backfilling old rows beyond what the first sweep does.

Open question decisions I'm taking as defaults, none of them blocking:
- Valid `WORK_ABANDON_AFTER` range is 1h–3h, guidance only. Config clamps to a minimum of `WORK_IDLE_AFTER`+30m and logs a warning outside 1h–3h.
- Reported work ended by the sweep goes to "Ready for your review" (task `review`). Work with no report goes to failed "Lost contact".
- Two new migration-free defaults: `WORK_IDLE_AFTER=30m`, `WORK_SWEEP_EVERY=1m`.
- Only migration 061 is new. It is the next free number: I checked, and 060 is the latest.

Stack: Go Fiber v3 with SQLite and the Svelte SPA (existing stack, kept). Bash black-box tests and Playwright, as the repo already uses.

Done when:
1. `test/blackbox/07-external-work-lifecycle.test.sh` passes and covers: log_message with no complete_work gives "Reported, not confirmed"; Confirm gives done with the "Marked complete by you…" receipt and `proven` false; a repeated complete_work succeeds; dry-run only logs, while `on` abandons after the window (reported gives review, unreported gives failed "Lost contact", waiting-on-person is untouched).
2. The Playwright flow shows a stale or reported task with no "Working" indicator, a "Mark complete" button and a Stop button. After Mark complete the task shows Done and not Proven.
3. `make verify` is green, Fast is green on the pushed commit, and `/api/health` on 8081 answers after `make redeploy-dev`.

Todo (one Developer delegation each, in order; every step leaves the app working):

1. **Developer: store the "reported" fact.**
   - Add migration `061_work_reported.sql` (nullable `reported_at` on `external_works`), the domain `DeriveWorkPhase` (terminal > awaiting person > blocked > reported > idle > running), and the four `WORK_*` settings in `platform/config` (default `WORK_SWEEP_MODE=dry-run`, `WORK_ABANDON_AFTER=2h`).
   - Make `UpdateWorkCall` bump the work's `updated_at`.
   - Use the injected clock in the store.
   - Verify: the app starts and migrates a copy of the existing database. Existing black-box suites still pass.
2. **Developer: assistant report sets `reported_at`.**
   - `log_message` with a `work_id` sets `reported_at`; any other activity clears it.
   - `complete_work` succeeds on a repeat of the same outcome whatever the summary (`already:true`), with clearer `ErrWorkClosed` and `ErrAwaitingPerson` text.
   - The MCP instructions now mention `complete_work`.
   - Verify: calling the MCP tools via curl gives the expected results and error text.
3. **Developer: the reconciler, in dry-run.**
   - Add `Works.Reconcile` and `RunReconciler` (startup and every minute, injectable clock, modelled on `handoff_dispatch.go`), wired in bootstrap.
   - It interrupts orphaned running calls, abandons after `WORK_ABANDON_AFTER`, skips waiting-on-person work, uses `Store.Commit` with a revision check, and calls Notify on each change.
   - `dry-run` only logs what it would do; `on` acts; `off` does nothing.
   - `Begin` keeps only `abandonIdleAtCap`.
   - Verify: with an aged work row, dry-run logs and changes nothing. With `on`, reported work goes to review and unreported work goes to failed "Lost contact".
4. **Developer: the Mark complete endpoint.**
   - Add `POST /api/tasks/:id/confirm` (`Service.Confirm` calling `Works.Confirm`). It is allowed only in the reported or idle phase, sets completed and done with the "Marked complete by you after the assistant reported its result" receipt, and leaves `proven` false.
   - It is idempotent, and returns 409 for stopped or abandoned work or a stale revision.
   - `WorkSummary` gains `phase` and `reportedAt`.
   - The SSE heartbeat pushes a snapshot when the phase changes.
   - Verify: curl the endpoint for the 200, repeat and 409 cases.
5. **Developer: the web status display.**
   - Add `work-phase.ts` returning `{label, hint, showWorking, canStop, canConfirm}`. `TaskHeader`, `TaskThread` and `working-indicator` all use it.
   - "Reported, not confirmed" gets Mark complete and Stop. Stale work gets the static text "No updates for over 30 min" and no Working spinner.
   - Use only existing daisyUI classes and theme roles. Files stay under 200 lines.
   - Verify: `svelte-check` and ESLint are clean, and the screenshots show the three states.
6. **Developer: tests, docs and shipping.**
   - Add `test/blackbox/07-external-work-lifecycle.test.sh`, one Playwright flow in `test/e2e/`, and a short mention of the new test in `agents/testing.md`. No white-box tests.
   - Update `docs/` (what the statuses mean, the settings table and the rollout switch), `agents/memory-backend.md` and `agents/memory-frontend.md`, and `docs/system-design.md` if folders moved.
   - Then run `make verify`, commit, push to `dev`, wait for the Fast run (fix and re-push while it is red), run `make redeploy-dev` and check `/api/health` on 8081.
   - Verify: Fast is green and health answers.

Rollout (person-controlled, after step 6):
- Deploy with `WORK_SWEEP_MODE=dry-run`, the default. Read the logs for a day or two to see which tasks would be touched.
- If the list looks right, set `WORK_SWEEP_MODE=on` and redeploy.

Rollback:
- Set `WORK_SWEEP_MODE=off` and redeploy. That stops all automatic ending.
- Migration 061 only adds a nullable column and can stay. No data is deleted.
- Work the sweep already ended stays ended, but a failed "Lost contact" or review task can be re-run by the person.

Open questions: none that block the work. I took one assumption you can overrule: reported-then-cleaned-up work lands in "Ready for your review", as you didn't state it explicitly. I changed no files.