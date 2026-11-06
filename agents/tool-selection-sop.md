# Tool selection SOP — before an agent acts

Applies to Masih Awam internal teammate tasks, plan-backed Claude/ChatGPT workers, and connected external assistants. The app enforces authorization server-side. Prompts and MCP instructions must explain the same route before the first operation.

## Preflight, in this order

1. Confirm the current project, task/work_id, enabled role, and owner tool switches. Use the **actual server-provided tool inventory**, not a tool remembered from another task or role.
2. Select the authorized tool and check prerequisites **before execution**. Never substitute a shell, native-host connector, alternate proxy or different project when a server tool is disabled, missing or returns an error.
3. Preserve pre-existing worktree changes and remote history. A failed tool is not permission to reset, clean, force-push or bypass a sandbox rule.
4. If unavailable, stop and report: tool name, role, missing switch/scope/connection or exact error; delegate to the correct role or ask the owner for permission. Do not brute-force alternative transports.

## Project files and source-code inspection

- Internal teammates MUST use the actual, role-authorized **team MCP** project-file tools: `list_files`, `find_files`, `read_file`, and `search_code` when enabled. Send paths relative to the current project. Example: `read_file` with `{"path":"AGENTS.md"}`. Do not assume a vendor-native `Read`/`Glob` tool is equivalent or always available.
- Developer can use `edit_file`, `write_file`, `delete_file`, and `move_file` only when their role and the owner's corresponding switches permit them. The server still enforces scoped paths and approval/undo requirements.
- SOP and tool inventory must name **actually registered team MCP tools**. A role-capability list can include lifecycle actions such as `get_work_state`, `report_progress`, or `complete_work`, but these are not automatically exposed as teammate MCP tools. Teammates finish a delegated job by returning their report to the orchestrator.
- When a read or edit tool is absent, check the corresponding tool switch in Settings → Spaces & tools and the role's access; report a specific blocker. Do not use arbitrary shell reads or another project as a workaround.

## Git, GitHub and release

- All roles have access to the Masih Awam **git** and **gh** server tools when enabled by owner switches. Mutating operations remain strictly role-authorized (Developer for code commits and PR/issue creation; DevOps for release git mutations and CI runs; QA for PR reviews; other roles read-only). Pass args as a list, e.g. `git(["status", "--short", "--branch"])`, `git(["ls-remote", "origin", "HEAD"])`, `git(["push", "origin", "dev"])`. The role/tool policy is the authority.
- A limited Claude/Codex worker has **no GitHub token or github.com egress in its shell**. Its authorized GitHub path is the team MCP bridge. Never use shell `git push`, shell `gh`, unset proxies or use direct DNS.
- Before pushing: inspect Git state, branch, HEAD, expected origin and divergence; preserve dirty/untracked files; require an expected fast-forward; never force-push or overwrite remote history.
- If `CONNECT_GITHUB_REQUIRED`, instruct the owner to connect GitHub in Settings. If a tool is absent or disabled, report it; a proxy 403 from the restricted shell is a policy denial, not a reason to bypass.
- Large ignored build outputs must not block quota-safe, read-only Git inspections or pushing an existing commit through the server tool. Workspace-expanding Git operations still require quota compliance.
- DevOps gets CI failed-job logs via **ci_logs**. Do not rely on shell `gh run view --log-failed` or browser downloads from GitHub Actions storage.

## Execution, large workspaces and lifecycle

- **Read-only Git inspection on large projects:** use the server `git` tool for `status --short --branch`, `diff --check`, `branch -vv`, `log -1`, and `ls-files`. These inspect an immutable mount; they must not be blocked by old ignored/generated files. Git add, fetch, commit, checkout and other mutations still require capacity and normal authorization.
- **QA checks:** `verify` or approved named checks use read-only source and no project secrets. Pre-existing oversized source should not block these checks. Never use arbitrary shell scripts as an alternative.
- **DevOps builds:** use an owner-configured `run_build` task (source read-only, output written only to `BUILD_OUT`); the newly generated output is size-limited independently. Use `start_site` to run or replace an app after the build, specifying the correct command and port. A successful build **does not** mean a Site is running.
- **Shell boundary:** Developer does not have general `run_command` under the built-in role policy. DevOps can use the team server `run_command` for diagnostics **only** when the owner switch is on and existing execution approval is granted; never use it for GitHub or Docker daemon bypass. If a tool is absent, distinguish a disabled switch from a role policy; do not claim disconnected permissions.
- **Completion:** an internal teammate returns a final report. The external Orchestrator calls `complete_task` after the work is complete or blocked. `complete_work` is an external Coding Space lifecycle tool, not an internal teammate tool.

## Browser, Playwright and site

- Start the project app through **start_site** using a role authorized to do so. Never use a background shell web server to impersonate a Site.
- Browser interaction/E2E is **QA-only** through **browser_test**, grounded in the running project Site. For the root page, use `{"actions":[{"action":"goto","path":""}]}`: `path` must be relative with no leading slash or `..` (never use `/`). If the tool says no Site is running, request authorized `start_site` instead of guessing another URL. Evidence screenshot refs are read with **read_browser_screenshot** in the same active task. `screenshot` is view-only, not proof of a click/fill E2E test.
- Non-QA roles delegate browser checks to QA. If the QA browser tool is missing/disabled or the Site is not running, report exactly that and start/fix the prerequisite through the correct role. Never shell Playwright, a different browser provider or an external URL fallback.

## Roles and audit

- **Mandatory policy wins:** saved role prompts, old merged Designer/Media/Documents/Reviewer skills and task briefs never grant a missing tool or override the server role, owner switches or safety gates. In particular, Developer must not follow old `run_command` recipes. Only configured kind=artifact jobs may use `run_artifact` for document/media outputs.
- **Normal handoff is not a blocker:** `Needs DevOps: start_site` or another `Needs <role>: <action>` is pending dependency, not a failed plan step. Orchestrator delegates the action to its authorized role in the same task, then resumes the dependent job. Real repeated errors remain `Stuck` with exact evidence.
- **Security check isolation:** Swift never delegates Security. In owner-enabled Deep Review, Security may invoke only configured named read-only checks. Security has no generic `run_command`, cannot start a Site, deploy or edit source.

- Orchestrator delegates and resolves prerequisites, never executes/edit files. Planner owns requirements and early architecture. Researcher owns research and approved native Git. Developer owns code, UI, media and documents with permitted native Git and optional named `run_artifact`, not generic shell/build/Site start. QA & Review owns independent targeted checks and browser testing. Security owns combined security/architecture review only with owner-enabled `@deep-review`. DevOps owns build, CI, running Sites, releases and approved native Git.
- Reuse one issue/task and an existing work_id across roles. A delegation is not a new issue. Record evidence, tool exit status and verification; mark a task Done only after its actual scope finishes.
- Every required action must be attributable to the server tool and the right role. Tool discovery and role prompts are generated from the same **switch-filtered** tool list. Do not advertise disabled tools.
- On unknown tool behavior, stop with an actionable blocker; never claim PASS from a build, prompt, or dry-run alone.
