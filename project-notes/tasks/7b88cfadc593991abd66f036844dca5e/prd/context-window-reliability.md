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
