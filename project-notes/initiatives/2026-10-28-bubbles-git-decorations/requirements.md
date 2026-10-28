# PRD — Consistent Task Bubbles & VS Code-Style Git File Decorations

- Date: 2026-10-28
- Project: agentic-ai-workspace
- Target: dev (read-only audit baseline fa5e610)
- Status: Ready for agent implementation; user authorized starting work
- Owner: Technical lead / Planner
- Delivery: Delegated to developer agents; lead does not implement
- Safety: Do not alter existing user changes in project-notes, worktrees, .env, local artifacts or user data.

## 1. Problem statement

Task chat/bubble rendering is inconsistent during streaming, event persistence, SSE reconnect and external-assistant lifecycle transitions. File Explorer currently marks paths from task action history, not from Git's actual uncommitted state. Consequently badges can survive a clean commit; opened code lacks VS Code-style changed-line gutter decorations. The Changes tab currently uses task-scoped file-tool history instead of the complete repository's working tree. User expects visually consistent activity and live Git state, with all active change decorations disappearing when Git is clean.

## 2. Existing implementation / grounded findings

- web/src/features/workspace/logic/thread.ts: EventSource delta text accumulates in an ephemeral draft; apply() only resets draft when the working indicator becomes false. Correlation to a persisted assistant message is not explicit; investigate duplication across while-working message finalization and reconnect.
- web/src/features/workspace/task/thread/TaskThread.svelte: thread renders persisted blocks and a draft/working indicator independently; the lifecycle/status labels appear before the conversation while working indicator appears after it.
- web/src/features/workspace/task/thread/EventItem.svelte, ChatBubble.svelte and blocks.ts: separate kinds (message/action/role/receipt) and collapsible actions, requiring consistent provenance, order and stable event identities.
- web/src/features/workspace/logic/work-status.ts and internal/modules/task/domain/work_phase.go: external phase derives running/idle/reported/completed; inactivity is not proof of completion. Default idle 30 minutes; abandon 2 hours. Deployment WORK_SWEEP_MODE defaults dry-run, verify effective runtime config.
- internal/modules/task/adapter/http/stream.go: SSE supports event versions and Last-Event-ID for updates, while delta events are ephemeral; verify reconciliation across reconnect and event boundary.
- web/src/features/workspace/task/panel/FilesTab.svelte + files/TreeLevel.svelte: changed set derives from task action events (even when Git is clean). FileView.svelte only shows Saved/Unsaved; CodeEditor / editor-runtime.ts has no Git gutter.
- Existing @codemirror/merge and diff-editor-runtime.ts already render task diffs in Changes. No editor migration required.
- /api/projects/:id/files and /file responses do not provide Git status or a Git base revision. Generic file layer intentionally denies .git and secret paths; maintain that boundary.
- Existing docs explicitly state Changes shows file-tool edits and misses shell/git modifications: docs/using-the-app.md, lines 104-120.

These are source-level observations and candidate root causes, not confirmed production reproduction. Require black-box proof.

## 3. Goals / success metrics

1. Every persisted conversation event appears exactly once, correctly ordered and attributed, throughout streaming, reload and reconnect. No spurious disappearance, duplicated finalized draft or indefinite working bubble.
2. User sees current lifecycle accurately (running, waiting, reported, idle, completed, failed), including external sessions. No automatic success on silence or assumption that a provider has finished.
3. Files Explorer reflects actual local Git staged/unstaged/untracked/renamed/deleted state per project, not action history. Changed folders are decorated by descendants. Clean files/folders are undecorated.
4. Opened text files show changed-line gutter marks for added/modified/deleted lines against a documented Git baseline; no Git marks once clean. Preserve unsaved local edit handling.
5. Changes displays current repository changes; historical task actions/undo remain available as separate activity/history, not falsely labeled active Git changes.
6. Work remains secure, fast and backward-compatible with non-Git projects, binary files, large files and mobile layouts.

## 4. In scope

### A — Chat/task bubble lifecycle

- Produce one canonical view-model from durable ordered task events plus explicit ephemeral streaming state, keyed by task + generation/turn/message identifier; dedupe deltas if replayed and clear a finalized draft upon matching durable message regardless of whether task is still working.
- Define transition rules for queued/planning/working/needs-input/review/done/failed and external running/awaiting_person/blocked/idle/reported/completed/failed/stopped/abandoned. Use same derived semantics for header, sidebar, thread and status indicator.
- Streaming bubble appears as soon as stream begins; replace it with the durable message atomically when persisted. Never treat transport connection or elapsed time as completion.
- On SSE reconnect/page reload/browser tab return, reconcile from durable server snapshot/event cursor and discard or correctly reconcile stale ephemeral text; avoid duplicated message and wrong status. Maintain event seq/version ordering and existing action grouping.
- Show an explicit unobtrusive connection/reconnecting state if appropriate; distinguish paused/idle from actively streaming.
- Prevent user-side auto-scroll jumps when reading older events; maintain follow-to-bottom behavior only when already near bottom.
- Preserve external vs internal authorship/provenance, and ensure external follow-up routes to internal orchestrator without misrepresenting it as a reply from provider.
- Audit whether WORK_SWEEP_MODE behavior is intentional; do not silently turn it on or mark expired work as successful.

### B — Git status and Explorer

