# Critical regression — delegation created duplicate top-level tasks

Status: **IMPLEMENTED / TESTED (2026-10-29)**. Project: `farismnrr/agentic-ai-workspace`, `dev`.

## Symptom and root cause
One user issue delegated to Planner, Researcher, Developer and QA could scatter across many tasks. MCP connector instructions added in `b7e9294` required `begin_work` before almost every workspace action, but `Works.Begin` created a new task whenever the call had no idempotency key. Distinct role-specific keys also created new tasks. Separately, `Tasks.List` displayed linked internal chat/approved handoff tasks as if they were independent issues.

Native orchestrator `delegate` itself already writes under the same `task.ID` and was not creating a task per role.

## Contract implemented
- One open issue per user/client/project is reused by default, even if the assistant repeats `begin_work` or changes role/key. Use `set_role` to track responsibility within the same work ID.
- New independently requested issues explicitly use `new_task: true`. Different clients/projects never share task IDs; closed/archived work cannot accidentally capture a fresh issue.
- Begin request keys, including keys aliased to resumed work, remain idempotent after task completion. A key with conflicting input is rejected.
- Linked internal chat and approved handoff tasks are excluded from primary and archived task lists; child execution/history and direct ID lookups continue working. No existing tasks or history were deleted.
- MCP tool descriptions, connected-app instructions and human documentation explain these semantics.

## E2E evidence
- `test/go/app/connectors/delegation_task_spam_test.go`: repeated begin/role, explicit independent task, cross-client/project isolation, completed and archived work, alias retry, linked chat and direct child ID — PASS.
- `test/go/app/tasks/review/plan_review_routes_test.go`: approved handoff, retry and one visible task — PASS.
- `go test ./... -count=1 -timeout 120s` — PASS.
- `cd web && npm run check` — PASS.
- `bash test/e2e/run.sh lifecycle` — PASS after marking independently seeded test tasks with `new_task:true`.
- `bash test/e2e/run.sh context` — PASS (no context-window regression).

## Operational notes
- External clients must only set `new_task:true` when the user intentionally starts another issue. One active work session is reused by default; this cannot infer task identity across different clients.
- Live authenticated ChatGPT/Gemini delegation on a Docker-enabled deployment was not executed from this sandbox; MCP+HTTP integration tests used the real application endpoints and repeatable fixtures.
