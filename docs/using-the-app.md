# Using the app

Add a model, create a project, and describe what you need. The AI shows every step and asks before it changes anything.

## Pages

| Address | What it is |
| --- | --- |
| `/` | Landing page |
| `/login`, `/register` | Sign in, create an account |
| `/app` | Your workspace: home, inbox, projects, tasks and settings |

Press Ctrl/Cmd+K anywhere in the workspace to search.

## Your first task

1. Add a model under Settings, AI models (see [Models](#models)).
2. Create a project. It gets its own folder under `DATA_DIR/projects`, and the AI can only touch files in it.
3. Describe what you need on Home. You can also pick a model and an effort for this task.
4. Watch the plan and the steps. The AI asks before it edits, deletes or moves files, runs commands, or starts a web app.
5. Review the result, then accept it or ask for changes.

## Reading a task

The task page has the conversation on one side and a panel on the other. On a phone or tablet, the panel is a row of tabs.

| Tab | Shows |
| --- | --- |
| Plan | The steps the AI intends to take, and where it is |
| Proof | What changed, how it was tested, and what was checked |
| Changes | Every edit, each with an Undo |
| Files | The files the task touched |

About Proof:

- **"Proven to work"** appears only when a check the app ran itself passed after the last change.
- **"Verified"** marks checks the app ran. **"Claimed"** marks things the AI only said.
- **A question that needs no changes** is answered in the conversation and ends as Done, with no review step.
- **The header** shows which model and effort the task used, and how full the AI's memory is.

## Models

You can add Anthropic, OpenAI, Google Gemini (key from [Google AI Studio](https://aistudio.google.com/apikey)), or an OpenAI-compatible gateway such as 9router.

- A gateway must run on the same computer as the app (`localhost`). The app refuses any other address, so nobody can point it at your internal network.
- Keys are stored encrypted and checked right away.
- Models are grouped by provider.
- With a gateway, each provider it serves is its own group. Codex, Claude Code and OpenCode have names. Others show the gateway's short name, such as `kr`.
- A provider added in the gateway appears within about 30 seconds, or at once when you press Refresh.

Two settings shape how a model works:

| Setting | What it does |
| --- | --- |
| Effort | How hard the AI thinks before answering. Higher is slower and costs more. Default leaves it to the provider. |
| Context window | How much the model keeps in mind at once. Near the limit, the AI summarizes its notes and keeps going. |

The context window comes from what the provider reports, else 200K, or 1M for names ending in `[1m]`.

## Agent team

An agent team works like a small software company. A lead takes your request and hands each part to the right teammate, who reports back.

- Turn on **Team** in the box where you type a task, or set it for every new task under Settings, Agent team.
- Without Team, one AI does the whole task, as before.
- The lead decides who to call and when. Only the lead talks to you.
- A teammate you did not need is simply never called.
- Teammates do not chat with each other. The lead passes on what each one needs, so nobody works from a different picture.

The lead sizes your request first and tells you:

| Size | What happens |
| --- | --- |
| Question | The lead answers. No plan, no team. |
| Small | A quick change to something that exists, such as a typo or a title. The lead does it alone, with no plan list. |
| Medium | Anything new, such as a page or a script, or three or more files. The Planner writes a short plan, then the Developer builds and QA checks. |
| Large | The Planner plans. Research, requirements and design are added only if needed. A Reviewer looks it over. |

When unsure, the lead picks the smaller size. A task can hand out at most 8 jobs.

| Role | What it does |
| --- | --- |
| Orchestrator (lead) | Plans, hands out work, answers you |
| Planner | Sizes the request and writes the plan |
| Researcher | Reads the project and gathers facts |
| Business Analyst | Turns your request into requirements |
| Architect | Designs the change |
| Developer | Writes the code |
| QA Tester | Runs the checks |
| Reviewer | Reads the finished work |
| DevOps | Builds, scripts and running the app |

Under Settings, Agent team you can change each role:

- Pick its own model and effort. Leave it empty to use your default.
- Write its personality and way of working.
- For a role you add, choose what it may do: look at files, change files, run commands. Built-in roles keep the access that fits them.
- Turn a role off, remove it, or add your own.
- Reset to built-in restores the default team, including roles added in later versions.

Good to know:

- Every step shows who did it, including the lead's own, with a card when work changes hands.
- The Team tab lists each job: who has it, which model, what was asked, how many steps, files and checks, and the report.
- The Proof tab names who ran each check.
- A teammate still asks before changing anything, with the same Allow button.
- Proof counts the checks any teammate ran after the last change.
- A team task may run 100 model turns and 30 minutes, shared by everyone on it.

## Checks

A check is a command a project defines, such as `go test ./...`. The AI runs checks to prove its work.

- Add them on a project's Checks tab.
- They run on this computer with the app's rights, so add only commands you trust.
- The AI can also run any command in the sandbox once you turn on "Run a command". It asks before every run.

## Tools and approvals

Settings, Spaces & tools has one list of tools. It applies to tasks in this app and to the AI apps you connect.

- A tool that's off can't be used. The AI isn't even told it exists.
- A tool that's on is allowed, but the AI still asks before risky steps.
- "Skip approvals" lets the AI use every tool that's on without asking. It never turns a tool on.
- On a new account, only looking around is on. Changing files, running commands and starting a web app stay off until you turn them on.

Connecting ChatGPT and other AI apps is covered in [connector.md](./connector.md).

## GitHub

Under Settings, Integrations, press Connect GitHub, approve, and choose which repositories the app may reach.

After that, the AI's commands in your projects act as you:

- They can clone those repos, commit under your name and push.
- The AI pushes only when you ask.
- No one else's projects can use your connection, and web apps never can.
- Disconnect removes it at once.

If you see "not set up", the server's owner has to add a GitHub App first (see [deployment.md](./deployment.md#github)).

## Sites

When the AI builds and runs a web app, it appears under Sites and in the sidebar.

- Starting one needs "Run a command" to be on.
- You can open it, make it public, or stop it.
- Details: [sites.md](./sites.md).

## Your data

Under Settings, Account you can:

- Change your password. This signs out other devices.
- Download everything as a zip. Saved AI keys are never included.
- Delete all tasks.
- Delete your account, with its projects and files.

Settings, Notifications can alert you in the browser when a task needs you or fails.