- Add an authenticated project-scoped read-only backend Git status capability. Resolve repository root under authorized project workspace only; never expose arbitrary directories, raw .git internals, secrets or untrusted path traversal.
- Use fixed Git arguments, process timeouts, repo-scoped cwd, no interpolated shell, null-delimited machine-readable output (porcelain v1/v2 -z), robust rename/space/unicode handling; preserve staged and unstaged distinctly.
- Include tracked modified/added/deleted/renamed, untracked, conflicts, ignored-file exclusion, and clear handling for no Git repository/initial HEAD/submodules. No automatic git init, stage, commit, checkout or reset.
- Return a bounded repository status snapshot/revision from backend; expose optional changed-file diff/original on demand (with size caps), never bulk file contents in status.
- Explorer uses actual Git state; badge conventions M/A/D/R/U with accessible labels/tooltips. Folder decorations aggregate currently changed visible descendants. No badge or colored dots for clean paths.
- Refresh status on initial load, relevant project file change notification, focus/reconnect and bounded fallback polling; coalesce bursts and prevent stale/out-of-order responses overwriting newer state.
- Keep task action history separate. If Git becomes clean after commit, immediately clear active decorations in tree and Changes.

### C — Open-file diff gutter

- Reuse CodeMirror 6 and @codemirror/merge or focused CM6 gutter/state extensions. Compare working contents to a consistent Git HEAD baseline (include staged + unstaged modifications); add/modified/deleted decorations aligned to current text lines, with color AND shape accessible semantics.
- If Git repo has unborn HEAD, no HEAD baseline or file is untracked, show appropriate whole-file added semantics rather than fail. Deleted/renamed files must not crash the editor.
- When the user types unsaved edits, keep editor state/conflict protection stable; distinguish unsaved draft changes from persisted Git working-tree state, avoid expensive per-keystroke backend reads. Gutter must be corrected on save, reload, external edit and commit.
- No active Git gutter or badges when working tree and index are clean. Binary/truncated files do not receive incorrect text-line markers.

### D — Changes tab semantics

- Active Changes shows current Git diff regardless of whether modified by AI file tool, shell or human. Include staged vs unstaged distinction when useful; no misleading task history as active changes.
- Preserve existing task steps, proof/provenance and Undo functionality in a clearly separate History/Steps section; never throw away audit data when Git is clean.
- For clean Git: display compact empty state (or hide count and active file groups); no historical 'changed' file badges. For no repository: explain unavailable Git tracking without claiming clean.
- Preserve existing mobile/desktop responsive layout, APIs where possible and permission boundaries.

## 5. Non-goals

- No replacement of Svelte, CodeMirror, SQLite, task event persistence or Git provider architecture.
- No new Git mutation UI (stage, commit, discard, checkout, push) in this scope.
- No silent external assistant auto-completion or inferred trust/proof from streamed text.
- No global filesystem scans or exposure of .git/.env/credentials.
- No broad unrelated UI redesign.

## 6. Black-box acceptance matrix (required)

AC01 Internal task: immediately visible working state, streaming draft, durable final event exactly once; no duplicate when another message follows while task remains working.
AC02 External task: role, progress, message and receipt remain correctly ordered/attributed across task switching.
AC03 Reconnect while mid-stream: after simulated network interruption and reconnect, no missing/duplicated persisted message or stuck draft; status converges to server truth.
AC04 Idle/reported/blocked/awaiting-person/completed/failed: no incorrect spinner; completion requires explicit valid transition.
AC05 Scroll up while new events arrive: scroll position remains readable, follow resumes near bottom.
AC06 Unstaged edit via editor and via shell: Explorer M badge + parent folder decoration; gutter changes match file.
AC07 New untracked file: U indication; ignored paths excluded; whitespace/unicode/renames handled.
AC08 Staged only, unstaged only, staged+unstaged, rename/delete and conflict: correct status and diff, with no data-loss actions.
AC09 After git commit makes project clean: Git badges, folder dots, gutter and active Changes all clear without full reload; task history remains.
AC10 Non-Git project: no fake clean/changed Git state, no errors or auto-initialization.
AC11 Multiple project owners: cross-project Git status/diff access forbidden; traversal and secret paths blocked.
AC12 External concurrent file edit during unsaved draft: conflict flow works; no overwritten user edits.
AC13 Large/binary files and many changes: limits/pagination/truncation and UI remain usable.
AC14 Playwright desktop + narrow mobile: no clipping or status layout regressions.
AC15 File changes triggered by shell/agent (not only file-tool events) are captured after refresh/reconnect.

## 7. Performance, reliability, observability

- Reuse existing SSE channels; provide bounded polling fallback and focus refresh, with debouncing around 250-500ms during bursts. Do not add a new server/process.
- Bound Git command execution time, diff size, file count and payload sizes; large files show safe fallback. No N+1 Git subprocess per tree node.
- Provide structured debug/logging for task event cursor, reconnects, stream turn correlation, work phase, Git scan revision, Git unavailable errors, without logging sensitive content.
- Handle stale async results with request revision and cancellation.

## 8. Delivery constraints

- Preserve checkout and all existing dirty/untracked files. Do not force reset/clean/stash/checkout; use safe repository worktree policy and verify remote base before any agent push.
- Align with AGENTS.md (architecture, documentation, security), existing Svelte 5 and CM6 conventions; prefer current dependency versions, no extra dependencies unless justified.
- Prioritize black-box API and Playwright E2E; avoid new white-box tests except strictly security-critical authorization/path containment checks.
- Run existing mandatory CI/guardrails and relevant black-box suites; don't disable required checks to meet test preference.
- Update human docs in docs/using-the-app.md and agent-only rules if behavior contract changes.
- Developer agent implements; reviewer and QA independently verify. Lead reviews evidence before declaring done.

## 9. Definition of done

All AC01-AC15 evidenced with black-box run logs or screenshots plus mandatory repository checks green; no unexplained unstable streaming/dirty decorations; no regression to external approvals, task history or security; code reviewed; all user changes preserved; implementation agent reports precise commit hashes, branch, files changed, tests, any skipped scenarios and blockers.
