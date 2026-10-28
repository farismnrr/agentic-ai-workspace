# Implementation Plan — Bubble Lifecycle and Live Git Decorations

Status: approved scope for developer delegation by user on 2026-10-28. This document is a plan only, NOT an implementation record.
Project: agentic-ai-workspace. Target remote branch: dev. Observed snapshot: fa5e610 (recheck before work).
PRD: requirements.md. Research: research.md.
Owner roles: Planner (planning), Architect/Security (review boundaries), Developer (code), QA (black-box E2E), Reviewer (independent review).

## Mandatory preflight before agent changes anything

1. Read AGENTS.md, agents/git-safety.md, relevant agents/ guides, docs/using-the-app.md, and the PRD/research here. Recheck upstream official sources for Git, CodeMirror 6 and SSE before coding.
2. Inspect current branch, origin/dev, git status, worktrees, symlinks and relevant runtime configuration read-only. There were pre-existing user edits under project-notes/tasks/3538373b8e689a4997439e0eb99974b3 (9 modified files); preserve all.
3. Follow repository worktree policy; use isolation only if permitted by the current guardrail policy. Never reset, clean, force push, stash or overwrite user work. Do not commit planning files unless deliberately assigned/approved. If isolation conflicts with one-worktree guardrail, stop for human decision rather than bypass it.
4. Capture actual black-box reproduction of bubble cases, including HTTP/SSE payload progression and browser screenshots, and inspect configured WORK_SWEEP_MODE. Do not call this bug fixed merely because source looks correct.
5. Document test baseline and explicit branch/commit before edits. Do not repeat unrelated expensive tests without reason.

## Stage 1 — P0: canonical chat bubble lifecycle

Owner: Developer (frontend), Architect reviews state contract.
Targets:
- web/src/features/workspace/logic/thread.ts, task-types.ts, work-status.ts, working-indicator.ts
- web/src/features/workspace/task/thread/TaskThread.svelte, EventItem.svelte, ChatBubble.svelte, blocks.ts
- internal/modules/task/adapter/http/stream.go and related events only if reproductions prove server change necessary

Deliverables:
- Document explicit durable-event versus ephemeral-delta state machine; define correlations for turn/generation and final message, atomic replacement and replay-safe reconciliation.
- Ensure persisted assistant messages replace matching draft even if task phase continues running. Reset only that turn, not unrelated subsequent deltas.
- Derive header/thread/sidebar statuses from shared canonical lifecycle logic; keep blocked/idle/reported/awaiting-person distinct.
- Reconcile SSE resume snapshots/version cursor, reconstruct after reconnect, make dropped ephemeral text non-authoritative; maintain scroll behavior.
- Retain external author/role/provenance and internal follow-up routing.
- Do not automatically finish external work on silence; examine 30m idle/2h sweep and mode= dry-run separately. No config flips without authorization.

Exit gate: AC01-AC05 captured by black-box Playwright/browser + API/SSE checks. No duplicate finalized messages, false spinner, lost events or scroll jump. Reviewer signs off.

## Stage 2 — P1: Git status backend (read-only)

