# Research and Source Trace — 2026-10-28

## Repo baseline (source inspected; runtime reproduction pending)

Project: agentic-ai-workspace; dev @ fa5e610 at read-only inspection. Existing 9 modified files inside project-notes/tasks/3538373b8e689a4997439e0eb99974b3; preserve. Worktree was not cleaned, reset, committed or pushed.

1. SSE task rendering
   - web/src/features/workspace/logic/thread.ts (draft accumulation, merge by event id and version, fallback refresh).
   - web/src/features/workspace/task/thread/TaskThread.svelte (durable event blocks and working draft separate; status in top, spinner at end).
   - web/src/features/workspace/task/thread/EventItem.svelte (messages, actions, role and receipts).
   - web/src/features/workspace/task/thread/blocks.ts (action grouping).
   - internal/modules/task/adapter/http/stream.go (Last-Event-ID version and update events; delta has no durable event id).
   - Hypothesis: persisted assistant message can render while ephemeral streaming draft remains until lifecycle transitions; must reproduce with black-box test.

2. External work lifecycle
   - internal/modules/task/domain/work_phase.go: derives phase; idle from updatedAt threshold, reporting/approval phases distinct.
   - internal/modules/task/application/works.go: default idle 30m and abandonment 2h; sweep initialized dry-run.
   - internal/modules/task/application/work_reconcile.go: sweep 'dry-run' logs candidates; 'on' closes qualifying idle work.
   - compose.dev.yaml, compose.prod.yaml, compose.latest.yaml default WORK_SWEEP_MODE=dry-run.
   - web/src/features/workspace/logic/work-status.ts: maps phase to working indicator and label.
   - test/e2e/04-work-lifecycle.spec.ts already checks reported/idle display; extend black-box scenarios rather than white-box expansion.
   - Inactivity does not prove completion, by design; do not alter this requirement.

3. Explorer and existing changes behavior
   - web/src/features/workspace/task/panel/FilesTab.svelte derives changed set from task change events.
   - web/src/features/workspace/files/TreeLevel.svelte shows a persistent 'changed' badge and folder dot.
   - web/src/features/workspace/files/FileView.svelte shows Saved/Unsaved only.
   - web/src/features/workspace/logic/files-api.ts and internal/modules/coding/adapter/http/files.go have no Git state fields.
   - web/src/features/workspace/logic/editor-runtime.ts uses CodeMirror 6; @codemirror/merge v6 already installed and used at diff-editor-runtime.ts.
   - docs/using-the-app.md describes 'Changes' as tool-action diffs; shell/git edits are omitted.
   - internal/modules/coding/domain/workspace.go denies .git/.env/private key access; maintain or strengthen protections.

## Upstream, official sources and design implications

1. VS Code Source Control documentation (updated 2026-10-21):
   https://code.visualstudio.com/docs/sourcecontrol/overview
   - Current changes vs staged changes, inline diff, commits and history are distinct concepts. Use current Git source rather than task event history for active decorations.

2. Git status machine-readable format:
   https://git-scm.com/docs/git-status
   - porcelain v1/v2 with -z is appropriate for stable parsing, including spaces, Unicode and renames; consider index and working-tree columns; ignored files excluded by default. No shell string interpolation.

3. CodeMirror 6 reference manual:
   https://codemirror.net/docs/ref/
   - @codemirror/merge supports unifiedMergeView with original and gutter; reuse it, or an equivalent focused CM6 gutter extension, rather than replacing CodeMirror with Monaco.

4. WHATWG HTML Server-Sent Events:
   https://html.spec.whatwg.org/multipage/server-sent-events.html
   - EventSource reconnect behavior and Last-Event-ID support durable ordered update reconciliation. Ephemeral text deltas should never be treated as independently durable on reconnect.

## Open investigation during implementation

- Is the observed bubble inconsistency due to draft/event double rendering, stale SSE update ordering, missing pending-phase reconciliation, server event publication, or multiple causes? Reproduce and record exact event IDs/versions and transitions before fixing.
- Do file/project writes via shell always signal files SSE? Inspect hub notification and fallback strategy; commits may not trigger file-stream events.
- Can a project lack Git metadata or be a nested/linked repo? Determine Git root and HEAD base policy and fail safely when unavailable.
- What are effective production WORK_SWEEP_MODE and threshold values? Do not infer from compose defaults alone.
- Confirm file-diff baseline (HEAD vs index vs working content) for staged+unstaged; present a single coherent choice across Explorer, gutter and Changes.
- Check CM6 large file performance and Svelte 5 state lifecycle under async refresh; preserve drafts and cursor.

## Scope and testing decisions

- User requested live Git decorations matching VS Code, no active changed marks after clean, and focus on black-box testing over white-box proliferation.
- Existing task audit/provenance must not be removed when Git is clean.
- Do not alter repo until delegated developer starts a separate implementation work session and follows AGENTS.md.
