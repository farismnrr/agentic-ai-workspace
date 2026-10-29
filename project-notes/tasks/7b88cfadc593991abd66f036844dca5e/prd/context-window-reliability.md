# PRD — Reliable Context Window & Automatic Compaction

## Problem
Context meter may show a full window indefinitely, especially on resumed Codex/Claude sessions. The API model may receive an oversized prompt before compaction, and failure handling has been reactive. Prior commit 9ec545f fixes safe API preflight and cumulative Codex usage display but lacks dedicated end-to-end coverage.

## Scope
- API preflight compaction for initially full/resumed transcripts, including oversized tool output, failed summary, retry, and truthful meter.
- Codex/Claude plan runtime: consume reliable *per-call* usage where available, do not label cumulative usage as active context, update the meter while streaming, and recover native context-full failures without unsafe infinite retries.
- UI: distinguish measured usage, unknown usage, compacting, and failed compact; never fake a reset or hide an actual failure.
- Preserve existing session state, task history, approval flow, and user work.

## Acceptance criteria
1. Initial context >= threshold triggers compaction before normal API model call; summary input fits model window and preserves recent user instructions.
2. Summary failure/overflow yields bounded retry or actionable error, not repeated identical oversized requests.
3. Context meter reports active-context occupancy only; cumulative usage must never render as 100% occupancy.
4. CLI progress updates context as trustworthy usage events arrive; unavailable usage is explicitly unknown, not stale red.
5. Native CLI full-context case follows supported vendor recovery or explicit actionable failure, with no silent session deletion.
6. Black-box E2E reproduces initial-full, compact success/failure, resumed session and meter; regression suite remains green.
7. Changes reviewed, PRD and test evidence linked, fast verification, push to dev only after passing relevant checks.

## Non-goals
- Full unit-test expansion, blanket CI redesign, vendor-private session manipulation, or fabricated context usage.

## Traceability
- PRD -> research -> plan -> implementation -> black-box evidence -> review -> commit.
- Previous partial fix: 9ec545f. New implementation must explicitly report any remaining unverified vendor behavior.

## Implementation and verification — 2026-10-29

Implemented on `dev`:

- API model: preflight at resume when transcript is already full; bounded, non-executable summary input retains original and latest instructions; summary failure is fail-closed and records `failed`, never replays a huge prompt.
- Overflow recovery: one bounded summary and one reduced retry, explicit failure if model refuses again.
- CLI sessions: measured-full and failed resume safely fork to a new vendor session without deleting old session; a context-limit failure retries at most once and only if no tool/command/file effect started. Task history carries forward. A fresh session can be saved only after success.
- Context meter: schema migration `064_task_context_state.sql` adds `unknown | measured | compacting | failed`; API and SSE expose states; UI displays each truthfully. Unknown vendor context does not masquerade as 0% or 100%.
- Usage: Claude last assistant model selection and per-call tokens; Codex last-call token event (not cumulative); updates during streaming when vendor data includes a trustworthy model window.

### Evidence / traceability

| AC | Test / fixture | Status |
| --- | --- | --- |
| Initially-full resume preflight | `go test ./test/go/app/tasks/execution -run TestInitiallyFull -count=1` (app API + scripted local model server + database) | PASS |
| Compaction input bounded and not executable | `go test ./internal/modules/task/application -run TestBoundedSummary -count=1` | PASS |
| CLI native error recovery safety / bounded retries | `go test ./internal/modules/task/application -run TestCLI -count=1` (scripted vendor-runner boundary) | PASS |
| Codex / Claude streaming usage and effects | `go test ./test/go/modules/clistream -count=1` (vendor output fixtures) | PASS |
| UI measured, compacting, failed and unknown | `bash test/e2e/run.sh context` (real app + Chromium) | PASS |
| Prior user-task lifecycle preserved | `bash test/e2e/run.sh lifecycle` | PASS |
| Relevant modules and API | `go test ./... -count=1 -timeout 120s` | PASS |
| Frontend diagnostics | `cd web && npm run check` | PASS |

**Environment boundary:** The runtime here does not have a Docker daemon or authenticated vendor CLI sessions. Actual live Claude Code/Codex service behavior cannot be attested from this environment; the parser, runner, and browser contracts are verified with repeatable boundary fixtures. A Docker-enabled host should run an additional real-vendor smoke test before labeling vendor service integration externally certified.

The critical task-spam regression is tracked separately in `project-notes/backlog/orchestrator-delegation-task-spam.md`; it is intentionally not part of this fix.
