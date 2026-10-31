# Deep research-only audit of internal/external Agent Team rol…

Task: 62b3bf7e9fe7d2ea2f554e43fc70ac3f
Status: working
Updated: 2026-10-31T18:05:10Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# Role Boundary Deep Audit: internal + external
**HEAD** dev 4fa793a, **issue** 62b3bf7e9fe7d2ea2f554e43fc70ac3f. Research ONLY: no code/policy edits or commits. Source-level inspection plus selected existing black-box tests.

## Sources
- NIST separation of duties: https://csrc.nist.gov/glossary/term/separation_of_duty
- OWASP AI Agent Security: https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html
- Microsoft AI agent least privilege: https://learn.microsoft.com/en-us/security/zero-trust/sfi/least-privilege-for-ai-agents
- Microsoft per-action tools and human gates: https://learn.microsoft.com/en-us/azure/security/fundamentals/shared-responsibility-ai-agent
- MCP auth: https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/docs/2026-07-28/tutorials/security/authorization.mdx
Theme: RBAC + separation of duties, least-privilege allowlists, per-operation server authorization, OS-level sandboxing. OAuth scope alone is a ceiling, not a sufficient policy.

## Actual existing roles (13)
- **Orchestrator:** preset AccessRead/Write/Exec. Internal *team-mode* lead only offered low-risk tools, but solo task bypasses team-lead restriction. Prompts explicitly let lead do small jobs.
- **Planner:** Read/Ask, WritesPRD, external plan lifecycle. Cannot edit/exec.
- **Researcher:** Read/Web, WritesResearch; no write/exec, research-first requirement internal.
- **Business Analyst:** Read/Ask and requirement writing. Overlaps Planner but don't merge without request.
- **Architect:** Read and ChecksArchitecture, no edits.
- **Developer:** Read/Write, NO run_command or run_task, MoLab exception. Good current basic tool gate; no generic shell; can't automatically inherit media/doc workflows by only deleting old roles.
- **Designer:** Read/Write/Exec/Ask and LooksAtScreens flag: screenshot after changed UI.
- **Media:** Read/Write/Exec/Images; image search/download and licensed assets.
- **Documents:** Read/Write/Exec; files generated via sandbox commands and previewed.
- **QA:** Read/Exec, run_task (arbitrary configured command), no write_file or run_command; not OS read-only.
- **Reviewer:** Read, ChecksLook; optional MCP approve_plan role behind separate off-by-default switch and coding:approve.
- **Security:** Read/Exec, generic run_command denied but run_task still granted and potentially mutable.
- **DevOps:** Read/Write/Exec, RunsApps, run_command, run_task, start/stop/deploy, CI logs; can edit source and run tests. Most serious gap.

## Evidence and severity
**P0 DevOps can code/test**: internal/modules/agentteam/domain/preset.go DevOps AccessWrite+Exec. internal/bootstrap/agentteam.go toolsFor maps group access into write_file/edit_file/delete_file/move_file and run_command/run_task. External guard (internal/modules/coding/adapter/mcp/guard.go) checks active role tool name, not action/path, so DevOps can edit source. run_command is arbitrary Bash (internal/modules/coding/application/commands.go; adapter/runner/runner.go), letting DevOps write source/test/deploy regardless of semantic role. Restrict actual filesystem/command effects, not just tool name/prompt.
**P0 QA test can mutate**: internal/modules/coding/domain/task.go named task is user-configured argv; runner uses writable workspace. Internal CLI maps run_task to unrestricted Claude Bash (internal/modules/cliagent/adapter/command/argv.go). CLI sandbox sets WorkspaceReadOnly=false if run_task in allowed set (internal/bootstrap/taskcli_env.go). Codex CLI executes inside Docker with --dangerously-bypass-approvals-and-sandbox, so repo mount permissions matter. QA/Security tests need read-only source mount and separate scratch output, not command keyword filters.
**P0 External self-switch isn't independent**: external SetRoleWithBrief (internal/modules/task/application/work_roles.go) allows any enabled non-lead role, not stage/planned roles bound. Assistant can self-switch Developer->DevOps->QA while retaining one work_id. Tool privileges are enforced by active role (internal/modules/task/application/work_calls.go), but swapping a label does not create separate reviewer identity. Need stage ownership/approval or explicitly mark same assistant review as not independent.
**P1 Solo lead bypass**: internal/modules/task/application/toolgate.go restricts team-mode lead from risky tools, but solo has full access. Decide intentional Swift exception vs hard orchestration boundary.
**P1 Internal scope only for direct file tools**: delegate_scope.go scopeHold checks write_file/edit_file/delete_file/move_file only, misses download_asset and commands; roleCanWrite ignores shell/run_task so parallel conflict may miss mutation. External no per-file scope.
**P1 Custom roles**: domain/role.go custom roles can have AccessWrite+Exec; roleMayUseTool key denies shell only for developer/qa/security, so custom role can bypass desired role taxonomy if broadly enabled.
**P1 Migration**: domain/team.go WithBuiltIns adds missing presets, never removes retired saved keys. Simply deleting designer/media/documents/reviewer presets can leave old roles as custom with previously stored exec/write. Explicit migration required, including active work IDs, saved enabled state, prompt/model, planned roles and historical role events. Never lose task history or create task spam.
**P1 Merge hooks**: domain/preset.go/role.go use LooksAtScreens (Designer) and ChecksLook (Reviewer). reviewhold.go reviewerOf and lookNotTaken, securityhold.go verification evidence, delegate_scope.go soloOnlyRoleKeys, agentteam.go toolsFor and SkillsFor/LibraryFor(key), system.go laneRule, reviewmodeprompt.go, docs and role-name tests rely on old roles. Must transfer flags/skills and update text/test fixtures.
**P1 Review approval**: approve_plan currently role=reviewer only + separate human switch off default + coding:approve. After merging route through QA only if explicitly permitted; preserve human-first approval and old idempotent replay.
**P2 UI semantics**: Security says AccessExec but cannot use generic shell; show effective rights rather than broad access label.

