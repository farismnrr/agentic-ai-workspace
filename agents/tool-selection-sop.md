# Tool selection SOP — before an agent acts

Applies to Masih Awam internal teammate tasks, plan-backed Claude/ChatGPT workers, and connected external assistants. The app enforces authorization server-side. Prompts and MCP instructions must explain the same route before the first operation.

## Preflight, in this order

1. Confirm the current project, task/work_id, enabled role, and owner tool switches. Use the **actual server-provided tool inventory**, not a tool remembered from another task or role.
2. Select the authorized tool and check prerequisites **before execution**. Never substitute a shell, native-host connector, alternate proxy or different project when a server tool is disabled, missing or returns an error.
3. Preserve pre-existing worktree changes and remote history. A failed tool is not permission to reset, clean, force-push or bypass a sandbox rule.
4. If unavailable, stop and report: tool name, role, missing switch/scope/connection or exact error; delegate to the correct role or ask the owner for permission. Do not brute-force alternative transports.

## Git, GitHub and release

- Researcher/Developer/DevOps use the Masih Awam **git** server tool when granted; Developer/QA/DevOps use **gh** where granted. Pass args as a list, e.g. `git(["status", "--short", "--branch"])`, `git(["ls-remote", "origin", "HEAD"])`, `git(["push", "origin", "dev"])`. The role/tool policy is the authority.
- A limited Claude/Codex worker has **no GitHub token or github.com egress in its shell**. Its authorized GitHub path is the team MCP bridge. Never use shell `git push`, shell `gh`, unset proxies or use direct DNS.
- Before pushing: inspect Git state, branch, HEAD, expected origin and divergence; preserve dirty/untracked files; require an expected fast-forward; never force-push or overwrite remote history.
- If `CONNECT_GITHUB_REQUIRED`, instruct the owner to connect GitHub in Settings. If a tool is absent or disabled, report it; a proxy 403 from the restricted shell is a policy denial, not a reason to bypass.
- Large ignored build outputs must not block quota-safe, read-only Git inspections or pushing an existing commit through the server tool. Workspace-expanding Git operations still require quota compliance.
- DevOps gets CI failed-job logs via **ci_logs**. Do not rely on shell `gh run view --log-failed` or browser downloads from GitHub Actions storage.

## Browser, Playwright and site

- Start the project app through **start_site** using a role authorized to do so. Never use a background shell web server to impersonate a Site.
- Browser interaction/E2E is **QA-only** through **browser_test**, grounded in the running project Site. For the root page, use `{"actions":[{"action":"goto","path":""}]}`: `path` must be relative with no leading slash or `..` (never use `/`). If the tool says no Site is running, request authorized `start_site` instead of guessing another URL. Evidence screenshot refs are read with **read_browser_screenshot** in the same active task. `screenshot` is view-only, not proof of a click/fill E2E test.
- Non-QA roles delegate browser checks to QA. If the QA browser tool is missing/disabled or the Site is not running, report exactly that and start/fix the prerequisite through the correct role. Never shell Playwright, a different browser provider or an external URL fallback.

## Roles and audit

- Planner plans; Researcher gathers facts and bounded Git reads where enabled; Developer edits; DevOps handles builds, releases and authorized Git; QA does interactive browser checks; Security is called only as required by review policy.
- Reuse one issue/task and an existing work_id across roles. A delegation is not a new issue. Record evidence, tool exit status and verification; mark a task Done only after its actual scope finishes.
- Every required action must be attributable to the server tool and the right role. Tool discovery and role prompts are generated from the same **switch-filtered** tool list. Do not advertise disabled tools.
- On unknown tool behavior, stop with an actionable blocker; never claim PASS from a build, prompt, or dry-run alone.
