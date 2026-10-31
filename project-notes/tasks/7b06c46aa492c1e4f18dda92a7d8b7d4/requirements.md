# Deep research external MCP role scope parity with internal O…

Task: 7b06c46aa492c1e4f18dda92a7d8b7d4
Status: working
Updated: 2026-10-31T17:05:03Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# PRD — Enforced External Role Scopes (shared with Internal Orchestrator)

**Issue:** 7b06c46aa492c1e4f18dda92a7d8b7d4 · **Branch:** dev · **Priority:** P0
**Requested behavior:** External ChatGPT/Gemini role is a real capability boundary—not a label. Reuse the internal Team's built-in/custom role access and one work_id per user issue. Research already saved under this task.

## Problems
- Internal teammates use roleGate(user tools ∩ role tools). External MCP checks work_id, enabled role, OAuth scope and global tool switches, but not whether active role permits the requested tool.
- External lifecycle tools all appear as read, with no role ownership. A Researcher can use execution/write operations if OAuth/global switches allow.
- Internal Developer currently explicitly runs relevant tests; user now wants stricter division of duties: Developer implements, QA executes tests. An arbitrary shell makes such separation impossible to promise with keyword filtering.

## Product requirements
1. **Role enforcement at server boundary.** Every connected tool operation must satisfy authenticated user/client/project/work, enabled live role, role's allowable action/tool, user's per-tool switch, OAuth scope and approval policy. Deny before executing any operation or starting its task event. Default deny on unavailable role/permissions. Never trust role sent by MCP args.
2. **Shared source.** Use same Agent Team role definitions and internal tool catalog that drives `roleGate` for the external checks; don't independently hardcode the main read/write/exec role matrix twice. Built-ins remain fixed; user-defined roles inherit user-configured access groups. Any stricter built-in rule introduced for Developer/QA must also apply to internal teammates.
3. **Responsibilities.** Researcher reads/web researches and saves research only; Planner/BA reads and creates PRDs/plans; Architect/Reviewer reads and reports; Developer edits source and performs MoLab with active Developer role but does **not** own QA tests or generic shell; QA checks/tests but cannot edit files or run arbitrary mutable shell; Security reviews/scans; DevOps handles command execution, builds, packaging, running site. Designer/Media/Documents operate only on their appropriate allowed groups. Role-allowed operation must be explicit; no silent escalation.
4. **Workflow tools.** `save_research` for Researcher; `save_requirements` for Planner/BA; `update_plan`/`submit_plan` for Planner/BA; `finish_research` for Researcher; `ask_question` for Planner/BA (and enabled roles with ask where appropriate). `get_work_state`, `log_message`, `report_progress`, `complete_work` available for ordinary task lifecycle; `set_role` available for recovery even if prior role disabled. `approve_plan` continues to require its separate disabled-by-default switch, coding:approve and an explicitly authorized review role; normal exec never implies approval. Reassess role stage transitions in subsequent design work, do not silently reclassify.
5. **Command capabilities.** For enforceable separation, Developer does not receive generic `run_command` or `run_task`; DevOps runs build/format commands; QA uses named approved checks through `run_task`, not generic `run_command`. Do not claim that a named task cannot mutate files just because it is called a check: tasks can be arbitrary commands. If read-only QA execution cannot be enforced safely in this pass, explicitly call it out as remaining limitation, don't claim perfect file isolation.
6. **MoLab is an explicit Developer-only exception** to the generic command split; still requires `coding:exec`, current tool toggle, active enabled Developer, work_id and per-chat ephemeral URL+token; no pairing/task spam/token persistence.
7. **Permissions response.** Denials name attempted tool and required role/capability, without OAuth reauthentication when problem is role. Keep role-at-call event attribution, and no recording successful effects on denial.
8. **Path restrictions.** Existing internal per-delegation file prefix remains unchanged. External project boundary enforced now; optional per-file assignment out of scope, cannot claim parity on folder scope.
9. **No task spam:** one task/work_id per issue; switch role through `set_role`. History and UI show active role. No role spoofing or forged independent QA reviewer claims.

## Acceptance
- Broad OAuth+enabled global switches: Researcher read/research succeeds, write/run_command/run_task denied; Planner can save requirements/plan, cannot edit or exec; Developer can edit + MoLab (if permitted) but `run_command` and `run_task` denied; QA can run configured named checks, but cannot write, delete, use generic shell or MoLab; Reviewer can't mutate; DevOps can execute and deploy within existing switches.
- Bad/disabled role/work_id/client/project fails closed; set_role recovers from disabled role. OAuth and global switches are always upper limits.
- Denied tool has no side effects (no file edit/command), and returns actionable message.
- MoLab remains Developer-only under work_id, ephemeral credential handling unchanged.
- Approval tools have role and separate approval capability, no bypass.
- Existing internal teammate tool gating matches role profile changes (especially Developer/QA).
- Focused black-box connector tests and relevant existing tests pass; frontend check/lint/build if changed. No noncritical whitebox suites.
- Unrelated files preserved; normal fast-forward push only after verification.

## Explicit limitations / follow-up
There is no secure generic method to identify a test hidden in a shell script. Therefore this pass enforces **no general shell for Developer or QA** and moves general commands to DevOps; QA's existing named user-configured tasks are still arbitrary programs that could modify generated files. A separate read-only execution mount and trusted-check management would be needed to guarantee no project filesystem writes during QA. No fake claim of complete isolation.

## Implementation steps
1. Define shared effective role tool policy and map role-to-lifecycle capabilities.
2. Add central MCP guard authorization before handlers for all tracked operations; preserve bootstrap/role-recovery paths.
3. Adjust internal tool allowlist where needed to honor strict Developer/QA division, maintain MoLab special case.
4. Add targeted cross-role black-box tests and focused regression tests, docs, commit/push on dev.