Owner: Developer (Go), Security reviewer required.
Targets:
- internal/modules/coding/domain (status contract as needed)
- internal/modules/coding/application (project-scoped Git status/diff abstraction)
- internal/modules/coding/adapter/* (safe Git CLI adapter)
- internal/modules/coding/adapter/http/handler.go, ports.go, focused API routes
- internal/bootstrap/files.go (authorized wiring)

Deliverables:
- Authenticated GET project Git status and per-file diff/base read API, with actual accessible repository root and hard project containment.
- Porcelain parser using -z, no string-parsed shell, fixed argv, cwd scoped to authorized project; index/worktree differentiation, untracked, deletion, rename, conflict, ignored exclusions.
- Strict filename/path validation, repo root enforcement, no access to .git internals or secrets, no write operations, bounded outputs, errors and timeouts.
- Repository absent/HEAD unborn handled explicitly; status snapshot revision and caching/debounce; no unbounded full-tree file content requests.
- DO NOT add stage/commit/discard/reset/checkout endpoints.

Exit gate: AC06-AC11 and AC15 backend observed via authenticated black-box HTTP tests, including cross-user and traversal rejection. Preserve sandbox isolation and existing API access controls.

## Stage 3 — P1: Explorer Git decorations and refreshing

Owner: Developer (Svelte), Designer/Reviewer checks visual consistency.
Targets:
- web/src/features/workspace/logic/files-api.ts, file-store.ts
- web/src/features/workspace/files/FileBrowser.svelte, TreeLevel.svelte
- optional separate focused git-status store/service (do not conflate task history)

Deliverables:
- Live per-file status (M/A/D/R/U + conflict) and aggregated parent-folder decorations; staged/unstaged accessible distinctions.
- No Git decoration for clean path. Suppress ignored entries. Handle deletes/renames even when path not present in directory listing (Changes view should expose deleted paths).
- Reconcile on opening, project file SSE, status panel focus, reconnect and bounded fallback polling; debounce burst events and prevent old responses from winning.
- Compact accessible badges/tooltips for mobile, no excessive visual clutter or decorative always-on success badges.
- Task-scoped change history remains independent and unchanged.

Exit gate: AC06-AC10, AC14-AC15; commit clears all active tree/folder decorations without hard reload.

## Stage 4 — P1: Changed-line gutter for opened files

Owner: Developer (CodeMirror), Reviewer validates performance.
Targets:
- web/src/features/workspace/components/CodeEditor.svelte
- web/src/features/workspace/logic/editor.ts, editor-runtime.ts, editor-theme.ts, file-view.ts
- web/src/features/workspace/files/FileView.svelte

Deliverables:
- On-demand original HEAD content from authorized backend for changed file, diff decoration in existing CM6 editor: added/modified/deleted lines; state-driven gutter updates.
- Support staged+unstaged correctly against documented baseline, plus untracked and initial commits.
- Keep Save / Ctrl+S / conflict flow; when a draft is unsaved, local draft behavior is distinct from saved Git status and no auto-write occurs.
- On commit, reload, file change and editor close, stale markers are removed. No markers on clean file.
- Binary, truncated or huge files degrade gracefully; avoid per-keypress Git calls, re-render storms and editor remount causing cursor loss.

Exit gate: AC06, AC07, AC08, AC09, AC12, AC13, AC14. Visual screenshots show before/after.

## Stage 5 — P1: Changes tab current state versus task history

Owner: Developer (frontend + diff read API).
Targets:
- web/src/features/workspace/task/panel/ChangesTab.svelte, FilesTab.svelte
- web/src/features/workspace/changes/*, web/src/features/workspace/logic/changes-*
- docs/using-the-app.md

Deliverables:
- Current Git changes drive the active Changes list and compare view; staged and unstaged included; external shell edits show up.
- When Git clean, no active files/diff/count are claimed. Task step history (including Undo, evidence) remains visible, labeled as historical, not active Git work.
- For non-repo projects, clearly communicate no Git tracking available; retain historical task action events.
- Source of truth for active state shared with Explorer, avoiding independent inconsistent snapshots.

Exit gate: AC08-AC10, AC15. No loss of task evidence/Undo.

## Stage 6 — black-box QA, review, documentation, release

Owner: QA + independent Reviewer; Developer resolves defects.
- Prefer existing Playwright and HTTP E2E patterns under test/e2e, including test/e2e/04-work-lifecycle.spec.ts; add scenarios for streaming persisted while still running, SSE disconnect/reconnect, idempotent event versions, external phase transitions, status clean after commit.
- Exercise auth negative cases and secret traversal against real app sandbox (container where supported).
- Exercise full Git lifecycle in disposable test repositories only: edit, save, stage, rename/delete, ignored/untracked, commit, clean; no destructive operations against user workspace.
- Desktop/mobile screenshot and interaction validation; smoke project/Files/Changes/Chat/task switching.
- Avoid creating white-box tests unless they guard critical security boundaries that cannot safely be verified externally. Do not delete mandatory existing checks or reduce CI standards.
- Run mandatory make verify/check-guardrails, existing CI and relevant E2E on proper host; do not claim tests pass when unrun. No forced rebuild/test repetition.
- Update docs/using-the-app.md (human readable) and only the relevant agent-only rule if behavior changes. Avoid unrelated formatting churn.
- Code review: concurrency correctness, SSE cursor/delta semantics, Git -z parse, symlink containment, access control, no secrets, diff performance, zero data loss.
- Deliver a concise report with specific test commands/results/screenshots, changed files, commit SHA(s), final Git status, remaining blockers and rollback notes.
- Push to origin/dev ONLY after verified and compatible with repo worktree/dirty-state guardrails, with safe normal fast-forward; never force push.

## Delivery checkpoints / acceptance

Gate A: Stage 1 black-box proof; do not block on visual styling.
Gate B: Stage 2 secure API proof before frontend makes backend assumptions.
Gate C: Stages 3-5 visual and state alignment.
Gate D: Stage 6 whole-feature E2E, security and clean-state evidence; separate reviewer approval.

Implementation agent should stop and report if Git status cannot be safely determined for a given project, if external-work reconciliation could destroy user intent, or if running tests would modify non-disposable user files.

## Suggested work delegation

Developer Agent: implement stages 1-5 in safe, reviewable increments and report verified changes.
QA Agent: independently run black-box cases AC01-AC15; no implementation.
Security/Reviewer Agent: verify data boundaries, status correctness, stale SSE and user changes preservation; no implementation unless independently delegated.
Technical lead (ChatGPT here): audit, approve plan criteria, review evidence; do not write source.
