# Deep research external MCP role scope parity with internal O…

Task: 7b06c46aa492c1e4f18dda92a7d8b7d4
Status: done
Updated: 2026-10-31T17:17:20Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# Deep research — external role scoping parity
Date 2026-10-31 · agentic-ai-workspace dev @ e4e7c42 · Research only, no implementation.
Issue task 7b06c46aa492c1e4f18dda92a7d8b7d4.

## Confirmed findings

**P0: external role is still only an attribution label for most tools.** Internal teammates use a real per-role tool allowlist, but MCP external verifies only that the work has a valid, enabled role; OAuth scope and user's global tool toggle are checked, not whether that role may use the selected tool. Consequently a Researcher with coding:write or coding:exec and enabled user switches can reach write/exec operations under current source control. This is a source-based conclusion, not yet experimentally reproduced.

**P0: Developer-not-testing is stricter than today's internal rules.** The internal Developer preset itself says “run the relevant build or tests after your last change” (internal/modules/agentteam/domain/preset.go:77-82), while QA Tester is charged with testing (:94-101). Both Developer and QA have generic exec access; neither is an exclusive boundary around tests.

**P0: arbitrary shell bypasses superficial policy.** MCP run_command accepts unrestricted shell syntax; internal Runner also supports generic commands. A Developer with exec can launch tests via shell or nested scripts, and a QA with exec can potentially mutate project files via shell despite not having write_file. Denying just run_task or matching "go test" is not a durable guard. Execution must be narrowed at the runner/sandbox or tests/build must be distinct bounded capabilities; do not promise perfect separation with prompts/string filters.

**P1: role self-switch.** set_role on external work accepts any enabled non-lead role with no planned-role/task-stage authority check; planned_roles is informational. One ChatGPT/Gemini assistant can mark itself as QA immediately after Developer. That transition is recorded, but is not an independent second agent or independent review.

**P1: path scope parity.** Internal delegation optionally restricts file changes to assigned prefixes. External MCP has a project/work_id selector but no equivalent assigned path prefixes; no file-level delegation boundary today.

**P1: privileged lifecycle commands.** External save_research, save_requirements, update_plan, submit_plan, report_progress, complete_work etc. share a read-group/tool switch, with no role ownership checks. approve_plan uses distinct coding:approve scope and off-by-default toggle, yet no role-specific restriction. Must distinguish the approval policy from generic exec.

## Code evidence

Internal:
- internal/modules/agentteam/domain/role.go:25-35,93-135 defines access read/write/exec/ask/web/images and immutable presets.
- internal/modules/agentteam/domain/preset.go:12-118 defines built-in tool categories and responsibilities.
- internal/bootstrap/agentteam.go:58-99 toolsFor(role) converts allowed access groups into tool names and adds specialized abilities; DevOps gets start/stop/build deployment tool, web-only roles can search.
- internal/modules/task/application/teamrunner.go:39-49,69-101 wraps internal child with roleGate: effective enabled = user settings AND role tool allowlist; own transcript and role-tagged events.
- internal/modules/task/application/toolgate.go:15-45 checks offered tools and rechecks before run; calls.go:42-105 applies scope/approval.
- internal/modules/task/application/delegate_scope.go:47-109,137-156 optionally fences writes to assigned project paths.

External:
- internal/modules/coding/adapter/mcp/guard.go:61-139 only checks OAuth identity, global switch, coding scopes and work_id. It does not inspect active role versus tool.
- internal/modules/task/application/work_calls.go:23-50 Validate checks work active and role still enabled; RequireRole special-cases MoLab developer.
- internal/modules/task/application/work_roles.go:23-59 explicitly says set_role is attribution, not permissions, and changes to any enabled non-lead role.
- internal/modules/coding/domain/tools.go:62-121 exposes routine lifecycle tools under coding:read; run_task and run_command under coding:exec; approve_plan under coding:approve.
- internal/modules/coding/adapter/mcp/tools_exec.go:37-69 and tools_write.go:70-155 call project operations after checking work validity, without per-tool role policy.
- internal/modules/coding/application/commands.go:10-23 and coding/adapter/runner/runner.go:70-98 run project shell commands, then observe changed files (observation is not prevention).
- internal/modules/task/application/work_approval.go:22-35 checks enabled role and work ownership before approval but no authorized role category.
- internal/modules/coding/adapter/mcp/target.go has project/work_id only, no path prefix role assignment.

