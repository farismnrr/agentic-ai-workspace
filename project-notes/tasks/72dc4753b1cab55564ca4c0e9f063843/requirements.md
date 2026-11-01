# Implement deep-audit fixes for deployed eight-role RBAC: ame…

Task: 72dc4753b1cab55564ca4c0e9f063843
Status: working
Updated: 2026-11-01T08:43:03Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# PRD — Eight-role follow-up, broad Git/GitHub and optional QA named checks
**Task:** 72dc4753b1cab55564ca4c0e9f063843 | **Branch:** dev | **Baseline deployed:** b038f41. This supersedes prior incomplete verification claims.

## Outcome
Preserve 8 built-in roles. Developer owns source/UI/media/docs plus general Git and GitHub Issues/PR workflows; QA owns tests/lint/typecheck and independent review; DevOps owns build, CI runs/workflows, runtime apps, releases, deploy. No role needs an unrestricted shell for routine functions. No unrelated dirty user files overwritten. Complete focused black-box testing and verified FF push to dev before claiming completion.

## QA verification — new requirement
**Named checks are optional, NOT required to run QA.**
- Expose QA-only `verify` tool taking a direct argv vector. Support common test/lint/typecheck tools across Go, JS, Python, Rust, .NET and Make, without hardcoding one project-specific `.workspace/tasks.json` file.
- Run in disposable sandbox with read-only source mount, no owner GitHub tokens, no project secrets or task-granted egress. Tests can execute in container but cannot mutate project source; do not map verify to vendor unrestricted Bash.
- Reuse existing `run_task` owner toggle for verify to avoid an extra enablement burden; `run_task` remains optional for explicitly configured named checks. `list_tasks` with no file should return an empty list (not an error).
- Developer and DevOps cannot call verify, QA cannot write source, run arbitrary general shell or deploy. Internal vendor tool bridge and external MCP server must follow identical role and owner-toggle enforcement.
- Verification output includes exit code and bounded logs; do not report PASS unless a command really returned exit 0.

## Git/GitHub
- Developer: Git status/branch/worktree/stash/cherry-pick/rebase/merge/add/commit/normal push and GitHub issues/PR create/update with read CI; QA can review PR/read CI; DevOps handles CI rerun/dispatch/release. Use separate `git` and `gh` direct argv tools.
- Destructive Git operations and high-impact gh API/admin changes cannot silently bypass approval; GitHub connection/token comes only from server-owned project owner with bounded egress.
- Keep approved actions auditable, no legacy arbitrary shell exposure, and preserve one visible task.

## Additional outstanding audit changes
- Lock down start_site direct command execution, preserve runtime-data writes separate from read-only source.
- Dedicated read-only build and media/doc artifact runner with explicitly writable output mount.
- Preserve custom role preferences, alias historical reviewer evidence, protect all write scope paths, and stop self-role-switches counting as independent review.
- Verify Docker volume Subpath read-only behavior and credential/network containment on real Docker-enabled host.

## Acceptance tests
1. No .workspace/tasks.json required: QA verify Go tests, JS lint, Python pytest, Cargo tests and typechecks; reject shell, source-write options and unsanctioned programs.
2. Owner toggle off and wrong role refuse; right role reaches verifier even if no named task config.
3. No project source changed by QA verifier (including malicious script attempts); GitHub secrets inaccessible.
4. Git/gh positive/negative operation tests, eight-role migration, review/approval, devops build and app startup smoke, responsive frontend checks.
5. Focused Go packages + frontend check/lint/build PASS; Git changes committed with no unrelated dirty files, safe fast-forward push to origin/dev, confirm remote SHA. If tools unavailable, keep task open and give exact blocker.