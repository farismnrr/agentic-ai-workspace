# Using the app

Add a model, create a project, and describe what you need. The AI shows every step and asks before it changes anything.

## Pages

| Address | What it is |
| --- | --- |
| `/` | Landing page |
| `/login`, `/register` | Sign in, create an account. If you are already signed in, they take you straight to the dashboard (or to the page in `?next=`) |
| `/app` | Your workspace: home, inbox, projects, tasks and settings |

Press Ctrl/Cmd+K anywhere in the workspace to search.

## Your first task

1. Add a model under Settings, AI models (see [Models](#models)).
2. Create a project. It gets its own folder under `DATA_DIR/projects`, and the AI can only touch files in it.
   - With GitHub connected, choose "From GitHub" to start from one of your repositories. Type to search; the list scrolls. It is copied into the new project with its history.
   - Or choose "New repository" to create a repository on your GitHub account, and a project from it. It starts with a README, and you pick public or private.
   - If GitHub does not let the app create it, you are told. Create it on GitHub, share it with the app, then pick it under "From GitHub".
3. Describe what you need on Home. You can also pick a model and an effort for this task.
4. Watch the plan and the steps. The AI asks before it edits, deletes or moves files, runs commands, or starts a web app.
5. Review the result, then accept it or ask for changes.

## Reading a task

The task page has the conversation on one side and a panel on the other. On a phone or tablet, the panel is a row of tabs.

| Tab | Shows |
| --- | --- |
| Plan | The steps the AI intends to take, and where it is |
| Proof | What changed, how it was tested, and what was checked |
| Changes | Each file the task changed, as a diff, with every step and its Undo |
| Files | The whole project: open any file, edit it by hand and save |

About Changes:

- **One diff per file,** against how the task found it, even if the AI edited the file many times.
- **Green** lines were added and **red** lines removed; each file shows its `+` and `−` counts.
- **Unified** shows one column. **Split** shows old and new side by side. A wide panel starts in Split.
- **It updates as the AI works.** Steps, with Undo, is listed under the files.

About Files:

- **The project page** has the same Files tab, so you can read the code without starting a task.
- **Live:** when the AI or a command changes a file, the tree and the open file update at once.
- **Your edits are safe.** If the AI changes a file you are editing, you choose: keep yours or see the new one.
- **Delete a folder:** hover a folder and press the bin. It removes everything inside, and it cannot be undone.
- **Hidden:** secrets such as `.env` files and keys never show.
- **Save** with the button or Ctrl/Cmd+S.

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
- The lead never changes anything itself, even a one-line fix. It reads, plans, asks you and hands every change to a teammate.
- A teammate you did not need is simply never called.
- Teammates do not chat with each other. The lead passes on what each one needs, so nobody works from a different picture.

The lead sizes your request first and tells you:

| Size | What happens |
| --- | --- |
| Question | The lead answers. No plan, no team. |
| Small | A quick change to something that exists, such as a typo or a title. The lead hands it straight to the Developer, with no Planner and no plan list. |
| Medium | Anything new, such as a page or a script, or three or more files. The Planner asks you a few questions and writes a short PRD, then the Developer builds and QA checks. |
| Large | The Planner plans the same way. Research, requirements and design are added only if needed. A Designer shapes anything you will see. A Reviewer looks it over. |

Research always comes first on a medium or large task, and again after the PRD for the stack and the look. Facts about the real world matter most: a business, a market, a style or a technology. The Researcher looks it up on the web and writes short notes with sources.

You can read them under the Plan tab, in Research notes. Later teammates build from them.

For a medium or large task:

1. The Planner asks only what blocks the scope, such as who it is for or what to leave out.
   - It shows one short form. Pick one option, tick several, or type your own answer, then send.
   - For the look of a page, each choice comes with a picture. The one it suggests comes first.
   - At most three questions for medium, five for large.
2. It writes a short PRD: overview, goals, what is not included, acceptance criteria, requirements and a todo. It shows in the thread and under the Plan tab.
3. The lead asks once whether to start building. Say what to change and the Planner revises it.
4. If you will see a screen or page, the Designer sets the look and builds it, and writes it down in `DESIGN.md` for the Developer.
5. Every teammate is shown the PRD with its job.
6. A screen ends with a design review by the Reviewer. Its list of fixes goes back to the Designer, at most two rounds.
7. The Designer and the Reviewer look at your running app through screenshots, on a desktop and a phone screen. You see each screenshot in the thread.
8. Work that includes a web page is never reported until its app is running. You get a link to open it.

When unsure, the lead picks the smaller size. There is no cap on how many jobs a team may hand out, as long as they keep getting somewhere.

| Role | What it does |
| --- | --- |
| Orchestrator (lead) | Plans, hands out work, answers you |
| Planner | Asks you what it needs in a short form, then writes the PRD. It follows a built-in playbook (its skill) that ships with the app |
| Designer | Sets the visual direction, writes `DESIGN.md` and builds the interface. It has built-in skills (design brief, frontend design, UI polish) and a library of craft rules, style families and page templates, and picks what fits the job |
| Researcher | Reads the project and looks things up on the web. It writes notes with sources before the PRD, the design or the build. It cannot change files |
| Media | Finds pictures you may use and edits them: resize, crop, convert, remove a background. Every picture it saves has its licence and credit written in `assets/CREDITS.md`, never on the page. It lists pictures that need no credit first |
| Business Analyst | Turns your request into requirements |
| Architect | Designs the change |
| Developer | Writes the code. A new app is built with Nuxt, SQLite and the latest stable versions, with its own register and login forms when it needs accounts |
| Security | Checks an app for known vulnerabilities, risky code and missing protections. An app with code is not started until it has reported. It cannot change files |
| QA Tester | Runs the checks |
| Reviewer | Reads the finished work |
| DevOps | Builds, scripts and running the app |

Under Settings, Agent team you can change each role:

- Pick its own model and effort. Leave it empty to use your default.
- Write its personality and way of working.
- For a role you add, choose what it may do: look at files, change files, run commands. Built-in roles keep the access that fits them.
- Turn a role off, or add your own. Built-in roles cannot be removed, and what you add is yours alone.
- Reset to built-in restores the default team, including roles added in later versions.

Good to know:

- Every step shows who did it, including the lead's own, with a card when work changes hands.
- The Team tab lists each job: who has it, which model, what was asked, how many steps, files and checks, and the report.
- The Proof tab names who ran each check.
- A teammate still asks before changing anything, with the same Allow button.
- Proof counts the checks any teammate ran after the last change.
- A team task may run 400 model turns and 2 hours, shared by everyone on it. A teammate may take 60 turns per job.
- If a limit is reached, the task pauses with Continue. Nothing is lost: the work stays and the team carries on.
- If six jobs in a row change nothing, run no check and write nothing, the lead is told the team is stuck and must change its approach or tell you what blocks it.

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
