# Implementation notes

Detail for agents working on the backend. People read the short versions in `docs/system-design.md`; this file keeps the long, exact ones. Update it in the same task when the code changes.

## Spaces and the coding MCP endpoint

- `modules/space` and `modules/coding`: each space is an MCP endpoint at `/api/<name>`. The first, `coding`, offers tools to list, read, search, change files in, run tasks and commands in, and start the web app of the caller's project for that app.
- Every call resolves the project from the verified token (user id plus the app's name from its grant). `projectapp.ForApp` finds or makes the live project whose `source` is that name, so one person can never name another's files and two apps never share a folder.
- The tool catalogue (`coding/domain.Tools()`) and the switches (`tool_permissions`, `approval_settings`) are the same ones the agent obeys. Paths are confined with Go's `os.Root`, and secret files are never shown.
- Each tool has a scope (`coding:read`, `coding:write`, `coding:exec`) and a switch the person controls. A call must pass the switch, then the token's scope, and is always written to the activity log.
- Edits save the old content first (under `DATA_DIR/backups`); that is what Undo restores.
- Tasks are named commands from `.workspace/tasks.json`, run without a shell, one at a time. The tools can never read or write that folder.

## OAuth discovery from `CONNECTOR_URL/api/coding`

1. A call without a token gets `401` and a pointer to `/.well-known/oauth-protected-resource/api/coding`.
2. That document names the authorization server and the permissions (`coding:read`).
3. `/.well-known/oauth-authorization-server` (and the OpenID Connect twin) lists the endpoints, PKCE `S256`, and the two ways to register: Client ID Metadata Documents and dynamic registration.
4. The app sends the person to `/oauth/authorize`, which opens the consent page in the app. After approval the app exchanges the code at `/oauth/token`.

Tokens are opaque and stored hashed, tied to one space, with rotating refresh tokens. A replayed code or refresh token revokes the whole connection.

## The sandbox

Tasks and commands run inside `platform/sandbox`, which starts one throwaway Docker container per run through the Docker Engine API (the `moby/moby/client` module, so the binary stays static and needs no Docker CLI).

**Image.** `SANDBOX_IMAGE` (default `ghcr.io/<owner>/agentic-ai-workspace-sandbox`, built from `sandbox/Dockerfile`: Python, Node.js, Go, Rust, Java, Maven, the GitHub CLI and Bash, pulled on first use). Toolchains live in the image, so every container shares the same read-only layers and nothing is downloaded per person. The image advertises them in the `io.masihawam.toolchains` label, which the agent prompt reads.

**Container.**
- no network interface, read-only root filesystem, private `/tmp` tmpfs, all capabilities dropped, `no-new-privileges`, Docker's default seccomp profile, an init process;
- `memory` (swap off), `cpu`, `pids` and file-size limits;
- runs as the server's own uid and gid, so files stay editable;
- `SANDBOX_RUNTIME` can select a hardened runtime such as gVisor.

**Mounts.** The person's workspace at `/workspace`, their package cache at `/cache` (`DATA_DIR/caches/<id>`, emptied above 5 GiB) and the proxy socket folder read-only at `/run/egress`. They are bind mounts when the server runs on the host, or the data volume narrowed with a volume `Subpath` (Engine API 1.45+) when the server itself runs in a container, so a run can never see another person's folder.

**Network.** Registry access goes through `platform/egressproxy`: an HTTPS `CONNECT` proxy on one Unix socket per workspace (the server still listens on one TCP port). It allows only `SANDBOX_EGRESS_HOSTS` on port 443, with every resolved address checked by `platform/egress` at dial time. Inside the container `sandbox-entry` bridges `127.0.0.1:3128` to that socket and the usual `HTTPS_PROXY` variables point there. `NO_PROXY` lists loopback, so a command can `curl` a server it started on `127.0.0.1`; without it the registry-only proxy answered 403 and the agent could not check its own site.

**Commands.** `run_command` (exec scope, off by default, asks every time) runs `bash -c` in the sandbox. Named tasks run their argument vector directly.

**Lifecycle.**
- Containers carry a label with an owner id derived from the data folder; at startup the server removes only leftovers with its own owner id.
- Docker is opened lazily and warmed in the background. If the daemon is unreachable, checks fail with a plain reason and nothing else breaks.
- A shared pool (`MAX_SANDBOXES`) bounds how many containers run at once. A watcher stops a run whose workspace grows past 2 GiB.

**Not covered.** Disk quotas (the watcher polls once a second, so use filesystem quotas for a hard limit); data smuggled out inside requests to an allowed registry (it has no credentials, but a request can still carry bytes); and the host kernel and Docker daemon, which a container always shares with the host. Whoever controls the Docker socket controls the host, so Docker rootless mode is the recommended mitigation.

Web apps the AI starts (`modules/site`) use the same sandbox with a long-lived container, reached through a Unix socket by a `/run/<name>/<uuid>/` proxy on the same port. See `docs/sites.md` for the isolation layers. The app's port is exposed as a Unix socket (`socat`, started by `sandbox-entry` when `SITE_PORT` is set) in `DATA_DIR/sites`, mounted at `/run/site`. Sites live in the `sites` table and reset at startup, because containers do not survive a restart.

## The agent loop

- There is no separate chat: every message starts a thread in a project (`modules/task`), and the agent decides whether to answer directly or to plan and work.
- Reads run on their own. Edits, deletes, moves, commands and starting a web app wait for a yes unless the person turned on Skip approvals for the space, which never overrides an off switch.
- A tool switched off in Spaces & tools is neither offered to the model nor run (checked again right before each call).
- A run that changed nothing ends as an answer (status Done, no receipt) and the thread stays open.
- The loop sends the conversation to the chosen model, runs tools in the project folder, and records plain-language steps and a provider-neutral transcript. Because the transcript is stored, a task can pause for approval and resume later.
- Proof is built only from checks the server ran after the last change. "How it was tested" only counts them ("Ran 3 checks: 2 passed, 1 failed"); each command is named under "What was checked". Claims the AI makes without running anything are listed as claimed.
- A project's checks live in `.workspace/tasks.json`; only the Checks tab changes them.

## The agent team

`modules/agentteam` holds a person's roles; `modules/task` runs them. Built on the loop above, not beside it.

- **Config.** `agent_roles` (one row per role, keyed `(user_id, key)`) and `agent_teams.use_by_default`. No saved team means the built-in SDLC presets (`agentteam/domain/preset.go`); saving replaces the whole team; reset deletes it. Exactly one lead, at most 12 roles. `access` is a list of groups (`read` always, plus `write`, `exec`), turned into tool names in `bootstrap/agentteam.go`; the lead is not limited by it. Built-in roles are the same for everyone and cannot be removed: `Team.WithBuiltIns` adds any missing one back on save and on read (so teams saved before a new built-in role existed get it), and `Role.normalize` forces their preset access. Only the orchestrator leads; a role the person adds never does. Roles live per person (`agent_roles.user_id`), so one person's edits never reach another. `MaxRoles` is 20, counting the 9 built-in.
- **Per task.** `tasks.team` is set at creation (`CreateInput.Team`, else `use_by_default`). A team task with no enabled teammate runs solo. With a team the lead's model is the composer's pick, else the lead role's, else the default.
- **The lead never changes anything.** On a team task the lead's runner (`Runner.leads`) is not offered any tool whose risk is not none (edit, write, delete, move, run a task or command, start or stop a site) and refuses it if called (`refusal`). A first test with a real model showed prompts alone are ignored: the lead said "medium, I will ask the Planner" and then wrote every file itself. Reads stay. A small job goes straight to the Developer. If no teammate can edit files, the lead is told to ask the person to turn on the Developer (`canChange`). Solo tasks and teammates are unchanged. If a real model still skips the Planner on a medium job, the next guard is to refuse delegating to a role that can change files until a PRD exists.
- **Delegation.** The lead gets a `delegate{role, brief, context}` tool and runs the teammate inside the same run (`delegate.go`): a second `Runner` (`teamrunner.go`) with its own transcript, a tool gate narrowed to the role, and every event labelled `Agent`. One level only: teammates get `report`, never `delegate`, `update_plan` or `finish`, and `ask_user` only with the `ask` access group. Hallucinated loop tools are refused in `handle`.
- **Transcript scope.** A teammate's blocks live in `task_transcript.scope` (the delegate call id; empty is the lead). `LoadTranscript` reads scope `''` only. `seq` stays shared per task. Compaction and the context meter follow the lead (`saveContext` skips teammates).
- **Pause and resume.** A teammate that needs an approval returns `stopWaiting` through `delegate`; the lead's `delegate` call stays unanswered. After the answer the task is requeued and the same teammate continues from its own transcript. `task_delegations` stores the model chosen at the start, so a resume never changes model. A finished delegation is idempotent: the stored report is replayed as the tool result.
- **Model for a role.** The role's provider and model if the person can reach it, else `Models.Default`, else the task's model. Usage is recorded against that model.
- **Limits.** One `budget` per run, drawn by the lead and every teammate (`TeamSteps` 400, `TeamWallTime` 2 hours). Each delegation also gets `DelegateSteps` 60 per segment; running out fails only that delegation (the lead reads an error result). Running out of the shared budget ends the run with the usual Continue hint.
- **Report and proof.** The result of `delegate` is the teammate's report plus what the server recorded for them (files changed, checks run). Their events sit in the same thread, so `buildProof` and `workDone` count them like any others; their claims never count.
- **Sizing and plans.** Research behind it: Anthropic's multi-agent write-up (sub-agents report only to the lead, effort is scaled by rules in the prompt, 50 sub-agents for a simple query was a real failure, coding is a poor fit, about 15x the tokens), Cognition's "Don't build multi-agents" (share context, parallel agents make conflicting implicit decisions) and Anthropic's "Building effective agents" (simplest thing that works). So: no free chat between teammates, the lead coordinates. `leadRules` (`system.go`) makes it size the request first: question, small (alone, no plan list), medium (Planner, Developer, QA), large (add Researcher/BA/Architect only for what is unknown, then Reviewer); when unsure take the smaller size, but making something new is never small (a first test let the lead call a four-file site "small"), and medium or large work ends with a check run by QA.
- **PRD and questions.** Research: Claude Code's AskUserQuestion (one decision per question, two to four options with the recommended one first, always a free-text "Other"), Codex plan mode's `request_user_input`, Kiro specs (requirements with checkable acceptance criteria) and agent-PRD templates (goals and explicit non-goals matter most). ngodingpakeai.com is only a marketing page, nothing to learn from its flow. The Planner (and Business Analyst) have the `ask` access group, which adds `ask_user` to the teammate's tools (`canAsk`); the question is a normal `EventQuestion` labelled with the teammate, and answering resumes the same teammate like an approval does. A teammate may ask at most 6 questions per task (`askedEnough`, counted per question, not per call). The Planner writes the PRD as the `document` of its `report`; `deliver` saves it as `tasks.prd` (migration 030), appends an `EventPRD`, returns it to the lead, and `briefWithPlan` adds it to every later brief. The lead asks the person once to approve before the build (`leadRules`). Questions are only for what blocks the scope; a sensible default goes under Assumptions.
- **Question form.** `ask_user` takes `{questions:[{header, question, type single|multi|text, options:[{label, description}]}]}`, one to four per call (`askargs.go`); options may be plain strings and the old `{question, options[]}` shape still parses. The event keeps `Questions` and also `Text`/`Options` of the first (old events and tests). The answer is `POST .../answer {answers: string[][]}` (one list of replies per question; `{choice}` still answers a one-question form), checked in `answerform.go` and stored as one summary line in `Answered`, so the transcript result stays "The person answered: ...". The person can always type their own reply, so the agent is told not to add an Other option. Web: `QuestionCard` + `QuestionField` (daisyUI radio/checkbox/input), logic in `logic/question-form.ts`.
- **Skills.** A skill is the playbook of a built-in role: `agentteam/domain/skills/<role>/SKILL.md` (frontmatter name/description, embedded with `go:embed`, `SkillFor`). It is not saved: `Role.Skill` is filled on read (`WithBuiltIns`, `normalize`) and `childPrompt` appends it after the role's prompt, saying the playbook wins. `WithBuiltIns` also forces a built-in role's current `Access` on read, because a team saved before a role gained `ask` otherwise kept `access=[read]` and its Planner never got `ask_user` (the likely reason it asked nothing). Only the Planner has a skill so far; next are the other roles, then skills the person adds.
- **Shared plan.** The lead's `update_plan` is the team's plan; each new teammate is shown it (`briefWithPlan`, read fresh from the task) with its brief. Briefs carry objective, what to hand back, files or tools, and limits. Failed checks go back to the Developer through the lead, two rounds at most.
- **Guard rails.** A team is not capped by how many jobs it hands out (a first cap of 8 stopped a real ERP build); `Limits.MaxIdleJobs` (6) refuses the next job only after that many in a row that changed no file, ran no check and wrote no document (`idleJobs`). A refused job is an error result, not a failure. The solo workflow in `prompt.go` also skips the plan for a small change; the first step that changes or runs something moves a plan-less task from planning to working (`tool()`).
- **Who did what.** On a team task every event carries `Event.Agent`: teammates by `labelledEvents`, the lead by a run-scoped copy of the Runner (`leading`; a Runner is shared by every task, never mutate it). Handoffs also carry `Event.Model`; `Check.By` names who ran a verified check. The web Team tab (`logic/team.ts`) derives jobs from the thread, so nothing extra is stored.
- **Thread.** `Event.Agent` names the teammate; `EventHandoff` (`action` assigned, reported or failed) marks work changing hands.

## Models, effort and context

**Providers.** Anthropic Messages and OpenAI Chat Completions over plain HTTP. Gemini uses Google's OpenAI-compatible endpoint (`/v1beta/openai`) with the same client. A gateway works if it runs on the same computer. Every outgoing model call goes through `platform/egress`, which checks the real IP when the connection opens: hosted providers may reach only public addresses, gateways only loopback and never this server's own port, with no redirects and no proxy variables. API keys are sealed with AES-256-GCM (`platform/secretbox`) using `DATA_DIR/secret.key`.

**GitHub.** `modules/github`: each person connects their own account through a GitHub App (web flow with PKCE; tokens sealed, renewed on demand). Commands in a project get the owner's token and, on a socket of their own, `github.com`, `api.github.com` and `codeload.github.com`; sites never do.

**Effort.**
- A task keeps the effort it was started with (`tasks.effort`; empty uses the provider's default; a person's default is `preferences.default_effort`).
- Which levels are offered comes from the provider. A gateway that describes its models (9router does) decides per model; a model it says thinks like Claude gets Low to Max, with no None or Minimal.
- A gateway's model list is read again when a page asks for models, at most every 30 seconds (`Service.SyncStale`). An unreachable gateway keeps its list. Refresh does it at once.
- The OpenAI-style client sends `reasoning_effort` (this also reaches gateways such as 9router); the Anthropic client sends `output_config.effort`. An empty level leaves the field out.
- Every model call, including the summary that shortens a long task, uses the task's effort.

**Limits and context.**
- 40 model turns and 15 minutes per run (team tasks: see The agent team). No total token budget per task. A project runs one task at a time. Interrupted tasks are marked and can be continued.
- A model's context window is the size the person chose in Settings, else the size its provider reported when the key was verified (`model_catalog`; 9router, OpenRouter and Anthropic list it in `/v1/models`), else 1M for a name ending in `[1m]`, else 200K.
- Before a turn, when the conversation nears the window (window minus 33K), the model writes a summary of the task so far. The summary replaces the earlier messages in what is sent; the full history stays on record, and the thread shows "I shortened my notes to keep going".
- The task row keeps the last measured size and the window (`context_used`, `context_window`), shown as a meter. If no summary comes back the task goes on, and after three misses in a run it stops trying.

## Data, live updates, limits

- **Your data** (`modules/mydata`): `GET /api/account/export` streams a zip built from sections that bootstrap assembles from the other features (`bootstrap/export.go`). `DELETE /api/tasks` clears a person's tasks, cancelling running ones. `POST /api/auth/delete` checks the password, then runs the erasers other features registered (tasks, then projects and their folders) before deleting the user row; the rest goes through foreign-key cascades.
- **Live updates:** task pages and the workspace follow changes over server-sent events (`GET /api/tasks/:id/stream` for steps and streamed text, `GET /api/stream` for list changes). A reconnecting client resumes from `Last-Event-ID`; the workspace falls back to a slow poll.
- **Rate limits:** sign-in, sign-up and the open OAuth endpoints are limited per visitor address; failed sign-ins also per account and address, so someone else's wrong guesses never lock you out. Expired sessions, codes, tokens and unused clients are swept every hour.
- **Migrations:** the schema changes only through numbered files in `internal/platform/database/migrations/`. Applied files are checksummed and can never be edited.

## Workspace screens (detail)

- **Sidebar:** New task, Search (Ctrl/Cmd+K), Inbox, All projects, recent tasks grouped by day with a status dot and relative time, and a collapsible Projects list. Settings and the account menu sit at the bottom. It is an icon rail on desktop and a drawer on phones.
- **Home:** one composer (a project picker that reads "Select a project" until chosen, a model and effort picker, and send), then a short "Needs you" list and what is running. There is no separate chat or approval-level choice: every message starts a task thread. The greeting is the one serif moment.
- **Inbox:** a list beside a detail pane, gathering what needs the user (a permission request, a question, a finished task to review, a task that could not finish). It is not a global task list.
- **Task:** a conversation next to a work panel. The user's message is a soft bubble; the assistant's reply is plain text. Consecutive actions fold into one expandable line; only questions, permission requests and results are cards. The work panel has Plan, Proof, Changes and Files, which become tabs on phones. Looking at files runs on its own; a step that changes a file or runs a check asks first. A question that needs no changes is answered in the thread and ends as Done, with no review step.
- **Settings:** a grouped left nav (AI, Preferences, Account) beside plain rows: label and short description on the left, control on the right, hairline dividers, no cards around single controls. Adding a provider opens a dialog using the shared `FormField` (visible label, hint in secondary text, error under the field once touched or submitted). Pages with editable values end with a sticky Save bar. On phones the nav becomes a scrolling row of tabs.
- **Status and approvals:** task status is always an icon plus a text label, never color alone. Permission requests show the exact action, a risk label and two choices (Allow, Don't allow). Project tiles pick `primary`, `secondary` or `accent` from the project id. The "Proven to work" stamp appears only for verified, passing checks. Screens never show sample data or pretend buttons. Destructive account actions ask for confirmation; deleting the account also asks for the password.