## Matrix: internal built-in capability groups TODAY

Researcher: read+web. Planner / Business Analyst: read+ask. Architect / Reviewer: read. Developer: read+write+exec. QA / Security: read+exec. DevOps: read+write+exec. Designer/Media/Documents: read+write+exec plus specialized tooling. Group "exec" is broad and allows shell, so these are coarse roles, not hard separation between tests/build.

## Proposed policy for PRD

- Researcher: read, web research, save_research. No write, shell, test, deploy.
- Planner / Business Analyst: read, plan/PRD/requirements/ask as appropriate. No shell or edits.
- Architect / Reviewer: read, review notes and verdict; no modifications.
- Developer: read, edit and build/format, MoLab. **No formal QA test or approval/deployment.**
- QA: read and bounded test commands, no edit_file or mutable general-purpose shell.
- Security: read and bounded security checks; no project mutation.
- DevOps: permitted build/package/start/stop; no independent review or arbitrary approvals.
- Designer/Media/Documents: restricted writes and relevant specialized operations only.
- Plan approval remains a separate explicit permission; human decision by default. A role with generic exec must not inherit approve_plan.
- Bootstrap whoami/list_projects/begin_work; set_role/get_work_state and safe close/recovery require intentional allowances when current role is disabled. Never allow role switching to bypass role policy silently.
- Existing MoLab Developer-only rule remains enforced; reuse one task per issue.

## Implementation strategy (do not implement until PRD)
1. Create one shared server-side policy resolver with canonical Agent Team access rules and role-specific tool capabilities, avoiding a second MCP-specific duplicate map. Effective authorization: [redacted] + client + project + work + active enabled role + role capability + user's global tool switch + OAuth scope + approvals. Deny if any check unavailable. Check before starting handler/sandbox.
2. Make lifecycle methods role-aware (research, PRD, planning, approval) rather than treating them all as read. Denied calls return short actionable role errors and are audited. Cross-client/work_id remain isolated.
3. For **strict QA-only testing**, split runner capabilities. A typed/bounded test runner for QA and a developer build/format path with actual execution restrictions are needed; developer's arbitrary shell otherwise lets it run tests, QA's shell lets it edit. A readonly project mount for QA/Security with writable scratch is one practical control, subject to validating how frameworks use generated files. Policy on build vs test must be shared with internal Orchestrator; currently internal Developer may test.
4. Consider stage-bound set_role (and explicit rules for QA/reviewer independence), recorded in same work_id. A single external LLM switching roles is not an independent reviewer. No task proliferation. Keep per-file scope optional unless needed.
5. Recheck the role and effective capabilities at call time and protect against role switching while a long command runs (pin role-at-start, cancellation/revocation when needed). Do not trust role labels from tool arguments.

## Focused black-box acceptance tests (later)
- With wide OAuth+global switches, Researcher can read but cannot edit, run tests, run shell. QA can run bounded tests but cannot write using file tools OR shell; Developer can edit/build/MoLab but cannot use test/deploy tools. Reviewer cannot mutate.
- Commands through aliases, scripts, make targets, interpreters do not silently bypass the promised separation. If strong command confinement is not feasible, explicitly weaken the guarantee rather than claiming it is airtight.
- set_role from Developer to QA is audited and constrained appropriately; stale role disabled, concurrent calls, role switch while running fail safely or keep role-at-start; no new tasks.
- User's off tool switch / missing OAuth permission / wrong client-project-work_id always deny even when role permits it; approve_plan additionally requires explicit independent permissions. Test these as focused black-box flows, no noncritical white-box suite.

## Public references
MCP official authorization (least-privilege OAuth scope does not imply local agent role policy):
https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/basic/authorization/index.mdx
Microsoft least privilege for AI agents (role + action + resource; auditable effective permissions):
https://learn.microsoft.com/en-us/security/zero-trust/sfi/least-privilege-for-ai-agents
OWASP AI Agent Security (per-tool privileges, separate trust groups):
https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html
OWASP AI Exchange general controls (server-side operation allowlists):
https://owaspai.org/docs/1_general_controls/

Status: only inspected source; no live unauthorized calls, tests, code changes, commits or pushes in this review. Keep this single issue open for PRD then fixing.

Source: https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/basic/authorization/index.mdx

Source: https://learn.microsoft.com/en-us/security/zero-trust/sfi/least-privilege-for-ai-agents

Source: https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html

Source: https://owaspai.org/docs/1_general_controls/