## Proposed exactly 9 roles (13 -> 9)
1. Orchestrator: lead, assign, monitor, collect evidence; no code/check/deploy in team mode. Explicit policy for solo Swift.
2. Researcher: source/repository/public research; no edit/run/deploy.
3. Business Analyst: requirements/acceptance criteria; no edit/run.
4. Planner: PRD/plan, no edit/run/approve.
5. Architect: architecture design and read-only review; no edit/run.
6. Developer (+Designer/Media/Documents): source/UI/media/assets/documents authoring, MoLab, visual screenshots after DevOps starts site, licensed image credits, document preview. No unit test execution, run_task, general shell, deploy/approve. Add *typed* restricted image processing/document creation capabilities; don't restore arbitrary shell just to merge roles.
7. QA (+Reviewer): named checks and static/dynamic review, independent verdict after last Developer change; ChecksLook=true, read-only source mount + writable scratch. No source edits/deploy/general shell. External role switch is not independent actor.
8. Security: independent read-only audits and bounded checks/scans; no edits/deploy.
9. DevOps: build/package/start/stop/deploy and scoped CI/deployment config only. No application source edits, QA tests, or unbounded shell with writable repo. Prefer typed approved build/deploy operations; deployment gated for high impact.

## Proposed backend plan (future, not implemented)
One shared effective policy for internal+external: identity/user/project/work + active enabled role + phase/assignment + tool/action + path/runtime mount + user toggle + OAuth + separate approval. Validate before effect; log denied operations. Assign roles in same task; prevent high-privilege cross-role self-escalation without scope permission. For QA/Security checks mount source read-only, scratch/output writable; DevOps build source read-only and output staged, deploy separately approved; Developer writes code/media/docs but no generic command. Preserve supported image and document flows via narrowly scoped tools; keep MoLab ephemeral chat URL/token behavior. Keep Swift targeted checks, no universal full suites.

## Future black-box acceptance
- Developer can edit code/UI/assets/docs and use MoLab, cannot run tests/deploy by any tool/shell.
- QA can run named test in read-only source sandbox, including malicious nested shell/make aliases with source unchanged; QA verdict after last Developer edit.
- DevOps can build/deploy authorized outputs and update scoped config, cannot mutate src via file tools/commands/scripts or run QA tests.
- Researcher/Planner/BA/Architect denied write/exec with broad OAuth/global switches; Security runs only bounded scans.
- Developer merged image/doc skills are intact; visual screenshot gate works for UI tasks.
- Explicit migration removes old keys from future roster, maps active work safely, preserves history and custom preferences.
- Plan approval remains human by default; same task across roles; authorization revoked mid-task fails closed.

## Focused existing verification
Ran `go test ./test/go/app/connectors ./test/go/app/tasks/review -run '^(TestExternalRolesEnforceToolsAndWorkflowOnServer|TestDisabledExternalRoleStopsCallsUntilRoleSwitch|TestConnectedRoleDelegationsKeepOneVisibleTask|TestAppPlanApprovalIsOwnerBoundAndIdempotent)$' -count=1 -timeout 80s` => exit 0 (existing guard/regression tests, not proof of desired new isolation). No code changes. Unrelated dirty prior plan.md preserved.

Source: https://csrc.nist.gov/glossary/term/separation_of_duty

Source: https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html

Source: https://learn.microsoft.com/en-us/security/zero-trust/sfi/least-privilege-for-ai-agents

Source: https://learn.microsoft.com/en-us/azure/security/fundamentals/shared-responsibility-ai-agent

Source: https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/docs/2026-07-28/tutorials/security/authorization.mdx