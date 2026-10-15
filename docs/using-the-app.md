# Using the app

## Pages

| Address | What it is |
| --- | --- |
| `/` | Landing page |
| `/login`, `/register` | Sign in, create an account |
| `/app` | Your workspace: home, inbox, projects, tasks and settings |

Press Ctrl/Cmd+K anywhere in the workspace to search. Everything works except notifications and data export, which are marked "coming soon".

## Your first task

1. Add a model under Settings, AI models (see below).
2. Create a project. It gets its own folder under `DATA_DIR/projects`, and the AI can only touch files in it.
3. Describe what you need on Home. The AI makes a plan and shows every step. It asks before it edits, deletes or moves files, runs commands, or starts a web app.
4. Review the result. You see what changed and what was checked. Every change can be undone.

"Proven to work" appears only when a check the app ran itself passed after the last change.

## Models

You can add Anthropic, OpenAI, Google Gemini (key from [Google AI Studio](https://aistudio.google.com/apikey)), or an OpenAI-compatible gateway such as 9router.

- A gateway must run on the same computer as the app (`localhost`). The app refuses any other address, so nobody can point it at your internal network.
- Keys are stored encrypted and checked right away.
- Models are grouped by provider.
- With a gateway, each provider it serves is its own group. Codex, Claude Code and OpenCode have names. Others show the gateway's short name, such as `kr`.
- A provider added in the gateway appears within about 30 seconds, or at once when you press Refresh.

Settings you can change:

| Setting | What it does |
| --- | --- |
| Effort | How hard the AI thinks before answering, from None up to Max. Higher is slower and costs more. Default leaves it to the provider. You can also set it for one task. |
| Context window | How much a model keeps in mind at once. It defaults to what the provider reports, else 200K, or 1M for names ending in `[1m]`. Near the limit, the AI summarizes its notes and keeps going. The task page shows how full it is. |

## Checks

Add checks on a project's Checks tab, such as `go test ./...`. The AI can run them to prove its work. They run on this computer with the app's rights, so add only commands you trust.

The AI can also run any command in the sandbox once you turn on "Run a command". It asks before every run.

## Tools and approvals

Settings, Spaces & tools has one list of tools. It applies to tasks in this app and to the AI apps you connect, like ChatGPT and Gemini.

- A tool that's off can't be used. The AI isn't even told it exists.
- A tool that's on is allowed, but the AI still asks before risky steps.
- "Skip approvals" lets the AI use every tool that's on without asking. It never turns a tool on.
- On a new account, only looking around is on. Changing files, running commands and starting a web app stay off until you turn them on.

Each connected AI app works in its own project, named after it (ChatGPT works in "ChatGPT"). These projects are marked "Connected app". Settings, Activity shows what each app did.

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

- change your password (this signs out other devices)
- download everything as a zip (saved AI keys are never included)
- delete all tasks
- delete your account, with its projects and files

Settings, Notifications can alert you in the browser when a task needs you or fails.
