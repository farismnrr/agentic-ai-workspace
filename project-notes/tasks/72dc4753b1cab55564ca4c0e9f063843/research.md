# Implement deep-audit fixes for deployed eight-role RBAC: ame…

Task: 72dc4753b1cab55564ca4c0e9f063843
Status: working
Updated: 2026-11-01T08:43:03Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# Deep Review — Git & GitHub CLI scope, 2026-11-01
**Task:** existing open follow-up 72dc4753b1cab55564ca4c0e9f063843. **No implementation in this audit.** Inspect live repository source only; the deployed server still lacks the newly coded 'git' tool. No claim of runtime verification or push.

## User intent
Developer must not be limited to commit/push or a short literal Git subcommand list. Preserve day-to-day full Git workflows plus GitHub Issues, Pull Requests, Actions/CI, workflow status, artifacts, releases, discussions, repo administration and generic GitHub API when authorized. Most GitHub features are in **gh**, not git.

## Authoritative sources
- Git official command reference https://git-scm.com/docs/git
- GitHub CLI reference https://cli.github.com/manual/gh
- GitHub CLI Issues https://cli.github.com/manual/gh_issue
- GitHub CLI PR https://cli.github.com/manual/gh_pr
- GitHub Actions runs https://cli.github.com/manual/gh_run and workflows https://cli.github.com/manual/gh_workflow
- Release https://cli.github.com/manual/gh_release
- Generic GitHub API https://cli.github.com/manual/gh_api
- Untrusted dynamic extensions https://cli.github.com/manual/gh_extension

## Code-grounded review findings
**P0 Functional: Current Git MCP handler cannot compile.** internal/modules/coding/adapter/mcp/tools_exec.go uses strings.Join for 'git' tool description but lacks strings import. internal/modules/coding/application/tasks.go still imports 'strings' though no longer uses it. Go rejects these imports. A working 'git' tool is NOT deployed.
**P0 Feature gap: gh CLI missing entirely.** sandbox/Dockerfile already pins/installs gh 2.102.0 and git. internal/modules/coding/adapter/runner/git.go already supports GH_TOKEN, scoped GitHub endpoints and commit identity, and internal/modules/task/application/prompt.go explicitly promises 'gh repo list, gh pr, gh issue, gh run'. New domain tools.go and mcp/tools_exec.go only register git argv; no gh tool. Existing ci_logs for DevOps is a special partial workaround but not GitHub issue/PR/Actions/release workflow.
**P0 Security: Git command allowlist is not a safety boundary.** internal/modules/coding/domain/git.go lists dozens of builtins but allows 'git config' modifications, 'git clean', 'git reset --hard', force push and custom fetch/push targets. '--no-pager' and hooksPath=/dev/null only disable some code execution paths: git filters, ssh/protocol helpers, config/aliases/external programs, submodule operations, remote helpers and user-defined diff/merge tools still need OS/process/network restrictions. Passing separate argv prevents shell concatenation but does NOT prove arbitrary subprocesses cannot run. Avoid treating a command name allowlist as secure.
**P1 Scope too narrow / brittle:** Git's official reference has more commands than git.go's static map, e.g. maintenance, sparse-checkout, update-ref, rerere, show-branch, rev-list variants, mailmap, debugging, plumbing, newer git versions. Extra 48-argument/512-char cap, hardcoded flag bans and no interactive fallback block legitimate recovery, cherry-pick conflict resolution, multi-file stage, and some rebase options. Some flags are banned even when passed as commit messages. Git subcommand coverage should not require application source changes each Git update.
**P1 GitHub operations not modeled:** gh issue {list,view,create,comment,edit,close,reopen,transfer}; gh pr {list,view,checkout,create,edit,checks,review,merge,comment,update-branch}; gh run {list,view,watch,download,rerun,cancel,delete}; gh workflow {list,view,run,enable,disable}; gh release {list,view,create,upload,edit,delete}; gh repo {view,list,clone,create,edit,delete}; gh project/discussion/search, gh api for unsupported endpoints. 'gh extension' installs executable code; gh alias/config/auth and 'gh api' mutating HTTP/GraphQL are privileged escape hatches, not all safe by default.
**P1 Owner and task boundaries:** internal/modules/task/application/work_roles.go lets connected assistant switch among any enabled non-lead roles; a single actor can self-switch Developer->QA->DevOps. Repo access must be bound to authorized owner/project/repo, not arbitrary '-R other/repo' or selected remote URL. Keep same task ID and independent review provenance.
**P1 Credentials and transport:** runner/run_exec.go sees WithGitOnly and withholds ProjectEnv, but still allows GitHub creds via GitSource and egress when available; error handling must distinguish not-connected, missing OAuth scopes, denied repo, no network, remote rejection. Protect [redacted] from logs and process output; check gh API/Actions scopes. Sandbox does not automatically guarantee credentials inaccessible to spawned Git helper commands when Git credential is available. GitHub API calls must be authenticated as owner, least-privilege and audited.
**P1 CI role separation:** Developer can read gh pr checks and gh run status but should not execute QA tests via run_task. QA owns local/integration checks and review; DevOps owns explicit CI reruns, workflow dispatch, release/deploy. Git push may implicitly trigger CI, which is normal and should not be mislabeled forbidden local QA. Allow issue/PR creation/update by Developer; restrict PR approval/merge and destructive CI/repo actions according to task-level approval.
**P1 High-impact actions:** git push --force, delete branch, reset --hard, clean -fdx, rebase public history, release publish/delete, gh run cancel/delete/rerun, gh workflow disable, gh repo delete, gh api POST/PATCH/DELETE need explicit user approval or established per-task authorization and audit; don't ban all writes. GitHub read operations generally available without human approvals. Normal commit/push to authorized dev branch can follow user's existing auto-push preference with remote SHA confirmation and non-force fast-forward.
**P2 Test coverage incomplete:** no focused black-box negative tests for bad Git args/config/helper/subprocesses, no Git/gh positive end-to-end for cherry-pick/rebase/issue/PR/CI/artifacts/release, no remote SHA consistency check, no Docker filesystem/network tests. Current MCP 'git' register code has compiler errors, and latest server has not been redeployed, so cannot claim availability.
**P2 Existing GitHub app infrastructure is relevant:** Use current GitSource, githubHosts, GitHub app permissions and existing ci_logs integration; no need to install an unrelated new GitHub connector simply to support workspace Git/GitHub.

