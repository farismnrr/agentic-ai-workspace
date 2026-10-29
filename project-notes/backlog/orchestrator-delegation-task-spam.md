# Critical regression — orchestrator delegation creates task spam

Status: **BACKLOG / DO NOT IMPLEMENT until context-window reliability PRD is finished and verified**.
Reported: 2026-10-29. Scope: `farismnrr/agentic-ai-workspace`, `dev`.

## User-observed regression
When the orchestrator delegates to agents, each delegation now appears to create a new top-level task. A small user issue can scatter into ~10 tasks, making the task list and review unreadable. This previously stayed within one task. The same problem is observed with ChatGPT-originated work.

## Expected behavior
One user issue / orchestration run = one top-level task through completion. Delegate roles, plans, research, tool calls, progress and reports appear as child activity within that task; no new user-facing task for each delegate. Explicit user-authored independent tasks are still allowed.

## Required follow-up after context PRD
1. Reproduce via actual orchestrator + external assistant integrations; log task IDs, parent/origin links and event lifecycle.
2. Trace delegate entrypoints and task creation (internal delegation vs linked external task) and identify regression commit or contract drift.
3. Write a separate scoped PRD and regression acceptance criteria with before/after trace.
4. Implement minimally with proper parent-child attribution and backward compatibility, no silent loss of existing task history.
5. Black-box E2E: one issue delegated to multiple roles remains one visible task until finished; no task spam after reconnect/resume.

Do not conflate this issue with the ongoing context-window fixes; preserve this note for next phase.