## Recommended interface (broad but bounded)
Prefer **one workspace VCS command interface** with typed 'program': 'git' | 'gh', plus argv []string, project, work_id, optional timeout. Or two thin tools 'git' and 'gh' backed by one shared execution/authorization policy. Do NOT enumerate dozens of command names in product code as the primary capability contract. Execute direct argv; no shell interpolation; bound repo and working dir; disable untrusted Git hooks/filters, gh extensions/aliases executing code, Git external helper and credential exfiltration paths; guard transport/host/token scope in sandbox.
Authorization:
[redacted] Read Git/gh repo/issue/PR/CI/release status: Developer and relevant QA/DevOps as role requires; read-only.
- Routine Developer source work: branch/worktree/stash/cherry-pick/rebase/commit/revert, fetch/pull, PR/issue create/edit/comment, normal push to assigned repo branch. Only Developer owns source edits.
- QA: read Git/gh diffs, PR checks/status and review; run named tests via dedicated check tool; no push/repo mutation by default.
- DevOps: read CI, manage authorized run rerun/workflow dispatch/build/release/deploy; no source edits or developer push by default.
- High impact: destructive Git reset/clean/force push, PR merge if required, run/workflow deletes/disables, repo delete/transfer, secret modifications, public releases, arbitrary API writes -> explicit approval and more precise scope, audit. Do not ask for routine reads/writes every time in Swift.
- Off-by-default optional admin capabilities for general gh api mutation, extension install, project/repo administration; don't silently grant via exec scope. Existing per-tool settings and OAuth remain upper bounds.

## Focused acceptance plan
1. Git: status/diff/add/branch/worktree/stash/cherry-pick conflict/continue/rebase abort/continue/merge/commit/push and remote SHA compare in sandbox, preserving dirty user files. Negative force push, clean/reset and code-execution helper paths.
2. gh: issue view/create/comment/edit, PR create/view/checks and authorized merge, run list/view/watch/download/rerun, workflow list/view/run, release list/view/upload, repo list/view. Check missing GitHub link and denied scopes gracefully.
3. Project and role isolation: researcher/qa/devops cannot push Developer source; Developer cannot invoke local QA tests or deployments; CLI connector and internal agent paths agree; role switches preserve one visible task.
4. Security: no token leaks in stdout/stderr, no Git external hook/filter/SSH/helper execution outside permitted sandbox, no unapproved arbitrary gh api mutation; test over existing Docker Subpath mount.
5. Fix both Go imports, run focused package tests and frontend checks; GitHub remote push verified; do not claim pass without running. Existing task remains open until verified.

No files modified by this audit.

Source: https://git-scm.com/docs/git

Source: https://cli.github.com/manual/gh

Source: https://cli.github.com/manual/gh_issue

Source: https://cli.github.com/manual/gh_pr

Source: https://cli.github.com/manual/gh_run

Source: https://cli.github.com/manual/gh_workflow

Source: https://cli.github.com/manual/gh_release

Source: https://cli.github.com/manual/gh_api

Source: https://cli.github.com/manual/gh_extension